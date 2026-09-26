package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglasvec"
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
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, errBorradorNoDisponibleEn()
	}
	if _, admitida := p.soporte.bolsasRef[bolsaRef]; !admitida {
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, dominiovec.ErrAutorizacionDenegada
	}
	return puertosbolsa.ContextoSituacionParticipacionResuelto{UnidadRef: p.soporte.unidadRef, AmbitoRef: p.soporte.ambitoRef}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) ResolverContextoContactosBolsa(_ context.Context, actor dominiovec.ContextoActor, bolsaRef string) (puertosbolsa.ContextoSituacionParticipacionResuelto, error) {
	if p == nil || p.soporte == nil || actor.PersonaRef == "" || bolsaRef == "" || p.soporte.unidadRef == "" || p.soporte.ambitoRef == "" {
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, errBorradorNoDisponibleEn()
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

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudRegistrarDatosContacto(ctx context.Context, entrada bolsahttp.EntradaRegistrarDatosContactoParticipacion) (puertosbolsa.SolicitudRegistrarDatosContactoParticipacion, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudRegistrarDatosContactoParticipacion{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudRegistrarDatosContactoParticipacion{}, err
	}
	return puertosbolsa.SolicitudRegistrarDatosContactoParticipacion{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: entrada.BolsaRef, ParticipacionRef: entrada.ParticipacionRef, Datos: entrada.Datos, Motivo: entrada.Motivo, ClaveIdempotencia: entrada.ClaveIdempotencia, Correlacion: correlacion, MotivoAutorizacion: motivoRegistrarDatosContactoParticipacionBolsaDesarrollo(), Origen: entrada.Origen}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudConsultarDatosContacto(ctx context.Context, bolsaRef, participacionRef string) (puertosbolsa.SolicitudConsultarDatosContactoParticipacion, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudConsultarDatosContactoParticipacion{}, err
	}
	return puertosbolsa.SolicitudConsultarDatosContactoParticipacion{ContextoActor: contexto.Resultado.Contexto, BolsaRef: bolsaRef, ParticipacionRef: participacionRef}, nil
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

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudRecuperarLlamamiento(ctx context.Context, bolsaRef, clave string) (puertosbolsa.SolicitudRecuperarLlamamiento, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudRecuperarLlamamiento{}, err
	}
	return puertosbolsa.SolicitudRecuperarLlamamiento{ContextoActor: contexto.Resultado.Contexto, BolsaRef: bolsaRef, ClaveIdempotencia: clave}, nil
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
		registrarRechazoBorradorLlamamientoDesarrollo("sesion_no_resuelta", err)
		return contextoSeguridadComunDesarrollo{}, ErrSeguridadComunDesarrolloDenegada
	}
	esperado := p.soporte.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef
	datos, err := contexto.Vinculo.Datos()
	if err != nil || !perfilActivoSeguridadComunValido(esperado) ||
		contexto.Resultado.Contexto.PerfilActivoRef != esperado || datos.PerfilActivoRef != esperado {
		registrarRechazoBorradorLlamamientoDesarrollo("perfil_distinto", err)
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

// errBorradorNoDisponibleEn conserva errBorradorLlamamientoDesarrolloNoDisponible
// para errors.Is y añade el fichero y la línea donde se rechazó el montaje.
// Hay decenas de comprobaciones que fallan cerradas con el mismo error; sin
// este dato el registro de arranque no permite saber cuál falló. Solo alcanza
// ese registro del servidor: ninguna respuesta HTTP lleva el detalle.
func errBorradorNoDisponibleEn() error {
	_, fichero, linea, ok := runtime.Caller(1)
	if !ok {
		return errBorradorLlamamientoDesarrolloNoDisponible
	}
	return fmt.Errorf("%w (%s:%d)", errBorradorLlamamientoDesarrolloNoDisponible, filepath.Base(fichero), linea)
}

var _ bolsahttp.PreparadorBorradorLlamamientoInterno = (*preparadorBorradorLlamamientoDesarrollo)(nil)
var _ puertosbolsa.ResolutorContextoBorradorLlamamiento = (*preparadorBorradorLlamamientoDesarrollo)(nil)
var _ puertosvec.GeneradorReferenciaDecisionAutorizacion = seguridadvec.GeneradorReferenciasCriptograficas{}

type emisorBorradorLlamamientoDesarrollo struct {
	crear, consultar, situacion, contacto, consultaContacto, datosContacto, emision *emisorMaterialRenovableCTDesarrollo
}

type manejadorParticipacionBolsaDesarrollo struct {
	situacion, operaciones, contacto, datosContacto, contratos, sanciones http.Handler
	preparador                                                            *preparadorBorradorLlamamientoDesarrollo
	servicio                                                              *aplicacionbolsa.ServicioContactoParticipacion
	// servicioSituacion permite componer después las reglas de transición.
	servicioSituacion *aplicacionbolsa.ServicioSituacionParticipacion
	// datos y emision reciben después la regla del contacto de origen CONVOCA.
	datos   *aplicacionbolsa.ServicioDatosContactoParticipacion
	emision *aplicacionbolsa.ServicioEmisionLlamamiento
	fuente  *fuenteCorreoParticipacionB7
}

func (m *manejadorParticipacionBolsaDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := bolsahttp.ReferenciasRutaOperacionesSituacion(r); ok {
		m.operaciones.ServeHTTP(w, r)
		return
	}
	if _, _, ok := bolsahttp.ReferenciasRutaContratosParticipacion(r); ok && m.contratos != nil {
		m.contratos.ServeHTTP(w, r)
		return
	}
	if _, _, _, ok := bolsahttp.ReferenciasRutaSancionesParticipacion(r); ok && m.sanciones != nil {
		m.sanciones.ServeHTTP(w, r)
		return
	}
	if _, _, _, ok := bolsahttp.ReferenciasRutaDatosContactoParticipacion(r); ok {
		m.datosContacto.ServeHTTP(w, r)
		return
	}
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
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errBorradorNoDisponibleEn()
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
	case puertosbolsa.AccionRegistrarDatosContactoParticipacion:
		if e != nil && e.datosContacto != nil {
			return e.datosContacto.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
		}
	case puertosbolsa.AccionEmitirLlamamiento:
		if e != nil && e.emision != nil {
			decision, confirmacion, exportador, err := e.emision.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
			if err != nil {
				registrarRechazoBorradorLlamamientoDesarrollo("material_emision", err)
			}
			return decision, confirmacion, exportador, err
		}
	}
	return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errBorradorNoDisponibleEn()
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
	personalizacion *fuentePersonalizacionB7,
) ([]vechttp.RutaExacta, []vechttp.RutaColeccion, http.Handler, catalogoFronterasComunDesarrollo, func(http.Handler) http.Handler, func(), error) {
	vacio := catalogoFronterasComunDesarrollo{}
	if ctx == nil || personalizacion == nil || dependenciasCT == nil || dependenciasCT.kms == nil || alta == nil || soporteBolsa == nil || identidadCT == nil || catalogoFronteras.identidad == nil || alta.soporte == nil || alta.postgresql.bolsa == nil || alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil || alta.postgresql.proveedorMaterialBorradorCrear == nil || alta.postgresql.proveedorMaterialBorradorConsulta == nil || alta.postgresql.proveedorMaterialSituacion == nil || alta.postgresql.proveedorMaterialContacto == nil || alta.postgresql.proveedorMaterialConsultaContacto == nil || alta.postgresql.proveedorMaterialDatosContacto == nil || alta.postgresql.proveedorMaterialEmision == nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	// El manifiesto y el contexto nominal Bolsa se cargan antes de declarar
	// rutas: cada frontera B-BACK queda ligada al perfil Bolsa, nunca al CT.
	var err error
	vinculo, err := soporteBolsa.soporteCanal.contexto.Vinculo.Datos()
	if err != nil || !perfilActivoSeguridadComunValido(vinculo.PerfilActivoRef) ||
		vinculo.PerfilActivoRef != soporteBolsa.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	auditoria, cerrarAuditoria, err := nuevaAuditoriaFronteraBorradorLlamamientoPostgreSQLDesarrollo(ctx, cfg)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if err := publicarContextoPostgreSQLBorradorBolsaDesarrollo(ctx, alta.postgresql.gobierno, soporteBolsa); err != nil {
		cerrarAuditoria()
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	esperadoRegistrado, err := contextoEsperadoRegistradoDesarrollo(ctx, identidadCT.resolutor, soporteBolsa.soporteCanal)
	if err != nil {
		cerrarAuditoria()
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	soporteBolsa.soporteCanal.contextoEsperadoRegistrado = esperadoRegistrado
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
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesionBolsa, dependenciasCT.reloj)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	autoridadBolsa := autoridadPostgreSQLDesarrollo{
		pool: alta.postgresql.gobierno, vinculo: soporteBolsa.soporteCanal.contexto.Vinculo,
		prefijoBloqueo: "vec:bolsa:bback:autorizacion:", actoControlRol: "acto:bolsa:bback:control-rol:v1",
		actoAsignacion: "acto:bolsa:bback:asignacion:v1", actoSesion: "acto:bolsa:bback:sesion:v1",
	}
	if !autoridadBolsa.validaConfiguracion() || vinculo.PrincipalID == "" {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if alta.soporte.registroDecisionesAnalisis == nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	politicaBolsa, err := nuevaPoliticaBorradorLlamamientoBolsaDesarrollo(soporteBolsa, &autoridadBolsa, alta.soporte.registroDecisionesAnalisis, dependenciasCT.reloj)
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(dependenciasCT.reloj.Ahora())
	if err != nil || !vigente {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoCrearBorradorLlamamientoBolsaDesarrollo(), motivoConsultarBorradorLlamamientoBolsaDesarrollo(),
	}, desde) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoCambiarSituacionParticipacionBolsaDesarrollo(),
	}, desde) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoRegistrarContactoParticipacionBolsaDesarrollo(),
		motivoConsultarContactoParticipacionBolsaDesarrollo(),
	}, desde) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoRegistrarDatosContactoParticipacionBolsaDesarrollo(),
	}, desde) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{
		motivoEmitirLlamamientoBolsaDesarrollo(),
	}, desde) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if politicaBolsa.PublicarInicial(ctx) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	politicaCT, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(alta.soporte, alta.soporte, alta.soporte, alta.soporte)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	politicaB, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(politicaBolsa, politicaBolsa, politicaBolsa, politicaBolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	descriptoresBolsa, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politicaB)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	descriptoresAutorizacion := append(descriptoresAutorizacionContratacionTemporalDesarrollo(politicaCT), descriptoresBolsa...)
	catalogoAutorizacion, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, descriptoresAutorizacion)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	pdp, err := nuevoAutorizadorComunDesarrollo(catalogoAutorizacion, dependenciasCT.reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	wrapper, ok := alta.autorizador.(*autorizadorAnalisisContratacionTemporalDesarrollo)
	if !ok || wrapper.instalarDelegadoComun(pdp) != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	emisorCrear, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialBorradorCrear)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	emisorConsulta, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialBorradorConsulta)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	emisorSituacion, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialSituacion)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	emisorContacto, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	emisorConsultaContacto, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialConsultaContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	emisorDatosContacto, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialDatosContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	emisorEmision, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, alta.postgresql.proveedorMaterialEmision)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	repositorio, err := postgresbolsa.NuevoRepositorioBorradorLlamamientoPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	preparador := &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, soporte: soporteBolsa}
	emisor := &emisorBorradorLlamamientoDesarrollo{crear: emisorCrear, consultar: emisorConsulta, situacion: emisorSituacion, contacto: emisorContacto, consultaContacto: emisorConsultaContacto, datosContacto: emisorDatosContacto, emision: emisorEmision}
	servicio, err := aplicacionbolsa.NuevoServicioBorradorLlamamiento(preparador, emisor, repositorio, repositorio)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handler, err := bolsahttp.NuevoHandlerBorradorLlamamiento(preparador, servicio)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	repositorioSituacion, err := postgresbolsa.NuevoRepositorioSituacionParticipacionPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	if err := publicarPoliticaSegregacionDesarrollo(ctx, cfg, repositorioSituacion, relojCalendariosDesarrollo{}); err != nil {
		return nil, nil, nil, vacio, nil, nil, err
	}
	servicioSituacion, err := aplicacionbolsa.NuevoServicioSituacionParticipacion(preparador, emisor, repositorioSituacion, dependenciasCT.reloj.Ahora)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerSituacion, err := bolsahttp.NuevoHandlerSituacionParticipacion(preparador, servicioSituacion)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerOperaciones, err := bolsahttp.NuevoHandlerOperacionesSituacion(preparador, servicioSituacion)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	// Sanciones (duda 62): mismo servicio de situación y autorización B8; el
	// catálogo solo existe con el paquete de reglas de ejemplo declarado.
	repositorioSanciones, err := postgresbolsa.NuevoRepositorioSancionesParticipacionPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	var catalogoSanciones puertosbolsa.CatalogoSancionesParticipacion
	if catalogo := reglasbolsa.NuevoCatalogoSanciones(dependenciasCT.reglasEjemplo.bolsa); catalogo != nil {
		catalogoSanciones = catalogo
	}
	servicioSanciones, err := aplicacionbolsa.NuevoServicioSancionesParticipacion(servicioSituacion, catalogoSanciones, repositorioSanciones)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerSanciones, err := bolsahttp.NuevoHandlerSancionesParticipacion(preparador, servicioSanciones)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerContratos, err := bolsahttp.NuevoHandlerContratosParticipacion(preparador, servicioSituacion)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	repositorioContacto, err := postgresbolsa.NuevoRepositorioContactoParticipacionPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	servicioContacto, err := aplicacionbolsa.NuevoServicioContactoParticipacion(preparador, emisor, repositorioContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerContacto, err := bolsahttp.NuevoHandlerContactoParticipacion(preparador, servicioContacto)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	repositorioEmision, err := postgresbolsa.NuevoRepositorioEmisionLlamamientoPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	repositorioDatos, err := postgresbolsa.NuevoRepositorioDatosContactoParticipacionPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	servicioDatos, err := aplicacionbolsa.NuevoServicioDatosContactoParticipacion(preparador, emisor, repositorioSituacion, dependenciasCT.kms, repositorioDatos, dependenciasCT.reloj.Ahora)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerDatos, err := bolsahttp.NuevoHandlerDatosContactoParticipacion(preparador, servicioDatos)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	correoSMTP, err := nuevoEnviadorCorreoLlamamientoDesarrollo(cfg)
	if err != nil || correoSMTP == nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	catalogoCorreo, err := cargarCatalogoCorreoLlamamientoBolsa()
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	fuenteCorreo := &fuenteCorreoParticipacionB7{repositorioDatos, dependenciasCT.kms}
	servicioEmision, err := aplicacionbolsa.NuevoServicioEmisionLlamamiento(preparador, emisor, repositorioEmision, fuenteCorreo, &emisorCorreoBolsaB7{correoSMTP}, dependenciasCT.reloj.Ahora, aplicacionbolsa.CorreoPersonalizadoLlamamiento{Catalogo: catalogoCorreo, Personalizacion: personalizacion, Huellas: repositorioEmision})
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerEmision, err := bolsahttp.NuevoHandlerEmisionLlamamiento(preparador, servicioEmision)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerCorreo, err := bolsahttp.NuevoHandlerCorreoLlamamiento(preparador, servicioEmision, catalogoCorreo.IdiomaDefecto())
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	repositorioOfertas, err := postgresbolsa.NuevoRepositorioOfertasPublicadasPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	servicioOfertas, err := aplicacionbolsa.NuevoServicioOfertasPublicadas(preparador, emisor, repositorioOfertas, dependenciasCT.plazosOfertasBolsa, dependenciasCT.reloj.Ahora)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	handlerOfertas, err := bolsahttp.NuevoHandlerOfertasPublicadas(preparador, servicioOfertas)
	if err != nil {
		return nil, nil, nil, vacio, nil, nil, errBorradorNoDisponibleEn()
	}
	mutador := &manejadorParticipacionBolsaDesarrollo{situacion: handlerSituacion, operaciones: handlerOperaciones, contratos: handlerContratos, sanciones: handlerSanciones, contacto: handlerContacto, datosContacto: handlerDatos, preparador: preparador, servicio: servicioContacto, servicioSituacion: servicioSituacion, datos: servicioDatos, emision: servicioEmision, fuente: fuenteCorreo}
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
	// No incorporación (CT124/Bolsa 000042): con la incorporación acreditada
	// encendida, el relevo lleva la baja a la bandeja de Bolsa.
	cerrar := cerrarAuditoria
	if incorporacionAcreditadaSolicitada(cfg) {
		var catalogo catalogoNoIncorporacionBolsa
		if c, ok := catalogoSanciones.(catalogoNoIncorporacionBolsa); ok && c != nil {
			catalogo = c
		}
		detener, err := iniciarEntregaNoIncorporacionesCTBolsaDesarrollo(ctx, cfg, alta.postgresql.ejecucion, alta.postgresql.bolsa, catalogo)
		if err != nil {
			slog.Error("entrega de no incorporaciones CT a Bolsa no iniciada", "causa", err)
			return nil, nil, nil, vacio, nil, nil, err
		}
		cerrar = func() {
			detener()
			cerrarAuditoria()
		}
	}
	completa = true
	return []vechttp.RutaExacta{{Ruta: bolsahttp.RutaBorradoresLlamamiento, Manejador: handler}, {Ruta: bolsahttp.RutaEmisionesLlamamiento, Manejador: handlerEmision}, {Ruta: bolsahttp.RutaOfertasPublicadas, Manejador: handlerOfertas}, {Ruta: bolsahttp.RutaResolucionesOferta, Manejador: handlerOfertas}, {Ruta: bolsahttp.RutaPlantillaCorreoLlamamiento, Manejador: handlerCorreo}, {Ruta: bolsahttp.RutaVistaPreviaCorreoLlamamiento, Manejador: handlerCorreo}}, []vechttp.RutaColeccion{{Prefijo: bolsahttp.RutaBorradoresLlamamiento, Manejador: handler}}, mutador, catalogoFronteras, envolver, cerrar, nil
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

// registrarRechazoBorradorLlamamientoDesarrollo deja en el registro del servidor
// por qué B-BACK denegó una operación. La respuesta HTTP sigue siendo el 403
// genérico; aquí solo van la fase y el texto del error, que son centinelas
// técnicos sin datos personales ni material de autorización.
func registrarRechazoBorradorLlamamientoDesarrollo(fase string, err error) {
	causa := "sin_error"
	if err != nil {
		causa = err.Error()
	}
	slog.Warn("bback: operacion denegada", "fase", fase, "causa", causa)
}
