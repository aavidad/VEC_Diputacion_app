package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	RutaPropuestaCobertura     = "/api/interno/v1/contratacion-temporal/cobertura/propuesta"
	RutaDecisionCobertura      = "/api/interno/v1/contratacion-temporal/cobertura/decisiones"
	RutaRectificacionCobertura = "/api/interno/v1/contratacion-temporal/cobertura/rectificaciones"
)

var ErrManejadorCoberturaInvalido = errors.New(
	"contratacion temporal http: manejador de cobertura invalido",
)

// ContextoCanalCobertura es el único dato que puede aportar la frontera
// corporativa al caso de uso. No contiene la intención del cliente ni se
// construye desde HTTP.
type ContextoCanalCobertura struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
}

func (c ContextoCanalCobertura) valido() bool {
	return (ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: c.AutenticacionRef,
		SesionRef:        c.SesionRef,
		PerfilRef:        c.PerfilRef,
	}).Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

// AutoridadContextoCanalCobertura debe estar compuesta por la superficie
// interna autenticada. El manejador no recibe la petición HTTP para impedir
// que cuerpo o cabeceras se conviertan en autoridad.
type AutoridadContextoCanalCobertura interface {
	ResolverContextoCanalCobertura(context.Context) (ContextoCanalCobertura, error)
}

type PresentadorPropuestaCobertura interface {
	Proponer(context.Context, application.SolicitudProponerCobertura) (application.PresentacionPropuestaCobertura, error)
}

type EjecutorDecisionCobertura interface {
	Decidir(context.Context, application.SolicitudDecidirCobertura) (cobertura.ReciboOperacionDecisionCobertura, error)
	Rectificar(context.Context, application.SolicitudRectificarCobertura) (cobertura.ReciboOperacionDecisionCobertura, error)
}

type manejadorCobertura struct {
	autoridad   AutoridadContextoCanalCobertura
	presentador PresentadorPropuestaCobertura
	decisor     EjecutorDecisionCobertura
}

var (
	_ http.Handler = (*manejadorCobertura)(nil)
)

// NuevoManejadorCobertura no compone identidad ni PostgreSQL. Si no se aporta
// una frontera confiable, el constructor falla cerrado.
func NuevoManejadorCobertura(
	autoridad AutoridadContextoCanalCobertura,
	presentador PresentadorPropuestaCobertura,
	decisor EjecutorDecisionCobertura,
) (http.Handler, error) {
	if dependenciaCoberturaNula(autoridad) || dependenciaCoberturaNula(presentador) || dependenciaCoberturaNula(decisor) {
		return nil, ErrManejadorCoberturaInvalido
	}
	return &manejadorCobertura{autoridad: autoridad, presentador: presentador, decisor: decisor}, nil
}

func (h *manejadorCobertura) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaCoberturaNula(h.autoridad) || dependenciaCoberturaNula(h.presentador) || dependenciaCoberturaNula(h.decisor) {
		responderErrorCobertura(w, errorServicioCoberturaNoDisponible)
		return
	}
	if !rutaCoberturaExacta(r) {
		responderErrorCobertura(w, errorRecursoCoberturaNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorCobertura(w, errorMetodoCoberturaNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorCobertura(w, clasificarErrorCobertura(err))
		return
	}
	if problema := validarMetadatosCobertura(r); problema != nil {
		responderErrorCobertura(w, *problema)
		return
	}
	contextoCanal, err := h.autoridad.ResolverContextoCanalCobertura(r.Context())
	if err != nil {
		responderErrorCobertura(w, clasificarErrorCobertura(err))
		return
	}
	if !contextoCanal.valido() {
		responderErrorCobertura(w, errorServicioCoberturaNoDisponible)
		return
	}
	switch r.URL.Path {
	case RutaPropuestaCobertura:
		h.servirPropuesta(w, r, contextoCanal)
	case RutaDecisionCobertura:
		h.servirDecision(w, r, contextoCanal)
	case RutaRectificacionCobertura:
		h.servirRectificacion(w, r, contextoCanal)
	default:
		responderErrorCobertura(w, errorRecursoCoberturaNoEncontrado)
	}
}

func (h *manejadorCobertura) servirPropuesta(w http.ResponseWriter, r *http.Request, contexto ContextoCanalCobertura) {
	entrada, err := propuestaCoberturaDesdePeticion(w, r)
	if err != nil {
		responderErrorCobertura(w, errorEntradaCobertura(err))
		return
	}
	propuesta, err := h.presentador.Proponer(r.Context(), application.SolicitudProponerCobertura{
		AutenticacionRef: contexto.AutenticacionRef, SesionRef: contexto.SesionRef,
		PerfilRef: contexto.PerfilRef, OrganizacionRef: contexto.OrganizacionRef,
		ExpedienteRef: entrada.ExpedienteRef, VersionEsperada: entrada.VersionEsperada,
	})
	if err != nil {
		responderErrorCobertura(w, clasificarErrorCobertura(err))
		return
	}
	salida, ok := proyectarPropuestaCobertura(propuesta)
	if !ok {
		responderErrorCobertura(w, errorResultadoCoberturaNoConfiable)
		return
	}
	responderJSONCobertura(w, http.StatusOK, envoltorioPropuestaCobertura{Data: salida})
}

func (h *manejadorCobertura) servirDecision(w http.ResponseWriter, r *http.Request, contexto ContextoCanalCobertura) {
	entrada, err := decisionCoberturaDesdePeticion(w, r, false)
	if err != nil {
		responderErrorCobertura(w, errorEntradaCobertura(err))
		return
	}
	recibo, err := h.decisor.Decidir(r.Context(), entrada.solicitud(contexto))
	if err != nil {
		responderErrorCobertura(w, clasificarErrorCobertura(err))
		return
	}
	salida, ok := proyectarReciboCobertura(recibo)
	if !ok {
		responderErrorCobertura(w, errorResultadoCoberturaNoConfiable)
		return
	}
	responderJSONCobertura(w, http.StatusCreated, envoltorioReciboCobertura{Data: salida})
}

func (h *manejadorCobertura) servirRectificacion(w http.ResponseWriter, r *http.Request, contexto ContextoCanalCobertura) {
	entrada, err := decisionCoberturaDesdePeticion(w, r, true)
	if err != nil {
		responderErrorCobertura(w, errorEntradaCobertura(err))
		return
	}
	recibo, err := h.decisor.Rectificar(r.Context(), entrada.rectificacion(contexto))
	if err != nil {
		responderErrorCobertura(w, clasificarErrorCobertura(err))
		return
	}
	salida, ok := proyectarReciboCobertura(recibo)
	if !ok {
		responderErrorCobertura(w, errorResultadoCoberturaNoConfiable)
		return
	}
	responderJSONCobertura(w, http.StatusCreated, envoltorioReciboCobertura{Data: salida})
}

func rutaCoberturaExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil || r.URL.Fragment != "" {
		return false
	}
	if r.URL.Path != RutaPropuestaCobertura && r.URL.Path != RutaDecisionCobertura && r.URL.Path != RutaRectificacionCobertura {
		return false
	}
	return r.URL.EscapedPath() == r.URL.Path && !strings.Contains(r.URL.Path, "%")
}

func dependenciaCoberturaNula(dependencia any) bool {
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
