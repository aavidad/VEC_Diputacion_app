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

const RutaResultadosFiscalizacion = "/api/vec/contratacion-temporal/fiscalizaciones/resultados"

var ErrManejadorFiscalizacionInvalido = errors.New(
	"contratacion temporal http: manejador de fiscalizacion invalido",
)

// ContextoCanalFiscalizacion contiene autoridad resuelta por la frontera
// confiable. Ninguno de estos datos procede del JSON del navegador.
type ContextoCanalFiscalizacion struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
}

func (c ContextoCanalFiscalizacion) valido() bool {
	return (ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: c.AutenticacionRef,
		SesionRef:        c.SesionRef,
		PerfilRef:        c.PerfilRef,
	}).Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

type AutoridadContextoCanalFiscalizacion interface {
	ResolverContextoCanalFiscalizacion(
		context.Context,
	) (ContextoCanalFiscalizacion, error)
}

type EjecutorFiscalizacion interface {
	RegistrarResultado(
		context.Context,
		application.SolicitudRegistrarResultadoFiscalizacion,
	) (ports.ReciboFiscalizacion, error)
}

type manejadorFiscalizacion struct {
	autoridad AutoridadContextoCanalFiscalizacion
	ejecutor  EjecutorFiscalizacion
}

var _ http.Handler = (*manejadorFiscalizacion)(nil)

func NuevoManejadorFiscalizacion(
	autoridad AutoridadContextoCanalFiscalizacion,
	ejecutor EjecutorFiscalizacion,
) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(ejecutor) {
		return nil, ErrManejadorFiscalizacionInvalido
	}
	return &manejadorFiscalizacion{autoridad: autoridad, ejecutor: ejecutor}, nil
}

func (h *manejadorFiscalizacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaNula(h.autoridad) || dependenciaNula(h.ejecutor) {
		responderErrorFiscalizacion(w, errorServicioFiscalizacionNoDisponible)
		return
	}
	if !rutaFiscalizacionValida(r) {
		responderErrorFiscalizacion(w, errorRecursoFiscalizacionNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorFiscalizacion(w, errorMetodoFiscalizacionNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorFiscalizacion(w, clasificarErrorFiscalizacionHTTP(err))
		return
	}
	if problema := validarMetadatosFiscalizacion(r); problema != nil {
		responderErrorFiscalizacion(w, *problema)
		return
	}

	entrada, err := fiscalizacionDesdePeticion(w, r)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorFiscalizacion(w, clasificarErrorFiscalizacionHTTP(errContexto))
		return
	}
	if err != nil {
		responderErrorFiscalizacion(w, errorEntradaFiscalizacion(err))
		return
	}

	contextoCanal, err := h.autoridad.ResolverContextoCanalFiscalizacion(r.Context())
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorFiscalizacion(w, clasificarErrorFiscalizacionHTTP(errContexto))
		return
	}
	if err != nil {
		responderErrorFiscalizacion(w, clasificarErrorFiscalizacionHTTP(err))
		return
	}
	if !contextoCanal.valido() {
		responderErrorFiscalizacion(w, errorServicioFiscalizacionNoDisponible)
		return
	}

	recibo, err := h.ejecutor.RegistrarResultado(
		r.Context(),
		entrada.solicitud(contextoCanal),
	)
	if err != nil {
		if !reciboFiscalizacionVacio(recibo) {
			responderErrorFiscalizacion(w, errorResultadoFiscalizacionNoConfiable)
			return
		}
		if errContexto := r.Context().Err(); errContexto != nil {
			responderErrorFiscalizacion(w, clasificarErrorFiscalizacionHTTP(errContexto))
			return
		}
		responderErrorFiscalizacion(w, clasificarErrorFiscalizacionHTTP(err))
		return
	}
	if !reciboFiscalizacionSeguro(recibo, contextoCanal, entrada) {
		responderErrorFiscalizacion(w, errorResultadoFiscalizacionNoConfiable)
		return
	}
	responderExitoFiscalizacion(w, recibo)
}

func rutaFiscalizacionValida(r *http.Request) bool {
	return r != nil && r.URL != nil &&
		r.URL.Path == RutaResultadosFiscalizacion &&
		r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == "" &&
		r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil &&
		r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" &&
		r.URL.EscapedPath() == r.URL.Path && !strings.Contains(r.URL.Path, "%")
}
