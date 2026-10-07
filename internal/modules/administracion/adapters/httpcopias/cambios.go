package httpcopias

import (
	"mime"
	"net/http"
	"strings"
	"time"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

func (h *Handler) post(w http.ResponseWriter, r *http.Request, s p.Sesion) {
	media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	for key := range params {
		if key != "charset" {
			h.denegar(w, r, s, p.ErrSolicitud, "escribir", "")
			return
		}
	}
	if err != nil || media != "application/json" || len(params) > 1 || params["charset"] != "" && strings.ToLower(params["charset"]) != "utf-8" || r.URL.RawQuery != "" {
		h.denegar(w, r, s, p.ErrSolicitud, "escribir", "")
		return
	}
	var result any
	var accion p.Operacion
	recurso := "copias"
	switch r.URL.Path {
	case PrefijoV1 + "/lanzamientos":
		accion = p.Lanzar
		var v p.SolicitudLanzamiento
		if !h.decode(w, r, s, &v) {
			return
		}
		if !referencia(v.OperacionRef) || v.Tipo != "completa" {
			h.denegar(w, r, s, p.ErrSolicitud, "lanzar", "")
			return
		}
		if !h.autorizado(w, r, s, p.Lanzar, "copias", h.servicio.Cambios) {
			return
		}
		var recibo p.Recibo
		recibo, err = h.servicio.Cambios.Lanzar(r.Context(), s, v)
		result = envolverRecibo(recibo, v.OperacionRef, &err)
	case PrefijoV1 + "/calendario":
		accion = p.ConfigurarCalendario
		var v p.SolicitudPolitica
		if !h.decode(w, r, s, &v) {
			return
		}
		if !referencia(v.OperacionRef) || !politicaValida(v.Politica) {
			h.denegar(w, r, s, p.ErrSolicitud, "calendario", "")
			return
		}
		if !h.autorizado(w, r, s, p.ConfigurarCalendario, "copias", h.servicio.Cambios) {
			return
		}
		var recibo p.Recibo
		recibo, err = h.servicio.Cambios.ConfigurarCalendario(r.Context(), s, v)
		result = envolverRecibo(recibo, v.OperacionRef, &err)
	case PrefijoV1 + "/retencion":
		accion = p.ConfigurarRetencion
		var v p.SolicitudPolitica
		if !h.decode(w, r, s, &v) {
			return
		}
		if !referencia(v.OperacionRef) || !politicaValida(v.Politica) {
			h.denegar(w, r, s, p.ErrSolicitud, "retencion", "")
			return
		}
		if !h.autorizado(w, r, s, p.ConfigurarRetencion, "copias", h.servicio.Cambios) {
			return
		}
		var recibo p.Recibo
		recibo, err = h.servicio.Cambios.ConfigurarRetencion(r.Context(), s, v)
		result = envolverRecibo(recibo, v.OperacionRef, &err)
	case PrefijoV1 + "/propuestas":
		accion = p.Proponer
		var v p.SolicitudPropuesta
		if !h.decode(w, r, s, &v) {
			return
		}
		if !referencia(v.OperacionRef) || !referencia(v.ConjuntoRef) || !referencia(v.DestinoRef) || !referencia(v.MotivoRef) || !referencia(v.VentanaRef) || !utc(v.VentanaInicio) || !utc(v.VentanaFin) || !utc(v.CaducaEn) || !v.VentanaInicio.Before(v.VentanaFin) || !time.Now().Before(v.CaducaEn) {
			h.denegar(w, r, s, p.ErrSolicitud, "proponer", "")
			return
		}
		recurso = v.ConjuntoRef
		if !h.autorizado(w, r, s, p.Proponer, v.ConjuntoRef, h.servicio.Control) {
			return
		}
		var vout p.Propuesta
		vout, err = h.servicio.Control.Proponer(r.Context(), s, v)
		if err == nil && (!propuestaValida(vout) || vout.PropuestaRef != v.OperacionRef || vout.ConjuntoRef != v.ConjuntoRef || vout.DestinoRef != v.DestinoRef || vout.MotivoRef != v.MotivoRef || vout.VentanaRef != v.VentanaRef || !vout.CaducaEn.Equal(v.CaducaEn) || !vout.VentanaInicio.Equal(v.VentanaInicio) || !vout.VentanaFin.Equal(v.VentanaFin)) {
			err = p.ErrNoDisponible
		}
		result = struct {
			Propuesta p.Propuesta `json:"propuesta"`
		}{vout}
	default:
		path := strings.TrimPrefix(r.URL.Path, PrefijoV1+"/propuestas/")
		parts := strings.Split(path, "/")
		if !strings.HasPrefix(r.URL.Path, PrefijoV1+"/propuestas/") || len(parts) != 2 || !referencia(parts[0]) || parts[1] != "revision" && parts[1] != "ejecucion" {
			h.denegarStatus(w, r, s, 404, "recurso_no_encontrado", "escribir", "")
			return
		}
		var v p.SolicitudControl
		if !h.decode(w, r, s, &v) {
			return
		}
		if !referencia(v.OperacionRef) || !referencia(v.DestinoRef) || v.VersionEsperada == 0 || !huella(v.PropuestaHuellaSHA256) {
			h.denegar(w, r, s, p.ErrSolicitud, "control", parts[0])
			return
		}
		op := p.Revisar
		if parts[1] == "ejecucion" {
			op = p.Ejecutar
		}
		accion, recurso = op, parts[0]
		if !h.autorizado(w, r, s, op, parts[0], h.servicio.Control) {
			return
		}
		if op == p.Revisar {
			if _, prepareErr := h.servicio.PrepararRevision(r.Context(), s, parts[0], v); prepareErr != nil {
				h.denegar(w, r, s, prepareErr, string(op), parts[0])
				return
			}
			var vout p.Propuesta
			vout, err = h.servicio.Control.Revisar(r.Context(), s, parts[0], v)
			if err == nil && (!propuestaValida(vout) || vout.PropuestaRef != parts[0] || vout.HuellaSHA256 != v.PropuestaHuellaSHA256 || vout.DestinoRef != v.DestinoRef) {
				err = p.ErrNoDisponible
			}
			result = struct {
				Propuesta p.Propuesta `json:"propuesta"`
			}{vout}
		} else {
			var recibo p.Recibo
			recibo, err = h.servicio.Control.Ejecutar(r.Context(), s, parts[0], v)
			result = envolverRecibo(recibo, v.OperacionRef, &err)
		}
	}
	if err != nil {
		h.denegar(w, r, s, err, string(accion), recurso)
		return
	}
	respuesta(w, 200, result)
}
