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

type autorizadorVecPrueba struct {
	certificadoCanal  string
	sinCanal          bool
	capacidadInvalida bool
	audiencia         string
	vistas            []ports.MaterialFirmaVec
}

func (d *autorizadorVecPrueba) AutorizarFirmaVec(_ context.Context, m ports.MaterialFirmaVec) (ports.CapacidadFirmaVec, error) {
	d.vistas = append(d.vistas, m)
	if d.sinCanal || d.certificadoCanal != m.CertificadoHuella {
		return ports.CapacidadFirmaVec{}, errors.New("canal sellado no acredita el certificado")
	}
	if d.capacidadInvalida {
		return ports.CapacidadFirmaVec{}, nil
	}
	r, err := RecursoFirmaVec(m)
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	contexto, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	hasta := instanteFirmaPrueba.Add(time.Minute)
	audiencia := ports.AudienciaFirmaVecV3
	if d.audiencia != "" {
		audiencia = d.audiencia
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision-firma-vec-001", strings.Repeat("a", 64), strings.Repeat("a", 64), "contexto-firma-vec-001", strings.Repeat("a", 64),
		ports.AccionRegistrarFirmaVec, r.Referencia, contexto, audiencia, hasta.Add(-5*time.Second), hasta,
	)
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte{7}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte("payload"), []byte("cose"), []byte("evidencia"), raiz,
	)
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	return ports.TransportarMaterialFirmaVec(material), nil
}

type registroVecPrueba struct {
	firmas     []ports.FirmaRegistrada
	registrado []ports.MaterialFirmaVec
}

func (d *registroVecPrueba) ConsultarFirmas(_ context.Context, _, _ string) ([]ports.FirmaRegistrada, error) {
	return append([]ports.FirmaRegistrada(nil), d.firmas...), nil
}

func (d *registroVecPrueba) RegistrarFirmaVec(_ context.Context, m ports.MaterialFirmaVec, c ports.CapacidadFirmaVec) (ports.ReciboFirmaDocumento, error) {
	if err := ValidarCapacidadFirmaVec(c, m); err != nil {
		return ports.ReciboFirmaDocumento{}, err
	}
	h, _ := m.HuellaSHA256()
	for _, previo := range d.registrado {
		if previo.ClaveIdempotencia == m.ClaveIdempotencia {
			hPrevio, _ := previo.HuellaSHA256()
			if hPrevio != h {
				return ports.ReciboFirmaDocumento{}, ports.ErrClaveFirmaDocumentoUsada
			}
			return reciboFirmaVecPrueba(m, h, true), nil
		}
	}
	d.registrado = append(d.registrado, m)
	d.firmas = append(d.firmas, ports.FirmaRegistrada{Via: m.Via, Documento: m.Documento, Secuencia: m.Secuencia,
		ExpedienteVersion: m.VersionExpediente, CatalogoHuella: m.CatalogoHuella, PasoOrden: m.PasoOrden,
		Resultado: domain.ResultadoFirmaFirmado, OriginalHuella: m.OriginalHuella, FirmadoHuella: m.FirmadoHuella,
		ClaveIdempotencia: m.ClaveIdempotencia, DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion})
	return reciboFirmaVecPrueba(m, h, false), nil
}

func reciboFirmaVecPrueba(m ports.MaterialFirmaVec, huellaSolicitud string, repetido bool) ports.ReciboFirmaDocumento {
	return ports.ReciboFirmaDocumento{FirmaRef: "firma:vec:001", ReciboRef: "recibo:firma:vec:001", Secuencia: m.Secuencia,
		Resultado: domain.ResultadoFirmaFirmado, ExpedienteVersion: m.VersionExpediente, ActorRef: m.FirmantePrincipalRef,
		PerfilRef: m.PerfilFirmanteRef, RegistradaEn: instanteFirmaPrueba, SolicitudHuella: huellaSolicitud, YaRegistrada: repetido,
		DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion}
}

func servicioFirmaVecPrueba(t *testing.T) (*ServicioFirmaVec, *registroVecPrueba, *originalExternoPrueba, *competenciaExternaPrueba, *autorizadorVecPrueba, *custodioPrueba) {
	t.Helper()
	original := &originalExternoPrueba{contenido: []byte("%PDF-1.7 original vec sintetico")}
	base, err := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}, verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada})
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
	registro, competencia := &registroVecPrueba{}, &competenciaExternaPrueba{}
	autorizador := &autorizadorVecPrueba{certificadoCanal: strings.Repeat("f", 64)}
	s, err := NuevoServicioFirmaVec(base, registro, autorizador, competencia)
	if err != nil {
		t.Fatal(err)
	}
	return s, registro, original, competencia, autorizador, custodio
}

func solicitudFirmaVecPrueba(original []byte, clave string) SolicitudFirmaVec {
	return SolicitudFirmaVec{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "informe_definitivo", PasoOrden: 1, OriginalRef: "ref:" + strings.Repeat("2", 64), OriginalVersion: 7,
		PDFFirmado: append(bytes.Clone(original), []byte(" firma-vec-pades-sintetica")...), ClaveIdempotencia: clave}
}

