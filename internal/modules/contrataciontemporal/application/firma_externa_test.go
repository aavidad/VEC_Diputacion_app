package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strconv"
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
	err                 error
	alterar             func(*ports.EvidenciaCompetenciaFirmante)
	avanzarComprobacion bool
	vistas              []ports.SolicitudCompetenciaFirmante
	competenciasVistas  []string
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
		UnidadFirmanteRef: "unidad:rrhh", PerfilActivoFirmanteRef: "perfil-activo:firmante:001", PuestoFirmanteRef: "puesto:firma",
		AmbitoFirmanteRef: "ambito:provincial", AsignacionFirmanteRef: "asignacion:firma:001",
		AsignacionFirmanteVersion: 4, AsignacionFirmanteHuella: strings.Repeat("b", 64),
		VersionRolFirmanteRef: "rol:ct:firmante:v1", VersionRolFirmanteHuella: strings.Repeat("c", 64),
		ControlVigenciaFirmanteRef: "rol:ct:firmante:v1", ControlVigenciaFirmanteRevision: 3,
		ControlVigenciaFirmanteHuella: strings.Repeat("d", 64),
		AsignacionVigenteDesde:        "2026-09-01T00:00:00Z", AsignacionVigenteHasta: "2026-12-01T00:00:00Z",
		CompetenciaComprobadaEn: "2026-10-02T10:00:00Z", ActoCompetenciaRef: "acto:competencia:001",
	}
	if d.avanzarComprobacion {
		e.CompetenciaComprobadaEn = "2026-10-02T10:00:0" + strconv.Itoa(len(d.vistas)-1) + "Z"
	}
	if d.alterar != nil {
		d.alterar(&e)
	}
	d.competenciasVistas = append(d.competenciasVistas, e.CompetenciaComprobadaEn)
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

type autorizadorConsultaFirmasR5Prueba struct {
	err       error
	audiencia string
	alterar   func(*ports.MaterialConsultaFirmasR5)
	vistas    []ports.MaterialConsultaFirmasR5
}

func (d *autorizadorConsultaFirmasR5Prueba) AutorizarConsultaFirmasR5(_ context.Context, m ports.MaterialConsultaFirmasR5) (ports.CapacidadConsultaFirmasR5, error) {
	d.vistas = append(d.vistas, m)
	if d.err != nil {
		return ports.CapacidadConsultaFirmasR5{}, d.err
	}
	materialAutorizado := m
	if d.alterar != nil {
		d.alterar(&materialAutorizado)
	}
	r, err := RecursoConsultaFirmasR5(materialAutorizado)
	if err != nil {
		return ports.CapacidadConsultaFirmasR5{}, err
	}
	contexto, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.CapacidadConsultaFirmasR5{}, err
	}
	audiencia := ports.AudienciaConsultaFirmasR5V3
	if d.audiencia != "" {
		audiencia = d.audiencia
	}
	hasta := instanteFirmaPrueba.Add(time.Minute)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision-consulta-firmas-r5-001", strings.Repeat("a", 64), strings.Repeat("a", 64),
		"contexto-consulta-firmas-r5-001", strings.Repeat("a", 64), ports.AccionConsultarFirmasR5,
		r.Referencia, contexto, audiencia, hasta.Add(-5*time.Second), hasta,
	)
	if err != nil {
		return ports.CapacidadConsultaFirmasR5{}, err
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{5}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		return ports.CapacidadConsultaFirmasR5{}, err
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte{4}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen,
		[]byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte("payload"), []byte("cose"), []byte("evidencia"), raiz,
	)
	if err != nil {
		return ports.CapacidadConsultaFirmasR5{}, err
	}
	return ports.TransportarMaterialConsultaFirmasR5(material), nil
}

