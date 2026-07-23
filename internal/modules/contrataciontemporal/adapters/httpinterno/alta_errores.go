package httpinterno

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrManejadorAltaInvalido = errors.New(
		"contratacion temporal http: manejador de alta invalido",
	)
	ErrContextoCanalAusente = errors.New(
		"contratacion temporal http: contexto de canal ausente",
	)
	ErrContextoCanalCaducado = errors.New(
		"contratacion temporal http: contexto de canal caducado",
	)
	ErrContextoCanalOrganizacionDenegada = errors.New(
		"contratacion temporal http: organizacion de canal denegada",
	)
	ErrContextoCanalNoDisponible = errors.New(
		"contratacion temporal http: autoridad de canal no disponible",
	)
	// ErrResultadoAltaIndeterminado será unido por O2-06 cuando el COMMIT no
	// pueda confirmarse ni descartarse. El cliente no debe repetir a ciegas.
	ErrResultadoAltaIndeterminado = errors.New(
		"contratacion temporal http: resultado de alta indeterminado",
	)
)

type errorPublicoAlta struct {
	estado    int
	codigo    string
	claveI18n string
}

var (
	errorPeticionNoValida = nuevoErrorPublico(
		http.StatusBadRequest, "peticion_no_valida", "api.contratacion_temporal.alta.error.peticion_no_valida",
	)
	errorPeticionNoPermitida = nuevoErrorPublico(
		http.StatusBadRequest, "peticion_no_permitida", "api.contratacion_temporal.alta.error.peticion_no_permitida",
	)
	errorAutenticacionRequerida = nuevoErrorPublico(
		http.StatusUnauthorized, "autenticacion_requerida", "api.contratacion_temporal.alta.error.autenticacion_requerida",
	)
	errorAccesoDenegado = nuevoErrorPublico(
		http.StatusForbidden, "acceso_denegado", "api.contratacion_temporal.alta.error.acceso_denegado",
	)
	errorRecursoNoEncontrado = nuevoErrorPublico(
		http.StatusNotFound, "recurso_no_encontrado", "api.contratacion_temporal.alta.error.recurso_no_encontrado",
	)
	errorMetodoNoPermitido = nuevoErrorPublico(
		http.StatusMethodNotAllowed, "metodo_no_permitido", "api.contratacion_temporal.alta.error.metodo_no_permitido",
	)
	errorRepresentacionNoAceptable = nuevoErrorPublico(
		http.StatusNotAcceptable, "representacion_no_aceptable", "api.contratacion_temporal.alta.error.representacion_no_aceptable",
	)
	errorPeticionCancelada = nuevoErrorPublico(
		http.StatusRequestTimeout, "peticion_cancelada", "api.contratacion_temporal.alta.error.peticion_cancelada",
	)
	errorClaveIdempotenciaReutilizada = nuevoErrorPublico(
		http.StatusConflict, "clave_idempotencia_reutilizada", "api.contratacion_temporal.alta.error.clave_idempotencia_reutilizada",
	)
	errorPeticionDemasiadoGrande = nuevoErrorPublico(
		http.StatusRequestEntityTooLarge, "peticion_demasiado_grande", "api.contratacion_temporal.alta.error.peticion_demasiado_grande",
	)
	errorTipoContenidoNoAdmitido = nuevoErrorPublico(
		http.StatusUnsupportedMediaType, "tipo_contenido_no_admitido", "api.contratacion_temporal.alta.error.tipo_contenido_no_admitido",
	)
	errorContenidoNoValido = nuevoErrorPublico(
		http.StatusUnprocessableEntity, "contenido_no_valido", "api.contratacion_temporal.alta.error.contenido_no_valido",
	)
	errorResultadoNoConfiable = nuevoErrorPublico(
		http.StatusBadGateway, "resultado_no_confiable", "api.contratacion_temporal.alta.error.resultado_no_confiable",
	)
	errorServicioNoDisponible = nuevoErrorPublico(
		http.StatusServiceUnavailable, "servicio_no_disponible", "api.contratacion_temporal.alta.error.servicio_no_disponible",
	)
	errorOperacionPendiente = nuevoErrorPublico(
		http.StatusServiceUnavailable, "operacion_pendiente", "api.contratacion_temporal.alta.error.operacion_pendiente",
	)
	errorPlazoAgotado = nuevoErrorPublico(
		http.StatusGatewayTimeout, "plazo_agotado", "api.contratacion_temporal.alta.error.plazo_agotado",
	)
	errorInterno = nuevoErrorPublico(
		http.StatusInternalServerError, "error_interno", "api.contratacion_temporal.alta.error.error_interno",
	)
)

