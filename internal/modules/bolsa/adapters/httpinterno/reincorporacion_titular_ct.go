package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const EsquemaReincorporacionesTitularCT = "vec.bolsa.rrhh.reincorporaciones_titular.v1"

type LectorReincorporacionesTitularCT interface {
	ListarReincorporacionesTitular(context.Context, ports.SolicitudCambiarSituacionParticipacion) ([]ports.ReincorporacionTitularFicha, error)
}

type HandlerReincorporacionesTitularCT struct {
	preparador PreparadorSituacionParticipacion
	lector     LectorReincorporacionesTitularCT
}

func NuevoHandlerReincorporacionesTitularCT(p PreparadorSituacionParticipacion, l LectorReincorporacionesTitularCT) (http.Handler, error) {
	if p == nil || l == nil {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	return &HandlerReincorporacionesTitularCT{preparador: p, lector: l}, nil
}

// ReferenciasRutaReincorporacionesTitularCT reconoce la ficha del candidato
// en /api/vec/bolsa/bolsas/{bolsa}/candidatos/{participacion}/reincorporaciones-titular.
func ReferenciasRutaReincorporacionesTitularCT(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.RequestURI != r.URL.Path || strings.Contains(r.URL.EscapedPath(), "%") {
		return "", "", false
	}
	partes := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if !strings.HasPrefix(r.URL.Path, RutaBolsasGestion+"/") || len(partes) != 4 ||
		partes[0] == "" || partes[1] != "candidatos" || partes[2] == "" || partes[3] != "reincorporaciones-titular" {
		return "", "", false
	}
	for _, v := range []string{partes[0], partes[2]} {
		if strings.ContainsAny(v, "?# \\%") || len(v) > 512 {
			return "", "", false
		}
	}
	return partes[0], partes[2], true
}

func (h *HandlerReincorporacionesTitularCT) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, ok := ReferenciasRutaReincorporacionesTitularCT(r)
	if !ok {
		responderOperacion(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderOperacion(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" || r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		responderOperacion(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	q, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{
		BolsaRef: bolsa, ParticipacionRef: participacion, Destino: domain.SituacionDisponible,
		Motivo: "consulta", ClaveIdempotencia: "consulta",
	})
	if err != nil {
		responderErrorReincorporacionesTitular(w, err)
		return
	}
	items, err := h.lector.ListarReincorporacionesTitular(r.Context(), q)
	if err != nil {
		responderErrorReincorporacionesTitular(w, err)
		return
	}
	salida := make([]map[string]any, 0, len(items))
	for _, item := range items {
		var disponible any
		if item.DisponibleDesde != nil {
			disponible = item.DisponibleDesde.Format(time.DateOnly)
		}
		salida = append(salida, map[string]any{
			"evento_ref": item.EventoRef, "expediente_ref": item.ExpedienteRef,
			"relacion_ref": item.RelacionRef, "fecha_efectiva": item.FechaEfectiva.Format(time.DateOnly),
			"recibo_ct_ref": item.ReciboCTRef, "cese_evento_ref": item.CeseEventoRef,
			"estado": item.Estado, "disponible_desde": disponible,
			"regla_version": item.ReglaVersion, "regla_huella_sha256": item.ReglaHuellaSHA256,
		})
	}
	responderSituacion(w, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": EsquemaReincorporacionesTitularCT, "items": salida,
	}})
}

func responderErrorReincorporacionesTitular(w http.ResponseWriter, err error) {
	if errors.Is(err, ports.ErrReincorporacionTitularNoDisponible) {
		responderOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	responderErrorOperacion(w, err)
}