type registroExternoPrueba struct {
	firmas               []ports.FirmaRegistrada
	firmantesPrincipales []string
	registrado           []ports.MaterialFirmaExterna
	consultas            []ports.MaterialConsultaFirmasR5
	lecturas             []ports.LecturaFirmasR5
	historiaRevision     uint64
	historiaHuella       string
	versionActual        uint64
}

func (d *registroExternoPrueba) ConsultarFirmasAutorizadas(_ context.Context, m ports.MaterialConsultaFirmasR5, c ports.CapacidadConsultaFirmasR5) (ports.LecturaFirmasR5, error) {
	if err := ValidarCapacidadConsultaFirmasR5(c, m); err != nil {
		return ports.LecturaFirmasR5{}, err
	}
	d.consultas = append(d.consultas, m)
	firmas := make([]ports.FirmaRegistrada, 0, len(d.firmas))
	for _, firma := range d.firmas {
		if firma.Documento == m.Documento {
			firmas = append(firmas, firma)
		}
	}
	exacta := false
	for _, firma := range firmas {
		if firma.Documento == m.Documento && firma.ExpedienteVersion == m.VersionExpediente &&
			firma.ClaveIdempotencia == m.ClaveIdempotencia {
			firmas = []ports.FirmaRegistrada{firma}
			exacta = true
			break
		}
	}
	if !exacta && m.VersionExpediente < d.versionActual {
		firmas = nil
	}
	for i := range firmas {
		for j, registrada := range d.firmas {
			if registrada.ClaveIdempotencia == firmas[i].ClaveIdempotencia {
				firmas[i].CoincideFirmanteCandidato = j < len(d.firmantesPrincipales) &&
					d.firmantesPrincipales[j] == m.FirmantePrincipalCandidatoRef
				break
			}
		}
	}
	coincide, acreditada := separacionGlobalFirmasR5Prueba(d.firmas, d.firmantesPrincipales, m, exacta)
	lectura := ports.LecturaFirmasR5{Firmas: firmas, HistoriaRevision: d.historiaRevision, HistoriaHuella: d.historiaHuella,
		CoincideFirmanteEnOtroPaso: coincide, HistoriaSeparacionAcreditada: acreditada}
	d.lecturas = append(d.lecturas, lectura)
	return lectura, nil
}

func separacionGlobalFirmasR5Prueba(firmas []ports.FirmaRegistrada, principales []string, m ports.MaterialConsultaFirmasR5, exacta bool) (bool, bool) {
	if exacta {
		return false, false
	}
	coincide, acreditada := false, true
	for i, firma := range firmas {
		if firma.Documento == m.Documento && firma.PasoOrden == m.PasoOrden && firma.CatalogoHuella == m.CatalogoHuella {
			continue
		}
		if !firma.FirmantePrincipalAcreditado || i >= len(principales) || principales[i] == "" {
			acreditada = false
			continue
		}
		if principales[i] == m.FirmantePrincipalCandidatoRef {
			coincide = true
		}
	}
	return coincide, acreditada
}

type politicaMismaPersonaAplicacionPrueba struct{}

