package bootstrap

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
	pgbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	fuentePersonal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	pgpersonal "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	apppersonal "vec-diputacion-granada/internal/modules/personal/application"
	httpapi "vec-diputacion-granada/internal/vec/adapters/httpapi"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type servicioCTIncorporacionB2 interface {
	RegistrarPlanNominalB2(context.Context, ct.SolicitudPlanNominalB2, core.ContextoActor) (ct.ContratoPlanNominalB2, error)
	LeerContratoPlanNominal(context.Context, string, string) (ct.ContratoPlanNominalB2, error)
	LeerOrigenIncorporacionB2(context.Context, string, string) (ct.OrigenIncorporacionPersonalB2, bool, error)
	ConfirmarOrigenIncorporacionB2(context.Context, ct.ConfirmacionOrigenIncorporacionB2, core.ContextoActor) (ct.OrigenIncorporacionPersonalB2, error)
}
type fuenteOpcionesIncorporacionB2 interface {
	ConsultarOpcionesIncorporacionB2(context.Context, string) (httpct.ProyeccionIncorporacionPersonalB2HTTP, error)
}
type consumidorEfectivoPersonalB2 interface {
	ConsultarPersonalB2(context.Context, string, string) (pp.EstadoPlanIncorporacionCT, error)
	ConfirmarPersonalB2(context.Context, string, string) (inc.ResultadoConsumidorPersonalB2, error)
}

// La fachada conecta el transporte nominal con las autoridades propietarias.
// Sólo Confirmar usa la fábrica que puede reservar y ejecutar Personal.
type fachadaIncorporacionPersonalB2 struct {
	organizacionRef string
	ct              servicioCTIncorporacionB2
	opciones        fuenteOpcionesIncorporacionB2
	planes          *inc.PlanesNominales
	personal        consumidorEfectivoPersonalB2
	autoridad       *autoridadIncorporacionPersonalB2
	fuentes         *fuentesIncorporacionPersonalB2
}

