package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaAltaSolicitudes = "/api/interno/v1/contratacion-temporal/solicitudes"

// AutoridadContextoCanal resuelve desde el contexto confiable que O2-07 ligará
// al canal interno las referencias de autenticación, sesión, perfil,
// organización e idempotencia. Debe devolver Solicitud vacía: el manejador
// incorpora después los únicos datos funcionales admitidos.
//
// La petición HTTP, su cuerpo y sus cabeceras nunca se entregan a esta
// frontera, por lo que no puede convertir texto del cliente en autoridad.
type AutoridadContextoCanal interface {
	ResolverContextoCanalAlta(
		context.Context,
	) (application.SolicitudRegistrarExpediente, error)
}

// EjecutorAlta es la superficie mínima del caso de uso compartido por web,
// escritorio, CLI y MCP.
type EjecutorAlta interface {
	Registrar(
		context.Context,
		application.SolicitudRegistrarExpediente,
	) (ports.ReciboAlta, error)
}

type manejadorAlta struct {
	autoridad AutoridadContextoCanal
	ejecutor  EjecutorAlta
	reloj     ports.Reloj
}

var (
	_ http.Handler = (*manejadorAlta)(nil)
	_ EjecutorAlta = (*application.ServicioRegistroSolicitud)(nil)
)

// NuevoManejadorAlta solo compone fronteras ya constituidas. Deliberadamente no
// ofrece un constructor de contexto autenticado a partir de cadenas.
func NuevoManejadorAlta(
	autoridad AutoridadContextoCanal,
	ejecutor EjecutorAlta,
	reloj ports.Reloj,
) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(ejecutor) || dependenciaNula(reloj) {
		return nil, ErrManejadorAltaInvalido
	}
	return &manejadorAlta{autoridad: autoridad, ejecutor: ejecutor, reloj: reloj}, nil
}

func (h *manejadorAlta) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || h == nil || dependenciaNula(h.autoridad) ||
		dependenciaNula(h.ejecutor) || dependenciaNula(h.reloj) {
		responderErrorAlta(w, errorServicioNoDisponible)
		return
	}
	if !rutaAltaExacta(r) {
		responderErrorAlta(w, errorRecursoNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorAlta(w, errorMetodoNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorAlta(w, clasificarErrorAlta(err))
		return
	}
	if problema := validarMetadatosAlta(r); problema != nil {
		responderErrorAlta(w, *problema)
		return
	}

	solicitud, err := solicitudCentroDesdePeticion(w, r)
	if err != nil {
		responderErrorAlta(w, errorEntradaAlta(err))
		return
	}
	contextoCanal, err := h.autoridad.ResolverContextoCanalAlta(r.Context())
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorAlta(w, clasificarErrorAlta(errContexto))
		return
	}
	if err != nil {
		responderErrorAlta(w, clasificarErrorAlta(err))
		return
	}
	comando, correcto := comandoDesdeContextoCanal(contextoCanal, solicitud)
	if !correcto {
		responderErrorAlta(w, errorContextoNoConfiable)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorAlta(w, clasificarErrorAlta(err))
		return
	}

	recibo, err := h.ejecutor.Registrar(r.Context(), comando)
	if reciboAltaSeguro(recibo, h.reloj.Ahora()) {
		// Un recibo válido confirma el COMMIT. Una cancelación observada a la
		// vez no degrada el éxito a un resultado ambiguo ni induce reintento.
		responderExitoAlta(w, recibo)
		return
	}
	if err != nil {
		responderErrorAlta(w, clasificarErrorAlta(err))
		return
	}
	responderErrorAlta(w, errorResultadoNoConfiable)
}

func rutaAltaExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.Path != RutaAltaSolicitudes ||
		r.URL.RawPath != "" || r.URL.Scheme != "" || r.URL.Host != "" ||
		r.URL.User != nil || r.URL.Fragment != "" {
		return false
	}
	escapada := r.URL.EscapedPath()
	return escapada == RutaAltaSolicitudes && !strings.Contains(escapada, "%")
}

func comandoDesdeContextoCanal(
	contexto application.SolicitudRegistrarExpediente,
	solicitud domain.SolicitudCentro,
) (application.SolicitudRegistrarExpediente, bool) {
	resolver := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: contexto.AutenticacionRef,
		SesionRef:        contexto.SesionRef,
		PerfilRef:        contexto.PerfilRef,
	}
	clon, err := solicitud.Clonar()
	if err != nil || resolver.Validar() != nil ||
		!domain.ReferenciaOpacaValida(contexto.OrganizacionRef) ||
		!ports.ClaveIdempotenciaValida(contexto.ClaveIdempotencia) ||
		!reflect.DeepEqual(contexto.Solicitud, domain.SolicitudCentro{}) {
		return application.SolicitudRegistrarExpediente{}, false
	}
	contexto.Solicitud = clon
	return contexto, true
}

func reciboAltaSeguro(recibo ports.ReciboAlta, ahora time.Time) bool {
	const toleranciaFuturo = time.Minute
	return domain.InstanteUTCCanonico(ahora) &&
		recibo.ValidarEstructura() == nil && recibo.Version <= MaximoVersionJSON &&
		!recibo.ConfirmadaEn.After(ahora.Add(toleranciaFuturo))
}

func dependenciaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func errorEntradaAlta(err error) errorPublicoAlta {
	if errors.Is(err, errCuerpoAltaDemasiadoGrande) {
		return errorPeticionDemasiadoGrande
	}
	if errors.Is(err, errContenidoAltaNoValido) {
		return errorContenidoNoValido
	}
	return errorPeticionNoValida
}
