package httpinscripcion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

const (
	RutaAlegacionesPropias = "/api/vec/bolsa/alegaciones"
	RutaAlegacionesRRHH    = "/api/vec/bolsa/rrhh/alegaciones"
	maximoCuerpoAlegacion  = 32 << 10
)

type PreparadorAlegaciones interface {
	PrepararLecturaAlegaciones(*http.Request, string, string, int, string) (inscripcion.Actor, error)
	PrepararPresentacionAlegacion(*http.Request, inscripcion.PresentacionAlegacion) (inscripcion.Actor, error)
}

type AplicacionAlegaciones interface {
	ObjetosAlegables(context.Context, inscripcion.Actor, int, string) (inscripcion.PaginaObjetosAlegacion, error)
	PresentarAlegacion(context.Context, inscripcion.Actor, inscripcion.PresentacionAlegacion) (inscripcion.ReciboAlegacion, error)
	AlegacionesPropias(context.Context, inscripcion.Actor, int, string) (inscripcion.PaginaAlegaciones, error)
	AlegacionesRRHH(context.Context, inscripcion.Actor, string, string, int, string) (inscripcion.PaginaAlegaciones, error)
	AmbitosAlegacionesRRHH(context.Context, inscripcion.Actor, int, string) (inscripcion.PaginaAmbitosAlegacionRRHH, error)
	AlegacionRRHH(context.Context, inscripcion.Actor, string) (inscripcion.Alegacion, error)
}

type HandlerAlegaciones struct {
	preparador PreparadorAlegaciones
	servicio   AplicacionAlegaciones
	canal      string
}

func NuevoAlegacionesExterno(p PreparadorAlegaciones, s AplicacionAlegaciones) (http.Handler, error) {
	if p == nil || s == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	return &HandlerAlegaciones{preparador: p, servicio: s, canal: "externa_personal"}, nil
}

func NuevoAlegacionesRRHH(p PreparadorAlegaciones, s AplicacionAlegaciones) (http.Handler, error) {
	if p == nil || s == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	return &HandlerAlegaciones{preparador: p, servicio: s, canal: "interna_corporativa"}, nil
}

func (h *HandlerAlegaciones) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if h == nil || h.preparador == nil || h.servicio == nil {
		responderError(w, 503, "servicio_no_disponible")
		return
	}
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.EscapedPath() != r.URL.Path || r.URL.ForceQuery ||
		strings.Contains(r.URL.Path, "//") || cabeceraProhibida(r.Header) ||
		(r.Header.Get("Accept") != "" && r.Header.Get("Accept") != "application/json") ||
		(r.Header.Get("Sec-Fetch-Site") != "" && r.Header.Get("Sec-Fetch-Site") != "same-origin") {
		responderError(w, 400, "peticion_no_permitida")
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	if h.canal == "externa_personal" {
		switch {
		case r.URL.Path == RutaAlegacionesPropias+"/objetos" && r.Method == http.MethodGet:
			h.objetos(w, r, q)
		case r.URL.Path == RutaAlegacionesPropias && r.Method == http.MethodGet:
			h.propias(w, r, q)
		case r.URL.Path == RutaAlegacionesPropias && r.Method == http.MethodPost:
			h.presentar(w, r, q)
		default:
			responderError(w, 404, "recurso_no_encontrado")
		}
		return
	}
	if h.canal != "interna_corporativa" {
		responderError(w, 503, "servicio_no_disponible")
		return
	}
	switch {
	case r.URL.Path == RutaAlegacionesRRHH+"/ambitos" && r.Method == http.MethodGet:
		h.ambitos(w, r, q)
	case r.URL.Path == RutaAlegacionesRRHH && r.Method == http.MethodGet:
		h.rrhh(w, r, q)
	case strings.HasPrefix(r.URL.Path, RutaAlegacionesRRHH+"/") && r.Method == http.MethodGet:
		h.detalleRRHH(w, r, q)
	default:
		responderError(w, 404, "recurso_no_encontrado")
	}
}

func parametrosPaginaAlegacion(q url.Values, extras ...string) (int, string, bool) {
	permitidas := map[string]bool{"limite": true, "cursor": true}
	for _, extra := range extras {
		permitidas[extra] = true
	}
	for clave, valores := range q {
		if !permitidas[clave] || len(valores) != 1 {
			return 0, "", false
		}
	}
	limite := 10
	if _, ok := q["limite"]; ok {
		var err error
		limite, err = strconv.Atoi(q.Get("limite"))
		if err != nil {
			var numero *strconv.NumError
			if errors.As(err, &numero) {
				slog.Warn("bolsa_alegaciones_limite_invalido", "clave", "limite", "esperado", "entero_1_20", "actual", "formato_invalido", "causa", numero.Err)
			} else {
				slog.Warn("bolsa_alegaciones_limite_invalido", "clave", "limite", "esperado", "entero_1_20", "actual", "conversion_invalida", "causa", err)
			}
			return 0, "", false
		}
		if limite < 1 || limite > 20 {
			return 0, "", false
		}
	}
	cursor := q.Get("cursor")
	if cursor != "" && (!referenciaAlegacionHTTP(cursor) || len(cursor) > 256) {
		return 0, "", false
	}
	return limite, cursor, true
}

func referenciaAlegacionHTTP(ref string) bool {
	if len(ref) < 3 || len(ref) > 512 {
		return false
	}
	for i, c := range ref {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			i > 0 && (c == ':' || c == '.' || c == '_' || c == '-') {
			continue
		}
		return false
	}
	return true
}