func (politicaMismaPersonaAplicacionPrueba) PoliticaMismaPersonaEnPasos(_ context.Context, ref, huella string) (ports.PoliticaMismaPersonaEnPasos, error) {
	return ports.PoliticaMismaPersonaEnPasos{CatalogoRef: ref, CatalogoHuella: huella, Permite: true}, nil
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
	if m.HistoriaRevision != d.historiaRevision || m.HistoriaHuella != d.historiaHuella {
		return ports.ReciboFirmaDocumento{}, ports.ErrFirmaDocumentoEnConflicto
	}
	d.registrado = append(d.registrado, m)
	recibo := reciboFirmaExternaPrueba(m, h, false)
	d.firmas = append(d.firmas, ports.FirmaRegistrada{Via: m.Via, FirmaRef: recibo.FirmaRef, ReciboRef: recibo.ReciboRef,
		Documento: m.Documento, Secuencia: m.Secuencia, ExpedienteVersion: m.VersionExpediente,
		CatalogoRef: m.CatalogoRef, CatalogoHuella: m.CatalogoHuella, PasoRef: m.PasoRef, PasoOrden: m.PasoOrden,
		Resultado: domain.ResultadoFirmaFirmado, OriginalHuella: m.OriginalHuella, FirmadoHuella: m.FirmadoHuella,
		OriginalRef: m.OriginalRef, OriginalVersion: m.OriginalVersion, FirmantePrincipalAcreditado: true,
		ReferenciaPortafirmasDeclarada: m.ReferenciaPortafirmasDeclarada, FechaPortafirmasDeclarada: m.FechaPortafirmasDeclarada,
		HistoriaRevision: m.HistoriaRevision, HistoriaHuella: m.HistoriaHuella,
		ClaveIdempotencia: m.ClaveIdempotencia, DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion})
	d.firmantesPrincipales = append(d.firmantesPrincipales, m.FirmantePrincipalRef)
	d.historiaRevision++
	d.historiaHuella = h
	return recibo, nil
}

