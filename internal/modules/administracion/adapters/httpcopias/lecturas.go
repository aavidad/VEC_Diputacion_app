package httpcopias

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

func (h *Handler) get(w http.ResponseWriter, r *http.Request, s p.Sesion) {
	if r.Body != nil {
		var b [1]byte
		n, err := r.Body.Read(b[:])
		if n != 0 || err != nil && err != io.EOF {
			h.denegar(w, r, s, p.ErrSolicitud, "consultar", "")
			return
		}
	}
	path := r.URL.Path
	if path == PrefijoV1+"/capacidades" && r.URL.RawQuery == "" {
		if !h.autorizado(w, r, s, p.Consultar, "copias", h.servicio.Lecturas) {
			return
		}
		c, err := h.servicio.Capacidades(r.Context(), s)
		if err != nil {
			h.denegar(w, r, s, err, string(p.Consultar), "copias")
			return
		}
		respuesta(w, 200, struct {
			Capacidades p.Capacidades `json:"capacidades"`
		}{c})
		return
	}
	ref := "copias"
	var result any
	var err error
	switch path {
	case PrefijoV1:
		q, e := url.ParseQuery(r.URL.RawQuery)
		if e != nil || len(q) > 2 || len(q["cursor"]) > 1 || len(q["limite"]) > 1 || len(q.Get("cursor")) > 256 {
			h.denegar(w, r, s, p.ErrSolicitud, "consultar", "")
			return
		}
		for k := range q {
			if k != "cursor" && k != "limite" {
				h.denegar(w, r, s, p.ErrSolicitud, "consultar", "")
				return
			}
		}
		limit := 25
		if q.Get("limite") != "" {
			limit, e = strconv.Atoi(q.Get("limite"))
			if e != nil || limit < 1 || limit > 100 {
				h.denegar(w, r, s, p.ErrSolicitud, "consultar", "")
				return
			}
		}
		if !h.autorizado(w, r, s, p.Consultar, "copias", h.servicio.Lecturas) {
			return
		}
		result, err = h.servicio.Lecturas.Listar(r.Context(), s, q.Get("cursor"), limit)
	case PrefijoV1 + "/opciones-restauracion":
		if r.URL.RawQuery != "" {
			h.denegar(w, r, s, p.ErrSolicitud, "consultar", "")
			return
		}
		if !h.autorizado(w, r, s, p.Consultar, "copias", h.servicio.Opciones) {
			return
		}
		result, err = h.servicio.Opciones.Opciones(r.Context(), s)
	case PrefijoV1 + "/calendario", PrefijoV1 + "/retencion", PrefijoV1 + "/propuestas":
		if r.URL.RawQuery != "" {
			h.denegar(w, r, s, p.ErrSolicitud, "consultar", "")
			return
		}
		if !h.autorizado(w, r, s, p.Consultar, "copias", h.servicio.Lecturas) {
			return
		}
		switch path {
		case PrefijoV1 + "/calendario":
			var v p.Configuracion
			v, err = h.servicio.Lecturas.Calendario(r.Context(), s)
			result = struct {
				Calendario p.Configuracion `json:"configuracion"`
			}{v}
		case PrefijoV1 + "/retencion":
			var v p.Configuracion
			v, err = h.servicio.Lecturas.Retencion(r.Context(), s)
			result = struct {
				Retencion p.Configuracion `json:"configuracion"`
			}{v}
		default:
			var v []p.Propuesta
			v, err = h.servicio.Lecturas.Propuestas(r.Context(), s)
			result = struct {
				Propuestas []p.Propuesta `json:"propuestas"`
			}{v}
		}
	default:
		ref = strings.TrimPrefix(path, PrefijoV1+"/")
		if !strings.HasPrefix(path, PrefijoV1+"/") || !referencia(ref) || r.URL.RawQuery != "" {
			h.denegarStatus(w, r, s, 404, "recurso_no_encontrado", "consultar", "")
			return
		}
		if !h.autorizado(w, r, s, p.Consultar, ref, h.servicio.Lecturas) {
			return
		}
		var v p.Copia
		v, err = h.servicio.Lecturas.Detalle(r.Context(), s, ref)
		if err == nil && v.CopiaRef != ref {
			err = p.ErrNoDisponible
		}
		result = struct {
			Copia p.Copia `json:"copia"`
		}{v}
	}
	if err != nil {
		h.denegar(w, r, s, err, string(p.Consultar), ref)
		return
	}
	respuesta(w, 200, result)
}