func (h *HandlerAlegaciones) lectura(w http.ResponseWriter, r *http.Request, accion, ref string, limite int, cursor string) (inscripcion.Actor, bool) {
	if !sinCuerpo(r) {
		responderError(w, 400, "peticion_no_permitida")
		return inscripcion.Actor{}, false
	}
	actor, err := h.preparador.PrepararLecturaAlegaciones(r, accion, ref, limite, cursor)
	if err != nil {
		responderFallo(w, r, err)
		return inscripcion.Actor{}, false
	}
	return actor, true
}

func (h *HandlerAlegaciones) objetos(w http.ResponseWriter, r *http.Request, q url.Values) {
	limite, cursor, ok := parametrosPaginaAlegacion(q)
	if !ok {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, ok := h.lectura(w, r, inscripcion.AccionObjetosAlegacion, "objetos", limite, cursor)
	if !ok {
		return
	}
	p, err := h.servicio.ObjetosAlegables(r.Context(), actor, limite, cursor)
	if err != nil {
		responderFallo(w, r, err)
		return
	}
	responder(w, 200, inscripcion.EsquemaObjetosAlegacion, p)
}

func (h *HandlerAlegaciones) propias(w http.ResponseWriter, r *http.Request, q url.Values) {
	limite, cursor, ok := parametrosPaginaAlegacion(q)
	if !ok {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, ok := h.lectura(w, r, inscripcion.AccionAlegacionesPropias, "propias", limite, cursor)
	if !ok {
		return
	}
	p, err := h.servicio.AlegacionesPropias(r.Context(), actor, limite, cursor)
	if err != nil {
		responderFallo(w, r, err)
		return
	}
	responder(w, 200, inscripcion.EsquemaAlegacionesPropias, p)
}

func (h *HandlerAlegaciones) presentar(w http.ResponseWriter, r *http.Request, q url.Values) {
	if len(q) != 0 || r.Header.Get("Content-Type") != "application/json" || r.Body == nil ||
		r.ContentLength < 2 || r.ContentLength > maximoCuerpoAlegacion || len(r.TransferEncoding) != 0 {
		responderError(w, 400, "peticion_no_permitida")
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoAlegacion))
	if err != nil || int64(len(contenido)) != r.ContentLength {
		responderError(w, 400, "peticion_no_permitida")
		return
	}
	var p inscripcion.PresentacionAlegacion
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&p) != nil || dec.Decode(new(any)) != io.EOF || p.Validar() != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, err := h.preparador.PrepararPresentacionAlegacion(r, p)
	if err != nil {
		responderFallo(w, r, err)
		return
	}
	recibo, err := h.servicio.PresentarAlegacion(r.Context(), actor, p)
	if err != nil {
		responderFallo(w, r, err)
		return
	}
	estado := http.StatusCreated
	if recibo.Repetida {
		estado = http.StatusOK
	}
	responder(w, estado, inscripcion.EsquemaAlegacionPresentada, recibo)
}

func (h *HandlerAlegaciones) ambitos(w http.ResponseWriter, r *http.Request, q url.Values) {
	limite, cursor, ok := parametrosPaginaAlegacion(q)
	if !ok {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, ok := h.lectura(w, r, inscripcion.AccionAlegacionesRRHH, "ambitos", limite, cursor)
	if !ok {
		return
	}
	p, err := h.servicio.AmbitosAlegacionesRRHH(r.Context(), actor, limite, cursor)
	if err != nil {
		responderFallo(w, r, err)
		return
	}
	responder(w, 200, "vec.bolsa.alegaciones.ambitos.v1", p)
}

func (h *HandlerAlegaciones) rrhh(w http.ResponseWriter, r *http.Request, q url.Values) {
	limite, cursor, ok := parametrosPaginaAlegacion(q, "convocatoria_ref", "bolsa_ref")
	if !ok || (q.Has("convocatoria_ref") == q.Has("bolsa_ref")) {
		responderError(w, 400, "datos_no_validos")
		return
	}
	tipo, ref := "convocatoria", q.Get("convocatoria_ref")
	if q.Has("bolsa_ref") {
		tipo, ref = "bolsa", q.Get("bolsa_ref")
	}
	if !referenciaAlegacionHTTP(ref) {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, ok := h.lectura(w, r, inscripcion.AccionAlegacionesRRHH, tipo+":"+ref, limite, cursor)
	if !ok {
		return
	}
	p, err := h.servicio.AlegacionesRRHH(r.Context(), actor, tipo, ref, limite, cursor)
	if err != nil {
		responderFallo(w, r, err)
		return
	}
	responder(w, 200, inscripcion.EsquemaAlegacionesRRHH, p)
}

func (h *HandlerAlegaciones) detalleRRHH(w http.ResponseWriter, r *http.Request, q url.Values) {
	if len(q) != 0 {
		responderError(w, 400, "datos_no_validos")
		return
	}
	ref := strings.TrimPrefix(r.URL.Path, RutaAlegacionesRRHH+"/")
	if !referenciaAlegacionHTTP(ref) {
		responderError(w, 404, "recurso_no_encontrado")
		return
	}
	actor, ok := h.lectura(w, r, inscripcion.AccionAlegacionesRRHH, ref, 1, "")
	if !ok {
		return
	}
	a, err := h.servicio.AlegacionRRHH(r.Context(), actor, ref)
	if err != nil {
		responderFallo(w, r, err)
		return
	}
	responder(w, 200, inscripcion.EsquemaDetalleAlegacion, a)
}
