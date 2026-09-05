package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

const RutaSeleccionLlamamiento = "" +
	"/api/vec/contratacion-temporal/llamamientos/seleccion"

var ErrManejadorSeleccionLlamamientoInvalido = errors.New(
	"contratacion temporal http: manejador de seleccion y llamamiento invalido",
)

// EjecutorSeleccionLlamamiento limita HTTP al único caso de uso autorizado.
// Identidad y acceso pertenecen a la AutoridadRutasExactas exterior.
type EjecutorSeleccionLlamamiento interface {
	SeleccionarYLlamarParaAdaptador(
		context.Context,
		application.SolicitudSeleccionLlamamiento,
	) (application.DatosReciboSeleccionLlamamientoParaAdaptador, error)
}

type manejadorSeleccionLlamamiento struct {
	ejecutor EjecutorSeleccionLlamamiento
}

var _ http.Handler = (*manejadorSeleccionLlamamiento)(nil)
var _ EjecutorSeleccionLlamamiento = (*application.ServicioSeleccionLlamamiento)(nil)

// NuevoManejadorSeleccionLlamamiento no acepta autoridad alternativa ni
// infraestructura. La fábrica exterior debe registrar la ruta de forma
// atómica junto con AutoridadRutasExactas.
func NuevoManejadorSeleccionLlamamiento(
	ejecutor EjecutorSeleccionLlamamiento,
) (http.Handler, error) {
	if dependenciaSeleccionLlamamientoNula(ejecutor) {
		return nil, ErrManejadorSeleccionLlamamientoInvalido
	}
	return &manejadorSeleccionLlamamiento{ejecutor: ejecutor}, nil
}

func (h *manejadorSeleccionLlamamiento) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaSeleccionLlamamientoNula(h.ejecutor) {
		responderErrorSeleccionLlamamiento(
			w,
			errorServicioSeleccionLlamamientoNoDisponible,
		)
		return
	}
	if !rutaSeleccionLlamamientoExacta(r) {
		responderErrorSeleccionLlamamiento(
			w,
			errorRecursoSeleccionLlamamientoNoEncontrado,
		)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorSeleccionLlamamiento(
			w,
			errorMetodoSeleccionLlamamientoNoPermitido,
		)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorSeleccionLlamamiento(
			w,
			clasificarErrorSeleccionLlamamiento(err),
		)
		return
	}
	if problema := validarMetadatosSeleccionLlamamiento(r); problema != nil {
		responderErrorSeleccionLlamamiento(w, *problema)
		return
	}
	entrada, err := seleccionLlamamientoDesdePeticion(w, r)
	if err != nil {
		responderErrorSeleccionLlamamiento(
			w,
			errorEntradaSeleccionLlamamiento(err),
		)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorSeleccionLlamamiento(
			w,
			clasificarErrorSeleccionLlamamiento(err),
		)
		return
	}
	recibo, err := h.ejecutor.SeleccionarYLlamarParaAdaptador(
		r.Context(),
		application.SolicitudSeleccionLlamamiento{
			ExpedienteRef:     entrada.ExpedienteRef,
			VersionEsperada:   entrada.VersionEsperada,
			ClaveIdempotencia: entrada.ClaveIdempotencia,
		},
	)
	if err != nil {
		responderErrorSeleccionLlamamiento(
			w,
			clasificarErrorSeleccionLlamamiento(err),
		)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorSeleccionLlamamiento(
			w,
			clasificarErrorSeleccionLlamamiento(err),
		)
		return
	}
	salida, valida := proyectarReciboSeleccionLlamamiento(recibo)
	if !valida {
		responderErrorSeleccionLlamamiento(
			w,
			errorResultadoSeleccionLlamamientoNoConfiable,
		)
		return
	}
	responderJSONCobertura(
		w,
		http.StatusOK,
		envoltorioReciboSeleccionLlamamiento{Data: salida},
	)
}

func rutaSeleccionLlamamientoExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.URL.Path != RutaSeleccionLlamamiento {
		return false
	}
	return r.URL.EscapedPath() == r.URL.Path &&
		!strings.Contains(r.URL.Path, "%")
}

func dependenciaSeleccionLlamamientoNula(dependencia any) bool {
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
