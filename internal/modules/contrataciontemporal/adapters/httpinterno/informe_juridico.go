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

const RutaPreparacionesInformeJuridico = "/api/vec/contratacion-temporal/informes-juridicos/preparaciones"

var ErrManejadorInformeJuridicoInvalido = errors.New(
	"contratacion temporal http: manejador de informe juridico invalido",
)

// ContextoCanalInformeJuridico contiene autoridad resuelta por la frontera
// confiable. Ninguno de estos datos procede del JSON del navegador.
type ContextoCanalInformeJuridico struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
}

func (c ContextoCanalInformeJuridico) valido() bool {
	return (ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: c.AutenticacionRef,
		SesionRef:        c.SesionRef,
		PerfilRef:        c.PerfilRef,
	}).Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

type AutoridadContextoCanalInformeJuridico interface {
	ResolverContextoCanalInformeJuridico(
		context.Context,
	) (ContextoCanalInformeJuridico, error)
}

type EjecutorInformeJuridico interface {
	Emitir(
		context.Context,
		application.SolicitudEmitirInformeJuridico,
	) (ports.ReciboInformeJuridico, error)
}

type manejadorInformeJuridico struct {
	autoridad AutoridadContextoCanalInformeJuridico
	ejecutor  EjecutorInformeJuridico
}

var _ http.Handler = (*manejadorInformeJuridico)(nil)
var _ EjecutorInformeJuridico = (*application.ServicioInformesJuridicos)(nil)

func NuevoManejadorInformeJuridico(
	autoridad AutoridadContextoCanalInformeJuridico,
	ejecutor EjecutorInformeJuridico,
) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(ejecutor) {
		return nil, ErrManejadorInformeJuridicoInvalido
	}
	return &manejadorInformeJuridico{
		autoridad: autoridad,
		ejecutor:  ejecutor,
	}, nil
}

func (h *manejadorInformeJuridico) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaNula(h.autoridad) || dependenciaNula(h.ejecutor) {
		responderErrorInformeJuridico(w, errorServicioInformeJuridicoNoDisponible)
		return
	}
	if !rutaInformeJuridicoValida(r) {
		responderErrorInformeJuridico(w, errorRecursoInformeJuridicoNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorInformeJuridico(w, errorMetodoInformeJuridicoNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorInformeJuridico(w, clasificarErrorInformeJuridicoHTTP(err))
		return
	}
	if problema := validarMetadatosInformeJuridico(r); problema != nil {
		responderErrorInformeJuridico(w, *problema)
		return
	}

	entrada, err := informeJuridicoDesdePeticion(w, r)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorInformeJuridico(w, clasificarErrorInformeJuridicoHTTP(errContexto))
		return
	}
	if err != nil {
		responderErrorInformeJuridico(w, errorEntradaInformeJuridico(err))
		return
	}

	contextoCanal, err := h.autoridad.ResolverContextoCanalInformeJuridico(r.Context())
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorInformeJuridico(w, clasificarErrorInformeJuridicoHTTP(errContexto))
		return
	}
	if err != nil {
		responderErrorInformeJuridico(w, clasificarErrorInformeJuridicoHTTP(err))
		return
	}
	if !contextoCanal.valido() {
		responderErrorInformeJuridico(w, errorServicioInformeJuridicoNoDisponible)
		return
	}

	recibo, err := h.ejecutor.Emitir(r.Context(), entrada.solicitud(contextoCanal))
	if err != nil {
		if recibo != (ports.ReciboInformeJuridico{}) {
			responderErrorInformeJuridico(w, errorResultadoInformeJuridicoNoConfiable)
			return
		}
		if errContexto := r.Context().Err(); errContexto != nil {
			responderErrorInformeJuridico(w, clasificarErrorInformeJuridicoHTTP(errContexto))
			return
		}
		responderErrorInformeJuridico(w, clasificarErrorInformeJuridicoHTTP(err))
		return
	}
	if !reciboInformeJuridicoSeguro(recibo, contextoCanal, entrada) {
		responderErrorInformeJuridico(w, errorResultadoInformeJuridicoNoConfiable)
		return
	}
	responderExitoInformeJuridico(w, recibo)
}

func rutaInformeJuridicoValida(r *http.Request) bool {
	return r != nil && r.URL != nil &&
		r.URL.Path == RutaPreparacionesInformeJuridico &&
		r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == "" &&
		r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil &&
		r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" &&
		r.URL.EscapedPath() == r.URL.Path && !strings.Contains(r.URL.Path, "%")
}
