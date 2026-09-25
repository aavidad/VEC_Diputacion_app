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

// EsquemaContratosParticipacion identifica la respuesta B13 de la ficha.
const EsquemaContratosParticipacion = "vec.bolsa.rrhh.contratos_participacion.v1"

type LectorContratosParticipacion interface {
	ListarContratos(context.Context, ports.SolicitudCambiarSituacionParticipacion) ([]ports.ContratoParticipacion, error)
}

// HandlerContratosParticipacion sirve el histórico B13 en solo lectura.
type HandlerContratosParticipacion struct {
	preparador PreparadorSituacionParticipacion
	lector     LectorContratosParticipacion
}

func NuevoHandlerContratosParticipacion(p PreparadorSituacionParticipacion, l LectorContratosParticipacion) (http.Handler, error) {
	if p == nil || l == nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	return &HandlerContratosParticipacion{preparador: p, lector: l}, nil
}

// ReferenciasRutaContratosParticipacion reconoce
// /api/vec/bolsa/bolsas/{bolsa}/candidatos/{participacion}/contratos.
func ReferenciasRutaContratosParticipacion(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.RequestURI != r.URL.Path || strings.Contains(r.URL.EscapedPath(), "%") {
		return "", "", false
	}
	partes := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if !strings.HasPrefix(r.URL.Path, RutaBolsasGestion+"/") || len(partes) != 4 || partes[0] == "" || partes[1] != "candidatos" || partes[2] == "" || partes[3] != "contratos" {
		return "", "", false
	}
	// Sin «%» no hay secuencias de escape que decodificar: el segmento es literal.
	for _, v := range []string{partes[0], partes[2]} {
		if strings.ContainsAny(v, "?# \\%") || len(v) > 512 {
			return "", "", false
		}
	}
	return partes[0], partes[2], true
}

func (h *HandlerContratosParticipacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, ok := ReferenciasRutaContratosParticipacion(r)
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
	q, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Destino: domain.SituacionDisponible, Motivo: "consulta", ClaveIdempotencia: "consulta"})
	if err != nil {
		responderErrorContratos(w, err)
		return
	}
	items, err := h.lector.ListarContratos(r.Context(), q)
	if err != nil {
		responderErrorContratos(w, err)
		return
	}
	salida := make([]map[string]any, 0, len(items))
	for _, c := range items {
		salida = append(salida, map[string]any{
			"evento_ref": c.EventoRef, "tipo": c.Tipo,
			"inicio": instanteOpcionalContrato(c.Inicio), "fin_previsto": instanteOpcionalContrato(c.FinPrevisto),
			"modalidad_clave": c.ModalidadClave, "categoria_ref": c.CategoriaRef, "causa_clave": c.CausaClave,
			"expediente_ref": c.ExpedienteRef, "ocurrido_en": c.OcurridoEn.UTC().Format(time.RFC3339Nano),
		})
	}
	responderSituacion(w, http.StatusOK, map[string]any{"data": map[string]any{"esquema": EsquemaContratosParticipacion, "items": salida}})
}

func instanteOpcionalContrato(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func responderErrorContratos(w http.ResponseWriter, err error) {
	if errors.Is(err, ports.ErrContratosParticipacionNoDisponible) {
		responderOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	responderErrorOperacion(w, err)
}
