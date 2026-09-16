package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaPropuestaFormalizacion = "" +
	"/api/vec/contratacion-temporal/formalizacion/propuestas"

var ErrManejadorPropuestaFormalizacionInvalido = errors.New(
	"contratacion temporal http: manejador de propuesta de formalizacion invalido",
)

// ContextoServidorPropuestaFormalizacion contiene la unica autoridad que el
// contrato local permite incorporar a la intencion recibida por HTTP.
type ContextoServidorPropuestaFormalizacion struct {
	OrganizacionRef string
}

func (c ContextoServidorPropuestaFormalizacion) valido() bool {
	return domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

// AutoridadServidorPropuestaFormalizacion resuelve la organizacion desde la
// frontera confiable. No recibe URL, cabeceras, cuerpo ni http.Request.
type AutoridadServidorPropuestaFormalizacion interface {
	ResolverContextoPropuestaFormalizacion(
		context.Context,
	) (ContextoServidorPropuestaFormalizacion, error)
}

// EjecutorPropuestaFormalizacion es la unica capacidad de aplicacion expuesta
// a este adaptador. El propio caso de uso conserva el unico commit local.
type EjecutorPropuestaFormalizacion interface {
	PrepararYConfirmar(
		context.Context,
		ports.SolicitudPropuestaFormalizacion,
	) (ports.ResultadoPropuestaFormalizacion, error)
}

type manejadorPropuestaFormalizacion struct {
	autoridad AutoridadServidorPropuestaFormalizacion
	ejecutor  EjecutorPropuestaFormalizacion
}

var _ http.Handler = (*manejadorPropuestaFormalizacion)(nil)
var _ EjecutorPropuestaFormalizacion = (*application.ServicioPropuestaFormalizacion)(nil)

// NuevoManejadorPropuestaFormalizacion no registra rutas, compone identidad,
// persiste estado ni incorpora capacidades documentales o de firma.
func NuevoManejadorPropuestaFormalizacion(
	autoridad AutoridadServidorPropuestaFormalizacion,
	ejecutor EjecutorPropuestaFormalizacion,
) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(ejecutor) {
		return nil, ErrManejadorPropuestaFormalizacionInvalido
	}
	return &manejadorPropuestaFormalizacion{
		autoridad: autoridad,
		ejecutor:  ejecutor,
	}, nil
}

func (h *manejadorPropuestaFormalizacion) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r == nil || h == nil || dependenciaNula(h.autoridad) ||
		dependenciaNula(h.ejecutor) {
		responderErrorPropuestaFormalizacion(
			w, r, errorServicioPropuestaFormalizacionNoDisponible,
		)
		return
	}
	if !rutaPropuestaFormalizacionExacta(r) {
		responderErrorPropuestaFormalizacion(
			w, r, errorRecursoPropuestaFormalizacionNoEncontrado,
		)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorPropuestaFormalizacion(
			w, r, errorMetodoPropuestaFormalizacionNoPermitido,
		)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorPropuestaFormalizacion(
			w, r, clasificarErrorPropuestaFormalizacionHTTP(err), err,
		)
		return
	}
	if problema := validarMetadatosPropuestaFormalizacion(r); problema != nil {
		responderErrorPropuestaFormalizacion(w, r, *problema)
		return
	}

	entrada, err := propuestaFormalizacionDesdePeticion(w, r)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorPropuestaFormalizacion(
			w, r, clasificarErrorPropuestaFormalizacionHTTP(errContexto), errContexto,
		)
		return
	}
	if err != nil {
		responderErrorPropuestaFormalizacion(
			w, r, errorEntradaPropuestaFormalizacion(err), err,
		)
		return
	}
	contextoServidor, err := h.autoridad.
		ResolverContextoPropuestaFormalizacion(r.Context())
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorPropuestaFormalizacion(
			w, r, clasificarErrorPropuestaFormalizacionHTTP(errContexto), errContexto,
		)
		return
	}
	if err != nil {
		responderErrorPropuestaFormalizacion(
			w, r, clasificarErrorPropuestaFormalizacionHTTP(err), err,
		)
		return
	}
	if !contextoServidor.valido() {
		responderErrorPropuestaFormalizacion(
			w, r, errorServicioPropuestaFormalizacionNoDisponible,
		)
		return
	}
	solicitud, err := entrada.solicitud(contextoServidor)
	if err != nil {
		responderErrorPropuestaFormalizacion(
			w, r, errorContenidoPropuestaFormalizacionInvalido,
		)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorPropuestaFormalizacion(
			w, r, clasificarErrorPropuestaFormalizacionHTTP(err), err,
		)
		return
	}

	resultado, err := h.ejecutor.PrepararYConfirmar(r.Context(), solicitud.Clonar())
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorPropuestaFormalizacion(
			w, r, clasificarErrorPropuestaFormalizacionHTTP(errContexto), errContexto,
		)
		return
	}
	if err != nil {
		if !resultado.EsCero() {
			responderErrorPropuestaFormalizacion(
				w, r, errorResultadoPropuestaFormalizacionNoConfiable, err,
			)
			return
		}
		responderErrorPropuestaFormalizacion(
			w, r, clasificarErrorPropuestaFormalizacionHTTP(err), err,
		)
		return
	}
	salida, estadoHTTP, valida := proyectarPropuestaFormalizacion(
		solicitud,
		resultado,
	)
	if !valida {
		responderErrorPropuestaFormalizacion(
			w, r, errorResultadoPropuestaFormalizacionNoConfiable,
		)
		return
	}
	responderJSONCobertura(
		w, r, estadoHTTP,
		envoltorioPropuestaFormalizacion{Data: salida},
	)
}

func rutaPropuestaFormalizacionExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.URL.Path != RutaPropuestaFormalizacion {
		return false
	}
	return r.URL.EscapedPath() == r.URL.Path &&
		!strings.Contains(r.URL.Path, "%")
}
