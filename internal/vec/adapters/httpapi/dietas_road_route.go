package httpapi

import (
	"net/http"

	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	"vec-diputacion-granada/internal/vec/domain"
)

// handleDietasRoadRoute conserva la frontera de autorizacion productiva. La
// decodificacion, los limites y la consulta OSRM viven en el adaptador comun de
// Cartografia; ninguna superficie puede obtener un calculador fabricando un
// Principal local.
func (h *Handler) handleDietasRoadRoute(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !principal.HasPermission(dietasmodule.PermissionRouteRead) {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	if h.roadRoute == nil {
		h.writeError(w, http.StatusServiceUnavailable, "Motor OSRM interno no configurado de forma completa y explicita.")
		return
	}
	h.roadRoute.ServeHTTP(w, r)
}