func (f *fachadaIncorporacionPersonalB2) Consultar(ctx context.Context, exp string) (httpct.ProyeccionIncorporacionPersonalB2HTTP, error) {
	var cero httpct.ProyeccionIncorporacionPersonalB2HTTP
	if f == nil || ctx == nil {
		return cero, httpct.ErrManejadorIncorporacionPersonalB2
	}
	p, e := f.opciones.ConsultarOpcionesIncorporacionB2(ctx, exp)
	if e != nil {
		return cero, errorHTTPNominalB2(ctx, e)
	}
	c, e := f.ct.LeerContratoPlanNominal(ctx, f.organizacionRef, exp)
	if errors.Is(e, ct.ErrPlanNominalB2NoEncontrado) {
		p.Estado = "sin_plan"
		return p, nil
	}
	if e != nil {
		return cero, errorHTTPNominalB2(ctx, e)
	}
	if e = f.fuentes.validarFuentesContratoB2(ctx, c); e != nil {
		return cero, errorHTTPNominalB2(ctx, e)
	}
	p.Plan = planHTTPIncorporacionB2(c)
	p.Estado = "plan_preparado"
	o, encontrado, e := f.ct.LeerOrigenIncorporacionB2(ctx, f.organizacionRef, exp)
	if e != nil {
		return cero, errorHTTPNominalB2(ctx, e)
	}
	if encontrado {
		actor, e := f.autoridad.actor(ctx, ct.AccionLeerPlanNominalB2)
		if e != nil {
			return cero, errorHTTPNominalB2(ctx, e)
		}
		if _, e = f.fuentes.VerificarHechosPersonalB2(ctx, c, o.Confirmacion.Hechos, actor); e != nil {
			return cero, errorHTTPNominalB2(ctx, e)
		}
		r := reciboHTTPIncorporacionB2(o)
		p.Estado = "incorporacion_confirmada"
		p.Recibo = &r
	}
	return p, nil
}
func (f *fachadaIncorporacionPersonalB2) Preparar(ctx context.Context, e httpct.EntradaPlanB2) (httpct.ProyeccionIncorporacionPersonalB2HTTP, error) {
	var cero httpct.ProyeccionIncorporacionPersonalB2HTTP
	if f == nil || ctx == nil || e.Validar() != nil {
		return cero, httpct.ErrPeticionIncorporacionPersonalB2
	}
	actor, err := f.autoridad.actor(ctx, ct.AccionRegistrarPlanNominalB2)
	if err != nil {
		return cero, errorHTTPNominalB2(ctx, err)
	}
	c, err := f.ct.RegistrarPlanNominalB2(ctx, ct.SolicitudPlanNominalB2{OrganizacionRef: f.organizacionRef, ExpedienteRef: e.ExpedienteRef, VersionExpediente: e.VersionExpediente, PuestoRef: e.PuestoRef, PlazaRef: e.PlazaRef, VersionPlantillaRef: e.VersionPlantillaRef, VersionRPTRef: e.VersionRPTRef, Regimen: domct.EntradaPlanPersonalB2{Ref: e.Regimen.Ref, Version: e.Regimen.Version}, Modalidad: domct.EntradaPlanPersonalB2{Ref: e.Modalidad.Ref, Version: e.Modalidad.Version}, ClaseOcupacion: e.ClaseOcupacion, Desde: e.Desde, Hasta: e.Hasta, MotivoClave: e.MotivoClave, DocumentoRef: e.DocumentoRef, DocumentoSHA256: e.DocumentoSHA256, ClaveIdempotencia: e.ClaveIdempotencia}, actor)
	if err != nil {
		return cero, errorHTTPNominalB2(ctx, err)
	}
	p, err := f.opciones.ConsultarOpcionesIncorporacionB2(ctx, e.ExpedienteRef)
	if err != nil {
		return cero, errorHTTPNominalB2(ctx, err)
	}
	p.Estado = "plan_preparado"
	p.Plan = planHTTPIncorporacionB2(c)
	return p, nil
}
func (f *fachadaIncorporacionPersonalB2) Confirmar(ctx context.Context, e httpct.EntradaConfirmacionB2) (httpct.ReciboIncorporacionPersonalB2HTTP, error) {
	var cero httpct.ReciboIncorporacionPersonalB2HTTP
	if f == nil || ctx == nil || e.Validar() != nil {
		return cero, httpct.ErrPeticionIncorporacionPersonalB2
	}
	c, err := f.ct.LeerContratoPlanNominal(ctx, f.organizacionRef, e.ExpedienteRef)
	if err != nil {
		return cero, errorHTTPNominalB2(ctx, err)
	}
	if c.PlanRef != e.PlanRef || c.PlanVersion != e.VersionPlan || c.Solicitud.ClaveIdempotencia != e.ClaveIdempotencia {
		return cero, httpct.ErrConflictoIncorporacionPersonalB2
	}
	// La clave se coteja con la intención original antes de cualquier reserva.
	plan, err := f.planes.Preparar(ctx, f.organizacionRef, e.ExpedienteRef)
	if err != nil {
		return cero, errorHTTPNominalB2(ctx, err)
	}
	if plan.Protocolo != inc.ProtocoloPersonalB2V1 || plan.Personal == nil {
		return cero, httpct.ErrConflictoIncorporacionPersonalB2
	}
	propio := plan.Personal.Reserva.Plan
	resultado, err := f.personal.ConfirmarPersonalB2(ctx, propio.PlanRef, propio.Datos.OrganismoRef)
	if err != nil {
		return cero, errorHTTPNominalB2(ctx, err)
	}
	hechos, err := hechosCTDesdePersonalB2(resultado)
	if err != nil {
		return cero, err
	}
	actor, err := f.autoridad.actor(ctx, ct.AccionConfirmarOrigenB2)
	if err != nil {
		return cero, errorHTTPNominalB2(ctx, err)
	}
	o, err := f.ct.ConfirmarOrigenIncorporacionB2(ctx, ct.ConfirmacionOrigenIncorporacionB2{OrganizacionRef: f.organizacionRef, UnidadCTRef: c.Material.UnidadCTRef, ExpedienteRef: e.ExpedienteRef, PlanRef: c.PlanRef, PlanVersion: c.PlanVersion, PlanSHA256: c.PlanSHA256, Hechos: hechos}, actor)
	if err != nil {
		return cero, errorHTTPNominalB2(ctx, err)
	}
	return reciboHTTPIncorporacionB2(o), nil
}
func planHTTPIncorporacionB2(c ct.ContratoPlanNominalB2) *httpct.PlanIncorporacionPersonalB2HTTP {
	m := c.Material
	return &httpct.PlanIncorporacionPersonalB2HTTP{PlanRef: c.PlanRef, Version: c.PlanVersion, SHA256: c.PlanSHA256, Intencion: httpct.EntradaPlanB2{ExpedienteRef: m.ExpedienteRef, VersionExpediente: m.VersionExpediente, PuestoRef: m.PuestoRef, PlazaRef: m.PlazaRef, VersionPlantillaRef: m.VersionPlazaRef, VersionRPTRef: m.VersionPuestoRef, Regimen: httpct.EntradaCatalogoB2{Ref: m.Regimen.Ref, Version: m.Regimen.Version}, Modalidad: httpct.EntradaCatalogoB2{Ref: m.Modalidad.Ref, Version: m.Modalidad.Version}, ClaseOcupacion: m.ClaseOcupacion, Desde: m.Desde, Hasta: m.Hasta, MotivoClave: m.MotivoClave, DocumentoRef: m.DocumentoRef, DocumentoSHA256: m.DocumentoSHA256, ClaveIdempotencia: c.Solicitud.ClaveIdempotencia}}
}
func reciboHTTPIncorporacionB2(o ct.OrigenIncorporacionPersonalB2) httpct.ReciboIncorporacionPersonalB2HTTP {
	m := o.Confirmacion
	return httpct.ReciboIncorporacionPersonalB2HTTP{Esquema: httpct.EsquemaReciboIncorporacionPersonalB2, ExpedienteRef: m.ExpedienteRef, PlanRef: m.PlanRef, PlanVersion: m.PlanVersion, ReciboRef: o.ReciboRef, RegistradaEn: o.RegistradoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), EmpleadoRef: m.Hechos.EmpleadoRef, RelacionRef: m.Hechos.RelacionRef, OcupacionRef: m.Hechos.OcupacionRef, FirmaOficial: o.FirmaOficial, EficaciaAdministrativa: o.EficaciaAdministrativa}
}
func hechosCTDesdePersonalB2(r inc.ResultadoConsumidorPersonalB2) (ct.HechosPersonalIncorporacionB2, error) {
	p, s := r.Estado.Plan, r.Hechos.Seleccion
	if r.Estado.ReciboAltaRelacion == nil || r.Estado.ReciboOcupacion == nil || !r.Uso.Encontrado || r.Uso.Uso == nil || r.Uso.Uso.Estado != "confirmado" || r.Uso.Uso.TerminalReciboRef == nil {
		return ct.HechosPersonalIncorporacionB2{}, httpct.ErrManejadorIncorporacionPersonalB2
	}
	planVersion, okPlan := versionB2DesdeInt64(p.Version)
	relacionVersion, okRelacion := versionB2DesdeInt64(s.VersionRelacion)
	ocupacionVersion, okOcupacion := versionB2DesdeInt64(s.VersionOcupacion)
	if !okPlan || !okRelacion || !okOcupacion {
		return ct.HechosPersonalIncorporacionB2{}, httpct.ErrManejadorIncorporacionPersonalB2
	}
	a, o := r.Estado.ReciboAltaRelacion, r.Estado.ReciboOcupacion
	h := ct.HechosPersonalIncorporacionB2{ModoPersonal: p.Modo, PersonalPlanRef: p.PlanRef, PersonalPlanVersion: planVersion, PersonalPlanReciboRef: p.ReciboRef, PersonalPlanSHA256: p.HuellaSHA256, EmpleadoRef: s.EmpleadoRef, RelacionRef: s.RelacionRef, RelacionVersion: relacionVersion, RelacionReciboRef: a.ReciboRef, OcupacionRef: s.OcupacionRef, OcupacionVersion: ocupacionVersion, OcupacionReciboRef: o.ReciboRef, RPTConfirmacionRef: p.ConfirmacionRPTRef, RPTReciboRef: *r.Uso.Uso.TerminalReciboRef, SeguimientoRef: r.Estado.EjecucionReciboRef}
	if p.Modo == "alta_empleado" {
		h.AltaReciboRef = a.ReciboRef
	}
	return h, nil
}
func errorHTTPNominalB2(ctx context.Context, e error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	switch {
	case errors.Is(e, ct.ErrConsultaRRHHNoDisponible), errors.Is(e, ct.ErrPlanNominalB2NoDisponible), errors.Is(e, personal.ErrRegistroEmpleadoB2NoDisponible), errors.Is(e, bp.ErrConsultaPersonaAceptacionCTNoDisponible), errors.Is(e, ct.ErrVinculoCategoriaRPTNoDisponible), errors.Is(e, vp.ErrLecturaRPTNoDisponible), errors.Is(e, vp.ErrLecturaRPTNoConfiable), errors.Is(e, vp.ErrFuenteAutorizacionNoDisponible), errors.Is(e, vp.ErrRegistroDecisionNoDisponible), errors.Is(e, vp.ErrRegistroDenegacionNoDisponible), errors.Is(e, vp.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible), errors.Is(e, vp.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible), errors.Is(e, vp.ErrInstantaneaAutorizacionObsoleta):
		return errors.Join(httpct.ErrManejadorIncorporacionPersonalB2, e)
	case errors.Is(e, ct.ErrPlanNominalB2Denegado), errors.Is(e, ct.ErrAutorizacionDenegada), errors.Is(e, ct.ErrDenegadaIncorporacionAplicacion), errors.Is(e, personal.ErrRegistroEmpleadoB2Denegado), errors.Is(e, vp.ErrUsoCategoriaRPTDenegado), errors.Is(e, vp.ErrLecturaRPTDenegada), errors.Is(e, bp.ErrConsultaPersonaAceptacionCTDenegada), errors.Is(e, ct.ErrVinculoCategoriaRPTDenegado), errors.Is(e, core.ErrAutorizacionDenegada), errors.Is(e, core.ErrPermissionDenied), errors.Is(e, vp.ErrDenegacionExplicitaAutorizacionLigadaV3):
		return errors.Join(httpct.ErrDenegadaIncorporacionPersonalB2, e)
	case errors.Is(e, ct.ErrPlanNominalB2Conflicto), errors.Is(e, ct.ErrConflictoIncorporacionAplicacion), errors.Is(e, ct.ErrVinculoCategoriaRPTConflicto), errors.Is(e, personal.ErrRegistroEmpleadoB2Conflicto), errors.Is(e, vp.ErrUsoCategoriaRPTConflicto):
		return errors.Join(httpct.ErrConflictoIncorporacionPersonalB2, e)
	case errors.Is(e, ct.ErrPreparacionIncorporacionPendiente), errors.Is(e, ct.ErrPlanNominalB2NoEncontrado), errors.Is(e, ct.ErrVinculoCategoriaRPTNoEncontrado), errors.Is(e, personal.ErrRegistroEmpleadoB2NoEncontrado):
		return errors.Join(httpct.ErrPreparacionPendienteIncorporacionPersonalB2, e)
	case errors.Is(e, ct.ErrPlanNominalB2Invalido), errors.Is(e, personal.ErrRegistroEmpleadoB2Invalido), errors.Is(e, ct.ErrVinculoCategoriaRPTInvalido):
		return errors.Join(httpct.ErrPeticionIncorporacionPersonalB2, e)
	default:
		return errors.Join(httpct.ErrManejadorIncorporacionPersonalB2, e)
	}
}

