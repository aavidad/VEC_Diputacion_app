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

const (
	RutaAsignaciones   = "/api/vec/contratacion-temporal/asignaciones"
	RutaReasignaciones = "/api/vec/contratacion-temporal/reasignaciones"
)

var ErrManejadorAsignacionInvalido = errors.New(
	"contratacion temporal http: manejador de asignacion invalido",
)

// ContextoCanalAsignacion contiene solo autoridad resuelta por la frontera
// confiable. La intención funcional llega por un contrato separado.
type ContextoCanalAsignacion struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
}

func (c ContextoCanalAsignacion) valido() bool {
	return (ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: c.AutenticacionRef,
		SesionRef:        c.SesionRef,
		PerfilRef:        c.PerfilRef,
	}).Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

// AutoridadContextoCanalAsignacion no recibe la petición HTTP. Composición
// debe ligarla a identidad, sesión, perfil y organización ya autenticados.
type AutoridadContextoCanalAsignacion interface {
	ResolverContextoCanalAsignacion(
		context.Context,
	) (ContextoCanalAsignacion, error)
}

// EjecutorAsignacion conserva las dos capacidades nominales de aplicación.
type EjecutorAsignacion interface {
	Asignar(
		context.Context,
		application.SolicitudAsignarUnidad,
	) (ports.ReciboAsignacion, error)
	Reasignar(
		context.Context,
		application.SolicitudReasignarUnidad,
	) (ports.ReciboAsignacion, error)
}

type manejadorAsignacion struct {
	autoridad AutoridadContextoCanalAsignacion
	ejecutor  EjecutorAsignacion
}

var _ http.Handler = (*manejadorAsignacion)(nil)
var _ EjecutorAsignacion = (*application.ServicioAsignacion)(nil)

// NuevoManejadorAsignacion no registra rutas ni crea autoridad de canal.
func NuevoManejadorAsignacion(
	autoridad AutoridadContextoCanalAsignacion,
	ejecutor EjecutorAsignacion,
) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(ejecutor) {
		return nil, ErrManejadorAsignacionInvalido
	}
	return &manejadorAsignacion{autoridad: autoridad, ejecutor: ejecutor}, nil
}

func (h *manejadorAsignacion) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaNula(h.autoridad) ||
		dependenciaNula(h.ejecutor) {
		responderErrorAsignacion(w, errorServicioAsignacionNoDisponible)
		return
	}
	operacion, rutaValida := operacionAsignacionHTTP(r)
	if !rutaValida {
		responderErrorAsignacion(w, errorRecursoAsignacionNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorAsignacion(w, errorMetodoAsignacionNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorAsignacion(w, clasificarErrorAsignacionHTTP(err))
		return
	}
	if problema := validarMetadatosAsignacion(r); problema != nil {
		responderErrorAsignacion(w, *problema)
		return
	}

	entrada, err := asignacionDesdePeticion(w, r, operacion)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorAsignacion(
			w,
			clasificarErrorAsignacionHTTP(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorAsignacion(w, errorEntradaAsignacion(err))
		return
	}

	contextoCanal, err := h.autoridad.
		ResolverContextoCanalAsignacion(r.Context())
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorAsignacion(
			w,
			clasificarErrorAsignacionHTTP(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorAsignacion(w, clasificarErrorAsignacionHTTP(err))
		return
	}
	if !contextoCanal.valido() {
		responderErrorAsignacion(w, errorServicioAsignacionNoDisponible)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorAsignacion(w, clasificarErrorAsignacionHTTP(err))
		return
	}

	recibo, err := ejecutarAsignacionHTTP(
		r.Context(),
		h.ejecutor,
		operacion,
		entrada,
		contextoCanal,
	)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorAsignacion(
			w,
			clasificarErrorAsignacionHTTP(errContexto),
		)
		return
	}
	if err != nil {
		if recibo != (ports.ReciboAsignacion{}) {
			responderErrorAsignacion(
				w,
				errorResultadoAsignacionNoConfiable,
			)
			return
		}
		responderErrorAsignacion(w, clasificarErrorAsignacionHTTP(err))
		return
	}
	if !reciboAsignacionSeguro(recibo, contextoCanal, entrada, operacion) {
		responderErrorAsignacion(w, errorResultadoAsignacionNoConfiable)
		return
	}
	responderExitoAsignacion(w, recibo)
}

func operacionAsignacionHTTP(
	r *http.Request,
) (ports.TipoOperacionAsignacion, bool) {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.URL.EscapedPath() != r.URL.Path || strings.Contains(r.URL.Path, "%") {
		return "", false
	}
	switch r.URL.Path {
	case RutaAsignaciones:
		return ports.OperacionRegistrarAsignacion, true
	case RutaReasignaciones:
		return ports.OperacionRegistrarReasignacion, true
	default:
		return "", false
	}
}

func ejecutarAsignacionHTTP(
	ctx context.Context,
	ejecutor EjecutorAsignacion,
	operacion ports.TipoOperacionAsignacion,
	entrada entradaAsignacionHTTP,
	contexto ContextoCanalAsignacion,
) (ports.ReciboAsignacion, error) {
	if operacion == ports.OperacionRegistrarAsignacion {
		return ejecutor.Asignar(ctx, entrada.solicitudAsignar(contexto))
	}
	return ejecutor.Reasignar(ctx, entrada.solicitudReasignar(contexto))
}
