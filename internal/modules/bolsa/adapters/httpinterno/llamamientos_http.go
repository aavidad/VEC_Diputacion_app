package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"strings"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const RutaPropuestasLlamamiento = "/api/vec/bolsa/propuestas-llamamiento"

var (
	ErrHandlerPropuestasLlamamientoInvalido = errors.New(
		"bolsa http interno: handler de propuestas de llamamiento invalido",
	)
	ErrDependenciaPropuestasLlamamientoNoDisponible = errors.New(
		"bolsa http interno: dependencia de propuestas de llamamiento no disponible",
	)
)

// EntradaPreparacionPropuestaLlamamientoInterno contiene exclusivamente el
// selector no confiable que el contrato HTTP admite. No transporta actor,
// perfil, sesion, autenticacion, autorizacion, roles, ambitos ni correlacion.
type EntradaPreparacionPropuestaLlamamientoInterno struct {
	NecesidadRef string
}

// PreparadorSolicitudPropuestaLlamamientoInterno es la unica frontera que
// puede constituir la solicitud interna. Recibe el contexto ya enriquecido por
// la composicion confiable y el selector no confiable, pero nunca recibe la
// peticion HTTP ni puede reconstruir identidad desde sus cabeceras o cuerpo.
// La autorizacion sigue perteneciendo al caso de uso y a su PDP; este handler
// no interpreta decisiones ni credenciales.
type PreparadorSolicitudPropuestaLlamamientoInterno interface {
	PrepararSolicitudPropuestaLlamamientoInterno(
		context.Context,
		EntradaPreparacionPropuestaLlamamientoInterno,
	) (puertosbolsa.SolicitudProponerLlamamiento, error)
}

// ProponentePrimerLlamamiento es la superficie minima del caso de aplicacion.
type ProponentePrimerLlamamiento interface {
	ProponerPrimerLlamamiento(
		context.Context,
		puertosbolsa.SolicitudProponerLlamamiento,
	) (dominiobolsa.PropuestaLlamamiento, error)
}

type HandlerPropuestasLlamamiento struct {
	preparador PreparadorSolicitudPropuestaLlamamientoInterno
	proponente ProponentePrimerLlamamiento
}

var (
	_ http.Handler                = (*HandlerPropuestasLlamamiento)(nil)
	_ ProponentePrimerLlamamiento = (*aplicacionbolsa.ServicioLlamamientos)(nil)
)

func NuevoHandlerPropuestasLlamamiento(
	preparador PreparadorSolicitudPropuestaLlamamientoInterno,
	proponente ProponentePrimerLlamamiento,
) (http.Handler, error) {
	if dependenciaNula(preparador) || dependenciaNula(proponente) {
		return nil, ErrHandlerPropuestasLlamamientoInvalido
	}
	return &HandlerPropuestasLlamamiento{
		preparador: preparador,
		proponente: proponente,
	}, nil
}

func (h *HandlerPropuestasLlamamiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	correlacion := nuevaCorrelacionErrorLlamamiento()
	if r == nil || h == nil || dependenciaNula(h.preparador) || dependenciaNula(h.proponente) {
		responderErrorLlamamiento(w, http.StatusServiceUnavailable, "servicio_no_disponible", correlacion)
		return
	}
	if !rutaPropuestaLlamamientoExacta(r) {
		responderErrorLlamamiento(w, http.StatusNotFound, "recurso_no_encontrado", correlacion)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorLlamamiento(w, http.StatusMethodNotAllowed, "metodo_no_permitido", correlacion)
		return
	}
	// El caso de aplicacion no consume una clave de intento ni puede garantizar
	// replay del mismo resultado HTTP. La unicidad de necesidad/version/huella
	// en la transaccion es una barrera semantica distinta; no se presenta como
	// idempotencia HTTP y una cabecera de ese tipo se rechaza expresamente.
	if cabeceraPresente(r.Header, "Idempotency-Key") {
		responderErrorLlamamiento(
			w,
			http.StatusBadRequest,
			"idempotencia_http_no_soportada",
			correlacion,
		)
		return
	}
	if r.ContentLength > maximoCuerpoSolicitudLlamamientoBytes {
		responderErrorLlamamiento(
			w,
			http.StatusRequestEntityTooLarge,
			"peticion_demasiado_grande",
			correlacion,
		)
		return
	}
	if !metadatosPropuestaLlamamientoPermitidos(r) {
		responderErrorLlamamiento(w, http.StatusBadRequest, "peticion_no_permitida", correlacion)
		return
	}

	entrada, err := entradaPropuestaLlamamientoDesdePeticion(w, r)
	if err != nil {
		responderErrorEntradaLlamamiento(w, err, correlacion)
		return
	}
	solicitud, err := h.preparador.PrepararSolicitudPropuestaLlamamientoInterno(r.Context(), entrada)
	if err != nil {
		responderErrorLlamamientoClasificado(w, err, correlacion)
		return
	}
	if correlacionBorradorValida(solicitud.CorrelacionRef) {
		correlacion = solicitud.CorrelacionRef
	}
	solicitudCanonica, err := solicitud.Clonar()
	if err != nil || solicitudCanonica.NecesidadRef != entrada.NecesidadRef {
		responderErrorLlamamiento(w, http.StatusInternalServerError, "error_interno", correlacion)
		return
	}

	propuesta, err := h.proponente.ProponerPrimerLlamamiento(r.Context(), solicitudCanonica)
	if err != nil {
		responderErrorLlamamientoClasificado(w, err, correlacion)
		return
	}
	respuesta, etag, err := nuevaRespuestaPropuestaLlamamiento(propuesta, entrada.NecesidadRef)
	if err != nil {
		responderErrorLlamamiento(w, http.StatusInternalServerError, "error_interno", correlacion)
		return
	}
	w.Header().Set("ETag", etag)
	responderJSONLlamamiento(w, http.StatusCreated, respuesta, correlacion)
}

func rutaPropuestaLlamamientoExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.Path != RutaPropuestasLlamamiento ||
		r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil || r.URL.Fragment != "" ||
		r.RequestURI != RutaPropuestasLlamamiento {
		return false
	}
	escapada := r.URL.EscapedPath()
	return escapada == RutaPropuestasLlamamiento && !strings.Contains(escapada, "%")
}

func responderErrorLlamamientoClasificado(w http.ResponseWriter, err error, correlacion string) {
	estado, codigo := clasificarErrorLlamamiento(err)
	responderErrorLlamamiento(w, estado, codigo, correlacion)
}

func clasificarErrorLlamamiento(err error) (int, string) {
	switch {
	case errors.Is(err, ErrAutenticacionInternaAusente):
		return http.StatusUnauthorized, "autenticacion_requerida"
	case errors.Is(err, puertosbolsa.ErrNecesidadLlamamientoYaPropuesta),
		errors.Is(err, puertosbolsa.ErrPropuestaLlamamientoYaExiste),
		errors.Is(err, puertosbolsa.ErrReferenciaLlamamientoYaUtilizada),
		errors.Is(err, puertosbolsa.ErrDecisionAutorizacionLlamamientoUsada):
		return http.StatusConflict, "propuesta_en_conflicto"
	case esDependenciaLlamamientoNoDisponible(err):
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada),
		errors.Is(err, dominiovec.ErrPermissionDenied),
		errors.Is(err, dominiobolsa.ErrSinParticipacionElegible):
		return http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, puertosbolsa.ErrRecursoNecesidadNoEncontrado),
		errors.Is(err, puertosbolsa.ErrDatosLlamamientoNoEncontrados):
		return http.StatusNotFound, "necesidad_no_disponible"
	default:
		return http.StatusInternalServerError, "error_interno"
	}
}

func esDependenciaLlamamientoNoDisponible(err error) bool {
	return errors.Is(err, ErrDependenciaPropuestasLlamamientoNoDisponible) ||
		errors.Is(err, puertosvec.ErrFuenteContextoActorNoDisponible) ||
		errors.Is(err, puertosvec.ErrRevalidacionAutenticacionActorNoDisponible) ||
		errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) ||
		errors.Is(err, puertosvec.ErrRegistroDecisionNoDisponible) ||
		errors.Is(err, puertosvec.ErrRegistroDenegacionNoDisponible) ||
		errors.Is(err, puertosbolsa.ErrRecursoNecesidadAmbiguo) ||
		errors.Is(err, puertosbolsa.ErrRecursoNecesidadNoConfiable) ||
		errors.Is(err, puertosbolsa.ErrDatosLlamamientoAmbiguos) ||
		errors.Is(err, puertosbolsa.ErrDatosLlamamientoNoConfiables) ||
		errors.Is(err, puertosbolsa.ErrMotorElegibilidadNoDisponible) ||
		errors.Is(err, puertosbolsa.ErrEvaluacionMotorNoConfiable) ||
		errors.Is(err, puertosbolsa.ErrGeneracionReferenciaLlamamiento) ||
		errors.Is(err, puertosbolsa.ErrPersistenciaPropuestaNoDisponible) ||
		errors.Is(err, puertosbolsa.ErrCapacidadMemoriaLlamamientosAgotada) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
