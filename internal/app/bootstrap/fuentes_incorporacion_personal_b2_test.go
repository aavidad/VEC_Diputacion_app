package bootstrap

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
)

type clasesIncorporacionB2Prueba struct {
	resultado pp.ResultadoClasesOcupacionCT
	err       error
	llamadas  int
}

func (c *clasesIncorporacionB2Prueba) ConsultarClasesOcupacion(context.Context, pp.ConsultaClasesOcupacionCT) (pp.ResultadoClasesOcupacionCT, error) {
	c.llamadas++
	return c.resultado, c.err
}

func TestIncorporacionB2ClaseProcedeDelCatalogoPersonal(t *testing.T) {
	base, ctx, _, publicador := escenarioNominalIncorporacion(t)
	refs := base.referencias
	refs.PerfilV3Ref = base.soporte.contexto.Resultado.Contexto.PerfilActivoRef
	if e := extenderPerfilesNominalesB2(base.nominales, refs, &archivoIncorporacionPersonalB2{Protocolo: "personal_b2_v1", OrganismoRef: "organismo:prueba", CatalogoRPTID: "categorias_rpt", ModuloRPTID: "personal"}, base.reloj.Ahora()); e != nil {
		t.Fatal(e)
	}
	for _, p := range base.nominales.todos() {
		p.contextoEsperadoRegistrado = p.contexto.Resultado
		p.sesionOperativa = &sesionNominalIncorporacionPrueba{contexto: p.contexto}
		publicador.publicadas[p.perfilRef()] = instantaneaPublicadaDesarrollo{instantanea: p.plantilla, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
	}
	ctx = context.WithValue(ctx, claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: "/api/interno/contratacion-temporal/incorporacion-personal-b2/plan/v1"})
	clases := &clasesIncorporacionB2Prueba{resultado: pp.ResultadoClasesOcupacionCT{Catalogo: personal.CatalogoClasesOcupacionCT{Ref: "catalogo:clases", Version: 4, HuellaSHA256: strings.Repeat("a", 64), Opciones: []personal.OpcionClaseOcupacionCT{{Valor: "temporal", TextoClave: "personal.clases.temporal"}}}}}
	f := &fuentesIncorporacionPersonalB2{organismoRef: "organismo:prueba", clases: clases, autoridad: &autoridadIncorporacionPersonalB2{perfiles: base.nominales, reloj: base.reloj}}
	if e := f.validarClaseOcupacion(ctx, "temporal"); e != nil {
		t.Fatalf("clase publicada de Personal rechazada: %v", e)
	}
	if e := f.validarClaseOcupacion(ctx, "provisional"); !errors.Is(e, ct.ErrPlanNominalB2Invalido) {
		t.Fatalf("clase ausente del catálogo admitida: %v", e)
	}
	clases.err = personal.ErrRegistroEmpleadoB2NoDisponible
	if e := f.validarClaseOcupacion(ctx, "temporal"); !errors.Is(e, personal.ErrRegistroEmpleadoB2NoDisponible) {
		t.Fatalf("caída del catálogo sustituyó fuente: %v", e)
	}
	if clases.llamadas != 3 {
		t.Fatal("no se releyó Personal para cada selección")
	}
	clases.err = nil
	clases.resultado.Catalogo.Version = 0
	if e := f.validarClaseOcupacion(ctx, "temporal"); !errors.Is(e, ct.ErrPlanNominalB2NoDisponible) {
		t.Fatalf("fuente sin versión admitida: %v", e)
	}
	clases.resultado.Catalogo.Version = 4
	clases.resultado.Catalogo.Opciones = append(clases.resultado.Catalogo.Opciones, personal.OpcionClaseOcupacionCT{Valor: "reserva", TextoClave: "personal.clases.reserva"})
	if e := f.validarClaseOcupacion(ctx, "temporal"); !errors.Is(e, ct.ErrPlanNominalB2NoDisponible) {
		t.Fatalf("catálogo con reserva efectiva admitido: %v", e)
	}
}

