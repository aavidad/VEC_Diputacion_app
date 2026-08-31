package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	RutaRegistroComunicacionLlamamiento = "" +
		"/api/vec/contratacion-temporal/llamamientos/comunicaciones"
	RutaResolucionComunicacionLlamamiento = "" +
		"/api/vec/contratacion-temporal/llamamientos/resoluciones"
)

var ErrManejadorComunicacionLlamamientoInvalido = errors.New(
	"contratacion temporal http: manejador de comunicacion de llamamiento invalido",
)

// EjecutorComunicacionLlamamiento limita HTTP a las dos operaciones locales
// del caso de uso. La transaccion resuelve autoridad, politica y persistencia;
// este adaptador no recibe ni fabrica esas capacidades.
type EjecutorComunicacionLlamamiento interface {
	Registrar(
		context.Context,
		ports.SolicitudRegistrarComunicacionLlamamiento,
	) (ports.ComunicacionProbatoria, error)
	Resolver(
		context.Context,
		ports.SolicitudResolverLlamamiento,
	) (ports.ResultadoResolucionLlamamiento, error)
}

type manejadorComunicacionLlamamiento struct {
	ejecutor EjecutorComunicacionLlamamiento
}

var _ http.Handler = (*manejadorComunicacionLlamamiento)(nil)
var _ EjecutorComunicacionLlamamiento = (*application.ServicioComunicacionLlamamiento)(nil)

// NuevoManejadorComunicacionLlamamiento no registra rutas, persiste estado ni
// conoce Bolsa. La composicion exterior debe proteger y registrar ambas rutas.
func NuevoManejadorComunicacionLlamamiento(
	ejecutor EjecutorComunicacionLlamamiento,
) (http.Handler, error) {
	if dependenciaNula(ejecutor) {
		return nil, ErrManejadorComunicacionLlamamientoInvalido
	}
	return &manejadorComunicacionLlamamiento{ejecutor: ejecutor}, nil
}

func (h *manejadorComunicacionLlamamiento) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r == nil || h == nil || dependenciaNula(h.ejecutor) {
		responderErrorComunicacionLlamamiento(
			w,
			errorServicioComunicacionLlamamientoNoDisponible,
		)
		return
	}
	if !rutaComunicacionLlamamientoExacta(r) {
		responderErrorComunicacionLlamamiento(
			w,
			errorRecursoComunicacionLlamamientoNoEncontrado,
		)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorComunicacionLlamamiento(
			w,
			errorMetodoComunicacionLlamamientoNoPermitido,
		)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorComunicacionLlamamiento(
			w,
			clasificarErrorComunicacionLlamamientoHTTP(err),
		)
		return
	}
	if problema := validarMetadatosComunicacionLlamamiento(r); problema != nil {
		responderErrorComunicacionLlamamiento(w, *problema)
		return
	}

	switch r.URL.Path {
	case RutaRegistroComunicacionLlamamiento:
		h.registrar(w, r)
	case RutaResolucionComunicacionLlamamiento:
		h.resolver(w, r)
	default:
		responderErrorComunicacionLlamamiento(
			w,
			errorRecursoComunicacionLlamamientoNoEncontrado,
		)
	}
}

func (h *manejadorComunicacionLlamamiento) registrar(
	w http.ResponseWriter,
	r *http.Request,
) {
	solicitud, err := solicitudRegistroComunicacionDesdePeticion(w, r)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorComunicacionLlamamiento(
			w,
			clasificarErrorComunicacionLlamamientoHTTP(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorComunicacionLlamamiento(
			w,
			errorEntradaComunicacionLlamamiento(err),
		)
		return
	}

	resultado, err := h.ejecutor.Registrar(r.Context(), solicitud)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorComunicacionLlamamiento(
			w,
			clasificarErrorComunicacionLlamamientoHTTP(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorComunicacionLlamamiento(
			w,
			clasificarErrorComunicacionLlamamientoHTTP(err),
		)
		return
	}
	salida, estadoHTTP, valida := proyectarRegistroComunicacionLlamamiento(
		solicitud,
		resultado,
	)
	if !valida {
		responderErrorComunicacionLlamamiento(
			w,
			errorResultadoComunicacionLlamamientoNoConfiable,
		)
		return
	}
	responderJSONCobertura(
		w,
		estadoHTTP,
		envoltorioRegistroComunicacionLlamamiento{Data: salida},
	)
}

func (h *manejadorComunicacionLlamamiento) resolver(
	w http.ResponseWriter,
	r *http.Request,
) {
	solicitud, err := solicitudResolucionLlamamientoDesdePeticion(w, r)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorComunicacionLlamamiento(
			w,
			clasificarErrorComunicacionLlamamientoHTTP(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorComunicacionLlamamiento(
			w,
			errorEntradaComunicacionLlamamiento(err),
		)
		return
	}

	resultado, err := h.ejecutor.Resolver(r.Context(), solicitud)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorComunicacionLlamamiento(
			w,
			clasificarErrorComunicacionLlamamientoHTTP(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorComunicacionLlamamiento(
			w,
			clasificarErrorComunicacionLlamamientoHTTP(err),
		)
		return
	}
	salida, estadoHTTP, valida := proyectarResolucionComunicacionLlamamiento(
		solicitud,
		resultado,
	)
	if !valida {
		responderErrorComunicacionLlamamiento(
			w,
			errorResultadoComunicacionLlamamientoNoConfiable,
		)
		return
	}
	responderJSONCobertura(
		w,
		estadoHTTP,
		envoltorioResolucionComunicacionLlamamiento{Data: salida},
	)
}

func rutaComunicacionLlamamientoExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		(r.URL.Path != RutaRegistroComunicacionLlamamiento &&
			r.URL.Path != RutaResolucionComunicacionLlamamiento) {
		return false
	}
	return r.URL.EscapedPath() == r.URL.Path &&
		!strings.Contains(r.URL.Path, "%")
}
