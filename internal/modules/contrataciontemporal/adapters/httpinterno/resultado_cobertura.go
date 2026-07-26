package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

const RutaResultadoCobertura = "/api/vec/contratacion-temporal/cobertura/resultados"

var ErrManejadorResultadoCoberturaInvalido = errors.New(
	"contratacion temporal http: manejador de resultado de cobertura invalido",
)

// ConsultorResultadoCobertura expone una sola lectura y ninguna capacidad de
// reservar, confirmar, rectificar ni reapropiar una operación.
type ConsultorResultadoCobertura interface {
	ConsultarParaAdaptador(
		context.Context,
		application.SolicitudConsultaResultadoCobertura,
	) (application.DatosConsultaResultadoCoberturaParaAdaptador, error)
}

type manejadorResultadoCobertura struct {
	consultor ConsultorResultadoCobertura
}

var _ http.Handler = (*manejadorResultadoCobertura)(nil)
var _ ConsultorResultadoCobertura = (*application.ServicioConsultaResultadoCobertura)(nil)

// NuevoManejadorResultadoCobertura no recibe autoridad ni la petición HTTP
// como origen de identidad. El caso de uso resuelve actor, perfil y
// organización desde el contexto confiable aportado por la composición.
func NuevoManejadorResultadoCobertura(
	consultor ConsultorResultadoCobertura,
) (http.Handler, error) {
	if dependenciaResultadoCoberturaNula(consultor) {
		return nil, ErrManejadorResultadoCoberturaInvalido
	}
	return &manejadorResultadoCobertura{consultor: consultor}, nil
}

func (h *manejadorResultadoCobertura) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaResultadoCoberturaNula(h.consultor) {
		responderErrorCobertura(w, errorResultadoConsultaCoberturaNoDisponible)
		return
	}
	if !rutaResultadoCoberturaExacta(r) {
		responderErrorCobertura(w, errorRecursoCoberturaNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorCobertura(w, errorMetodoCoberturaNoPermitido)
		return
	}
	if r.Context().Err() != nil {
		responderErrorCobertura(
			w,
			clasificarErrorResultadoCobertura(r.Context().Err()),
		)
		return
	}
	if problema := validarMetadatosCobertura(r); problema != nil {
		responderErrorCobertura(w, *problema)
		return
	}
	entrada, err := resultadoCoberturaDesdePeticion(w, r)
	if err != nil {
		responderErrorCobertura(w, errorEntradaCobertura(err))
		return
	}
	resultado, err := h.consultor.ConsultarParaAdaptador(
		r.Context(),
		application.SolicitudConsultaResultadoCobertura{
			ExpedienteRef:     entrada.ExpedienteRef,
			ClaveIdempotencia: entrada.ClaveIdempotencia,
		},
	)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorCobertura(
			w,
			clasificarErrorResultadoCobertura(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorCobertura(w, clasificarErrorResultadoCobertura(err))
		return
	}
	salida, estadoHTTP, valida := proyectarResultadoConsultaCobertura(resultado)
	if !valida {
		responderErrorCobertura(w, errorResultadoConsultaCoberturaNoDisponible)
		return
	}
	responderJSONCobertura(
		w,
		estadoHTTP,
		envoltorioResultadoConsultaCobertura{Data: salida},
	)
}

func rutaResultadoCoberturaExacta(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" ||
		r.URL.Path != RutaResultadoCobertura {
		return false
	}
	return r.URL.EscapedPath() == r.URL.Path &&
		!strings.Contains(r.URL.Path, "%")
}

func dependenciaResultadoCoberturaNula(dependencia any) bool {
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