type rutaPeticionIncorporacionB2 struct{ metodo, ruta string }
type claveRutaPeticionIncorporacionB2 struct{}

func ligarContextoIncorporacionPersonalB2(h http.Handler, soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		previo := rutaPeticionIncorporacionB2{r.Method, r.URL.Path}
		if ((previo.ruta == httpct.RutaPlanB2 || previo.ruta == httpct.RutaVinculoCategoriaRPTB2) && (previo.metodo == http.MethodGet || previo.metodo == http.MethodPost)) || (previo.ruta == httpct.RutaConfirmacionB2 && previo.metodo == http.MethodPost) || (previo.ruta == httpct.RutaCategoriasRPTB2 && previo.metodo == http.MethodGet) || (previo.ruta == rutaCatalogosRegistroEmpleadoB2 && previo.metodo == http.MethodPost) {
			if ctx, ok := contextoNominalIncorporacionPersonalB2(r.Context(), soporte, fronteras, previo); ok {
				r = r.WithContext(ctx)
			}
		}
		h.ServeHTTP(w, r)
	})
}

// contextoNominalIncorporacionPersonalB2 deriva de la petición autenticada el
// contexto con el que operan los perfiles nominales B2 en la ruta indicada.
func contextoNominalIncorporacionPersonalB2(ctx context.Context, soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo, ruta rutaPeticionIncorporacionB2) (context.Context, bool) {
	if _, ok := soporte.capacidadValida(ctx); !ok {
		return nil, false
	}
	ctx, e := subconsultaDetalleContratacionTemporalDesarrollo(ctx, soporte, fronteras)
	if e != nil {
		return nil, false
	}
	ctx = context.WithValue(ctx, claveIncorporacionV2Desarrollo{}, soporte.sello)
	ctx = context.WithValue(ctx, claveRutaPeticionIncorporacionB2{}, ruta)
	return context.WithValue(ctx, claveContextosNominalesIncorporacion{}, &capturaContextosNominalesIncorporacion{}), true
}
func (a *autoridadIncorporacionPersonalB2) ResolverContextoIncorporacionEjercicioV2(ctx context.Context) error {
	_, e := a.contexto(ctx, ct.AccionLeerPlanNominalB2)
	if e == nil {
		return nil
	}
	return errorHTTPNominalB2(ctx, e)
}

