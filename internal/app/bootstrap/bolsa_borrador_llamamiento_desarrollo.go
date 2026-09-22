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
	smtpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/smtp"
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

func (p *preparadorBorradorLlamamientoDesarrollo) ResolverContextoSituacionParticipacion(_ context.Context, actor dominiovec.ContextoActor, bolsaRef, participacionRef string) (puertosbolsa.ContextoSituacionParticipacionResuelto, error) {
	if p == nil || p.soporte == nil || actor.PersonaRef == "" || bolsaRef == "" || participacionRef == "" || p.soporte.unidadRef == "" || p.soporte.ambitoRef == "" {
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if _, admitida := p.soporte.bolsasRef[bolsaRef]; !admitida {
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, dominiovec.ErrAutorizacionDenegada
	}
	return puertosbolsa.ContextoSituacionParticipacionResuelto{UnidadRef: p.soporte.unidadRef, AmbitoRef: p.soporte.ambitoRef}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) ResolverContextoContactosBolsa(_ context.Context, actor dominiovec.ContextoActor, bolsaRef string) (puertosbolsa.ContextoSituacionParticipacionResuelto, error) {
	if p == nil || p.soporte == nil || actor.PersonaRef == "" || bolsaRef == "" || p.soporte.unidadRef == "" || p.soporte.ambitoRef == "" {
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if _, admitida := p.soporte.bolsasRef[bolsaRef]; !admitida {
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, dominiovec.ErrAutorizacionDenegada
	}
	return puertosbolsa.ContextoSituacionParticipacionResuelto{UnidadRef: p.soporte.unidadRef, AmbitoRef: p.soporte.ambitoRef}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudCambiarSituacion(ctx context.Context, entrada bolsahttp.EntradaCambiarSituacionParticipacion) (puertosbolsa.SolicitudCambiarSituacionParticipacion, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudCambiarSituacionParticipacion{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudCambiarSituacionParticipacion{}, err
	}
	return puertosbolsa.SolicitudCambiarSituacionParticipacion{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: entrada.BolsaRef, ParticipacionRef: entrada.ParticipacionRef, Destino: entrada.Destino, Motivo: entrada.Motivo, ClaveIdempotencia: entrada.ClaveIdempotencia, FechaDisponible: entrada.FechaDisponible, Correlacion: correlacion, MotivoAutorizacion: motivoCambiarSituacionParticipacionBolsaDesarrollo()}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudRegistrarContacto(ctx context.Context, entrada bolsahttp.EntradaRegistrarContactoParticipacion) (puertosbolsa.SolicitudRegistrarContactoParticipacion, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudRegistrarContactoParticipacion{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudRegistrarContactoParticipacion{}, err
	}
	return puertosbolsa.SolicitudRegistrarContactoParticipacion{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: entrada.BolsaRef, ParticipacionRef: entrada.ParticipacionRef, LlamamientoRef: entrada.LlamamientoRef, Canal: entrada.Canal, Instante: entrada.Instante, Resultado: entrada.Resultado, Anotacion: entrada.Anotacion, ClaveIdempotencia: entrada.ClaveIdempotencia, Correlacion: correlacion, MotivoAutorizacion: motivoRegistrarContactoParticipacionBolsaDesarrollo()}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudEmitirLlamamiento(ctx context.Context, entrada bolsahttp.EntradaEmitirLlamamiento) (puertosbolsa.SolicitudEmitirLlamamiento, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudEmitirLlamamiento{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudEmitirLlamamiento{}, err
	}
	return puertosbolsa.SolicitudEmitirLlamamiento{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: entrada.BolsaRef, Participaciones: entrada.Participaciones, Configuracion: entrada.Configuracion, ClaveIdempotencia: entrada.ClaveIdempotencia, Correlacion: correlacion, MotivoAutorizacion: motivoEmitirLlamamientoBolsaDesarrollo()}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararConsultaContactos(ctx context.Context, bolsaRef, participacionRef, cursor string, limite int) (puertosbolsa.ConsultaContactosParticipacion, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.ConsultaContactosParticipacion{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.ConsultaContactosParticipacion{}, err
	}
	return puertosbolsa.ConsultaContactosParticipacion{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: bolsaRef, ParticipacionRef: participacionRef, Cursor: cursor, Limite: limite, Correlacion: correlacion, MotivoAutorizacion: motivoConsultarContactoParticipacionBolsaDesarrollo()}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararConsultaContactosBolsa(ctx context.Context, bolsaRef, cursor string, limite int) (puertosbolsa.ConsultaContactosBolsa, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.ConsultaContactosBolsa{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.ConsultaContactosBolsa{}, err
	}
	return puertosbolsa.ConsultaContactosBolsa{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: bolsaRef, Cursor: cursor, Limite: limite, Correlacion: correlacion, MotivoAutorizacion: motivoConsultarContactoParticipacionBolsaDesarrollo()}, nil
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
	crear, consultar, situacion, contacto, consultaContacto, emision *emisorMaterialRenovableCTDesarrollo
}

type manejadorParticipacionBolsaDesarrollo struct {
	situacion, contacto http.Handler
	preparador          *preparadorBorradorLlamamientoDesarrollo
	servicio            *aplicacionbolsa.ServicioContactoParticipacion
}

func (m *manejadorParticipacionBolsaDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := bolsahttp.ReferenciasRutaContactosParticipacion(r); ok {
		m.contacto.ServeHTTP(w, r)
		return
	}
	m.situacion.ServeHTTP(w, r)
}
func (m *manejadorParticipacionBolsaDesarrollo) ListarContactosBolsa(ctx context.Context, bolsa, cursor string, limite int) (puertosbolsa.PaginaContactosParticipacion, error) {
	q, err := m.preparador.PrepararConsultaContactosBolsa(ctx, bolsa, cursor, limite)
	if err != nil {
		return puertosbolsa.PaginaContactosParticipacion{}, err
	}
	return m.servicio.ListarContactosBolsa(ctx, q)
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
	case puertosbolsa.AccionCambiarSituacionParticipacion:
		if e != nil && e.situacion != nil {
			return e.situacion.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
		}
	case puertosbolsa.AccionRegistrarContactoParticipacion:
		if e != nil && e.contacto != nil {
			return e.contacto.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
		}
	case puertosbolsa.AccionConsultarContactoParticipacion:
		if e != nil && e.consultaContacto != nil {
			return e.consultaContacto.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
		}
	case puertosbolsa.AccionEmitirLlamamiento:
		if e != nil && e.emision != nil {
			return e.emision.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
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
) ([]vechttp.RutaExacta, []vechttp.RutaColeccion, http.Handler, catalogoFronterasComunDesarrollo, func(http.Handler) http.Handler, func(), error) {
	vacio := catalogoFronterasComunDesarrollo{}
	if ctx == nil || dependenciasCT == nil || dependenciasCT.kms == nil || alta == nil || soporteBolsa == nil || identidadCT == nil || catalogoFronteras.identidad == nil || alta.soporte == nil || alta.postgresql.bolsa == nil || alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil || alta.postgresql.proveedorMaterialBorradorCrear == nil || alta.postgresql.proveedorMaterialBorradorConsulta == nil || alta.postgresql.proveedorMaterialSituacion == nil || alta.postgresql.proveedorMaterialContacto == nil || alta.postgresql.proveedorMaterialConsultaContacto == nil || alta.postgresql.proveedorMaterialEmision == nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	// El manifiesto y el contexto nominal Bolsa se cargan antes de declarar
	// rutas: cada frontera B-BACK queda ligada al perfil Bolsa, nunca al CT.
	var err error
	vinculo, err := soporteBolsa.soporteCanal.contexto.Vinculo.Datos()
	if err != nil || !perfilActivoSeguridadComunValido(vinculo.PerfilActivoRef) ||
		vinculo.PerfilActivoRef != soporteBolsa.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	auditoria, cerrarAuditoria, err := nuevaAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx, cfg)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if err := publicarContextoPostgreSQLBorradorBolsaDesarrollo(ctx, alta.postgresql.gobierno, soporteBolsa); err != nil {
		cerrarAuditoria()
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
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
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesionBolsa, dependenciasCT.reloj)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	autoridadBolsa := autoridadPostgreSQLDesarrollo{
		pool: alta.postgresql.gobierno, vinculo: soporteBolsa.soporteCanal.contexto.Vinculo,
		prefijoBloqueo: "vec:bolsa:bback:autorizacion:", actoControlRol: "acto:bolsa:bback:control-rol:v1",
		actoAsignacion: "acto:bolsa:bback:asignacion:v1", actoSesion: "acto:bolsa:bback:sesion:v1",
	}
	if !autoridadBolsa.validaConfiguracion() || vinculo.PrincipalID == "" {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if alta.soporte.registroDecisionesAnalisis == nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	politicaBolsa, err := nuevaPoliticaBorradorLlamamientoBolsaDesarrollo(soporteBolsa, &autoridadBolsa, alta.soporte.registroDecisionesAnalisis, dependenciasCT.reloj)
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(dependenciasCT.reloj.Ahora())
	if err != nil || !vigente {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoCrearBorradorLlamamientoBolsaDesarrollo(), motivoConsultarBorradorLlamamientoBolsaDesarrollo(),
	}, desde) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoCambiarSituacionParticipacionBolsaDesarrollo(),
	}, desde) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoRegistrarContactoParticipacionBolsaDesarrollo(),
		motivoConsultarContactoParticipacionBolsaDesarrollo(),
		motivoEmitirLlamamientoBolsaDesarrollo(),
	}, desde) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	if politicaBolsa.PublicarInicial(ctx) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	politicaCT, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(alta.soporte, alta.soporte, alta.soporte, alta.soporte)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	politicaB, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(politicaBolsa, politicaBolsa, politicaBolsa, politicaBolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	descriptoresBolsa, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politicaB)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	descriptoresAutorizacion := append(descriptoresAutorizacionContratacionTemporalDesarrollo(politicaCT), descriptoresBolsa...)
	catalogoAutorizacion, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, descriptoresAutorizacion)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	pdp, err := nuevoAutorizadorComunDesarrollo(catalogoAutorizacion, dependenciasCT.reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	wrapper, ok := alta.autorizador.(*autorizadorAnalisisContratacionTemporalDesarrollo)
	if !ok || wrapper.instalarDelegadoComun(pdp) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	emisorCrear, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialBorradorCrear)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	emisorConsulta, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialBorradorConsulta)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	emisorSituacion, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialSituacion)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	emisorContacto, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	emisorConsultaContacto, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialConsultaContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	emisorEmision, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialEmision)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	repositorio, err := postgresbolsa.NuevoRepositorioBorradorLlamamientoPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	preparador := &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, soporte: soporteBolsa}
	emisor := &emisorBorradorLlamamientoDesarrollo{crear: emisorCrear, consultar: emisorConsulta, situacion: emisorSituacion, contacto: emisorContacto, consultaContacto: emisorConsultaContacto, emision: emisorEmision}
	servicio, err := aplicacionbolsa.NuevoServicioBorradorLlamamiento(preparador, emisor, repositorio, repositorio)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	handler, err := bolsahttp.NuevoHandlerBorradorLlamamiento(preparador, servicio)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	repositorioSituacion, err := postgresbolsa.NuevoRepositorioSituacionParticipacionPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	servicioSituacion, err := aplicacionbolsa.NuevoServicioSituacionParticipacion(preparador, emisor, repositorioSituacion, dependenciasCT.reloj.Ahora)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	handlerSituacion, err := bolsahttp.NuevoHandlerSituacionParticipacion(preparador, servicioSituacion)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	repositorioContacto, err := postgresbolsa.NuevoRepositorioContactoParticipacionPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	servicioContacto, err := aplicacionbolsa.NuevoServicioContactoParticipacion(preparador, emisor, repositorioContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	handlerContacto, err := bolsahttp.NuevoHandlerContactoParticipacion(preparador, servicioContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	repositorioEmision, err := postgresbolsa.NuevoRepositorioEmisionLlamamientoPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	repositorioDatos, err := postgresbolsa.NuevoRepositorioDatosContactoParticipacionPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	correoSMTP, err := nuevoEnviadorCorreoLlamamientoDesarrollo(cfg)
	if err != nil || correoSMTP == nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	servicioEmision, err := aplicacionbolsa.NuevoServicioEmisionLlamamiento(preparador, emisor, repositorioEmision, &fuenteCorreoParticipacionB7{repositorioDatos, dependenciasCT.kms}, &emisorCorreoBolsaB7{correoSMTP}, servicioContacto, dependenciasCT.reloj.Ahora)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	handlerEmision, err := bolsahttp.NuevoHandlerEmisionLlamamiento(preparador, servicioEmision, repositorioEmision)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorLlamamientoDesarrolloNoDisponible
	}
	mutador := &manejadorParticipacionBolsaDesarrollo{situacion: handlerSituacion, contacto: handlerContacto, preparador: preparador, servicio: servicioContacto}
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
	return []vechttp.RutaExacta{{Ruta: bolsahttp.RutaBorradoresLlamamiento, Manejador: handler}, {Ruta: bolsahttp.RutaEmisionesLlamamiento, Manejador: handlerEmision}}, []vechttp.RutaColeccion{{Prefijo: bolsahttp.RutaBorradoresLlamamiento, Manejador: handler}}, mutador, catalogoFronteras, envolver, cerrarAuditoria, nil
}

