package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// verificadorExternoPrueba ajusta el doble legado para el contrato CT170:
// la referencia criptográfica del firmante es exactamente la huella de su
// certificado, sin introducir una identidad declarada por RRHH.
type verificadorExternoPrueba struct {
	motivo       docports.MotivoVerificacionFirma
	firmanteByte byte
}

func (v verificadorExternoPrueba) VerificarMotivado(ctx context.Context, s docports.SolicitudVerificacionFirma) (docports.VerificacionFirmaMotivada, error) {
	r, err := (verificadorPrueba{motivo: v.motivo}).VerificarMotivado(ctx, s)
	if err == nil && r.Resultado.FirmanteRef != "" {
		identificador := byte('f')
		if v.firmanteByte != 0 {
			identificador = v.firmanteByte
		}
		r.Resultado.CertificadoHuellaSHA256 = strings.Repeat(string(identificador), 64)
		r.Resultado.FirmanteRef = "ref:" + r.Resultado.CertificadoHuellaSHA256
	}
	return r, err
}

type originalExternoPrueba struct {
	contenido []byte
	err       error
	alterar   func(*ports.OriginalFirmaAutorizado)
}

func (d *originalExternoPrueba) ObtenerOriginalFirma(_ context.Context, q ports.SolicitudOriginalFirma) (ports.OriginalFirmaAutorizado, error) {
	if d.err != nil {
		return ports.OriginalFirmaAutorizado{}, d.err
	}
	r := ports.OriginalFirmaAutorizado{Solicitud: q, Contenido: bytes.Clone(d.contenido), HuellaSHA256: huella(d.contenido)}
	if d.alterar != nil {
		d.alterar(&r)
	}
	return r, nil
}

type competenciaExternaPrueba struct {
	err     error
	alterar func(*ports.EvidenciaCompetenciaFirmante)
	vistas  []ports.SolicitudCompetenciaFirmante
}

func (d *competenciaExternaPrueba) AcreditarCompetenciaFirmante(_ context.Context, q ports.SolicitudCompetenciaFirmante) (ports.EvidenciaCompetenciaFirmante, error) {
	d.vistas = append(d.vistas, q)
	if d.err != nil {
		return ports.EvidenciaCompetenciaFirmante{}, d.err
	}
	firmante := "per_firmante_sintetico_001"
	if q.PasoOrden == 2 {
		firmante = "per_jefatura_sintetica_002"
	}
	e := ports.EvidenciaCompetenciaFirmante{
		Solicitud: q, Vigente: true, FirmantePrincipalRef: firmante,
		PerfilFirmanteRef: q.PerfilFirmanteRef, CargoFirmante: q.CargoFirmante,
		UnidadFirmanteRef: "unidad:rrhh", PuestoFirmanteRef: "puesto:firma",
		AmbitoFirmanteRef: "ambito:provincial", AsignacionFirmanteRef: "asignacion:firma:001",
		AsignacionFirmanteVersion: 4, AsignacionFirmanteHuella: strings.Repeat("b", 64),
		AsignacionVigenteDesde: "2026-09-01T00:00:00Z", AsignacionVigenteHasta: "2026-12-01T00:00:00Z",
		CompetenciaComprobadaEn: "2026-10-02T10:00:00Z", ActoCompetenciaRef: "acto:competencia:001",
	}
	if d.alterar != nil {
		d.alterar(&e)
	}
	return e, nil
}

type autorizadorExternoPrueba struct{ vistas []ports.MaterialFirmaExterna }