func nuevoErrorPublico(estado int, codigo, clave string) errorPublicoAlta {
	return errorPublicoAlta{estado: estado, codigo: codigo, claveI18n: clave}
}

func clasificarErrorAlta(err error) errorPublicoAlta {
	switch {
	case errors.Is(err, context.Canceled):
		return errorPeticionCancelada
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoAgotado
	case errors.Is(err, ErrContextoCanalAusente), errors.Is(err, ErrContextoCanalCaducado):
		return errorAutenticacionRequerida
	case errors.Is(err, ErrContextoCanalOrganizacionDenegada),
		errors.Is(err, ports.ErrAutorizacionDenegada):
		return errorAccesoDenegado
	case errors.Is(err, ports.ErrClaveIdempotenciaUsada):
		return errorClaveIdempotenciaReutilizada
	case errors.Is(err, ErrResultadoAltaIndeterminado):
		return errorOperacionPendiente
	case errors.Is(err, application.ErrResultadoRegistroNoConfiable),
		errors.Is(err, ports.ErrOrdenAltaInvalida):
		return errorResultadoNoConfiable
	case errors.Is(err, application.ErrSolicitudRegistroInvalida),
		errors.Is(err, domain.ErrDatoInvalido):
		return errorContenidoNoValido
	case errors.Is(err, ErrContextoCanalNoDisponible),
		errors.Is(err, application.ErrServicioRegistroInvalido),
		errors.Is(err, ports.ErrPersistenciaNoDisponible),
		errors.Is(err, ports.ErrFlujoNoDisponible),
		errors.Is(err, ports.ErrMotivoAutorizacionNoDisponible):
		return errorServicioNoDisponible
	default:
		return errorInterno
	}
}

type envelopeErrorAlta struct {
	Error detalleErrorAlta `json:"error"`
}

type detalleErrorAlta struct {
	Codigo         string `json:"codigo"`
	ClaveI18n      string `json:"clave_i18n"`
	CorrelacionRef string `json:"correlacion_ref"`
}

func responderErrorAlta(w http.ResponseWriter, problema errorPublicoAlta) {
	correlacion := nuevaCorrelacionErrorAlta()
	responderJSONAlta(w, problema.estado, envelopeErrorAlta{Error: detalleErrorAlta{
		Codigo:         problema.codigo,
		ClaveI18n:      problema.claveI18n,
		CorrelacionRef: correlacion,
	}})
}

func responderJSONAlta(w http.ResponseWriter, estado int, valor any) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > 16*1024 {
		estado = http.StatusInternalServerError
		contenido, _ = json.Marshal(envelopeErrorAlta{Error: detalleErrorAlta{
			Codigo:         errorInterno.codigo,
			ClaveI18n:      errorInterno.claveI18n,
			CorrelacionRef: nuevaCorrelacionErrorAlta(),
		}})
	}
	aplicarCabecerasAlta(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

func aplicarCabecerasAlta(w http.ResponseWriter) {
	for _, nombre := range []string{
		"Set-Cookie",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Expose-Headers",
		"Content-Encoding",
		"Location",
		"Retry-After",
	} {
		w.Header().Del(nombre)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("X-Frame-Options", "DENY")
}

func nuevaCorrelacionErrorAlta() string {
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		return "corr_no_disponible"
	}
	return "corr_" + hex.EncodeToString(aleatorio)
}
