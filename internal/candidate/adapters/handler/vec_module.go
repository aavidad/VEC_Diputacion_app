package handler

import (
	"net/http"

	"vec-diputacion-granada/internal/candidate/ports"
	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
)

func (h *Handler) handleVECModuleRoute(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	principal ports.AuthPrincipal,
) {
	if !h.requireStaff(w, principal) || !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if path == "/modules/bolsa/healthz" {
		h.writeJSON(w, http.StatusOK, responseEnvelope{
			Data: map[string]string{"module_ref": "vec.module.bolsa", "status": "ok"},
		})
		return
	}
	h.writeJSON(w, http.StatusOK, responseEnvelope{
		Message: h.t("api.module.bolsa.manifest_loaded"),
		Data:    bolsamodule.ModuleManifestForCandidatePortal(),
	})
}

func (h *Handler) handleAdminRoute(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	principal ports.AuthPrincipal,
) {
	if !h.requireStaff(w, principal) || !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	switch path {
	case "/admin/status":
		h.writeJSON(w, http.StatusOK, responseEnvelope{
			Message: h.t("api.admin.status_loaded"),
			Data:    h.status,
		})
	case "/admin/capabilities":
		h.writeJSON(w, http.StatusOK, responseEnvelope{
			Message: h.t("api.admin.capabilities_loaded"),
			Data:    bolsamodule.AdminCapabilitiesContract(),
		})
	}
}
