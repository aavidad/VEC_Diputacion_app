package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const RutaParticipacionesPropias = "/api/vec/bolsa/mi-bolsa"

var ErrHandlerParticipacionesPropiasInvalido = errors.New("bolsa http interno: handler de participaciones propias invalido")

type PreparadorOrdenConsultaParticipacionesPropias interface {
	PrepararOrdenConsultaParticipacionesPropias(context.Context) (aplicacionbolsa.OrdenConsultaParticipacionesPropias, error)
}
type ConsultorParticipacionesPropias interface {
	Consultar(context.Context, aplicacionbolsa.OrdenConsultaParticipacionesPropias) (puertosbolsa.ResultadoParticipacionesPropias, error)
}
type HandlerParticipacionesPropias struct {
	preparador PreparadorOrdenConsultaParticipacionesPropias
	consultor  ConsultorParticipacionesPropias
}

func NuevoHandlerParticipacionesPropias(p PreparadorOrdenConsultaParticipacionesPropias, c ConsultorParticipacionesPropias) (http.Handler, error) {
	if dependenciaNula(p) || dependenciaNula(c) {
		return nil, ErrHandlerParticipacionesPropiasInvalido
	}
	return &HandlerParticipacionesPropias{p, c}, nil
}
func (h *HandlerParticipacionesPropias) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || h == nil || dependenciaNula(h.preparador) || dependenciaNula(h.consultor) {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.Method == http.MethodHead {
		w = escritorSinCuerpo{ResponseWriter: w}
	}
	if r.URL == nil || r.URL.Path != RutaParticipacionesPropias || r.URL.RawPath != "" || r.URL.EscapedPath() != RutaParticipacionesPropias {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !entradaVaciaYPermitida(r) || cabeceraPresente(r.Header, "Authorization") {
		responderError(w, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	orden, err := h.preparador.PrepararOrdenConsultaParticipacionesPropias(r.Context())
	if err != nil {
		responderErrorClasificado(w, err)
		return
	}
	resultado, err := h.consultor.Consultar(r.Context(), orden)
	if err != nil {
		responderErrorClasificado(w, err)
		return
	}
	responderJSON(w, http.StatusOK, respuestaParticipacionesPropias{Data: datosParticipacionesPropias{Esquema: resultado.Esquema, ConsultadaEn: resultado.ConsultadaEn, Participaciones: resultado.Participaciones}})
}

type respuestaParticipacionesPropias struct {
	Data datosParticipacionesPropias `json:"data"`
}
type datosParticipacionesPropias struct {
	Esquema         string                             `json:"esquema"`
	ConsultadaEn    time.Time                          `json:"consultada_en"`
	Participaciones []puertosbolsa.ParticipacionPropia `json:"participaciones"`
}
