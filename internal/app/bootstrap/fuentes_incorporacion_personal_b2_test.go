package bootstrap

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type catalogosIncorporacionB2Prueba func(context.Context, personal.SolicitudConsultaCatalogoEmpleadoB2) (pp.ResultadoConsultaCatalogoEmpleadoB2, error)

func (c catalogosIncorporacionB2Prueba) Consultar(ctx context.Context, s personal.SolicitudConsultaCatalogoEmpleadoB2) (pp.ResultadoConsultaCatalogoEmpleadoB2, error) {
	return c(ctx, s)
}

func TestIncorporacionB2CatalogosParalelosConActorPropioYDosLecturas(t *testing.T) {
	a, ctx, _ := escenarioCapturaAsignacionB2Prueba(t)
	ctx, cancelar := context.WithTimeout(ctx, 2*time.Second)
	defer cancelar()
	inicios := make(chan personal.SolicitudConsultaCatalogoEmpleadoB2, 2)
	liberar := make(chan struct{})
	var una sync.Once
	abrir := func() { una.Do(func() { close(liberar) }) }
	defer abrir()
	f := &fuentesIncorporacionPersonalB2{autoridad: a, organismoRef: "organismo:prueba"}
	f.catalogos = catalogosIncorporacionB2Prueba(func(ctx context.Context, s personal.SolicitudConsultaCatalogoEmpleadoB2) (pp.ResultadoConsultaCatalogoEmpleadoB2, error) {
		if s.Actor.Principal.Attributes == nil {
			s.Actor.Principal.Attributes = map[string]string{}
		}
		s.Actor.Principal.Attributes["prueba_paralelo"] = s.Tipo
		inicios <- s
		select {
		case <-liberar:
		case <-ctx.Done():
			return pp.ResultadoConsultaCatalogoEmpleadoB2{}, ctx.Err()
		}
		if s.Actor.Principal.Attributes["prueba_paralelo"] != s.Tipo {
			return pp.ResultadoConsultaCatalogoEmpleadoB2{}, errors.New("actor compartido entre consultas")
		}
		return pp.ResultadoConsultaCatalogoEmpleadoB2{OrganismoRef: s.OrganismoRef, Entradas: []personal.EntradaCatalogoRegistroEmpleadoB2{{Tipo: s.Tipo, Ref: "catalogo:" + s.Tipo}}}, nil
	})
	type resultado struct {
		catalogos [2]pp.ResultadoConsultaCatalogoEmpleadoB2
		err       error
	}
	termino := make(chan resultado, 1)
	go func() { r, err := f.consultarCatalogosB2(ctx); termino <- resultado{r, err} }()
	solicitudes := map[string]personal.SolicitudConsultaCatalogoEmpleadoB2{}
	for range 2 {
		select {
		case s := <-inicios:
			solicitudes[s.Tipo] = s
		case <-ctx.Done():
			t.Fatal("las dos consultas no empezaron antes de terminar la primera")
		}
	}
	regimen, modalidad := solicitudes["regimen"], solicitudes["modalidad"]
	if len(solicitudes) != 2 || regimen.Actor.PerfilActivoRef != modalidad.Actor.PerfilActivoRef || regimen.Actor.PersonaRef != modalidad.Actor.PersonaRef ||
		regimen.OrganismoRef != f.organismoRef || modalidad.OrganismoRef != f.organismoRef || regimen.Estado != "publicada" || modalidad.Estado != "publicada" || regimen.Limite != 100 || modalidad.Limite != 100 {
		t.Fatal("las lecturas alteraron actor, organismo o selector")
	}
	abrir()
	r := <-termino
	if r.err != nil || r.catalogos[0].Entradas[0].Tipo != "regimen" || r.catalogos[1].Entradas[0].Tipo != "modalidad" {
		t.Fatalf("se mezclaron resultados: %+v", r)
	}
	actor, err := a.actor(ctx, personal.AccionConsultarCatalogoEmpleadoB2)
	if err != nil || actor.Principal.Attributes["prueba_paralelo"] != "" {
		t.Fatalf("consulta alteró actor nominal: %v", err)
	}
}

