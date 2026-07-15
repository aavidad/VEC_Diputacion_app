package httpapi

import (
	"net/http"

	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	"vec-diputacion-granada/internal/vec/domain"
)

// El mensaje es deliberadamente generico. Las rutas heredadas/de demostracion
// no pueden deducir de un Principal, con seguridad, una persona empleada, una
// relacion jerarquica ni un ambito organizativo. Hasta que el resolver de
// recursos del servidor y el PDP aporten esa evidencia, toda peticion falla
// cerrada y ningun dato o mutacion alcanza el servicio de aplicacion de Cronos.
const mensajeSuperficieHTTPCronosNoDisponible = "superficie HTTP de Cronos no disponible"

func (h *Handler) handleCronosTimecards(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	switch r.Method {
	case http.MethodGet:
		if !principal.HasPermission(cronosmodule.PermissionTimeRead) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.escribirSuperficieHTTPCronosNoDisponible(w)
	case http.MethodPost:
		if !principal.HasPermission(cronosmodule.PermissionTimeManage) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.escribirSuperficieHTTPCronosNoDisponible(w)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleCronosLeaveRequests(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	switch r.Method {
	case http.MethodGet:
		if !principal.HasPermission(cronosmodule.PermissionLeaveRead) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.escribirSuperficieHTTPCronosNoDisponible(w)
	case http.MethodPost:
		if !principal.HasPermission(cronosmodule.PermissionLeaveManage) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.escribirSuperficieHTTPCronosNoDisponible(w)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) escribirSuperficieHTTPCronosNoDisponible(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	h.writeError(w, http.StatusServiceUnavailable, mensajeSuperficieHTTPCronosNoDisponible)
}