func (d *autorizadorExternoPrueba) AutorizarRegistroFirmaExterna(_ context.Context, m ports.MaterialFirmaExterna) (ports.CapacidadFirmaExterna, error) {
	d.vistas = append(d.vistas, m)
	r, err := RecursoFirmaExterna(m)
	if err != nil {
		return ports.CapacidadFirmaExterna{}, err
	}
	contexto, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.CapacidadFirmaExterna{}, err
	}
	hasta := instanteFirmaPrueba.Add(time.Minute)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision-firma-externa-001", strings.Repeat("a", 64), strings.Repeat("a", 64), "contexto-firma-externa-001", strings.Repeat("a", 64),
		ports.AccionRegistrarFirmaExterna, r.Referencia, contexto, ports.AudienciaFirmaExternaV3, hasta.Add(-5*time.Second), hasta,
	)
	if err != nil {
		return ports.CapacidadFirmaExterna{}, err
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{8}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		return ports.CapacidadFirmaExterna{}, err
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte{6}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte("payload"), []byte("cose"), []byte("evidencia"), raiz,
	)
	if err != nil {
		return ports.CapacidadFirmaExterna{}, err
	}
	return ports.TransportarMaterialFirmaExterna(material), nil
}

type registroExternoPrueba struct {
	firmas     []ports.FirmaRegistrada
	registrado []ports.MaterialFirmaExterna
}

func (d *registroExternoPrueba) ConsultarFirmas(_ context.Context, _, _ string) ([]ports.FirmaRegistrada, error) {
	return append([]ports.FirmaRegistrada(nil), d.firmas...), nil
}

func (d *registroExternoPrueba) RegistrarFirmaExterna(_ context.Context, m ports.MaterialFirmaExterna, c ports.CapacidadFirmaExterna) (ports.ReciboFirmaDocumento, error) {
	if err := ValidarCapacidadFirmaExterna(c, m); err != nil {
		return ports.ReciboFirmaDocumento{}, err
	}
	h, _ := m.HuellaSHA256()
	for _, previo := range d.registrado {
		if previo.ClaveIdempotencia == m.ClaveIdempotencia {
			hPrevio, _ := previo.HuellaSHA256()
			if hPrevio != h {
				return ports.ReciboFirmaDocumento{}, ports.ErrClaveFirmaDocumentoUsada
			}
			return reciboFirmaExternaPrueba(m, h, true), nil
		}
	}
	d.registrado = append(d.registrado, m)
	d.firmas = append(d.firmas, ports.FirmaRegistrada{Via: m.Via, Documento: m.Documento, Secuencia: m.Secuencia,
		ExpedienteVersion: m.VersionExpediente, CatalogoHuella: m.CatalogoHuella, PasoOrden: m.PasoOrden,
		Resultado: domain.ResultadoFirmaFirmado, OriginalHuella: m.OriginalHuella, FirmadoHuella: m.FirmadoHuella,
		ClaveIdempotencia: m.ClaveIdempotencia, DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion})
	return reciboFirmaExternaPrueba(m, h, false), nil
}

func reciboFirmaExternaPrueba(m ports.MaterialFirmaExterna, huellaSolicitud string, repetido bool) ports.ReciboFirmaDocumento {
	return ports.ReciboFirmaDocumento{FirmaRef: "firma:externa:001", ReciboRef: "recibo:firma:externa:001", Secuencia: m.Secuencia,
		Resultado: domain.ResultadoFirmaFirmado, ExpedienteVersion: m.VersionExpediente, ActorRef: "per_registrador_rrhh_001",
		PerfilRef: "perfil:rrhh", RegistradaEn: instanteFirmaPrueba, SolicitudHuella: huellaSolicitud, YaRegistrada: repetido,
		DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion}
}

func servicioFirmaExternaPrueba(t *testing.T) (*ServicioFirmaExterna, *registroExternoPrueba, *originalExternoPrueba, *competenciaExternaPrueba, *autorizadorExternoPrueba, *custodioPrueba) {
	t.Helper()
	original := &originalExternoPrueba{contenido: []byte("%PDF-1.7 original sintetico")}
	registroBase := &registroFirmaPrueba{}
	base, err := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, registroBase, &autorizadorFirmaPrueba{}, verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada})
	if err != nil {
		t.Fatal(err)
	}
	custodio := &custodioPrueba{}
	if err := base.ComponerOriginalAutorizado(original); err != nil {
		t.Fatal(err)
	}
	if err := base.ComponerCustodia(custodio, map[string]string{"informe_definitivo": tipoCustodiaPrueba}); err != nil {
		t.Fatal(err)
	}
	registro, competencia, autorizador := &registroExternoPrueba{}, &competenciaExternaPrueba{}, &autorizadorExternoPrueba{}
	s, err := NuevoServicioFirmaExterna(base, registro, autorizador, competencia)
	if err != nil {
		t.Fatal(err)
	}
	return s, registro, original, competencia, autorizador, custodio
}