func TestIncorporacionB2CatalogosEsperanAmbasYConservanErrores(t *testing.T) {
	for _, caso := range []string{"denegada", "caida_y_denegada", "cancelada"} {
		t.Run(caso, func(t *testing.T) {
			a, ctx, _ := escenarioCapturaAsignacionB2Prueba(t)
			ctx, cancelar := context.WithCancel(ctx)
			defer cancelar()
			var llamadas atomic.Int32
			f := &fuentesIncorporacionPersonalB2{autoridad: a, organismoRef: "organismo:prueba"}
			f.catalogos = catalogosIncorporacionB2Prueba(func(_ context.Context, s personal.SolicitudConsultaCatalogoEmpleadoB2) (pp.ResultadoConsultaCatalogoEmpleadoB2, error) {
				llamadas.Add(1)
				if caso == "cancelada" {
					if s.Tipo == "regimen" {
						cancelar()
						return pp.ResultadoConsultaCatalogoEmpleadoB2{}, context.Canceled
					}
					return pp.ResultadoConsultaCatalogoEmpleadoB2{}, personal.ErrRegistroEmpleadoB2NoDisponible
				}
				if caso == "caida_y_denegada" && s.Tipo == "modalidad" {
					return pp.ResultadoConsultaCatalogoEmpleadoB2{}, personal.ErrRegistroEmpleadoB2NoDisponible
				}
				return pp.ResultadoConsultaCatalogoEmpleadoB2{}, personal.ErrRegistroEmpleadoB2Denegado
			})
			r, err := f.consultarCatalogosB2(ctx)
			if llamadas.Load() != 2 || r[0].Entradas != nil || r[1].Entradas != nil {
				t.Fatalf("no esperó ambas o devolvió resultado parcial: %d %+v", llamadas.Load(), r)
			}
			if caso != "cancelada" && !errors.Is(err, personal.ErrRegistroEmpleadoB2Denegado) ||
				caso != "denegada" && !errors.Is(err, personal.ErrRegistroEmpleadoB2NoDisponible) || caso == "cancelada" && !errors.Is(err, context.Canceled) {
				t.Fatalf("se ocultó causa de una consulta: %v", err)
			}
		})
	}
}

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
	ctx = context.WithValue(ctx, claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: "/api/vec/contratacion-temporal/incorporacion-personal-b2/plan/v1"})
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

// lectorDetalleIncorporacionB2Prueba cuenta las lecturas autorizadas del
// detalle para comprobar que una misma petición no lo lee dos veces.
type lectorDetalleIncorporacionB2Prueba struct {
	detalle  ct.DetalleExpedienteRRHH
	err      error
	llamadas int
}

func (l *lectorDetalleIncorporacionB2Prueba) Consultar(context.Context, ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error) {
	l.llamadas++
	return l.detalle, l.err
}

// escenarioFuentesIncorporacionB2Detalle monta las fuentes con la autoridad
// nominal de prueba y un doble del lector de detalle cuya asignación coincide
// con la unidad del expediente. La autoridad del escenario no tiene material
// emisor: la lectura de antecedentes falla por dependencia, nunca por permiso.
func escenarioFuentesIncorporacionB2Detalle(t *testing.T, metodo string) (*fuentesIncorporacionPersonalB2, *lectorDetalleIncorporacionB2Prueba, context.Context, ReferenciasCTIncorporacionDesarrollo) {
	t.Helper()
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
	ctx = context.WithValue(ctx, claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: metodo, ruta: httpct.RutaPlanB2})
	detalle := &lectorDetalleIncorporacionB2Prueba{detalle: ct.DetalleExpedienteRRHH{
		Resumen:    ct.ResumenExpedienteRRHH{ExpedienteRef: "expediente:ejercicio:ct:0001", OrganizacionRef: refs.OrganizacionRef, Version: 8, UnidadRef: refs.UnidadRef},
		Solicitud:  ct.SolicitudOperativaRRHH{MotivoClave: "sustitucion"},
		Asignacion: &ct.AsignacionOperativaRRHH{UnidadRef: refs.UnidadRef},
	}}
	f := &fuentesIncorporacionPersonalB2{organizacionRef: refs.OrganizacionRef, organismoRef: "organismo:prueba", autoridad: &autoridadIncorporacionPersonalB2{perfiles: base.nominales, reloj: base.reloj}, detalle: detalle}
	return f, detalle, ctx, refs
}

