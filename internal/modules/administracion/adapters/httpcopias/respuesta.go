package httpcopias

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

func envolverRecibo(v p.Recibo, op string, err *error) any {
	if *err == nil && (v.OperacionRef != op || !referencia(v.ReciboRef) || !referencia(v.RecursoRef) || v.Estado == "" || !utc(v.RegistradoEn)) {
		*err = p.ErrNoDisponible
	}
	return struct {
		Recibo p.Recibo `json:"recibo"`
	}{v}
}
func politicaValida(v p.Politica) bool {
	if v.Formato != 1 || !referencia(v.Referencia) || !referencia(v.Destino) || len(v.ZonaHoraria) > 96 || v.ZonaHoraria == "" || v.ZonaHoraria == "Local" || v.CadaDias < 1 || v.CadaDias > 366 || len(v.DiasSemana) > 7 || len(v.Retencion.Protegidas) > 1024 || v.Retencion.ConservarMinimo < 1 || v.Retencion.ConservarMinimo > 10000 || v.Retencion.EdadMaximaDias < 1 || v.Retencion.EdadMaximaDias > 36500 {
		return false
	}
	if _, e := time.Parse("2006-01-02", v.FechaInicial); e != nil {
		return false
	}
	for _, value := range []string{v.Ventana.Inicio, v.Ventana.Fin} {
		t, e := time.Parse("15:04", value)
		if e != nil || t.Format("15:04") != value {
			return false
		}
	}
	seen := map[int]bool{}
	for _, day := range v.DiasSemana {
		if day < 0 || day > 6 || seen[day] {
			return false
		}
		seen[day] = true
	}
	return true
}
func utc(t time.Time) bool { _, offset := t.Zone(); return !t.IsZero() && offset == 0 }
func referencia(s string) bool {
	if len(s) < 1 || len(s) > 192 || !(s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z' || s[0] >= '0' && s[0] <= '9') {
		return false
	}
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == ':' || c == '.' {
			continue
		}
		return false
	}
	return true
}
func huella(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' {
			continue
		}
		return false
	}
	return true
}
func (h *Handler) denegar(w http.ResponseWriter, r *http.Request, s p.Sesion, err error, accion, ref string) {
	status, code := errorHTTP(err)
	h.denegarStatus(w, r, s, status, code, accion, ref)
}
func (h *Handler) denegarStatus(w http.ResponseWriter, r *http.Request, s p.Sesion, status int, code, accion, ref string) {
	accion, ref = accionAuditoria(r, accion, ref)
	if h.auditor(context.WithoutCancel(r.Context()), Denegacion{code, accion, ref, s.Actor.PersonaRef, s.Actor.PerfilActivoRef, s.CorrelacionRef}) != nil {
		fallo(w, p.ErrNoDisponible)
		return
	}
	falloCodigo(w, status, code)
}
func errorHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, p.ErrAutenticacion):
		return 401, "autenticacion_requerida"
	case errors.Is(err, p.ErrDenegado):
		return 403, "acceso_denegado"
	case errors.Is(err, p.ErrSolicitud):
		return 400, "solicitud_invalida"
	case errors.Is(err, p.ErrConflicto):
		return 409, "conflicto_estado"
	case errors.Is(err, p.ErrNoEncontrado):
		return 404, "recurso_no_encontrado"
	default:
		return 503, "servicio_no_disponible"
	}
}
func fallo(w http.ResponseWriter, err error) { s, c := errorHTTP(err); falloCodigo(w, s, c) }
func falloCodigo(w http.ResponseWriter, status int, code string) {
	respuesta(w, status, struct {
		Error struct {
			Codigo    string `json:"codigo"`
			ClaveI18N string `json:"clave_i18n"`
		} `json:"error"`
	}{Error: struct {
		Codigo    string `json:"codigo"`
		ClaveI18N string `json:"clave_i18n"`
	}{code, "api.admin.copias.error." + code}})
}
func respuesta(w http.ResponseWriter, status int, v any) {
	for _, key := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location"} {
		w.Header().Del(key)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