type fuenteCorreoParticipacionB7 struct {
	repositorio puertosbolsa.RepositorioDatosContactoParticipacion
	kms         *emisorKMSDesarrollo
}

func (f *fuenteCorreoParticipacionB7) CorreoParticipacion(ctx context.Context, participacion string) (string, error) {
	r, err := f.repositorio.DatosContactoVigentes(ctx, participacion)
	if err != nil {
		return "", err
	}
	var correo string
	err = f.kms.ConDatosContactoParticipacionDescifrados(ctx, participacion, r.Sobre, func(claro []byte) error {
		d, e := dominiobolsa.DatosContactoParticipacionDesdeCanonico(participacion, claro)
		if e == nil {
			correo = d.Correo
		}
		return e
	})
	return correo, err
}

type emisorCorreoBolsaB7 struct {
	smtp enviadorCorreoLlamamientoDesarrollo
}

func (e *emisorCorreoBolsaB7) EnviarCorreo(ctx context.Context, destino, asunto, cuerpo, messageID string, fecha time.Time) bool {
	return e != nil && e.smtp != nil && e.smtp.Enviar(ctx, smtpct.Mensaje{Destino: destino, Asunto: asunto, Cuerpo: cuerpo, MessageID: messageID, FechaOrigen: fecha}).Estado == smtpct.AceptadoPorRelay
}