func TestFirmaVecRegistraR5ConCertificadoCanalSellado(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
	sol := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-000001")
	r, err := s.Firmar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	if len(registro.registrado) != 1 || len(competencia.vistas) != 1 || len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 ||
		r.Material.Via != ports.ViaFirmaCertificadoVEC || r.Material.CertificadoHuella != autorizador.certificadoCanal ||
		r.Recibo.ActorRef != r.Material.FirmantePrincipalRef || r.Custodiado.HuellaSHA256 != huella(sol.PDFFirmado) {
		t.Fatalf("registro R5 incompleto: %+v", r)
	}
}

func TestFirmaVecDosPasosConservanOriginalYSeparanFirmantes(t *testing.T) {
	s, registro, original, competencia, _, custodio := servicioFirmaVecPrueba(t)
	primera := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-paso-001")
	primera.PDFFirmado = append(bytes.Clone(original.contenido), []byte(" visto-bueno-pades-sintetico")...)
	r1, err := s.Firmar(context.Background(), primera)
	if err != nil {
		t.Fatal(err)
	}

	s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada, firmanteByte: 'd'}
	segunda := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-paso-002")
	segunda.PasoOrden = 2
	segunda.PDFFirmado = append(bytes.Clone(original.contenido), []byte(" resolucion-pades-sintetica")...)
	autorizador := s.autorizador.(*autorizadorVecPrueba)
	autorizador.certificadoCanal = strings.Repeat("d", 64)
	r2, err := s.Firmar(context.Background(), segunda)
	if err != nil {
		t.Fatal(err)
	}

	if r1.Material.OriginalRef != r2.Material.OriginalRef ||
		r1.Material.OriginalVersion != r2.Material.OriginalVersion ||
		r1.Material.OriginalHuella != r2.Material.OriginalHuella ||
		r1.Material.FirmadoHuella == r2.Material.FirmadoHuella ||
		r1.Material.FirmanteRef == r2.Material.FirmanteRef ||
		r1.Material.FirmantePrincipalRef == r2.Material.FirmantePrincipalRef ||
		r1.Material.CargoFirmante != "Técnico" || r2.Material.CargoFirmante != "Jefatura" ||
		len(registro.registrado) != 2 || len(competencia.vistas) != 2 || len(custodio.ordenes) != 2 {
		t.Fatalf("los dos pasos VEC no conservaron el original o no separaron firmantes: p1=%+v p2=%+v", r1.Material, r2.Material)
	}
}

func TestFirmaVecRechazaPDFODictamenNoAcreditadoSinEfectos(t *testing.T) {
	casos := map[string]func(*ServicioFirmaVec, *originalExternoPrueba, *SolicitudFirmaVec){
		"PDF desligado del original": func(_ *ServicioFirmaVec, _ *originalExternoPrueba, sol *SolicitudFirmaVec) {
			sol.PDFFirmado = []byte("%PDF-1.7 firmado sin el original autorizado")
		},
		"dictamen de integridad negativo": func(s *ServicioFirmaVec, _ *originalExternoPrueba, _ *SolicitudFirmaVec) {
			s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoIntegridadNoValida}
		},
		"firmante criptografico indeterminado": func(s *ServicioFirmaVec, _ *originalExternoPrueba, _ *SolicitudFirmaVec) {
			s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoFirmanteNoIdentificado}
		},
	}
	for nombre, preparar := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
			sol := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-dictamen-001")
			preparar(s, original, &sol)
			_, err := s.Firmar(context.Background(), sol)
			if !errors.Is(err, ErrFirmaNoVerificada) || len(registro.registrado) != 0 || len(competencia.vistas) != 0 ||
				len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("PDF o dictamen no acreditado produjo efectos: %v", err)
			}
		})
	}
}

func TestFirmaVecRechazaOriginalAjenoONoDisponible(t *testing.T) {
	casos := map[string]struct {
		preparar func(*originalExternoPrueba)
		err      error
	}{
		"fuente no disponible": {
			preparar: func(d *originalExternoPrueba) { d.err = errors.New("fuente sintetica no disponible") },
			err:      ports.ErrFuenteOriginalFirmaNoDisponible,
		},
		"referencia ajena": {
			preparar: func(d *originalExternoPrueba) {
				d.alterar = func(o *ports.OriginalFirmaAutorizado) { o.Solicitud.OriginalRef = "documento:ajeno" }
			},
			err: ports.ErrOriginalFirmaNoAutorizado,
		},
		"huella ajena": {
			preparar: func(d *originalExternoPrueba) {
				d.alterar = func(o *ports.OriginalFirmaAutorizado) { o.HuellaSHA256 = strings.Repeat("0", 64) }
			},
			err: ports.ErrOriginalFirmaNoAutorizado,
		},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
			caso.preparar(original)
			_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-original-001"))
			if !errors.Is(err, caso.err) || len(registro.registrado) != 0 || len(competencia.vistas) != 0 ||
				len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("original no autorizado produjo efectos: %v", err)
			}
		})
	}
}

