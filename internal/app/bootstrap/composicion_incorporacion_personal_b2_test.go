package bootstrap

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type servicioCTFacadeB2Prueba struct {
	contrato             ct.ContratoPlanNominalB2
	err                  error
	lecturas, mutaciones int
}

func TestIncorporacionB2HechosRechazanVersionesFirmadasInvalidas(t *testing.T) {
	recibo := "recibo:rpt"
	base := inc.ResultadoConsumidorPersonalB2{
		Estado: pp.EstadoPlanIncorporacionCT{Plan: personal.PlanIncorporacionCT{Version: 1}, ReciboAltaRelacion: &pp.ReciboActoRegistroEmpleadoB2{}, ReciboOcupacion: &pp.ReciboActoRegistroEmpleadoB2{}},
		Hechos: pp.HechosIncorporacionCT{Seleccion: pp.SeleccionHechosIncorporacionCT{VersionRelacion: 1, VersionOcupacion: 1}},
		Uso:    vp.ResultadoUsoCategoriaRPT{Encontrado: true, Uso: &vp.UsoCategoriaRPT{Estado: "confirmado", TerminalReciboRef: &recibo}},
	}
	for nombre, cambiar := range map[string]func(*inc.ResultadoConsumidorPersonalB2){
		"plan":      func(r *inc.ResultadoConsumidorPersonalB2) { r.Estado.Plan.Version = -1 },
		"relacion":  func(r *inc.ResultadoConsumidorPersonalB2) { r.Hechos.Seleccion.VersionRelacion = 0 },
		"ocupacion": func(r *inc.ResultadoConsumidorPersonalB2) { r.Hechos.Seleccion.VersionOcupacion = math.MinInt64 },
	} {
		r := base
		cambiar(&r)
		if _, err := hechosCTDesdePersonalB2(r); !errors.Is(err, httpct.ErrManejadorIncorporacionPersonalB2) {
			t.Fatalf("%s: versión inválida admitida: %v", nombre, err)
		}
	}
	base.Estado.Plan.Version = math.MaxInt64
	base.Hechos.Seleccion.VersionRelacion = math.MaxInt64
	base.Hechos.Seleccion.VersionOcupacion = math.MaxInt64
	h, err := hechosCTDesdePersonalB2(base)
	if err != nil || h.PersonalPlanVersion != math.MaxInt64 || h.RelacionVersion != math.MaxInt64 || h.OcupacionVersion != math.MaxInt64 {
		t.Fatalf("límite válido alterado: %+v, %v", h, err)
	}
}

func (s *servicioCTFacadeB2Prueba) LeerContratoPlanNominal(context.Context, string, string) (ct.ContratoPlanNominalB2, error) {
	s.lecturas++
	return s.contrato, s.err
}
func (s *servicioCTFacadeB2Prueba) LeerOrigenIncorporacionB2(context.Context, string, string) (ct.OrigenIncorporacionPersonalB2, bool, error) {
	s.lecturas++
	return ct.OrigenIncorporacionPersonalB2{}, false, s.err
}
func (s *servicioCTFacadeB2Prueba) RegistrarPlanNominalB2(context.Context, ct.SolicitudPlanNominalB2, core.ContextoActor) (ct.ContratoPlanNominalB2, error) {
	s.mutaciones++
	return s.contrato, s.err
}
func (s *servicioCTFacadeB2Prueba) ConfirmarOrigenIncorporacionB2(context.Context, ct.ConfirmacionOrigenIncorporacionB2, core.ContextoActor) (ct.OrigenIncorporacionPersonalB2, error) {
	s.mutaciones++
	return ct.OrigenIncorporacionPersonalB2{}, s.err
}

type opcionesFacadeB2Prueba struct {
	err      error
	lecturas int
}