func solicitudFirmaExternaPrueba(original []byte, clave string) SolicitudFirmaExterna {
	firmado := append(bytes.Clone(original), []byte(" firma-pades-sintetica")...)
	return SolicitudFirmaExterna{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "informe_definitivo", PasoOrden: 1, OriginalRef: "ref:" + strings.Repeat("1", 64), OriginalVersion: 7, PDFFirmado: firmado,
		ReferenciaPortafirmasDeclarada: "PF-SINTETICO-001", FechaPortafirmasDeclarada: "2026-10-02T10:00:00Z", ClaveIdempotencia: clave}
}

func TestFirmaExternaRegistraSoloPDFVerificadoConOriginalYCompetencia(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	sol := solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-0001")
	r, err := s.Registrar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	if len(registro.registrado) != 1 || len(competencia.vistas) != 1 || len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 ||
		r.Material.FirmantePrincipalRef != "per_firmante_sintetico_001" || r.Recibo.ActorRef == r.Material.FirmantePrincipalRef ||
		r.Custodiado.HuellaSHA256 != huella(sol.PDFFirmado) || r.MotivoVerificacion != docports.MotivoFirmaVerificada {
		t.Fatalf("registro externo incompleto: %+v, registros=%d competencia=%d autorizacion=%d custodia=%d", r, len(registro.registrado), len(competencia.vistas), len(autorizador.vistas), len(custodio.ordenes))
	}
}

func TestFirmaExternaRechazaPDFNoVerificadoAntesDeCompetenciaYEfectos(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoIntegridadNoValida}
	_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-0002"))
	if !errors.Is(err, ErrFirmaNoVerificada) || len(registro.registrado) != 0 || len(competencia.vistas) != 0 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("PDF no verificado produjo efecto: %v", err)
	}
}

func TestFirmaExternaDosPasosConservaOriginalYSeparaFirmantes(t *testing.T) {
	s, registro, original, competencia, _, custodio := servicioFirmaExternaPrueba(t)
	primera := solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-paso-01")
	primera.PDFFirmado = append(bytes.Clone(original.contenido), []byte(" firma-tecnico-sintetica")...)
	r1, err := s.Registrar(context.Background(), primera)
	if err != nil {
		t.Fatal(err)
	}
	s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada, firmanteByte: 'd'}
	segunda := solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-paso-02")
	segunda.PasoOrden = 2
	segunda.PDFFirmado = append(bytes.Clone(original.contenido), []byte(" firma-jefatura-sintetica")...)
	r2, err := s.Registrar(context.Background(), segunda)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Material.OriginalRef != r2.Material.OriginalRef || r1.Material.OriginalVersion != r2.Material.OriginalVersion ||
		r1.Material.OriginalHuella != r2.Material.OriginalHuella || r1.Material.FirmadoHuella == r2.Material.FirmadoHuella ||
		r1.Material.FirmanteRef == r2.Material.FirmanteRef || r1.Material.FirmantePrincipalRef == r2.Material.FirmantePrincipalRef ||
		r1.Material.CargoFirmante != "Técnico" || r2.Material.CargoFirmante != "Jefatura" || len(registro.registrado) != 2 || len(competencia.vistas) != 2 || len(custodio.ordenes) != 2 {
		t.Fatalf("ronda externa no conserva/separa los datos exigidos: p1=%+v p2=%+v", r1.Material, r2.Material)
	}
}

func TestFirmaExternaRechazaDictamenMultifirmaIndeterminadoSinEfectos(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoFirmanteNoIdentificado}
	_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-multifirma"))
	if !errors.Is(err, ErrFirmaNoVerificada) || len(registro.registrado) != 0 || len(competencia.vistas) != 0 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("dictamen multifirma indeterminado produjo efecto: %v", err)
	}
}

