package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const RutaComprobacionesOfrecimiento = "/api/vec/bolsa/bolsas/"

type EntradaComprobacionOfrecimiento struct {
	Contexto          ports.ContextoComprobacion
	Hechos            map[string]ports.HechoComprobacion
	VersionEsperada   int64
	ClaveIdempotencia string
}

type PreparadorComprobacionOfrecimiento interface {
	PrepararComprobacionOfrecimiento(context.Context, EntradaComprobacionOfrecimiento) (ports.SolicitudComprobacion, error)
}

type HandlerComprobacionOfrecimiento struct {
	preparador PreparadorComprobacionOfrecimiento
	servicio   ports.ServicioComprobacion
}

// El montaje puede omitir la dependencia opcional: solo esta ruta devuelve 503.
func NuevoHandlerComprobacionOfrecimiento(p PreparadorComprobacionOfrecimiento, s ports.ServicioComprobacion) http.Handler {
	return &HandlerComprobacionOfrecimiento{p, s}
}

func (h *HandlerComprobacionOfrecimiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, ok := ReferenciasRutaComprobacionOfrecimiento(r)
	if !ok {
		responderCodigoComprobacion(w, 404, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderCodigoComprobacion(w, 405, "metodo_no_permitido")
		return
	}
	if h == nil || h.preparador == nil || h.servicio == nil {
		responderCodigoComprobacion(w, 503, "comprobacion_ofrecimiento_no_habilitada")
		return
	}
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > 49152 || len(r.TransferEncoding) != 0 || len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" || len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" || len(r.Header.Values("Idempotency-Key")) != 1 {
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
		return
	}
	clave := r.Header.Get("Idempotency-Key")
	if len(clave) < 8 || len(clave) > 256 || strings.TrimSpace(clave) != clave {
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
		return
	}
	var cuerpo struct {
		Contexto        ports.ContextoComprobacion         `json:"contexto"`
		Hechos          map[string]ports.HechoComprobacion `json:"hechos"`
		VersionEsperada int64                              `json:"version_esperada"`
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 49153))
	d.DisallowUnknownFields()
	if d.Decode(&cuerpo) != nil || d.Decode(new(any)) != io.EOF {
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
		return
	}
	if cuerpo.Contexto.BolsaRef != bolsa || cuerpo.Contexto.ParticipacionRef != participacion {
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
		return
	}
	q, err := h.preparador.PrepararComprobacionOfrecimiento(r.Context(), EntradaComprobacionOfrecimiento{cuerpo.Contexto, cuerpo.Hechos, cuerpo.VersionEsperada, clave})
	if err != nil {
		responderErrorComprobacion(w, err)
		return
	}
	// El preparador solo aporta identidad y metadatos de frontera. No puede
	// sustituir la intención validada del cuerpo y de la cabecera.
	if q.Contexto != cuerpo.Contexto || q.VersionEsperada != cuerpo.VersionEsperada || q.ClaveIdempotencia != clave || !mismosHechosComprobacion(q.Hechos, cuerpo.Hechos) {
		responderCodigoComprobacion(w, 503, "comprobacion_ofrecimiento_no_habilitada")
		return
	}
	recibo, err := h.servicio.Registrar(r.Context(), q)
	if err != nil {
		responderErrorComprobacion(w, err)
		return
	}
	status := http.StatusCreated
	if recibo.Reutilizada {
		status = http.StatusOK
	}
	responderComprobacion(w, status, map[string]any{"data": recibo})
}

func mismosHechosComprobacion(a, b map[string]ports.HechoComprobacion) bool {
	if len(a) != len(b) {
		return false
	}
	for k, x := range a {
		y, ok := b[k]
		if !ok || x.DocumentoRef != y.DocumentoRef || x.Huella != y.Huella || !jsonIgualComprobacion(x.Valor, y.Valor) {
			return false
		}
	}
	return true
}
func jsonIgualComprobacion(a, b json.RawMessage) bool {
	var x, y bytes.Buffer
	return json.Compact(&x, a) == nil && json.Compact(&y, b) == nil && bytes.Equal(x.Bytes(), y.Bytes())
}

func ReferenciasRutaComprobacionOfrecimiento(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.RequestURI != r.URL.Path || strings.Contains(r.URL.EscapedPath(), "%") || !strings.HasPrefix(r.URL.Path, RutaComprobacionesOfrecimiento) {
		return "", "", false
	}
	s := strings.Split(strings.TrimPrefix(r.URL.Path, RutaComprobacionesOfrecimiento), "/")
	if len(s) != 4 || s[0] == "" || s[1] != "candidatos" || s[2] == "" || s[3] != "comprobaciones-ofrecimiento" {
		return "", "", false
	}
	for _, v := range []string{s[0], s[2]} {
		if _, err := url.PathUnescape(v); err != nil || strings.ContainsAny(v, "?# \\%") || len(v) > 512 {
			return "", "", false
		}
	}
	return s[0], s[2], true
}

func responderErrorComprobacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrComprobacionResultadoNoConfirmado):
		responderCodigoComprobacion(w, 503, "resultado_no_confirmado")
	case errors.Is(err, vd.ErrAutorizacionDenegada), errors.Is(err, vd.ErrPermissionDenied):
		responderCodigoComprobacion(w, 403, "acceso_denegado")
	case errors.Is(err, ports.ErrComprobacionConflicto):
		responderCodigoComprobacion(w, 409, "comprobacion_en_conflicto")
	case errors.Is(err, ports.ErrComprobacionInvalida):
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
	case errors.Is(err, ports.ErrComprobacionDependencia):
		responderCodigoComprobacion(w, 503, "comprobacion_ofrecimiento_no_habilitada")
	default:
		responderCodigoComprobacion(w, 503, "servicio_no_disponible")
	}
}

func responderCodigoComprobacion(w http.ResponseWriter, status int, codigo string) {
	responderComprobacion(w, status, map[string]any{"error": map[string]string{"codigo": codigo}})
}
func responderComprobacion(w http.ResponseWriter, status int, valor any) {
	b, err := json.Marshal(valor)
	if err != nil {
		status = http.StatusServiceUnavailable
		b = []byte(`{"error":{"codigo":"servicio_no_disponible"}}`)
		slog.Error("bolsa: respuesta de comprobacion no serializable", "error", err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if _, err = w.Write(append(b, '\n')); err != nil {
		slog.Warn("bolsa: respuesta de comprobacion no entregada", "error", err)
	}
}