func (s *opcionesFacadeB2Prueba) ConsultarOpcionesIncorporacionB2(context.Context, string) (httpct.ProyeccionIncorporacionPersonalB2HTTP, error) {
	s.lecturas++
	return httpct.ProyeccionIncorporacionPersonalB2HTTP{VersionExpedienteActual: 8}, s.err
}
func TestIncorporacionB2GETSinPlanNoEjecutaPreparacion(t *testing.T) {
	repo := &servicioCTFacadeB2Prueba{err: ct.ErrPlanNominalB2NoEncontrado}
	op := &opcionesFacadeB2Prueba{}
	// Fábrica y ejecutor ausentes: consultarlos en GET provocaría un fallo.
	f := &fachadaIncorporacionPersonalB2{ct: repo, opciones: op, organizacionRef: "organizacion:prueba"}
	r, e := f.Consultar(context.Background(), "expediente:prueba")
	if e != nil || r.Estado != "sin_plan" || r.Plan != nil || r.Recibo != nil || repo.mutaciones != 0 || repo.lecturas != 1 || op.lecturas != 1 {
		t.Fatalf("GET alteró o ejecutó preparación: %+v err=%v", r, e)
	}
}
func TestIncorporacionB2ConfirmacionAjenaFrenaAntesDePersonal(t *testing.T) {
	repo := &servicioCTFacadeB2Prueba{contrato: ct.ContratoPlanNominalB2{PlanRef: "plan:original", PlanVersion: 1, Solicitud: ct.SolicitudPlanNominalB2{ClaveIdempotencia: "11111111-1111-4111-8111-111111111111"}}}
	f := &fachadaIncorporacionPersonalB2{ct: repo, organizacionRef: "organizacion:prueba"}
	for _, entrada := range []httpct.EntradaConfirmacionB2{
		{ExpedienteRef: "expediente:prueba", PlanRef: "plan:ajeno", VersionPlan: 1, ClaveIdempotencia: repo.contrato.Solicitud.ClaveIdempotencia},
		{ExpedienteRef: "expediente:prueba", PlanRef: repo.contrato.PlanRef, VersionPlan: 2, ClaveIdempotencia: repo.contrato.Solicitud.ClaveIdempotencia},
		{ExpedienteRef: "expediente:prueba", PlanRef: repo.contrato.PlanRef, VersionPlan: 1, ClaveIdempotencia: "22222222-2222-4222-8222-222222222222"},
	} {
		_, e := f.Confirmar(context.Background(), entrada)
		if !errors.Is(e, httpct.ErrConflictoIncorporacionPersonalB2) || repo.mutaciones != 0 {
			t.Fatalf("confirmación distinta alcanzó Personal: %v", e)
		}
	}
}
func TestIncorporacionB2DependenciaCaidaNoPareceDenegacion(t *testing.T) {
	for _, e := range []error{ct.ErrConsultaRRHHNoDisponible, ct.ErrPlanNominalB2NoDisponible, bp.ErrConsultaPersonaAceptacionCTNoDisponible, errors.New("dependencia caída"), errors.Join(ct.ErrAutorizacionDenegada, vp.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible)} {
		r := errorHTTPNominalB2(context.Background(), e)
		if !errors.Is(r, httpct.ErrManejadorIncorporacionPersonalB2) || errors.Is(r, httpct.ErrDenegadaIncorporacionPersonalB2) {
			t.Fatalf("caída clasificada como permiso: %v", r)
		}
	}
	for _, e := range []error{ct.ErrAutorizacionDenegada, ct.ErrDenegadaIncorporacionAplicacion, bp.ErrConsultaPersonaAceptacionCTDenegada, core.ErrAutorizacionDenegada, core.ErrPermissionDenied, vp.ErrDenegacionExplicitaAutorizacionLigadaV3} {
		if r := errorHTTPNominalB2(context.Background(), e); !errors.Is(r, httpct.ErrDenegadaIncorporacionPersonalB2) {
			t.Fatalf("denegación oculta: %v", r)
		}
	}
}

// El servicio real de aplicación conserva su validación, canon y replay;
// sólo sus fuentes y repositorio se sustituyen por el escenario sintético.
type fuentesRegistroPreparacionB2Prueba struct {
	ct.FuentePreparacionPlanNominalB2
	fuente     *fuentesIncorporacionPersonalB2
	material   domct.PlanIncorporacionPersonalB2
	proyeccion httpct.ProyeccionIncorporacionPersonalB2HTTP
	capturar   bool
}

func (s *fuentesRegistroPreparacionB2Prueba) ResolverPlanNominalB2(ctx context.Context, _ ct.SolicitudPlanNominalB2, _ core.ContextoActor) (domct.PlanIncorporacionPersonalB2, error) {
	if s.capturar {
		s.fuente.capturarPreparacionB2(ctx, s.material, s.proyeccion)
	}
	return s.material, nil
}

