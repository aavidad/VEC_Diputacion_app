package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// RutaPlazoRespuestaLlamamiento es la consulta de solo lectura con la que el
// asistente B7 propone el plazo de respuesta. La autorización es la de las
// demás lecturas RRHH de Bolsa y la aplica la frontera de rutas exactas.
const RutaPlazoRespuestaLlamamiento = "/api/vec/bolsa/llamamientos/plazo-respuesta"

var ErrHandlerPlazoRespuestaInvalido = errors.New("bolsa http interno: handler de plazo de respuesta invalido")

// ConsultorPlazoRespuestaLlamamiento es la superficie mínima del caso de uso.
type ConsultorPlazoRespuestaLlamamiento interface {
	ConsultarPlazoRespuesta(context.Context) (puertosbolsa.PlazoRespuestaLlamamiento, error)
}

type handlerPlazoRespuestaLlamamiento struct {
	consultor ConsultorPlazoRespuestaLlamamiento
}

func NuevoHandlerPlazoRespuestaLlamamiento(consultor ConsultorPlazoRespuestaLlamamiento) (http.Handler, error) {
	if dependenciaNula(consultor) {
		return nil, ErrHandlerPlazoRespuestaInvalido
	}
	return &handlerPlazoRespuestaLlamamiento{consultor: consultor}, nil
}

func (h *handlerPlazoRespuestaLlamamiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || h == nil || dependenciaNula(h.consultor) {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.Method == http.MethodHead {
		w = escritorSinCuerpo{ResponseWriter: w}
	}
	if r.URL == nil || r.URL.Path != RutaPlazoRespuestaLlamamiento || r.URL.RawPath != "" {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !entradaVaciaYPermitida(r) {
		responderError(w, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	plazo, err := h.consultor.ConsultarPlazoRespuesta(r.Context())
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderError(w, http.StatusForbidden, "acceso_denegado")
		return
	case err != nil:
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	responderJSON(w, http.StatusOK, respuestaPlazoRespuesta{Data: nuevosDatosPlazoRespuesta(plazo)})
}

type respuestaPlazoRespuesta struct {
	Data datosPlazoRespuesta `json:"data"`
}

// datosPlazoRespuesta es un DTO cerrado: solo publica lo que necesita la
// pantalla de RRHH. Sin catálogo, solo esquema y configurada=false.
type datosPlazoRespuesta struct {
	Esquema     string                     `json:"esquema"`
	Configurada bool                       `json:"configurada"`
	Regla       *reglaPlazoRespuesta       `json:"regla,omitempty"`
	Vencimiento *vencimientoPlazoRespuesta `json:"vencimiento,omitempty"`
}

type reglaPlazoRespuesta struct {
	Etiqueta   string `json:"etiqueta"`
	Texto      string `json:"texto"`
	Referencia string `json:"referencia"`
	Origen     string `json:"origen"`
	Articulo   string `json:"articulo,omitempty"`
	Ejemplo    bool   `json:"ejemplo"`
}

type vencimientoPlazoRespuesta struct {
	CalculadoEn  time.Time `json:"calculado_en"`
	UltimoDia    string    `json:"ultimo_dia"`
	VenceEn      time.Time `json:"vence_en"`
	VenceAntesDe time.Time `json:"vence_antes_de"`
}

func nuevosDatosPlazoRespuesta(plazo puertosbolsa.PlazoRespuestaLlamamiento) datosPlazoRespuesta {
	datos := datosPlazoRespuesta{Esquema: puertosbolsa.EsquemaPlazoRespuestaLlamamiento}
	if !plazo.Configurada {
		return datos
	}
	datos.Configurada = true
	datos.Regla = &reglaPlazoRespuesta{
		Etiqueta: plazo.Regla.Etiqueta, Texto: plazo.Regla.Texto, Referencia: plazo.Regla.Referencia,
		Origen: plazo.Regla.Origen, Articulo: plazo.Regla.Articulo, Ejemplo: plazo.Regla.Ejemplo,
	}
	datos.Vencimiento = &vencimientoPlazoRespuesta{
		CalculadoEn: plazo.CalculadoEn.UTC(), UltimoDia: plazo.UltimoDia,
		VenceEn: plazo.VenceEn.UTC(), VenceAntesDe: plazo.VenceAntesDe.UTC(),
	}
	return datos
}