func TestIncorporacionB2PrevioPersonaConservaDenegacionYCaida(t *testing.T) {
	ctx := context.Background()
	if e := errorPrevioPersonaIncorporacionB2(ctx, ct.ErrPreparacionIncorporacionPendiente); e != nil {
		t.Fatalf("ausencia pendiente no es error de consulta: %v", e)
	}
	for _, caso := range []struct {
		fuente error
		clase  error
	}{
		{bp.ErrConsultaPersonaAceptacionCTDenegada, httpct.ErrDenegadaIncorporacionPersonalB2},
		{bp.ErrConsultaPersonaAceptacionCTNoDisponible, httpct.ErrManejadorIncorporacionPersonalB2},
		{errors.Join(ct.ErrPreparacionIncorporacionPendiente, bp.ErrConsultaPersonaAceptacionCTNoDisponible), httpct.ErrManejadorIncorporacionPersonalB2},
	} {
		err := errorPrevioPersonaIncorporacionB2(ctx, caso.fuente)
		if err == nil || !errors.Is(err, caso.fuente) || !errors.Is(errorHTTPNominalB2(ctx, err), caso.clase) {
			t.Fatalf("fuente ocultada o mal clasificada: %v", err)
		}
	}
	cancelado, cancel := context.WithCancel(ctx)
	cancel()
	if e := errorPrevioPersonaIncorporacionB2(cancelado, ct.ErrPreparacionIncorporacionPendiente); !errors.Is(e, context.Canceled) {
		t.Fatalf("cancelación ocultada: %v", e)
	}
}

func TestIncorporacionB2SeleccionNoSustituyeFuentesCT(t *testing.T) {
	a := ct.AntecedentesPlanNominalB2{DocumentoRef: "documento:formalizacion", DocumentoSHA256: strings.Repeat("a", 64)}
	d := ct.DetalleExpedienteRRHH{Solicitud: ct.SolicitudOperativaRRHH{MotivoClave: "sustitucion"}, Analisis: &ct.AnalisisOperativoRRHH{PeriodoInicio: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), PeriodoFin: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)}}
	s := ct.SolicitudPlanNominalB2{DocumentoRef: a.DocumentoRef, DocumentoSHA256: a.DocumentoSHA256, Desde: "2026-09-30", Hasta: "2026-12-01", MotivoClave: "sustitucion"}
	if e := validarSeleccionFuenteCTB2(s, a, d); e != nil {
		t.Fatal(e)
	}
	for _, cambiar := range []func(*ct.SolicitudPlanNominalB2){
		func(s *ct.SolicitudPlanNominalB2) { s.DocumentoRef = "documento:ajeno" },
		func(s *ct.SolicitudPlanNominalB2) { s.DocumentoSHA256 = strings.Repeat("b", 64) },
		func(s *ct.SolicitudPlanNominalB2) { s.Desde = "2026-10-01" },
		func(s *ct.SolicitudPlanNominalB2) { s.Hasta = "" },
		func(s *ct.SolicitudPlanNominalB2) { s.MotivoClave = "otra_causa" },
	} {
		alterada := s
		cambiar(&alterada)
		if e := validarSeleccionFuenteCTB2(alterada, a, d); !errors.Is(e, ct.ErrPlanNominalB2Conflicto) {
			t.Fatalf("selección ajena admitida: %v", e)
		}
	}
	d.Analisis = nil
	if e := validarSeleccionFuenteCTB2(s, a, d); !errors.Is(e, ct.ErrPlanNominalB2Conflicto) {
		t.Fatalf("ausencia de análisis admitida: %v", e)
	}
}