func reciboFirmaExternaPrueba(m ports.MaterialFirmaExterna, huellaSolicitud string, repetido bool) ports.ReciboFirmaDocumento {
	return ports.ReciboFirmaDocumento{FirmaRef: "firma:externa:" + strconv.Itoa(m.Secuencia),
		ReciboRef: "recibo:firma:externa:" + strconv.Itoa(m.Secuencia), Secuencia: m.Secuencia,
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
	registro := &registroExternoPrueba{historiaHuella: strings.Repeat("1", 64), versionActual: 7}
	consulta := &autorizadorConsultaFirmasR5Prueba{}
	competencia, autorizador := &competenciaExternaPrueba{}, &autorizadorExternoPrueba{}
	s, err := NuevoServicioFirmaExterna(base, registro, consulta, autorizador, competencia)
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

func firmaLegadaPasoUnoPrueba(original []byte) ports.FirmaRegistrada {
	return ports.FirmaRegistrada{
		FirmaRef: "firma:legada:001", ReciboRef: "recibo:firma:legada:001",
		Documento: "informe_definitivo", Secuencia: 1, ExpedienteVersion: 7,
		CatalogoHuella: strings.Repeat("c", 64), PasoOrden: 1, Resultado: domain.ResultadoFirmaFirmado,
		OriginalHuella: huella(original), FirmadoHuella: strings.Repeat("e", 64),
		RegistradaEn: instanteFirmaPrueba, ClaveIdempotencia: "clave-firma-legada-0001",
	}
}

func comprobarCanonFirmaR5Prueba(t *testing.T, canon []byte, clavesEsperadas int, externa bool) {
	t.Helper()
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(canon, &campos); err != nil {
		t.Fatal(err)
	}
	if len(campos) != clavesEsperadas {
		t.Fatalf("canon R5 con %d claves, esperadas %d: %s", len(campos), clavesEsperadas, canon)
	}
	for _, clave := range []string{"Secuencia", "HistoriaRevision", "HistoriaHuella"} {
		if _, ok := campos[clave]; !ok {
			t.Fatalf("canon R5 sin %s: %s", clave, canon)
		}
	}
	posSecuencia := bytes.Index(canon, []byte(`"Secuencia"`))
	posRevision := bytes.Index(canon, []byte(`"HistoriaRevision"`))
	posHuella := bytes.Index(canon, []byte(`"HistoriaHuella"`))
	if posSecuencia < 0 || posRevision <= posSecuencia || posHuella <= posRevision {
		t.Fatalf("cabeza de historia fuera de orden tras Secuencia: %s", canon)
	}
	if _, existe := campos["CompetenciaComprobadaEn"]; existe {
		t.Fatalf("el instante de comprobación entró en material idempotente: %s", canon)
	}
	_, tieneReferencia := campos["ReferenciaPortafirmasDeclarada"]
	_, tieneFecha := campos["FechaPortafirmasDeclarada"]
	if externa != (tieneReferencia && tieneFecha) {
		t.Fatalf("declaraciones Portafirmas inesperadas, externa=%v: %s", externa, canon)
	}
}

func TestFirmaExternaRegistraSoloPDFVerificadoConOriginalYCompetencia(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	sol := solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-0001")
	r, err := s.Registrar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	if len(registro.registrado) != 1 || len(competencia.vistas) != 1 || len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 ||
		len(registro.consultas) != 1 || registro.consultas[0].FirmantePrincipalCandidatoRef != r.Material.FirmantePrincipalRef ||
		r.Material.FirmantePrincipalRef != "per_firmante_sintetico_001" || r.Recibo.ActorRef == r.Material.FirmantePrincipalRef ||
		r.Custodiado.HuellaSHA256 != huella(sol.PDFFirmado) || r.MotivoVerificacion != docports.MotivoFirmaVerificada {
		t.Fatalf("registro externo incompleto: %+v, registros=%d competencia=%d autorizacion=%d custodia=%d", r, len(registro.registrado), len(competencia.vistas), len(autorizador.vistas), len(custodio.ordenes))
	}
}

func TestFirmaExternaDeniegaConsultaAntesDeLeerHistoria(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	consulta := s.consulta.(*autorizadorConsultaFirmasR5Prueba)
	consulta.err = errors.New("consulta denegada de prueba")

	_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-consulta-denegada"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(consulta.vistas) != 1 ||
		consulta.vistas[0].FirmantePrincipalCandidatoRef != "per_firmante_sintetico_001" || len(registro.consultas) != 0 ||
		len(registro.registrado) != 0 || len(competencia.vistas) != 1 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("la consulta denegada alcanzó historia o efectos: %v", err)
	}
}

func TestFirmaExternaRechazaCapacidadDeConsultaLigadaAOtroCandidato(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	consulta := s.consulta.(*autorizadorConsultaFirmasR5Prueba)
	consulta.alterar = func(m *ports.MaterialConsultaFirmasR5) {
		m.FirmantePrincipalCandidatoRef = "per_candidato_adulterado_999"
	}

	_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-candidato-adulterado"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(consulta.vistas) != 1 || len(registro.consultas) != 0 ||
		len(registro.registrado) != 0 || len(competencia.vistas) != 1 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("capacidad ligada a otro candidato alcanzó historia o efectos: %v", err)
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
	s, registro, original, competencia, autorizadorFirma, custodio := servicioFirmaExternaPrueba(t)
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
	registro.historiaRevision++
	registro.historiaHuella = strings.Repeat("8", 64)
	registro.versionActual = 8
	cabezaActualRevision, cabezaActualHuella := registro.historiaRevision, registro.historiaHuella
	r2Repetido, err := s.Registrar(context.Background(), segunda)
	if err != nil {
		t.Fatal(err)
	}
	versionDistinta := segunda
	versionDistinta.VersionExpediente++
	_, err = s.Registrar(context.Background(), versionDistinta)
	if !errors.Is(err, ports.ErrClaveFirmaDocumentoUsada) {
		t.Fatalf("la clave histórica aceptó otra versión de expediente: %v", err)
	}
	claveNueva := versionDistinta
	claveNueva.ClaveIdempotencia = "clave-firma-externa-nueva-version"
	_, err = s.Registrar(context.Background(), claveNueva)
	if !errors.Is(err, ErrPasoFirmaNoPendiente) {
		t.Fatalf("una clave nueva reutilizó el paso completo: %v", err)
	}
	filasPasoDos := 0
	for _, firma := range registro.firmas {
		if firma.PasoOrden == 2 {
			filasPasoDos++
		}
	}
	if r1.Material.OriginalRef != r2.Material.OriginalRef || r1.Material.OriginalVersion != r2.Material.OriginalVersion ||
		r1.Material.OriginalHuella != r2.Material.OriginalHuella || r1.Material.FirmadoHuella == r2.Material.FirmadoHuella ||
		r1.Material.CertificadoHuella == r2.Material.CertificadoHuella || r1.Material.FirmanteRef == r2.Material.FirmanteRef ||
		r1.Material.FirmantePrincipalRef == r2.Material.FirmantePrincipalRef ||
		r1.Material.CargoFirmante != "Técnico" || r2.Material.CargoFirmante != "Jefatura" ||
		!r2Repetido.Recibo.YaRegistrada || r2Repetido.Recibo.ReciboRef != r2.Recibo.ReciboRef ||
		r2Repetido.Recibo.SolicitudHuella != r2.Recibo.SolicitudHuella ||
		r2Repetido.Material.OriginalHuella != r2.Material.OriginalHuella ||
		r2Repetido.Material.HistoriaRevision != r2.Material.HistoriaRevision ||
		r2Repetido.Material.HistoriaHuella != r2.Material.HistoriaHuella ||
		registro.historiaRevision != cabezaActualRevision || registro.historiaHuella != cabezaActualHuella ||
		len(registro.registrado) != 2 || filasPasoDos != 1 || len(registro.consultas) != 5 ||
		registro.consultas[2].ClaveIdempotencia != segunda.ClaveIdempotencia ||
		registro.consultas[1].PasoOrden != 2 || registro.consultas[1].CatalogoHuella != r2.Material.CatalogoHuella ||
		registro.consultas[1].FirmantePrincipalCandidatoRef != r2.Material.FirmantePrincipalRef ||
		len(registro.lecturas[2].Firmas) != 1 || registro.lecturas[2].Firmas[0].Secuencia != 2 ||
		len(registro.lecturas[4].Firmas) != 2 || len(competencia.vistas) != 5 ||
		len(autorizadorFirma.vistas) != 3 || len(custodio.ordenes) != 3 {
		t.Fatalf("ronda externa no conserva/separa los datos exigidos: p1=%+v p2=%+v", r1.Material, r2.Material)
	}
	consulta := s.consulta.(*autorizadorConsultaFirmasR5Prueba)
	consulta.err = errors.New("concesión de lectura revocada")
	if _, err := s.Registrar(context.Background(), segunda); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) ||
		len(registro.consultas) != 5 || len(registro.registrado) != 2 || len(autorizadorFirma.vistas) != 3 || len(custodio.ordenes) != 3 {
		t.Fatalf("la revocación permitió leer o repetir el efecto: %v", err)
	}
}