type autoridadRegistroPreparacionB2Prueba struct{}

func (autoridadRegistroPreparacionB2Prueba) AutorizarPlanNominalB2(context.Context, string, []byte, core.ContextoActor) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

type repositorioRegistroPreparacionB2Prueba struct {
	ct.RepositorioPlanNominalB2
	contrato  ct.ContratoPlanNominalB2
	registros int
	ahora     func() time.Time
}

func (r *repositorioRegistroPreparacionB2Prueba) RegistrarPlanNominalB2(_ context.Context, m ct.RegistroPlanNominalB2, _ vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ct.ContratoPlanNominalB2, error) {
	r.registros++
	if r.contrato.PlanRef == "" {
		h, _ := domct.SHA256PlanPersonalB2(m)
		r.contrato = ct.ContratoPlanNominalB2{Protocolo: ct.ProtocoloIncorporacionPersonalB2, Material: m.Material, Solicitud: m.Solicitud, PlanRef: "plan:prueba", PlanVersion: 1, PlanReciboRef: "recibo:plan", PlanSHA256: h, IntencionRef: "intencion:prueba", IntencionVersion: 1, IntencionReciboRef: "recibo:intencion", SolicitudPersonalRef: "solicitud:personal", IdempotenciaPersonalUUID: "33333333-3333-4333-8333-333333333333", RegistradoEn: r.ahora().UTC()}
	}
	return r.contrato, nil
}

func TestIncorporacionB2PrepararConServicioRealActivaCapturaYReplay(t *testing.T) {
	fuente, ctx, c, esperada, _, detalle := escenarioProyeccionPreparacionB2(t)
	holder := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	origen := &fuentesRegistroPreparacionB2Prueba{fuente: fuente, material: c.Material, proyeccion: copiarProyeccionPreparacionB2(holder.preparacionB2.proyeccion), capturar: true}
	holder.preparacionB2 = nil
	repo := &repositorioRegistroPreparacionB2Prueba{ahora: fuente.reloj.Ahora}
	servicio, err := appct.NuevoServicioPlanNominalB2(origen, autoridadRegistroPreparacionB2Prueba{}, repo)
	if err != nil {
		t.Fatal(err)
	}
	fuente.servicioCT = servicio
	fachada := &fachadaIncorporacionPersonalB2{organizacionRef: fuente.organizacionRef, ct: servicio, opciones: fuente, fuentes: fuente, autoridad: fuente.autoridad}
	entrada := planHTTPIncorporacionB2(ct.ContratoPlanNominalB2{Material: c.Material, Solicitud: ct.SolicitudPlanNominalB2{ClaveIdempotencia: "44444444-4444-4444-8444-444444444444"}}).Intencion
	var primera httpct.ProyeccionIncorporacionPersonalB2HTTP
	for i := range 2 {
		r, err := fachada.Preparar(ctx, entrada)
		if err != nil {
			t.Fatal(err)
		}
		esperada.Estado = "plan_preparado"
		esperada.Plan = planHTTPIncorporacionB2(repo.contrato)
		if !reflect.DeepEqual(r, esperada) || r.Plan.Intencion != entrada {
			t.Fatalf("respuesta de preparación distinta: %+v", r)
		}
		if i == 0 {
			primera = r
		} else if !reflect.DeepEqual(primera, r) {
			t.Fatal("replay cambió respuesta o recibo")
		}
	}
	if repo.registros != 2 || detalle.llamadas != 0 {
		t.Fatalf("no activó reutilización en instancias reales: registros=%d detalle=%d", repo.registros, detalle.llamadas)
	}
	// Un registro alternativo que no captura conserva la consulta completa.
	origen.capturar = false
	holder.preparacionB2 = nil
	_, err = fachada.Preparar(ctx, entrada)
	if !errors.Is(err, detalle.err) || detalle.llamadas != 1 {
		t.Fatalf("ausencia de captura bloqueó fallback completo: %v", err)
	}
	// Unas opciones externas nunca reciben la captura de la fuente nominal.
	op := &opcionesFacadeB2Prueba{}
	fachada.opciones = op
	if _, err = fachada.Preparar(ctx, entrada); err != nil || op.lecturas != 1 {
		t.Fatalf("opciones alternativas no conservaron su servicio: %v", err)
	}
}