func TestIncorporacionB2TraduccionConservaInstrumentosYProcedencia(t *testing.T) {
	plantilla := "11111111-1111-4111-8111-111111111111"
	rpt := "22222222-2222-4222-8222-222222222222"
	m := domct.PlanIncorporacionPersonalB2{PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 7, PersonaFuente: domct.FuentePlanPersonalB2{Ref: "fuente:bolsa", Version: 8, SHA256: strings.Repeat("f", 64)}, PersonaReciboBolsaRef: "recibo:bolsa", VersionPlazaRef: "plantilla:" + plantilla, VersionPuestoRef: "rpt:" + rpt, FuentePlantilla: domct.InstrumentoPlanPersonalB2{Ref: plantilla, Revision: 2, FuenteRef: "fuente:plantilla", FuenteSHA256: strings.Repeat("a", 64)}, FuenteRPT: domct.InstrumentoPlanPersonalB2{Ref: rpt, Revision: 3, FuenteRef: "fuente:rpt", FuenteSHA256: strings.Repeat("b", 64)}, RevisionPlaza: 4, RevisionPuesto: 5, Regimen: domct.EntradaPlanPersonalB2{Version: 1}, Modalidad: domct.EntradaPlanPersonalB2{Version: 1}}
	c := ct.ContratoPlanNominalB2{Material: m, PlanRef: "plan:ct", PlanReciboRef: "recibo:plan", PlanVersion: 1, PlanSHA256: strings.Repeat("c", 64), IdempotenciaPersonalUUID: "33333333-3333-4333-8333-333333333333"}
	r, err := contratoAplicacionB2(c)
	if err != nil {
		t.Fatal(err)
	}
	if r.Protocolo != inc.ProtocoloPersonalB2V1 || r.PersonaRef != m.PersonaRef || r.PersonaVersion != m.PersonaVersion || r.PersonaFuente != m.PersonaFuente || r.SeleccionReciboRef != m.PersonaReciboBolsaRef || r.DatosPersonal.VersionPlantillaRef != m.VersionPlazaRef || r.DatosPersonal.VersionRPTRef != m.VersionPuestoRef || r.DatosPersonal.RevisionPlaza != 4 || r.DatosPersonal.RevisionPuesto != 5 || r.FuenteRPT.Version != 3 || r.FuenteRPT.HuellaSHA256 != m.FuenteRPT.FuenteSHA256 {
		t.Fatal("traducción mezcló instrumentos, revisiones o fuente")
	}
	p := r.DatosPersonal.Procedencia
	if p.ActoRef != c.PlanRef || p.FuenteRef != c.PlanRef || p.FuenteVersion != 1 || p.FuenteHuellaSHA256 != c.PlanSHA256 || p.IdempotenciaRef != c.IdempotenciaPersonalUUID {
		t.Fatal("traducción sustituyó la procedencia del plan durable")
	}
}

func TestIncorporacionB2ConversionesVersionRechazanDesbordamientos(t *testing.T) {
	for _, v := range []int64{math.MinInt64, -1, 0} {
		if n, ok := versionB2DesdeInt64(v); ok || n != 0 {
			t.Fatalf("versión firmada inválida %d aceptada: %d", v, n)
		}
	}
	for _, v := range []int64{1, math.MaxInt64} {
		if n, ok := versionB2DesdeInt64(v); !ok || n != uint64(v) {
			t.Fatalf("versión firmada válida %d alterada: %d", v, n)
		}
	}
	for _, v := range []uint64{0, uint64(math.MaxInt64) + 1, math.MaxUint64} {
		if n, ok := versionB2HaciaInt64(v); ok || n != 0 {
			t.Fatalf("versión sin signo inválida %d aceptada: %d", v, n)
		}
	}
	for _, v := range []uint64{1, math.MaxInt64} {
		if n, ok := versionB2HaciaInt64(v); !ok || n != int64(v) {
			t.Fatalf("versión sin signo válida %d alterada: %d", v, n)
		}
	}
}

func TestIncorporacionB2TraduccionNoTruncaVersionesCT(t *testing.T) {
	m := domct.PlanIncorporacionPersonalB2{Regimen: domct.EntradaPlanPersonalB2{Version: 1}, Modalidad: domct.EntradaPlanPersonalB2{Version: 1}, RevisionPlaza: 1, RevisionPuesto: 1}
	for nombre, cambiar := range map[string]func(*domct.PlanIncorporacionPersonalB2){
		"regimen":   func(m *domct.PlanIncorporacionPersonalB2) { m.Regimen.Version = uint64(math.MaxInt64) + 1 },
		"modalidad": func(m *domct.PlanIncorporacionPersonalB2) { m.Modalidad.Version = math.MaxUint64 },
		"plaza":     func(m *domct.PlanIncorporacionPersonalB2) { m.RevisionPlaza = math.MaxUint64 },
		"puesto":    func(m *domct.PlanIncorporacionPersonalB2) { m.RevisionPuesto = 0 },
	} {
		alterado := m
		cambiar(&alterado)
		if _, err := contratoAplicacionDesdeMaterialB2(alterado); !errors.Is(err, ct.ErrPlanNominalB2Conflicto) {
			t.Fatalf("%s: versión fuera de rango aceptada: %v", nombre, err)
		}
	}
	for _, version := range []uint64{0, uint64(math.MaxInt64) + 1, math.MaxUint64} {
		if _, err := contratoAplicacionB2(ct.ContratoPlanNominalB2{Material: m, PlanVersion: version}); !errors.Is(err, ct.ErrPlanNominalB2Conflicto) {
			t.Fatalf("plan %d: procedencia truncada: %v", version, err)
		}
	}
}
