package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	RutaConsultaCuadroRRHH  = "/api/vec/contratacion-temporal/cuadro/consultas"
	RutaConsultaDetalleRRHH = "/api/vec/contratacion-temporal/expedientes/consultas"
)

var ErrManejadorConsultaRRHHInvalido = errors.New(
	"contratacion temporal http: manejador de consulta RRHH invalido",
)

// ConsultorCuadroRRHH mantiene al adaptador desacoplado de la implementación
// del caso de uso. La identidad y el ámbito se resuelven dentro de aplicación.
type ConsultorCuadroRRHH interface {
	Consultar(context.Context, ports.SolicitudCuadroRRHH) (ports.PaginaCuadroRRHH, error)
}

// ConsultorDetalleRRHH expone únicamente la intención mínima de lectura.
type ConsultorDetalleRRHH interface {
	Consultar(context.Context, ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error)
}

type manejadorConsultaCuadroRRHH struct {
	consultor ConsultorCuadroRRHH
}

type manejadorConsultaDetalleRRHH struct {
	consultor ConsultorDetalleRRHH
}

var (
	_ http.Handler         = (*manejadorConsultaCuadroRRHH)(nil)
	_ http.Handler         = (*manejadorConsultaDetalleRRHH)(nil)
	_ ConsultorCuadroRRHH  = (*application.ServicioConsultaCuadroRRHH)(nil)
	_ ConsultorDetalleRRHH = (*application.ServicioConsultaDetalleRRHH)(nil)
)

func NuevoManejadorConsultaCuadroRRHH(
	consultor ConsultorCuadroRRHH,
) (http.Handler, error) {
	if dependenciaConsultaRRHHNula(consultor) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	return &manejadorConsultaCuadroRRHH{consultor: consultor}, nil
}

func NuevoManejadorConsultaDetalleRRHH(
	consultor ConsultorDetalleRRHH,
) (http.Handler, error) {
	if dependenciaConsultaRRHHNula(consultor) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	return &manejadorConsultaDetalleRRHH{consultor: consultor}, nil
}

func (h *manejadorConsultaCuadroRRHH) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaConsultaRRHHNula(h.consultor) {
		responderErrorConsultaRRHH(w, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaConsultaCuadroRRHH) {
		responderErrorConsultaRRHH(w, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorConsultaRRHH(w, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if problema := validarMetadatosConsultaRRHH(r, MaximoCuerpoConsultaCuadroRRHHBytes); problema != nil {
		responderErrorConsultaRRHH(w, *problema)
		return
	}
	solicitud, err := solicitudCuadroRRHHDesdePeticion(w, r)
	if err != nil {
		responderErrorConsultaRRHH(w, errorEntradaConsultaRRHH(err))
		return
	}
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	pagina, err := h.consultor.Consultar(r.Context(), solicitud)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	if err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if pagina.ValidarContenidoPublicablePara(solicitud) != nil {
		responderErrorConsultaRRHH(w, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONConsultaRRHH(
		w,
		http.StatusOK,
		envoltorioCuadroRRHH{Data: proyectarPaginaCuadroRRHH(pagina)},
	)
}

func (h *manejadorConsultaDetalleRRHH) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaConsultaRRHHNula(h.consultor) {
		responderErrorConsultaRRHH(w, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaConsultaDetalleRRHH) {
		responderErrorConsultaRRHH(w, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorConsultaRRHH(w, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if problema := validarMetadatosConsultaRRHH(r, MaximoCuerpoConsultaDetalleRRHHBytes); problema != nil {
		responderErrorConsultaRRHH(w, *problema)
		return
	}
	solicitud, err := solicitudDetalleRRHHDesdePeticion(w, r)
	if err != nil {
		responderErrorConsultaRRHH(w, errorEntradaConsultaRRHH(err))
		return
	}
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	detalle, err := h.consultor.Consultar(r.Context(), solicitud)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(errContexto))
		return
	}
	if err != nil {
		responderErrorConsultaRRHH(w, clasificarErrorConsultaRRHH(err))
		return
	}
	if detalle.ValidarContenidoPublicablePara(solicitud) != nil {
		responderErrorConsultaRRHH(w, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONConsultaRRHH(
		w,
		http.StatusOK,
		envoltorioDetalleRRHH{Data: proyectarDetalleRRHH(detalle)},
	)
}

func rutaConsultaRRHHExacta(r *http.Request, esperada string) bool {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.URL.RawPath != "" || r.URL.Scheme != "" || r.URL.Host != "" ||
		r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" ||
		r.URL.RawFragment != "" || r.URL.Path != esperada {
		return false
	}
	return r.URL.EscapedPath() == esperada && !strings.Contains(r.URL.Path, "%")
}

func dependenciaConsultaRRHHNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
