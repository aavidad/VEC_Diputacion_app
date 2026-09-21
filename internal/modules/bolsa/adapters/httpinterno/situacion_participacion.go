package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const RutaBolsasGestion = "/api/vec/bolsa/bolsas"

type EntradaCambiarSituacionParticipacion struct {
	BolsaRef, ParticipacionRef, Destino, Motivo, ClaveIdempotencia string
	FechaDisponible                                                *time.Time
}

type PreparadorSituacionParticipacion interface {
	PrepararSolicitudCambiarSituacion(context.Context, EntradaCambiarSituacionParticipacion) (puertosbolsa.SolicitudCambiarSituacionParticipacion, error)
}

type OperadorSituacionParticipacion interface {
	Cambiar(context.Context, puertosbolsa.SolicitudCambiarSituacionParticipacion) (puertosbolsa.RegistroSituacionParticipacion, error)
}

type HandlerSituacionParticipacion struct {
	preparador PreparadorSituacionParticipacion
	operador   OperadorSituacionParticipacion
}

func NuevoHandlerSituacionParticipacion(p PreparadorSituacionParticipacion, o OperadorSituacionParticipacion) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, errors.New("bolsa http interno: situacion no disponible")
	}
	return &HandlerSituacionParticipacion{p, o}, nil
}

func (h *HandlerSituacionParticipacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, ok := ReferenciasRutaSituacionParticipacion(r)
	if !ok {
		responderSituacion(w, http.StatusNotFound, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderSituacion(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
		return
	}
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > 4096 || len(r.TransferEncoding) != 0 || len(r.Header.Values("Content-Type")) != 1 || len(r.Header.Values("Accept")) != 1 || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	clave := r.Header.Get("Idempotency-Key")
	if len(r.Header.Values("Idempotency-Key")) != 1 || clave == "" || strings.TrimSpace(clave) != clave || len(clave) > 256 {
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	var cuerpo struct {
		Situacion       string     `json:"situacion"`
		Motivo          string     `json:"motivo"`
		FechaDisponible *time.Time `json:"fecha_disponible"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4097))
	dec.DisallowUnknownFields()
	if dec.Decode(&cuerpo) != nil || dec.Decode(&struct{}{}) != io.EOF || strings.TrimSpace(cuerpo.Motivo) != cuerpo.Motivo || cuerpo.Motivo == "" || len(cuerpo.Motivo) > 1000 {
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	solicitud, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Destino: cuerpo.Situacion, Motivo: cuerpo.Motivo, ClaveIdempotencia: clave, FechaDisponible: cuerpo.FechaDisponible})
	if err != nil {
		responderErrorSituacion(w, err)
		return
	}
	resultado, err := h.operador.Cambiar(r.Context(), solicitud)
	if err != nil {
		responderErrorSituacion(w, err)
		return
	}
	estado := http.StatusCreated
	if resultado.Reutilizada {
		estado = http.StatusOK
	}
	responderSituacion(w, estado, map[string]any{"data": map[string]any{"participacion_ref": resultado.ParticipacionRef, "situacion": resultado.Situacion, "desde": resultado.Desde.UTC().Format(time.RFC3339Nano), "fecha_disponible": resultado.FechaDisponible, "recibo_ref": resultado.ReciboRef, "reutilizada": resultado.Reutilizada}})
}

func ReferenciasRutaSituacionParticipacion(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.RequestURI != r.URL.Path || strings.Contains(r.URL.EscapedPath(), "%") {
		return "", "", false
	}
	segmentos := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if len(segmentos) != 4 || segmentos[0] == "" || segmentos[1] != "candidatos" || segmentos[2] == "" || segmentos[3] != "situacion" {
		return "", "", false
	}
	for _, v := range []string{segmentos[0], segmentos[2]} {
		if _, err := url.PathUnescape(v); err != nil || strings.ContainsAny(v, "?# \\%") || len(v) > 512 {
			return "", "", false
		}
	}
	return segmentos[0], segmentos[2], true
}

func responderErrorSituacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderSituacion(w, http.StatusForbidden, map[string]any{"error": map[string]string{"codigo": "acceso_denegado"}})
	case errors.Is(err, dominiobolsa.ErrCambioSituacionParticipacionInvalido):
		responderSituacion(w, http.StatusConflict, map[string]any{"error": map[string]string{"codigo": "cambio_en_conflicto"}})
	case errors.Is(err, puertosbolsa.ErrSituacionParticipacionNoEncontrada):
		responderSituacion(w, http.StatusNotFound, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
	default:
		responderSituacion(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]string{"codigo": "servicio_no_disponible"}})
	}
}

func responderSituacion(w http.ResponseWriter, estado int, valor any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(valor)
}