var _ httpct.EjecutorIncorporacionPersonalB2 = (*fachadaIncorporacionPersonalB2)(nil)

type montajeIncorporacionPersonalB2 struct {
	fachada   *fachadaIncorporacionPersonalB2
	autoridad *autoridadIncorporacionPersonalB2
	// cese es nil si la configuración no fija cese_fecha_efecto.
	cese       *inc.CesePersonalB2
	vinculo    *fachadaVinculoCategoriaRPTB2
	categorias *fachadaCategoriasRPTB2
	// gobiernoCatalogo es nil si la configuración no habilita publicar y
	// retirar entradas del catálogo de registro de empleado.
	gobiernoCatalogo *apppersonal.ServicioCatalogosRegistroEmpleadoB2
	cerrar           func()
}

// fachadaVinculoCategoriaRPTB2 reutiliza el ServicioVinculoCategoriaRPT que el
// montaje B2 ya compone con su autoridad nominal; sólo completa la intención
// con la organización y el catálogo RPT de la configuración del servidor.
type fachadaVinculoCategoriaRPTB2 struct {
	organizacionRef, catalogoID, moduloID string
	servicio                              *appct.ServicioVinculoCategoriaRPT
}

func (f *fachadaVinculoCategoriaRPTB2) ConsultarVinculo(ctx context.Context, exp string) (ct.LecturaVinculoCategoriaRPT, error) {
	if f == nil || f.servicio == nil || ctx == nil {
		return ct.LecturaVinculoCategoriaRPT{}, httpct.ErrManejadorIncorporacionPersonalB2
	}
	c, e := ct.NuevaConsultaVinculoCategoriaRPT(f.organizacionRef, exp)
	if e != nil {
		return ct.LecturaVinculoCategoriaRPT{}, errorHTTPNominalB2(ctx, e)
	}
	l, e := f.servicio.Consultar(ctx, c)
	if e != nil {
		return ct.LecturaVinculoCategoriaRPT{}, errorHTTPNominalB2(ctx, e)
	}
	return l, nil
}
func (f *fachadaVinculoCategoriaRPTB2) RegistrarVinculo(ctx context.Context, e httpct.EntradaVinculoCategoriaRPTB2) (ct.ReciboVinculoCategoriaRPT, error) {
	if f == nil || f.servicio == nil || ctx == nil {
		return ct.ReciboVinculoCategoriaRPT{}, httpct.ErrManejadorIncorporacionPersonalB2
	}
	var anterior *string
	if e.AnteriorReciboRef != "" {
		a := e.AnteriorReciboRef
		anterior = &a
	}
	m := ct.RegistroVinculoCategoriaRPT{Esquema: ct.EsquemaRegistroVinculoCategoriaRPT, OrganizacionRef: f.organizacionRef,
		ExpedienteRef: e.ExpedienteRef, VersionExpedienteEsperada: e.VersionExpedienteEsperada, AnalisisVersion: e.AnalisisVersion,
		AnalisisReciboRef: e.AnalisisReciboRef, AnalisisHuellaSHA256: e.AnalisisHuellaSHA256, CategoriaRef: e.CategoriaRef,
		CatalogoID: f.catalogoID, ModuloID: f.moduloID, CatalogoVersion: e.CatalogoVersion, CatalogoHuellaSHA256: e.CatalogoHuellaSHA256,
		CategoriaID: e.CategoriaID, FuenteRef: e.FuenteRef, MotivoRef: e.MotivoRef, AprobacionRef: e.AprobacionRef,
		RevisionEsperada: e.RevisionEsperada, AnteriorReciboRef: anterior, ClaveIdempotencia: e.ClaveIdempotencia}
	r, err := f.servicio.Registrar(ctx, m)
	if err != nil {
		return ct.ReciboVinculoCategoriaRPT{}, errorHTTPNominalB2(ctx, err)
	}
	return r, nil
}