type circuitoFirmaMutantePrueba struct{ lecturas int }

func (c *circuitoFirmaMutantePrueba) CircuitoFirma(ctx context.Context) (domain.CircuitoFirma, error) {
	circuito, err := (circuitoFirmaPrueba{}).CircuitoFirma(ctx)
	c.lecturas++
	if c.lecturas > 1 {
		circuito.HuellaCatalogo = strings.Repeat("e", 64)
	}
	return circuito, err
}

func TestFirmaExternaReplayTrasReconstruirServicioYCambioCatalogo(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	sol := solicitudFirmaExternaPrueba(original.contenido, "clave-externa-reinicio-p1")
	primero, err := s.Registrar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	baseNueva, err := NuevoServicioFirmaDocumento(s.base.circuito, &registroFirmaPrueba{}, &autorizadorFirmaPrueba{},
		verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada})
	if err != nil {
		t.Fatal(err)
	}
	if err := baseNueva.ComponerOriginalAutorizado(original); err != nil {
		t.Fatal(err)
	}
	if err := baseNueva.ComponerCustodia(custodio, map[string]string{"informe_definitivo": tipoCustodiaPrueba}); err != nil {
		t.Fatal(err)
	}
	reconstruido, err := NuevoServicioFirmaExterna(baseNueva, registro, s.consulta, autorizador, competencia)
	if err != nil {
		t.Fatal(err)
	}
	repetido, err := reconstruido.Registrar(context.Background(), sol)
	if err != nil || !repetido.Recibo.YaRegistrada || repetido.Recibo.ReciboRef != primero.Recibo.ReciboRef ||
		len(registro.registrado) != 1 || len(registro.lecturas[1].Firmas) != 1 {
		t.Fatalf("servicio reconstruido no recuperó la firma exacta: %v", err)
	}
	s.base.circuito = &circuitoFirmaMutantePrueba{}
	_, err = s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-externa-catalogo-mutante"))
	if !errors.Is(err, ErrCircuitoFirmaNoDisponible) || len(registro.registrado) != 1 || len(custodio.ordenes) != 2 {
		t.Fatalf("cambio de catálogo entre lecturas produjo efecto: %v", err)
	}
}