func TestFirmaVecRechazaCompetenciaNoCoincidente(t *testing.T) {
	casos := map[string]func(*ports.EvidenciaCompetenciaFirmante){
		"cargo":     func(e *ports.EvidenciaCompetenciaFirmante) { e.CargoFirmante = "Otro cargo" },
		"perfil":    func(e *ports.EvidenciaCompetenciaFirmante) { e.PerfilFirmanteRef = "perfil:ajeno" },
		"solicitud": func(e *ports.EvidenciaCompetenciaFirmante) { e.Solicitud.PasoRef = "paso:ajeno" },
		"vigencia":  func(e *ports.EvidenciaCompetenciaFirmante) { e.Vigente = false },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
			competencia.alterar = alterar
			_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-competencia-001"))
			if !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) || len(registro.registrado) != 0 ||
				len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("competencia no coincidente produjo efectos: %v", err)
			}
		})
	}
}

func TestFirmaVecRechazaCapacidadV3DeOtraAudienciaSinCustodiar(t *testing.T) {
	s, registro, original, _, autorizador, custodio := servicioFirmaVecPrueba(t)
	autorizador.audiencia = ports.AudienciaFirmaExternaV3
	_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-capacidad-001"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(autorizador.vistas) != 1 ||
		len(registro.registrado) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("capacidad V3 de otra audiencia produjo efectos: %v", err)
	}
}

func TestFirmaVecRechazaCertificadoDistintoAlCanalSellado(t *testing.T) {
	s, registro, original, _, autorizador, custodio := servicioFirmaVecPrueba(t)
	autorizador.certificadoCanal = strings.Repeat("d", 64)
	_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-canal-001"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(autorizador.vistas) != 1 ||
		len(registro.registrado) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("certificado ajeno al canal produjo efectos: %v", err)
	}
}

func TestFirmaVecReplayRecuperaMismoMaterialYRechazaConflicto(t *testing.T) {
	s, registro, original, _, _, custodio := servicioFirmaVecPrueba(t)
	sol := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-replay-001")
	primero, err := s.Firmar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	repetido, err := s.Firmar(context.Background(), sol)
	if err != nil || !repetido.Recibo.YaRegistrada || repetido.Recibo.Secuencia != primero.Recibo.Secuencia ||
		len(registro.registrado) != 1 || len(custodio.ordenes) != 2 {
		t.Fatalf("replay identico no recuperado: %+v, %v", repetido, err)
	}

	sol.PDFFirmado = append(bytes.Clone(sol.PDFFirmado), 'x')
	if _, err := s.Firmar(context.Background(), sol); !errors.Is(err, ports.ErrClaveFirmaDocumentoUsada) || len(registro.registrado) != 1 {
		t.Fatalf("replay con material distinto admitido: %v", err)
	}
}

func TestMaterialFirmaVecCanonicoLigaOriginalPDFYPasoSinDeclaracionPortafirmas(t *testing.T) {
	s, _, original, _, _, _ := servicioFirmaVecPrueba(t)
	r, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-canon-001"))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := r.Material.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	for _, fragmento := range [][]byte{
		[]byte(`"Via":"certificado_vec"`),
		[]byte(`"OriginalRef":"` + r.Material.OriginalRef + `"`),
		[]byte(`"OriginalVersion":7`),
	} {
		if !bytes.Contains(canon, fragmento) {
			t.Fatalf("el canon VEC no contiene %s: %s", fragmento, canon)
		}
	}
	for _, ausente := range [][]byte{[]byte("ReferenciaPortafirmasDeclarada"), []byte("FechaPortafirmasDeclarada")} {
		if bytes.Contains(canon, ausente) {
			t.Fatalf("el canon VEC contiene una declaración de Portafirmas: %s", canon)
		}
	}

	base, err := r.Material.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	variantes := map[string]func(*ports.MaterialFirmaVec){
		"referencia original": func(m *ports.MaterialFirmaVec) { m.OriginalRef = "ref:" + strings.Repeat("3", 64) },
		"version original":    func(m *ports.MaterialFirmaVec) { m.OriginalVersion++ },
		"PDF firmado":         func(m *ports.MaterialFirmaVec) { m.FirmadoHuella = strings.Repeat("1", 64) },
		"paso":                func(m *ports.MaterialFirmaVec) { m.PasoOrden++ },
	}
	for nombre, variar := range variantes {
		t.Run(nombre, func(t *testing.T) {
			m := r.Material
			variar(&m)
			h, err := m.HuellaSHA256()
			if err != nil || h == base {
				t.Fatalf("la variación no quedó ligada a la huella: %s, %v", h, err)
			}
		})
	}
}