func (m *montajeIncorporacionPersonalB2) rutas(soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo, registrador vp.RegistradorAuditoriaFronteraRutaExacta) ([]httpapi.RutaExacta, error) {
	if m == nil || m.fachada == nil || m.autoridad == nil {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	h, e := httpct.NuevoManejadorIncorporacionPersonalB2(m.autoridad, m.fachada)
	if e != nil {
		return nil, e
	}
	h = ligarContextoIncorporacionPersonalB2(h, soporte, fronteras)
	v, e := httpct.NuevoManejadorVinculoCategoriaRPTB2(m.vinculo)
	if e != nil {
		return nil, e
	}
	v = ligarContextoIncorporacionPersonalB2(v, soporte, fronteras)
	l, e := httpct.NuevoManejadorCategoriasRPTB2(m.categorias)
	if e != nil {
		return nil, e
	}
	l = ligarContextoIncorporacionPersonalB2(l, soporte, fronteras)
	r := []httpapi.RutaExacta{{Ruta: httpct.RutaPlanB2, Manejador: h}, {Ruta: httpct.RutaConfirmacionB2, Manejador: h}, {Ruta: httpct.RutaVinculoCategoriaRPTB2, Manejador: v}, {Ruta: httpct.RutaCategoriasRPTB2, Manejador: l}}
	if m.gobiernoCatalogo != nil {
		c, e := rutaCatalogosEmpleadoB2(m.gobiernoCatalogo, m.autoridad, registrador, soporte, fronteras)
		if e != nil {
			return nil, e
		}
		r = append(r, c)
	}
	return r, nil
}

// Se invoca al cargar el archivo privado, antes de montar los handlers. Cada
// conexión nueva tiene su LOGIN y familia técnica cerrados; nada se provisiona
// en esta función ni al recibir peticiones.
func cargarMontajeIncorporacionPersonalB2(ctx context.Context, raiz *os.Root, c *archivoIncorporacionPersonalB2, nominales *perfilesNominalesIncorporacion, existentes map[string]*pgxpool.Pool, registroPool *pgxpool.Pool, material *proveedorMaterialAltaContratacionTemporalDesarrollo, detalle *appct.ServicioConsultaDetalleRRHH, org string, reloj ct.Reloj) (*montajeIncorporacionPersonalB2, error) {
	if c == nil {
		return nil, nil
	}
	nuevos, cerrar, e := cargarPoolsIncorporacionB2(ctx, raiz, c, existentes)
	if e != nil {
		return nil, e
	}
	completo := false
	defer func() {
		if !completo {
			cerrar()
		}
	}()
	all := map[string]*pgxpool.Pool{}
	for k, v := range existentes {
		all[k] = v
	}
	for k, v := range nuevos {
		all[k] = v
	}
	autoridad, e := cargarAutoridadIncorporacionB2(raiz, c, nominales, all, registroPool, material, reloj)
	if e != nil {
		return nil, e
	}
	autoridad.rptPool, autoridad.catalogoRPTID, autoridad.moduloRPTID = nuevos["rpt"], c.CatalogoRPTID, c.ModuloRPTID
	lectorCT, e := pgct.NuevaFuentePlanNominalB2PostgreSQL(existentes["registro_ct"])
	if e != nil {
		return nil, e
	}
	repoFicha, e := pgpersonal.NuevoRepositorioRegistroEmpleadoB2PostgreSQL(nuevos["personal_consultas"])
	if e != nil {
		return nil, e
	}
	ficha, e := apppersonal.NuevoServicioRegistroEmpleadoB2(autoridad, repoFicha)
	if e != nil {
		return nil, e
	}
	fuenteFicha, e := fuentePersonal.NuevaFuenteRegistroEmpleadoB2(ficha)
	if e != nil {
		return nil, e
	}
	hechos, e := apppersonal.NuevoServicioConsultaIncorporacionCT(fuenteFicha)
	if e != nil {
		return nil, e
	}
	catalogos, e := apppersonal.NuevoServicioCatalogosRegistroEmpleadoB2(autoridad, repoFicha)
	if e != nil {
		return nil, e
	}
	repoPlanes, e := pgpersonal.NuevoRepositorioRegistroEmpleadoB2PostgreSQL(nuevos["personal_planes"])
	if e != nil {
		return nil, e
	}
	repoActos, e := pgpersonal.NuevoRepositorioActosPlanIncorporacionCTPostgreSQL(nuevos["personal_actos"])
	if e != nil {
		return nil, e
	}
	actos, e := apppersonal.NuevoServicioActosRegistroEmpleadoB2(autoridad, repoActos)
	if e != nil {
		return nil, e
	}
	descriptor := vp.DescriptorCatalogoRPT{CatalogoID: c.CatalogoRPTID, ModuloID: c.ModuloRPTID}
	lectorRPT, e := pgvec.NuevoLectorCategoriasRPTPostgreSQL(nuevos["rpt"], descriptor)
	if e != nil {
		return nil, e
	}
	gestorRPT, e := pgvec.NuevoGestorUsosCategoriaRPTPostgreSQL(nuevos["rpt"], descriptor, "personal")
	if e != nil {
		return nil, e
	}
	autoridad.preparadorUsosRPT = gestorRPT
	emisorBolsa, e := autoridad.emisorMaterial(bp.AccionConsultaPersonaAceptacionCT)
	if e != nil {
		return nil, e
	}
	correlacion := func(ctx context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error) {
		return core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
	}
	proveedorBolsa, e := appbolsa.NuevoProveedorNominalConsultaPersonaAceptacionCT(emisorBolsa, c.Operaciones["bolsa_persona"].Motivo, correlacion, reloj.Ahora)
	if e != nil {
		return nil, e
	}
	repoBolsa, e := pgbolsa.NuevoRepositorioConsultaPersonaAceptacionCTPostgreSQL(nuevos["bolsa_persona"], reloj.Ahora)
	if e != nil {
		return nil, e
	}
	bolsa, e := appbolsa.NuevoServicioConsultaPersonaAceptacionCT(proveedorBolsa, repoBolsa, reloj.Ahora)
	if e != nil {
		return nil, e
	}
	fuenteVinculos, e := pgct.NuevaFuenteVinculoCategoriaRPTPostgreSQL(existentes["registro_ct"])
	if e != nil {
		return nil, e
	}
	publicacion, e := pgct.NuevaFuentePublicacionCategoriaRPTPostgreSQL(nuevos["rpt"])
	if e != nil {
		return nil, e
	}
	vinculos, e := appct.NuevoServicioVinculoCategoriaRPT(autoridad, fuenteVinculos, publicacion, reloj)
	if e != nil {
		return nil, e
	}
	externas, e := NuevasFuentesContextoIncorporacionB2(ConfiguracionFuentesContextoIncorporacionB2{Bolsa: bolsa, ActorBolsa: func(ctx context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
		actual, e := autoridad.contexto(ctx, bp.AccionConsultaPersonaAceptacionCT)
		return bp.ActorConfiablePersonaAceptacionCT{Vinculo: actual.Vinculo, Resultado: actual.Resultado}, e
	}, Vinculos: vinculos, Publicacion: publicacion, AutoridadRPT: autoridad})
	if e != nil {
		return nil, e
	}
	emisorAnclaje, e := autoridad.emisorMaterial(bp.AccionConsultaAnclajeAceptacionCT)
	if e != nil {
		return nil, e
	}
	proveedorAnclaje, e := appbolsa.NuevoProveedorNominalConsultaAnclajeAceptacionCT(emisorAnclaje, c.Operaciones["bolsa_anclaje"].Motivo, correlacion, reloj.Ahora)
	if e != nil {
		return nil, e
	}
	repoAnclaje, e := pgbolsa.NuevoRepositorioConsultaAnclajeAceptacionCTPostgreSQL(nuevos["bolsa_persona"], reloj.Ahora)
	if e != nil {
		return nil, e
	}
	anclaje, e := appbolsa.NuevoServicioConsultaAnclajeAceptacionCT(proveedorAnclaje, repoAnclaje, reloj.Ahora)
	if e != nil {
		return nil, e
	}
	fuentes := &fuentesIncorporacionPersonalB2{organizacionRef: org, organismoRef: c.OrganismoRef, autoridad: autoridad, ct: lectorCT, detalle: detalle, ficha: fuenteFicha, hechos: hechos, vacantes: ficha, catalogos: catalogos, externas: externas, lectorRPT: lectorRPT, reloj: reloj}
	personalServicio, e := apppersonal.NuevoServicioPlanIncorporacionCT(autoridad, repoPlanes, actos, fuentes)
	if e != nil {
		return nil, e
	}
	fuentes.personal = personalServicio
	fuentes.clases = personalServicio
	fuentes.anclaje = anclaje
	servicioCT, e := appct.NuevoServicioPlanNominalB2(fuentes, autoridad, lectorCT)
	if e != nil {
		return nil, e
	}
	fuentes.servicioCT = servicioCT
	consumidor, e := inc.NuevoConsumidorPersonalB2(inc.ConfiguracionConsumidorPersonalB2{Personal: personalServicio, Ficha: fuenteFicha, Hechos: hechos, RPT: gestorRPT, AutoridadRPT: autoridad, Actores: autoridad, Reloj: reloj})
	if e != nil {
		return nil, e
	}
	planes, e := inc.NuevosPlanesNominales(inc.ConfiguracionPlanesNominales{Contratos: fuentes, IntencionCT: fuentes, CT124: fuentes, Bolsa: externas, RPT: externas, Personal: consumidor})
	if e != nil {
		return nil, e
	}
	fachada := &fachadaIncorporacionPersonalB2{organizacionRef: org, ct: servicioCT, opciones: fuentes, planes: planes, personal: consumidor, autoridad: autoridad, fuentes: fuentes}
	cese, e := componerCesePersonalB2(c, nuevos["personal_actos"], autoridad, fuentes, servicioCT, fuenteFicha, reloj)
	if e != nil {
		return nil, e
	}
	var gobiernoCatalogo *apppersonal.ServicioCatalogosRegistroEmpleadoB2
	if operacionConfiguradaB2(c, claveCatalogoPublicarB2) {
		// Publicar y retirar escriben con el LOGIN de actos de Personal, no con
		// el de consultas que usa el plan para leer el catálogo.
		repoGobierno, e := pgpersonal.NuevoRepositorioRegistroEmpleadoB2PostgreSQL(nuevos["personal_actos"])
		if e != nil {
			return nil, e
		}
		if gobiernoCatalogo, e = apppersonal.NuevoServicioCatalogosRegistroEmpleadoB2(autoridad, repoGobierno); e != nil {
			return nil, e
		}
	}
	completo = true
	vinculo := &fachadaVinculoCategoriaRPTB2{organizacionRef: org, catalogoID: c.CatalogoRPTID, moduloID: c.ModuloRPTID, servicio: vinculos}
	categorias := &fachadaCategoriasRPTB2{catalogoID: c.CatalogoRPTID, autoridad: autoridad, lector: lectorRPT}
	return &montajeIncorporacionPersonalB2{fachada: fachada, autoridad: autoridad, cese: cese, vinculo: vinculo, categorias: categorias, gobiernoCatalogo: gobiernoCatalogo, cerrar: cerrar}, nil
}