func TestFirmaExternaReparoExigeOriginalNuevo(t *testing.T) {
	s, registro, original, _, autorizador, custodio := servicioFirmaExternaPrueba(t)
	primera := solicitudFirmaExternaPrueba(original.contenido, "clave-externa-reparo-v7")
	if _, err := s.Registrar(context.Background(), primera); err != nil {
		t.Fatal(err)
	}
	ronda := &rondaInformePrueba{inicio: 8}
	politica := fuenteInformeTrasSubsanacionPrueba{politica: domain.PoliticaInformeTrasSubsanacion{ExigeInformeNuevo: true, DocumentoFirma: "informe_definitivo"}}
	if err := s.base.AbrirRondaInformeNuevo(politica, ronda); err != nil {
		t.Fatal(err)
	}
	registro.versionActual = 8
	vieja := primera
	vieja.VersionExpediente = 8
	vieja.ClaveIdempotencia = "clave-externa-reparo-v8-vieja"
	_, err := s.Registrar(context.Background(), vieja)
	if !errors.Is(err, ports.ErrOriginalTrasReparoNoNuevo) || len(registro.registrado) != 1 ||
		len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 {
		t.Fatalf("el reparo aceptó el original anterior: %v", err)
	}
	original.contenido = []byte("%PDF-1.7 original nuevo tras reparo sintetico")
	nueva := solicitudFirmaExternaPrueba(original.contenido, "clave-externa-reparo-v8-nueva")
	nueva.VersionExpediente, nueva.OriginalVersion = 8, 8
	nueva.OriginalRef = "ref:" + strings.Repeat("9", 64)
	resultado, err := s.Registrar(context.Background(), nueva)
	if err != nil || resultado.Material.OriginalRef != nueva.OriginalRef || len(registro.registrado) != 2 {
		t.Fatalf("el reparo no aceptó el original nuevo: %v", err)
	}
}

func TestFirmaExternaConsultaV3NoAceptaOtroPasoOCatalogo(t *testing.T) {
	for nombre, alterar := range map[string]func(*ports.MaterialConsultaFirmasR5){
		"paso":     func(m *ports.MaterialConsultaFirmasR5) { m.PasoOrden++ },
		"catalogo": func(m *ports.MaterialConsultaFirmasR5) { m.CatalogoHuella = strings.Repeat("9", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, _, autorizador, custodio := servicioFirmaExternaPrueba(t)
			consulta := s.consulta.(*autorizadorConsultaFirmasR5Prueba)
			consulta.alterar = alterar
			_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-externa-consulta-otro-"+nombre))
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(registro.consultas) != 0 ||
				len(registro.registrado) != 0 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("capacidad V3 de otro %s alcanzó la historia o el efecto: %v", nombre, err)
			}
		})
	}
}

