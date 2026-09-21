package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// preparadorBorradorLlamamientoDesarrollo une una entrada HTTP mínima a la
// sesión nominal de Bolsa revalidada en esa misma petición. Unidad y ámbito
// llegan del manifiesto privado publicado durante composición, nunca del
// cliente.
type preparadorBorradorLlamamientoDesarrollo struct {
	sesion  *seguridadComunDesarrollo
	soporte *soporteSesionBorradorBolsaDesarrollo
	generar seguridadvec.GeneradorReferenciasCriptograficas
}

func (p *preparadorBorradorLlamamientoDesarrollo) ResolverContextoBorradorLlamamiento(
	_ context.Context, actor dominiovec.ContextoActor,
) (puertosbolsa.ContextoBorradorLlamamientoResuelto, error) {
	if p == nil || p.soporte == nil || p.soporte.soporteCanal == nil || actor.PersonaRef == "" ||
		p.soporte.unidadRef == "" || p.soporte.ambitoRef == "" {
		return puertosbolsa.ContextoBorradorLlamamientoResuelto{}, puertosbolsa.ErrSolicitudBorradorLlamamientoInvalida
	}
	return puertosbolsa.ContextoBorradorLlamamientoResuelto{UnidadRef: p.soporte.unidadRef, AmbitoRef: p.soporte.ambitoRef}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudCrearBorradorLlamamientoInterno(
	ctx context.Context, entrada bolsahttp.EntradaCrearBorradorLlamamientoInterno,
) (puertosbolsa.SolicitudCrearBorradorLlamamiento, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudCrearBorradorLlamamiento{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudCrearBorradorLlamamiento{}, err
	}
	return puertosbolsa.SolicitudCrearBorradorLlamamiento{
		Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado,
		ClaveIdempotencia: entrada.ClaveIdempotencia,
		Contenido:         dominiobolsa.ContenidoBorradorLlamamiento{Resumen: entrada.Resumen},
		Correlacion:       correlacion, Motivo: motivoCrearBorradorLlamamientoBolsaDesarrollo(),
	}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudConsultarBorradorLlamamientoInterno(
	ctx context.Context, entrada bolsahttp.EntradaConsultarBorradorLlamamientoInterno,
) (puertosbolsa.SolicitudConsultarBorradorLlamamiento, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudConsultarBorradorLlamamiento{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudConsultarBorradorLlamamiento{}, err
	}
	return puertosbolsa.SolicitudConsultarBorradorLlamamiento{
		Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado,
		BorradorRef: entrada.BorradorRef, Correlacion: correlacion,
		Motivo: motivoConsultarBorradorLlamamientoBolsaDesarrollo(),
	}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) contextoRevalidado(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	if p == nil || p.sesion == nil || ctx == nil {
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	contexto, err := p.sesion.ResolverContexto(ctx)
	if err != nil || contexto.Resultado.Validar() != nil || contexto.Vinculo.ValidarPara(contexto.Resultado) != nil {
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	esperado := p.soporte.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef
	datos, err := contexto.Vinculo.Datos()
	if err != nil || !perfilActivoSeguridadComunValido(esperado) ||
		contexto.Resultado.Contexto.PerfilActivoRef != esperado || datos.PerfilActivoRef != esperado {
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	if holder, ok := ctx.Value(claveHolderActorBorradorLlamamientoDesarrollo{}).(*holderActorBorradorLlamamientoDesarrollo); ok {
		holder.fijar(contexto.Resultado.Contexto.PersonaRef)
	}
	return contexto, nil
}

// holderActorBorradorLlamamientoDesarrollo permite que la auditoría exterior
// lea sólo la persona que el preparador ya revalidó. Cada petición instala su
// propio holder antes de pasar por la protección.
type holderActorBorradorLlamamientoDesarrollo struct {
	mu     sync.Mutex
	actor  string
	fijado bool
}

type claveHolderActorBorradorLlamamientoDesarrollo struct{}

type actorBorradorLlamamientoDesdeContextoDesarrollo struct{}

func (actorBorradorLlamamientoDesdeContextoDesarrollo) ActorVerificadoParaAuditoriaBorradorLlamamiento(ctx context.Context) (string, bool) {
	holder, ok := ctx.Value(claveHolderActorBorradorLlamamientoDesarrollo{}).(*holderActorBorradorLlamamientoDesarrollo)
	if !ok {
		return "", false
	}
	return holder.ActorVerificadoParaAuditoriaBorradorLlamamiento(ctx)
}

func (h *holderActorBorradorLlamamientoDesarrollo) fijar(actor string) {
	if h == nil || actor == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.fijado {
		h.actor, h.fijado = actor, true
	}
}

func (h *holderActorBorradorLlamamientoDesarrollo) ActorVerificadoParaAuditoriaBorradorLlamamiento(context.Context) (string, bool) {
	if h == nil {
		return "", false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.actor, h.fijado
}

var errBorradorLlamamientoDesarrolloNoDisponible = errors.New("bootstrap: borrador de llamamiento no disponible")

var _ bolsahttp.PreparadorBorradorLlamamientoInterno = (*preparadorBorradorLlamamientoDesarrollo)(nil)
var _ puertosbolsa.ResolutorContextoBorradorLlamamiento = (*preparadorBorradorLlamamientoDesarrollo)(nil)
var _ puertosvec.GeneradorReferenciaDecisionAutorizacion = seguridadvec.GeneradorReferenciasCriptograficas{}

type emisorBorradorLlamamientoDesarrollo struct {
	crear, consultar *emisorMaterialRenovableCTDesarrollo
}

func (e *emisorBorradorLlamamientoDesarrollo) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := solicitud.Datos()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	switch datos.Accion {
	case puertosbolsa.AccionCrearBorradorLlamamientoInterno:
		if e != nil && e.crear != nil {
			return e.crear.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
		}
	case puertosbolsa.AccionConsultarBorradorLlamamientoInterno:
		if e != nil && e.consultar != nil {
			return e.consultar.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
		}
	}
	return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errBorradorLlamamientoDesarrolloNoDisponible
}

// nuevasDependenciasBorradorLlamamientoDesarrollo conecta el único caso B-BACK
// sin ensanchar los casos de uso CT: la composición aporta identidad, política,
// PDP común, material por audiencia y el adaptador PostgreSQL de Bolsa.
func nuevasDependenciasBorradorLlamamientoDesarrollo(
	ctx context.Context,
	cfg config.Config,
	dependenciasCT *DependenciasCT,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	soporteBolsa *soporteSesionBorradorBolsaDesarrollo,
	catalogoFronteras catalogoFronterasComunDesarrollo,
	identidadCT *proveedorSesionConsultaRRHHDesarrollo,
) ([]vechttp.RutaExacta, []vechttp.RutaColeccion, catalogoFronterasComunDesarrollo, func(http.Handler) http.Handler, func(), error) {
	vacio := catalogoFronterasComunDesarrollo{}
	if ctx == nil || dependenciasCT == nil || alta == nil || soporteBolsa == nil || identidadCT == nil || catalogoFronteras.identidad == nil || alta.soporte == nil || alta.postgresql.bolsa == nil || alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil || alta.postgresql.proveedorMaterialBorradorCrear == nil || alta.postgresql.proveedorMaterialBorradorConsulta == nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	// El manifiesto y el contexto nominal Bolsa se cargan antes de declarar
	// rutas: cada frontera B-BACK queda ligada al perfil Bolsa, nunca al CT.
	var err error
	vinculo, err := soporteBolsa.soporteCanal.contexto.Vinculo.Datos()
	if err != nil || !perfilActivoSeguridadComunValido(vinculo.PerfilActivoRef) ||
		vinculo.PerfilActivoRef != soporteBolsa.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	auditoria, cerrarAuditoria, err := nuevaAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx, cfg)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if err := publicarContextoPostgreSQLBorradorBolsaDesarrollo(ctx, alta.postgresql.gobierno, soporteBolsa); err != nil {
		cerrarAuditoria()
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	completa := false
	defer func() {
		if !completa {
			cerrarAuditoria()
		}
	}()
	sesionBolsa, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(
		soporteBolsa.soporteCanal, identidadCT.registro, identidadCT.revalidador, identidadCT.reloj, identidadCT.resolutor, catalogoFronteras,
	)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesionBolsa, dependenciasCT.reloj)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	autoridadBolsa := autoridadPostgreSQLDesarrollo{
		pool: alta.postgresql.gobierno, vinculo: soporteBolsa.soporteCanal.contexto.Vinculo,
		prefijoBloqueo: "vec:bolsa:bback:autorizacion:", actoControlRol: "acto:bolsa:bback:control-rol:v1",
		actoAsignacion: "acto:bolsa:bback:asignacion:v1", actoSesion: "acto:bolsa:bback:sesion:v1",
	}
	if !autoridadBolsa.validaConfiguracion() || vinculo.PrincipalID == "" {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if alta.soporte.registroDecisionesAnalisis == nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	politicaBolsa, err := nuevaPoliticaBorradorLlamamientoBolsaDesarrollo(soporteBolsa, &autoridadBolsa, alta.soporte.registroDecisionesAnalisis, dependenciasCT.reloj)
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(dependenciasCT.reloj.Ahora())
	if err != nil || !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoCrearBorradorLlamamientoBolsaDesarrollo(), motivoConsultarBorradorLlamamientoBolsaDesarrollo(),
	}, desde) != nil || politicaBolsa.PublicarInicial(ctx) != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	politicaCT, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(alta.soporte, alta.soporte, alta.soporte, alta.soporte)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	politicaB, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(politicaBolsa, politicaBolsa, politicaBolsa, politicaBolsa)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	descriptoresBolsa, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politicaB)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	descriptoresAutorizacion := append(descriptoresAutorizacionContratacionTemporalDesarrollo(politicaCT), descriptoresBolsa...)
	catalogoAutorizacion, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, descriptoresAutorizacion)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	pdp, err := nuevoAutorizadorComunDesarrollo(catalogoAutorizacion, dependenciasCT.reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	wrapper, ok := alta.autorizador.(*autorizadorAnalisisContratacionTemporalDesarrollo)
	if !ok || wrapper.instalarDelegadoComun(pdp) != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	emisorCrear, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialBorradorCrear)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	emisorConsulta, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialBorradorConsulta)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	repositorio, err := postgresbolsa.NuevoRepositorioBorradorLlamamientoPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	preparador := &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, soporte: soporteBolsa}
	servicio, err := aplicacionbolsa.NuevoServicioBorradorLlamamiento(preparador, &emisorBorradorLlamamientoDesarrollo{crear: emisorCrear, consultar: emisorConsulta}, repositorio, repositorio)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	handler, err := bolsahttp.NuevoHandlerBorradorLlamamiento(preparador, servicio)
	if err != nil {
		return nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	envolver := func(siguiente http.Handler) http.Handler {
		auditada, auditErr := bolsahttp.NuevaAuditoriaBorradorLlamamiento(siguiente, auditoria, seguridadvec.GeneradorReferenciasCriptograficas{}, actorBorradorLlamamientoDesdeContextoDesarrollo{})
		if auditErr != nil {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			})
		}
		metodos, metodoErr := bolsahttp.EnvolverRechazoHEADDetalleBorradorLlamamiento(auditada)
		if metodoErr != nil {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			})
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), claveHolderActorBorradorLlamamientoDesarrollo{}, &holderActorBorradorLlamamientoDesarrollo{})
			metodos.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	completa = true
	return []vechttp.RutaExacta{{Ruta: bolsahttp.RutaBorradoresLlamamiento, Manejador: handler}}, []vechttp.RutaColeccion{{Prefijo: bolsahttp.RutaBorradoresLlamamiento, Manejador: handler}}, catalogoFronteras, envolver, cerrarAuditoria, nil
}