func TestFirmaExternaRechazaOriginalAjenoOFuenteCaida(t *testing.T) {
	for nombre, preparar := range map[string]func(*originalExternoPrueba){
		"fuente caida": func(d *originalExternoPrueba) { d.err = errors.New("caida sintetica") },
		"huella alterada": func(d *originalExternoPrueba) {
			d.alterar = func(o *ports.OriginalFirmaAutorizado) { o.HuellaSHA256 = strings.Repeat("0", 64) }
		},
		"solicitud ajena": func(d *originalExternoPrueba) {
			d.alterar = func(o *ports.OriginalFirmaAutorizado) { o.Solicitud.OriginalRef = "documento:ajeno" }
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
			preparar(original)
			_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-0003"))
			if !(errors.Is(err, ports.ErrFuenteOriginalFirmaNoDisponible) || errors.Is(err, ports.ErrOriginalFirmaNoAutorizado)) || len(registro.registrado) != 0 || len(competencia.vistas) != 0 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("original no seguro produjo efecto: %v", err)
			}
		})
	}
}

func TestFirmaExternaRechazaFirmanteOCargoNoAcreditados(t *testing.T) {
	for nombre, alterar := range map[string]func(*ports.EvidenciaCompetenciaFirmante){
		"cargo distinto":  func(e *ports.EvidenciaCompetenciaFirmante) { e.CargoFirmante = "Otro cargo" },
		"perfil distinto": func(e *ports.EvidenciaCompetenciaFirmante) { e.PerfilFirmanteRef = "perfil:ajeno" },
		"no vigente":      func(e *ports.EvidenciaCompetenciaFirmante) { e.Vigente = false },
	} {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
			competencia.alterar = alterar
			_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-0004"))
			if !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) || len(registro.registrado) != 0 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("competencia no acreditada produjo efecto: %v", err)
			}
		})
	}
}

func TestFirmaExternaRechazaFuenteCompetenciaCaidaSinEfectos(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	competencia.err = errors.New("fuente de competencia caida sintetica")
	_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-0006"))
	if !errors.Is(err, ports.ErrCompetenciaFirmanteNoDisponible) || len(registro.registrado) != 0 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("fuente de competencia caida produjo efecto: %v", err)
	}
}

func TestFirmaExternaReintentoIgualRecuperaYMaterialDistintoFalla(t *testing.T) {
	s, registro, original, _, _, custodio := servicioFirmaExternaPrueba(t)
	sol := solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-0005")
	primero, err := s.Registrar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	reintento, err := s.Registrar(context.Background(), sol)
	if err != nil || !reintento.Recibo.YaRegistrada || reintento.Recibo.Secuencia != primero.Recibo.Secuencia || len(registro.registrado) != 1 || len(custodio.ordenes) != 2 {
		t.Fatalf("reintento no recuperado: %+v, %v", reintento, err)
	}
	for nombre, alterar := range map[string]func(*SolicitudFirmaExterna){
		"pdf":                    func(s *SolicitudFirmaExterna) { s.PDFFirmado = append(bytes.Clone(s.PDFFirmado), 'x') },
		"referencia Portafirmas": func(s *SolicitudFirmaExterna) { s.ReferenciaPortafirmasDeclarada = "PF-SINTETICO-OTRA" },
		"fecha Portafirmas":      func(s *SolicitudFirmaExterna) { s.FechaPortafirmasDeclarada = "2026-10-02T11:00:00Z" },
		"version original":       func(s *SolicitudFirmaExterna) { s.OriginalVersion++ },
		"paso":                   func(s *SolicitudFirmaExterna) { s.PasoOrden = 2 },
	} {
		t.Run(nombre, func(t *testing.T) {
			otro := sol
			alterar(&otro)
			if _, err := s.Registrar(context.Background(), otro); !errors.Is(err, ports.ErrClaveFirmaDocumentoUsada) || len(registro.registrado) != 1 {
				t.Fatalf("material distinto con misma clave admitido: %v", err)
			}
		})
	}
}