func TestFirmaExternaMismaPersonaEnPasosRequierePoliticaVersionada(t *testing.T) {
	for _, permite := range []bool{false, true} {
		t.Run(strconv.FormatBool(permite), func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
			if _, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-externa-separacion-p1")); err != nil {
				t.Fatal(err)
			}
			competencia.alterar = func(e *ports.EvidenciaCompetenciaFirmante) {
				e.FirmantePrincipalRef = "per_firmante_sintetico_001"
				e.ActoCompetenciaRef = "acto:competencia:jefatura:002"
			}
			s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada, firmanteByte: 'd'}
			segunda := solicitudFirmaExternaPrueba(original.contenido, "clave-externa-separacion-p2")
			segunda.PasoOrden = 2
			if permite {
				if err := s.ComponerPoliticaMismaPersonaEnPasos(politicaMismaPersonaAplicacionPrueba{}); err != nil {
					t.Fatal(err)
				}
			}
			r, err := s.Registrar(context.Background(), segunda)
			if !permite {
				if !errors.Is(err, ports.ErrMismaPersonaEnOtroPasoR5) || len(registro.registrado) != 1 ||
					len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 ||
					!registro.lecturas[1].CoincideFirmanteEnOtroPaso || !registro.lecturas[1].HistoriaSeparacionAcreditada {
					t.Fatalf("misma persona pasó sin política: %v", err)
				}
				return
			}
			if err != nil || len(registro.registrado) != 2 || r.Material.FirmantePrincipalRef != "per_firmante_sintetico_001" ||
				r.Material.PerfilFirmanteRef != "perfil:ct:jefatura" || r.Material.CargoFirmante != "Jefatura" ||
				r.Material.ActoCompetenciaRef != "acto:competencia:jefatura:002" || len(autorizador.vistas) != 2 {
				t.Fatalf("política no conservó competencia del paso 2: %+v %v", r.Material, err)
			}
		})
	}
}

func TestFirmaExternaLegadoEnOtroDocumentoNoAcreditaSeparacion(t *testing.T) {
	s, registro, original, _, autorizador, custodio := servicioFirmaExternaPrueba(t)
	legada := firmaLegadaPasoUnoPrueba(original.contenido)
	legada.Documento = "resolucion"
	registro.firmas = append(registro.firmas, legada)
	registro.firmantesPrincipales = append(registro.firmantesPrincipales, "")
	_, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-externa-legado-otro-documento"))
	if !errors.Is(err, ports.ErrMismaPersonaEnOtroPasoR5) || len(registro.registrado) != 0 ||
		len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 ||
		len(registro.lecturas[0].Firmas) != 0 || registro.lecturas[0].HistoriaSeparacionAcreditada {
		t.Fatalf("historia global legada acreditó separación: %v", err)
	}
}

func TestFirmaExternaAntecedenteLegadoNoAcreditaPasoR5(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	registro.firmas = append(registro.firmas, firmaLegadaPasoUnoPrueba(original.contenido))
	sol := solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-paso-legacy")
	sol.PasoOrden = 2

	_, err := s.Registrar(context.Background(), sol)
	if !errors.Is(err, ports.ErrAntecedenteFirmaR5NoAcreditado) || len(registro.registrado) != 0 ||
		len(competencia.vistas) != 1 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("la firma legada acreditó indebidamente el paso R5: %v", err)
	}
}

