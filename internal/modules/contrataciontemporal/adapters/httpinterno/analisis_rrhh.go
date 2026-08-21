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
	RutaRegistroAnalisisRRHH      = "/api/vec/contratacion-temporal/analisis/registros"
	RutaRectificacionAnalisisRRHH = "/api/vec/contratacion-temporal/analisis/rectificaciones"
)

var ErrManejadorAnalisisRRHHInvalido = errors.New(
	"contratacion temporal http: manejador de analisis RRHH invalido",
)

// ContextoCanalAnalisisRRHH contiene exclusivamente referencias resueltas por
// la frontera confiable del servidor. La intención funcional nunca se obtiene
// de este contexto y la autoridad no recibe la petición HTTP.
type ContextoCanalAnalisisRRHH struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
}

func (c ContextoCanalAnalisisRRHH) valido() bool {
	return (ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: c.AutenticacionRef,
		SesionRef:        c.SesionRef,
		PerfilRef:        c.PerfilRef,
	}).Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

// AutoridadContextoCanalAnalisisRRHH debe estar ligada por composición a una
// superficie interna autenticada. No puede derivar autoridad de cuerpo, URL o
// cabeceras aportadas por el cliente.
type AutoridadContextoCanalAnalisisRRHH interface {
	ResolverContextoCanalAnalisisRRHH(
		context.Context,
	) (ContextoCanalAnalisisRRHH, error)
}

// EjecutorAnalisisRRHH conserva únicamente las dos capacidades nominales del
// caso de uso compartido por cualquier adaptador cliente.
type EjecutorAnalisisRRHH interface {
	Registrar(
		context.Context,
		application.SolicitudRegistrarAnalisis,
	) (ports.ReciboOperacionAnalisis, error)
	Rectificar(
		context.Context,
		application.SolicitudRectificarAnalisis,
	) (ports.ReciboOperacionAnalisis, error)
}

type manejadorAnalisisRRHH struct {
	autoridad AutoridadContextoCanalAnalisisRRHH
	ejecutor  EjecutorAnalisisRRHH
}

var _ http.Handler = (*manejadorAnalisisRRHH)(nil)
var _ EjecutorAnalisisRRHH = (*application.ServicioOperacionAnalisis)(nil)

// NuevoManejadorAnalisisRRHH no registra rutas ni construye identidad,
// autorización o dependencias del caso de uso.
func NuevoManejadorAnalisisRRHH(
	autoridad AutoridadContextoCanalAnalisisRRHH,
	ejecutor EjecutorAnalisisRRHH,
) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(ejecutor) {
		return nil, ErrManejadorAnalisisRRHHInvalido
	}
	return &manejadorAnalisisRRHH{
		autoridad: autoridad,
		ejecutor:  ejecutor,
	}, nil
}

func (h *manejadorAnalisisRRHH) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r == nil || h == nil || dependenciaNula(h.autoridad) ||
		dependenciaNula(h.ejecutor) {
		responderErrorCobertura(w, errorServicioCoberturaNoDisponible)
		return
	}
	if !rutaAnalisisRRHHExacta(r) {
		responderErrorCobertura(w, errorRecursoCoberturaNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorCobertura(w, errorMetodoCoberturaNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorCobertura(w, clasificarErrorAnalisisRRHH(err))
		return
	}
	if problema := validarMetadatosCobertura(r); problema != nil {
		responderErrorCobertura(w, *problema)
		return
	}

	entrada, err := operacionAnalisisRRHHDesdePeticion(w, r)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorCobertura(
			w,
			clasificarErrorAnalisisRRHH(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorCobertura(w, errorEntradaAnalisisRRHH(err))
		return
	}
	contextoCanal, err := h.autoridad.
		ResolverContextoCanalAnalisisRRHH(r.Context())
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorCobertura(
			w,
			clasificarErrorAnalisisRRHH(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorCobertura(w, clasificarErrorAnalisisRRHH(err))
		return
	}
	if !contextoCanal.valido() {
		responderErrorCobertura(w, errorServicioCoberturaNoDisponible)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorCobertura(w, clasificarErrorAnalisisRRHH(err))
		return
	}

	recibo, err := h.ejecutar(r.Context(), contextoCanal, entrada)
	if reciboAnalisisRRHHEsSeguro(recibo, contextoCanal, entrada) {
		responderExitoAnalisisRRHH(w, recibo)
		return
	}
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorCobertura(
			w,
			clasificarErrorAnalisisRRHH(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorCobertura(w, clasificarErrorAnalisisRRHH(err))
		return
	}
	responderErrorCobertura(w, errorResultadoCoberturaNoConfiable)
}

func (h *manejadorAnalisisRRHH) ejecutar(
	ctx context.Context,
	contexto ContextoCanalAnalisisRRHH,
	entrada entradaOperacionAnalisisRRHH,
) (ports.ReciboOperacionAnalisis, error) {
	if entrada.operacion == ports.OperacionRegistrarAnalisis {
		return h.ejecutor.Registrar(ctx, entrada.solicitudRegistro(contexto))
	}
	return h.ejecutor.Rectificar(
		ctx,
		entrada.solicitudRectificacion(contexto),
	)
}

func rutaAnalisisRRHHExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" ||
		(r.URL.Path != RutaRegistroAnalisisRRHH &&
			r.URL.Path != RutaRectificacionAnalisisRRHH) {
		return false
	}
	return r.URL.EscapedPath() == r.URL.Path &&
		!strings.Contains(r.URL.Path, "%")
}

// clasificarErrorAnalisisRRHH reutiliza el vocabulario público y las claves
// i18n ya existentes para efectos de contratación temporal. Ninguna causa
// privada cruza la frontera HTTP.
func clasificarErrorAnalisisRRHH(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, context.Canceled):
		return errorCancelacionCobertura
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoCobertura
	case errors.Is(err, ErrContextoCanalAusente),
		errors.Is(err, ErrContextoCanalCaducado):
		return errorAutenticacionCoberturaRequerida
	case errors.Is(err, ErrContextoCanalOrganizacionDenegada),
		errors.Is(err, ports.ErrAutorizacionDenegada),
		errors.Is(err, application.ErrOperacionAnalisisDenegada):
		return errorAccesoCoberturaDenegado
	case errors.Is(err, application.ErrOperacionAnalisisEnConflicto),
		errors.Is(err, ports.ErrClaveIdempotenciaOperacionAnalisisUsada),
		errors.Is(err, ports.ErrConjuntoFuentesAnalisisYaConsumido),
		errors.Is(err, domain.ErrVersionEnConflicto):
		return errorConflictoCobertura
	case errors.Is(err, application.ErrSolicitudOperacionAnalisisInvalida),
		errors.Is(err, domain.ErrDatoInvalido):
		return errorContenidoCoberturaInvalido
	case errors.Is(err, application.ErrResultadoOperacionAnalisisNoConfiable),
		errors.Is(err, ports.ErrResultadoOperacionAnalisisNoConfiable),
		errors.Is(err, ports.ErrOrdenOperacionAnalisisInvalida):
		return errorResultadoCoberturaNoConfiable
	case errors.Is(err, ErrContextoCanalNoDisponible),
		errors.Is(err, application.ErrServicioOperacionAnalisisInvalido),
		errors.Is(err, application.ErrDependenciaOperacionAnalisisNoDisponible),
		errors.Is(err, ports.ErrPersistenciaOperacionAnalisisNoDisponible):
		return errorServicioCoberturaNoDisponible
	default:
		return errorInternoCobertura
	}
}