func TestIncorporacionB2OpcionesLeenDetalleExpedienteUnaSolaVez(t *testing.T) {
	f, detalle, ctx, _ := escenarioFuentesIncorporacionB2Detalle(t, "GET")
	if _, e := f.ConsultarOpcionesIncorporacionB2(ctx, "expediente:ejercicio:ct:0001"); !errors.Is(e, ct.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("la consulta de opciones no llegó a autorizar sus antecedentes: %v", e)
	}
	if detalle.llamadas != 1 {
		t.Fatalf("el detalle del expediente se leyó %d veces en la misma petición", detalle.llamadas)
	}
}

func TestIncorporacionB2OpcionesComparanUnidadSinReleerDetalle(t *testing.T) {
	f, detalle, ctx, _ := escenarioFuentesIncorporacionB2Detalle(t, "GET")
	detalle.detalle.Asignacion.UnidadRef = "unidad:ajena"
	if _, e := f.ConsultarOpcionesIncorporacionB2(ctx, "expediente:ejercicio:ct:0001"); !errors.Is(e, ct.ErrPreparacionIncorporacionPendiente) {
		t.Fatalf("la unidad desigual del expediente pasó sin su comprobación: %v", e)
	}
	if detalle.llamadas != 1 {
		t.Fatalf("la comprobación de unidad volvió a leer el detalle: %d lecturas", detalle.llamadas)
	}
}