func TestFirmaExternaPasoDosRechazaOtroOriginalAunqueConserveLosBytes(t *testing.T) {
	variantes := map[string]func(*SolicitudFirmaExterna){
		"referencia": func(s *SolicitudFirmaExterna) { s.OriginalRef = "ref:" + strings.Repeat("9", 64) },
		"version":    func(s *SolicitudFirmaExterna) { s.OriginalVersion++ },
	}
	for nombre, variar := range variantes {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
			if _, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-original-p1")); err != nil {
				t.Fatal(err)
			}
			segunda := solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-original-p2")
			segunda.PasoOrden = 2
			variar(&segunda)

			_, err := s.Registrar(context.Background(), segunda)
			if !errors.Is(err, ports.ErrAntecedenteFirmaR5NoAcreditado) || len(registro.registrado) != 1 ||
				len(competencia.vistas) != 2 || len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 {
				t.Fatalf("otro original con los mismos bytes heredó el antecedente R5: %v", err)
			}
		})
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
		"cargo distinto":        func(e *ports.EvidenciaCompetenciaFirmante) { e.CargoFirmante = "Otro cargo" },
		"perfil distinto":       func(e *ports.EvidenciaCompetenciaFirmante) { e.PerfilFirmanteRef = "perfil:ajeno" },
		"no vigente":            func(e *ports.EvidenciaCompetenciaFirmante) { e.Vigente = false },
		"perfil activo ausente": func(e *ports.EvidenciaCompetenciaFirmante) { e.PerfilActivoFirmanteRef = "" },
		"rol sin huella":        func(e *ports.EvidenciaCompetenciaFirmante) { e.VersionRolFirmanteHuella = "" },
		"control de otro rol":   func(e *ports.EvidenciaCompetenciaFirmante) { e.ControlVigenciaFirmanteRef = "rol:ct:otro:v1" },
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
	s, registro, original, competencia, autorizador, custodio := servicioFirmaExternaPrueba(t)
	competencia.avanzarComprobacion = true
	sol := solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-0005")
	primero, err := s.Registrar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	registro.historiaRevision++
	registro.historiaHuella = strings.Repeat("9", 64)
	cabezaAvanzadaRevision, cabezaAvanzadaHuella := registro.historiaRevision, registro.historiaHuella
	reintento, err := s.Registrar(context.Background(), sol)
	hPrimero, errHuellaPrimero := primero.Material.HuellaSHA256()
	hReintento, errHuellaReintento := reintento.Material.HuellaSHA256()
	if err != nil || errHuellaPrimero != nil || errHuellaReintento != nil || hPrimero != hReintento ||
		!reintento.Recibo.YaRegistrada || reintento.Recibo.Secuencia != primero.Recibo.Secuencia || len(registro.registrado) != 1 ||
		len(custodio.ordenes) != 2 || len(competencia.competenciasVistas) != 2 ||
		competencia.competenciasVistas[0] != "2026-10-02T10:00:00Z" || competencia.competenciasVistas[1] != "2026-10-02T10:00:01Z" {
		t.Fatalf("reintento no recuperado: %+v, %v", reintento, err)
	}
	if reintento.Material.HistoriaRevision != primero.Material.HistoriaRevision ||
		reintento.Material.HistoriaHuella != primero.Material.HistoriaHuella ||
		registro.historiaRevision != cabezaAvanzadaRevision || registro.historiaHuella != cabezaAvanzadaHuella {
		t.Fatalf("el replay no conservó la cabeza original: primero=%+v replay=%+v", primero.Material, reintento.Material)
	}
	conflicto := primero.Material
	conflicto.ClaveIdempotencia = "clave-firma-externa-cabeza-obsoleta"
	conflicto.DocumentoCustodiaRef = ports.DocumentoCustodiaRef(conflicto.OrganizacionRef, conflicto.ExpedienteRef, conflicto.ClaveIdempotencia)
	capacidadConflicto, err := autorizador.AutorizarRegistroFirmaExterna(context.Background(), conflicto)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registro.RegistrarFirmaExterna(context.Background(), conflicto, capacidadConflicto); !errors.Is(err, ports.ErrFirmaDocumentoEnConflicto) {
		t.Fatalf("una clave nueva con cabeza obsoleta no produjo conflicto: %v", err)
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

func TestMaterialFirmaExternaCanonicoConservaContratoCT170(t *testing.T) {
	s, _, original, _, _, _ := servicioFirmaExternaPrueba(t)
	r, err := s.Registrar(context.Background(), solicitudFirmaExternaPrueba(original.contenido, "clave-firma-externa-canon-r5"))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := r.Material.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	comprobarCanonFirmaR5Prueba(t, canon, 45, true)
}
