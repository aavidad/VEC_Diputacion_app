package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const RutaEmisionesLlamamiento = "/api/vec/bolsa/llamamientos/emisiones"

type EntradaEmitirLlamamiento struct {
	BolsaRef          string
	Participaciones   []string
	Configuracion     ports.ConfiguracionLlamamiento
	ClaveIdempotencia string
}
type PreparadorEmisionLlamamiento interface {
	PrepararSolicitudEmitirLlamamiento(context.Context, EntradaEmitirLlamamiento) (ports.SolicitudEmitirLlamamiento, error)
	PrepararSolicitudRecuperarLlamamiento(context.Context, string, string) (ports.SolicitudRecuperarLlamamiento, error)
}
type OperadorEmisionLlamamiento interface {
	EmitirLlamamiento(context.Context, ports.SolicitudEmitirLlamamiento) (ports.EmisionLlamamiento, error)
	RecuperarLlamamiento(context.Context, ports.SolicitudRecuperarLlamamiento) (ports.EmisionLlamamiento, error)
}

type HandlerEmisionLlamamiento struct {
	preparador PreparadorEmisionLlamamiento
	operador   OperadorEmisionLlamamiento
}

func NuevoHandlerEmisionLlamamiento(p PreparadorEmisionLlamamiento, o OperadorEmisionLlamamiento) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	return &HandlerEmisionLlamamiento{p, o}, nil
}
func (h *HandlerEmisionLlamamiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != RutaEmisionesLlamamiento || r.URL.RawPath != "" {
		responderEmision(w, 404, map[string]any{"error": map[string]string{"codigo": "no_encontrado"}})
		return
	}
	if r.Method == http.MethodGet {
		h.recuperar(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		responderEmision(w, 405, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
		return
	}
	if r.Body == nil || r.ContentLength < 1 || r.ContentLength > 32768 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		responderEmision(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	clave := r.Header.Get("Idempotency-Key")
	if len(r.Header.Values("Idempotency-Key")) != 1 || len(clave) < 8 || len(clave) > 256 || strings.TrimSpace(clave) != clave {
		responderEmision(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	var cuerpo struct {
		BolsaRef        string                         `json:"bolsa_ref"`
		Participaciones []string                       `json:"participaciones"`
		Configuracion   ports.ConfiguracionLlamamiento `json:"configuracion"`
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 32769))
	d.DisallowUnknownFields()
	if d.Decode(&cuerpo) != nil || d.Decode(&struct{}{}) != io.EOF {
		responderEmision(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	q, err := h.preparador.PrepararSolicitudEmitirLlamamiento(r.Context(), EntradaEmitirLlamamiento{cuerpo.BolsaRef, cuerpo.Participaciones, cuerpo.Configuracion, clave})
	if err != nil {
		responderErrorEmision(w, err)
		return
	}
	out, err := h.operador.EmitirLlamamiento(r.Context(), q)
	if err != nil {
		responderErrorEmision(w, err)
		return
	}
	estado := 201
	if out.Reutilizada {
		estado = 200
	}
	responderEmision(w, estado, map[string]any{"data": out})
}
func (h *HandlerEmisionLlamamiento) recuperar(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength != 0 || r.Header.Get("Accept") != "application/json" {
		responderEmision(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	v := r.URL.Query()
	bolsa, clave := v.Get("bolsa_ref"), v.Get("clave_idempotencia")
	if len(v) != 2 || bolsa == "" || clave == "" {
		responderEmision(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	q, err := h.preparador.PrepararSolicitudRecuperarLlamamiento(r.Context(), bolsa, clave)
	if err != nil {
		responderErrorEmision(w, err)
		return
	}
	out, err := h.operador.RecuperarLlamamiento(r.Context(), q)
	if err != nil {
		responderErrorEmision(w, err)
		return
	}
	responderEmision(w, 200, map[string]any{"data": out})
}
func responderErrorEmision(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderEmision(w, 403, map[string]any{"error": map[string]string{"codigo": "acceso_denegado"}})
	case errors.Is(err, ports.ErrEmisionLlamamientoConflicto):
		responderEmision(w, 409, map[string]any{"error": map[string]string{"codigo": "clave_divergente"}})
	case errors.Is(err, ports.ErrEmisionLlamamientoInvalida):
		responderEmision(w, 422, map[string]any{"error": map[string]string{"codigo": "emision_invalida"}})
	default:
		responderEmision(w, 503, map[string]any{"error": map[string]string{"codigo": "servicio_no_disponible"}})
	}
}
func responderEmision(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}