func TestIncorporacionB2PlanNominalNoRepiteDetalleExpediente(t *testing.T) {
	f, detalle, ctx, refs := escenarioFuentesIncorporacionB2Detalle(t, "POST")
	actor, e := f.autoridad.actor(ctx, ct.AccionRegistrarPlanNominalB2)
	if e != nil {
		t.Fatal(e)
	}
	s := ct.SolicitudPlanNominalB2{OrganizacionRef: refs.OrganizacionRef, ExpedienteRef: "expediente:ejercicio:ct:0001", VersionExpediente: 8}
	if _, e := f.ResolverPlanNominalB2(ctx, s, actor); !errors.Is(e, ct.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("el plan nominal no llegó a autorizar sus antecedentes: %v", e)
	}
	if detalle.llamadas != 1 {
		t.Fatalf("el detalle del expediente se leyó %d veces en la misma petición", detalle.llamadas)
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

type vacantesPreparacionB2Prueba func(context.Context, personal.SolicitudVacantesB2) (pp.ResultadoVacantesB2, error)

func (v vacantesPreparacionB2Prueba) ConsultarVacantes(ctx context.Context, q personal.SolicitudVacantesB2) (pp.ResultadoVacantesB2, error) {
	return v(ctx, q)
}

type anclajePreparacionB2Prueba func(context.Context, bp.SolicitudConsultaAnclajeAceptacionCT) (bp.ResultadoConsultaAnclajeAceptacionCT, error)

func (a anclajePreparacionB2Prueba) ConsultarAnclajeAceptacionCT(ctx context.Context, q bp.SolicitudConsultaAnclajeAceptacionCT) (bp.ResultadoConsultaAnclajeAceptacionCT, error) {
	return a(ctx, q)
}

func escenarioProyeccionPreparacionB2(t *testing.T) (*fuentesIncorporacionPersonalB2, context.Context, ct.ContratoPlanNominalB2, httpct.ProyeccionIncorporacionPersonalB2HTTP, *lectorAsignacionB2ContadoPrueba, *lectorDetalleIncorporacionB2Prueba) {
	t.Helper()
	a, ctx, lector := escenarioCapturaAsignacionB2Prueba(t)
	if _, err := a.actor(ctx, ct.AccionRegistrarPlanNominalB2); err != nil {
		t.Fatal(err)
	}
	detalle := &lectorDetalleIncorporacionB2Prueba{err: errors.New("lectura completa del detalle")}
	f := &fuentesIncorporacionPersonalB2{autoridad: a, organizacionRef: "organizacion:prueba", organismoRef: "organismo:prueba", reloj: a.reloj, detalle: detalle}
	m := materialPreparacionB2Prueba(t)
	m.OrganizacionRef, m.OrganismoRef = f.organizacionRef, f.organismoRef
	m.ExpedienteRef = "expediente:prueba"
	d := ct.DetalleExpedienteRRHH{Resumen: ct.ResumenExpedienteRRHH{Version: m.VersionExpediente}, Solicitud: ct.SolicitudOperativaRRHH{MotivoClave: domct.ClaveCatalogo(m.MotivoClave)}, Analisis: &ct.AnalisisOperativoRRHH{PeriodoInicio: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)}}
	antecedentes := ct.AntecedentesPlanNominalB2{ExpedienteRef: m.ExpedienteRef, VersionExpediente: m.VersionExpediente, CategoriaRef: m.CategoriaRef, DocumentoRef: m.DocumentoRef, DocumentoSHA256: m.DocumentoSHA256, AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis", AnalisisSHA256: strings.Repeat("b", 64), AceptacionRef: m.AceptacionRef, AceptacionReciboRef: m.AceptacionReciboRef, Bolsa: m.Bolsa}
	p := proyeccionCTIncorporacionB2(m.ExpedienteRef, d, antecedentes)
	f.vacantes = vacantesPreparacionB2Prueba(func(_ context.Context, q personal.SolicitudVacantesB2) (pp.ResultadoVacantesB2, error) {
		if q.Actor.PerfilActivoRef != a.perfiles.b2[personal.AccionVacantesB2].perfilRef() || q.Limite != 100 || string(q.Corte.VigenteEn) != m.Desde {
			t.Fatal("selector nominal de vacantes cambiado")
		}
		return pp.ResultadoVacantesB2{Pagina: personal.PaginaVacantesB2{Vacantes: []personal.VacantePlazaB2{{PlazaRef: "plaza:una", PuestoRef: "puesto:uno", VersionRPTRef: "rpt:uno", CodigoPlazaFuente: "001", PuestoDenominacion: "Puesto uno"}, {PlazaRef: "plaza:dos", PuestoRef: "puesto:dos", VersionRPTRef: "rpt:dos", CodigoPlazaFuente: "002", PuestoDenominacion: "Puesto dos"}}}}, nil
	})
	f.catalogos = catalogosIncorporacionB2Prueba(func(_ context.Context, q personal.SolicitudConsultaCatalogoEmpleadoB2) (pp.ResultadoConsultaCatalogoEmpleadoB2, error) {
		return pp.ResultadoConsultaCatalogoEmpleadoB2{Entradas: []personal.EntradaCatalogoRegistroEmpleadoB2{{Ref: q.Tipo + ":uno", Version: 1, Denominacion: "Uno", VigenteDesde: "2026-01-01"}, {Ref: q.Tipo + ":dos", Version: 2, Denominacion: "Dos", VigenteDesde: "2026-01-01"}}}, nil
	})
	f.clases = &clasesIncorporacionB2Prueba{resultado: pp.ResultadoClasesOcupacionCT{Catalogo: personal.CatalogoClasesOcupacionCT{Ref: "catalogo:clases", Version: 3, HuellaSHA256: strings.Repeat("c", 64), Opciones: []personal.OpcionClaseOcupacionCT{{Valor: "temporal", TextoClave: "personal.clases.temporal"}, {Valor: "sustitucion", TextoClave: "personal.clases.sustitucion"}}}}}
	f.anclaje = anclajePreparacionB2Prueba(func(_ context.Context, q bp.SolicitudConsultaAnclajeAceptacionCT) (bp.ResultadoConsultaAnclajeAceptacionCT, error) {
		if q.Selector.LlamamientoRef != m.Bolsa.LlamamientoRef {
			t.Fatal("anclaje cambió selector")
		}
		return bp.ResultadoConsultaAnclajeAceptacionCT{Estado: "pendiente"}, nil
	})
	if err := f.capturarPreparacionB2(ctx, m, p); err != nil {
		t.Fatal(err)
	}
	completo, err := f.completarOpcionesIncorporacionB2(ctx, copiarProyeccionPreparacionB2(p), antecedentes)
	if err != nil {
		t.Fatal(err)
	}
	return f, ctx, ct.ContratoPlanNominalB2{Material: m}, completo, lector, detalle
}

func TestIncorporacionB2PreparacionReutilizaCTYConservaOpcionesCompletas(t *testing.T) {
	f, ctx, c, esperada, _, detalle := escenarioProyeccionPreparacionB2(t)
	for range 2 { // La misma captura también sirve tras el replay del registro nominal.
		r, err := f.opcionesTrasPreparacionB2(ctx, c)
		if err != nil || !reflect.DeepEqual(r, esperada) {
			t.Fatalf("respuesta distinta: %+v err=%v", r, err)
		}
		if len(r.Opciones.Vacantes) != 2 || len(r.Opciones.Regimenes) != 2 || len(r.Opciones.Modalidades) != 2 || len(r.Opciones.ClasesOcupacion) != 2 {
			t.Fatal("se sustituyó lista por selección")
		}
		r.Prerrequisitos[0].Cumplido = true
		r.Opciones.Motivos[0] = "ajeno"
	}
	if detalle.llamadas != 0 {
		t.Fatalf("repitió detalle %d veces", detalle.llamadas)
	}
}

func TestIncorporacionB2PreparacionUsaConsultaCompletaSiCapturaNoCoincide(t *testing.T) {
	for _, caso := range []string{"ausente", "peticion", "fuente", "expediente", "version", "selector", "actor", "sin_holder"} {
		t.Run(caso, func(t *testing.T) {
			f, ctx, c, _, _, detalle := escenarioProyeccionPreparacionB2(t)
			holder := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
			switch caso {
			case "ausente":
				holder.preparacionB2 = nil
			case "peticion":
				ctx = context.WithValue(ctx, claveContextosNominalesIncorporacion{}, &capturaContextosNominalesIncorporacion{})
			case "fuente":
				otra := *f
				f = &otra
			case "expediente":
				c.Material.ExpedienteRef = "expediente:otro"
			case "version":
				c.Material.VersionExpediente++
			case "selector":
				c.Material.Bolsa.LlamamientoRef = "llamamiento:otro"
			case "actor":
				holder.preparacionB2.identidad.PrincipalID = "persona:otra"
			case "sin_holder":
				ctx = context.Background()
			}
			_, err := f.opcionesTrasPreparacionB2(ctx, c)
			if !errors.Is(err, detalle.err) || detalle.llamadas != 1 {
				t.Fatalf("fallback incompleto: lecturas=%d err=%v", detalle.llamadas, err)
			}
		})
	}
}

func TestIncorporacionB2PreparacionRevalidaRevocacionYFalloEnMismaPeticion(t *testing.T) {
	for _, caso := range []string{"detalle_revocado", "plan_revocado", "fuente_caida", "cancelada", "vacantes_caidas", "catalogo_denegado", "clases_caidas", "anclaje_caido"} {
		t.Run(caso, func(t *testing.T) {
			f, ctx, c, _, lector, detalle := escenarioProyeccionPreparacionB2(t)
			// Los dos perfiles ya están capturados cuando cambia la fuente.
			for _, accion := range []string{ct.AccionConsultarDetalleRRHH, ct.AccionLeerPlanNominalB2} {
				if _, err := f.autoridad.actor(ctx, accion); err != nil {
					t.Fatal(err)
				}
			}
			esperado := ct.ErrAutorizacionDenegada
			switch caso {
			case "detalle_revocado", "plan_revocado":
				accion := ct.AccionLeerPlanNominalB2
				if caso == "detalle_revocado" {
					accion = ct.AccionConsultarDetalleRRHH
				}
				perfil := f.autoridad.perfiles.b2[accion]
				publicada := lector.base.publicadas[perfil.perfilRef()]
				publicada.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(publicada.instantanea)
				publicada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
				lector.base.publicadas[perfil.perfilRef()] = publicada
			case "fuente_caida":
				lector.base.errorLectura = errors.New("caida")
				esperado = ct.ErrConsultaRRHHNoDisponible
			case "cancelada":
				var cancelar context.CancelFunc
				ctx, cancelar = context.WithCancel(ctx)
				cancelar()
				esperado = context.Canceled
			case "vacantes_caidas":
				esperado = personal.ErrRegistroEmpleadoB2NoDisponible
				f.vacantes = vacantesPreparacionB2Prueba(func(context.Context, personal.SolicitudVacantesB2) (pp.ResultadoVacantesB2, error) {
					return pp.ResultadoVacantesB2{}, esperado
				})
			case "catalogo_denegado":
				esperado = personal.ErrRegistroEmpleadoB2Denegado
				f.catalogos = catalogosIncorporacionB2Prueba(func(context.Context, personal.SolicitudConsultaCatalogoEmpleadoB2) (pp.ResultadoConsultaCatalogoEmpleadoB2, error) {
					return pp.ResultadoConsultaCatalogoEmpleadoB2{}, esperado
				})
			case "clases_caidas":
				esperado = personal.ErrRegistroEmpleadoB2NoDisponible
				f.clases.(*clasesIncorporacionB2Prueba).err = esperado
			case "anclaje_caido":
				esperado = bp.ErrConsultaPersonaAceptacionCTNoDisponible
				f.anclaje = anclajePreparacionB2Prueba(func(context.Context, bp.SolicitudConsultaAnclajeAceptacionCT) (bp.ResultadoConsultaAnclajeAceptacionCT, error) {
					return bp.ResultadoConsultaAnclajeAceptacionCT{}, esperado
				})
			}
			r, err := f.opcionesTrasPreparacionB2(ctx, c)
			if !errors.Is(err, esperado) || !reflect.ValueOf(r).IsZero() || detalle.llamadas != 0 {
				t.Fatalf("captura evitó control o devolvió parcial: %+v err=%v", r, err)
			}
		})
	}
}

func materialPreparacionB2Prueba(t *testing.T) domct.PlanIncorporacionPersonalB2 {
	t.Helper()
	h := strings.Repeat("a", 64)
	fuente := domct.FuentePlanPersonalB2{Ref: "fuente:original", Version: 1, SHA256: h}
	m := domct.PlanIncorporacionPersonalB2{
		OrganizacionRef: "org:uno", UnidadCTRef: "unidad:rrhh", ExpedienteRef: "expediente:uno", VersionExpediente: 8,
		AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis", AnalisisSHA256: h, PropuestaReciboRef: "recibo:propuesta",
		AceptacionRef: "aceptacion:uno", AceptacionReciboRef: "recibo:aceptacion",
		Bolsa: domct.SelectorBolsaPlanB2{UnidadRef: "unidad:uno", CategoriaRef: "categoria:uno", NecesidadRef: "necesidad:uno",
			AceptacionOperacionRef: "operacion:aceptacion", AceptacionRegistroSHA256: h, AperturaOperacionRef: "operacion:apertura",
			AperturaRegistroSHA256: h, LlamamientoRef: "llamamiento:uno", PropuestaRef: "propuesta:uno"},
		PersonaRef: "per_aaaaaaaaaaaaaaaaaaaaaaaa", PersonaVersion: 1, PersonaFuente: fuente, PersonaReciboBolsaRef: "recibo:bolsa",
		OrganismoRef: "organismo:uno", UnidadRef: "unidad:uno", FuenteOrganizacion: domct.FuenteSinVersionPlanB2{Ref: fuente.Ref, SHA256: h},
		FuentePlantilla: domct.InstrumentoPlanPersonalB2{Ref: "11111111-1111-4111-8111-111111111111", Revision: 1, FuenteRef: fuente.Ref, FuenteSHA256: h},
		FuenteRPT:       domct.InstrumentoPlanPersonalB2{Ref: "22222222-2222-4222-8222-222222222222", Revision: 1, FuenteRef: fuente.Ref, FuenteSHA256: h},
		VersionPlazaRef: "plantilla:uno", VersionPuestoRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 1,
		PuestoRef: "puesto:uno", PlazaRef: "plaza:uno", CatalogoRPTID: "catalogo:rpt", CatalogoRPTModulo: "personal",
		CatalogoRPTVersion: 1, CatalogoRPTSHA256: h, CategoriaRef: "categoria:uno", VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo",
		Regimen: domct.EntradaPlanPersonalB2{Ref: "regimen:uno", Version: 1}, Modalidad: domct.EntradaPlanPersonalB2{Ref: "modalidad:uno", Version: 1},
		ClaseOcupacion: "temporal", Desde: "2026-09-30", MotivoClave: "incorporacion", DocumentoRef: "documento:uno", DocumentoSHA256: h,
		EjercicioSintetico: true,
	}
	if err := m.Validar(); err != nil {
		t.Fatalf("plan de prueba inválido: %v", err)
	}
	return m
}
