package httpapi

import (
	"net/http"

	"vec-diputacion-granada/internal/vec/domain"
)

// handleWorkspace conserva la ruta publicada, pero no construye ni retiene
// datos demostrativos. El antiguo snapshot mezclaba registros de varias
// personas y modulos sin un resolver de recursos en el servidor; ademas, no
// existia ningun camino HTTP que lo entregase. Mantenerlo dentro del adaptador
// aumentaba el riesgo de exponerlo accidentalmente en una refactorizacion.
func (h *Handler) handleWorkspace(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !principal.HasPermission("vec.workspace.read") {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	// La ruta solo podra abrirse cuando el PDP aporte un ambito positivo exacto
	// para cada seccion y campo. Hasta entonces falla cerrada sin datos.
	h.escribirSuperficieHTTPCronosNoDisponible(w)
}
