package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/candidate/ports"
)

var errInvalidRoute = errors.New("invalid route")

func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request, principal ports.AuthPrincipal) {
	path := apiPath(r.URL.Path)
	switch {
	case path == "/":
		h.handleAPIRoot(w, r, principal)
	case path == "/demo":
		h.handleDemoRoute(w, r, principal)
	case path == "/portal":
		h.handleProfessionalPortalRoute(w, r, principal)
	case path == "/admin/status" || path == "/admin/capabilities":
		h.handleAdminRoute(w, r, path, principal)
	case path == "/modules/bolsa" || path == "/modules/bolsa/manifest" || path == "/modules/bolsa/healthz":
		h.handleVECModuleRoute(w, r, path, principal)
	case path == "/notifications" || isNotificationActionPath(path):
		h.handleNotificationsRoute(w, r, path, principal)
	case path == "/audit":
		h.handleAuditRoute(w, r, principal)
	case path == "/candidates":
		h.handleCreateCandidate(w, r, principal)
	case isCandidateDocumentsPath(path):
		h.handleCandidateDocumentsRoute(w, r, path, principal)
	case isCandidateClaimsPath(path):
		h.handleCandidateClaimsRoute(w, r, path, principal)
	case isCandidateNotificationsPath(path):
		h.handleCandidateNotificationsRoute(w, r, path, principal)
	case isCandidateAuditPath(path):
		h.handleCandidateAuditRoute(w, r, path, principal)
	case strings.HasPrefix(path, "/candidates/"):
		h.handleCandidateAction(w, r, path, principal)
	default:
		h.writeError(w, http.StatusNotFound, "api.error.not_found", errInvalidRoute)
	}
}

func (h *Handler) handleDemoRoute(
	w http.ResponseWriter,
	r *http.Request,
	principal ports.AuthPrincipal,
) {
	if !h.requireStaff(w, principal) || !h.requireMethod(w, r, http.MethodPost) {
		return
	}
	h.handleDemo(w, r)
}

func (h *Handler) handleAPIRoot(w http.ResponseWriter, r *http.Request, principal ports.AuthPrincipal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	h.writeJSON(w, http.StatusOK, responseEnvelope{
		Data: map[string][]string{"routes": apiRootRoutes(principal.Role)},
	})
}

func apiRootRoutes(role ports.AuthRole) []string {
	switch role {
	case ports.AuthRoleCiudadano:
		return []string{
			"/api/candidates",
			"/api/candidates/{id}/merits",
			"/api/candidates/{id}/baremo",
			"/api/candidates/{id}/expediente",
			"/api/candidates/{id}/documents",
			"/api/candidates/{id}/claims",
		}
	case ports.AuthRolePersonalInterno, ports.AuthRoleValidatorL2:
		return append(apiRoutes(),
			"/api/modules/bolsa",
			"/api/modules/bolsa/manifest",
			"/api/modules/bolsa/healthz",
		)
	case ports.AuthRoleSystemAdmin:
		return []string{
			"/api/admin/status",
			"/api/admin/capabilities",
			"/api/modules/bolsa",
			"/api/modules/bolsa/manifest",
			"/api/modules/bolsa/healthz",
		}
	default:
		return nil
	}
}

func apiPath(path string) string {
	if path == "/api" {
		return "/"
	}
	if strings.HasPrefix(path, "/api/") {
		return strings.TrimPrefix(path, "/api")
	}
	return path
}

func parseCandidateAction(path string) (string, string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "candidates" || !exactCandidateReference(parts[1]) {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func isCandidateDocumentsPath(path string) bool {
	_, action, ok := parseCandidateAction(path)
	return ok && action == "documents"
}

func isCandidateClaimsPath(path string) bool {
	_, action, ok := parseCandidateAction(path)
	return ok && action == "claims"
}

func isCandidateNotificationsPath(path string) bool {
	_, action, ok := parseCandidateAction(path)
	return ok && action == "notifications"
}

func isCandidateAuditPath(path string) bool {
	_, action, ok := parseCandidateAction(path)
	return ok && action == "audit"
}

func parseNotificationAction(path string) (string, string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "notifications" || parts[1] == "" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func isNotificationActionPath(path string) bool {
	_, action, ok := parseNotificationAction(path)
	return ok && (action == "send" || action == "read")
}

func (h *Handler) requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	h.writeError(w, http.StatusMethodNotAllowed, "api.error.method_not_allowed", nil)
	return false
}

func (h *Handler) handleCandidateClaimsRoute(w http.ResponseWriter, r *http.Request, path string, principal ports.AuthPrincipal) {
	if !h.requireAdministrative(w) {
		return
	}
	candidateID, _, _ := parseCandidateAction(path)
	if !h.requireCandidateOwner(w, principal, candidateID) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		claims, err := h.administrative.ListCandidateClaims(r.Context(), candidateID, r.URL.Query().Get("solicitud_id"))
		h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.claims_listed", claims, err)
	case http.MethodPost:
		// El recibo/CSV de la alegacion heredada procede del navegador. La
		// presentacion se mantiene cerrada hasta que registro y cotejo emitan la
		// evidencia desde el servidor.
		h.writeError(w, http.StatusServiceUnavailable, "api.error.probative_flow_unavailable", errFlujoProbatorioSeguroNoDisponible)
	default:
		w.Header().Set("Allow", "GET, POST")
		h.writeError(w, http.StatusMethodNotAllowed, "api.error.method_not_allowed", nil)
	}
}

func (h *Handler) handleCandidateNotificationsRoute(w http.ResponseWriter, r *http.Request, path string, principal ports.AuthPrincipal) {
	if !h.requireStaff(w, principal) || !h.requireAdministrative(w) {
		return
	}
	candidateID, _, _ := parseCandidateAction(path)
	switch r.Method {
	case http.MethodGet:
		notifications, err := h.administrative.ListCandidateNotifications(r.Context(), candidateID)
		h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.notifications_listed", notifications, err)
	case http.MethodPost:
		var request administrativeNotificationRequest
		if err := decodeJSON(r, &request); err != nil {
			h.writeError(w, http.StatusBadRequest, "api.error.bad_request", err)
			return
		}
		notification, err := h.administrative.CreateCandidateNotification(r.Context(), candidateID, principal, request)
		h.writeAdministrativeResult(w, http.StatusCreated, "api.candidate.notification_created", notification, err)
	default:
		w.Header().Set("Allow", "GET, POST")
		h.writeError(w, http.StatusMethodNotAllowed, "api.error.method_not_allowed", nil)
	}
}

func (h *Handler) handleCandidateAuditRoute(w http.ResponseWriter, r *http.Request, path string, principal ports.AuthPrincipal) {
	if !h.requireStaff(w, principal) || !h.requireAdministrative(w) || !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	candidateID, _, _ := parseCandidateAction(path)
	audit, err := h.administrative.ListCandidateAudit(r.Context(), candidateID)
	h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.audit_listed", audit, err)
}

func (h *Handler) handleNotificationsRoute(w http.ResponseWriter, r *http.Request, path string, principal ports.AuthPrincipal) {
	if !h.requireStaff(w, principal) || !h.requireAdministrative(w) {
		return
	}
	if path == "/notifications" {
		h.handleGlobalNotifications(w, r, principal)
		return
	}
	notificationID, action, _ := parseNotificationAction(path)
	request, ok := h.notificationReceiptRequest(w, r, notificationID)
	if !ok {
		return
	}
	switch action {
	case "send":
		notification, err := h.administrative.SendNotification(r.Context(), principal, request)
		h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.notification_sent", notification, err)
	case "read":
		notification, err := h.administrative.MarkNotificationRead(r.Context(), principal, request)
		h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.notification_read", notification, err)
	}
}

func (h *Handler) handleGlobalNotifications(w http.ResponseWriter, r *http.Request, principal ports.AuthPrincipal) {
	switch r.Method {
	case http.MethodGet:
		candidateID, ok := exactOnlyQueryValue(r.URL.Query(), "candidate_id", exactCandidateReference)
		if !ok {
			h.writeError(w, http.StatusBadRequest, "api.error.bad_request", nil)
			return
		}
		notifications, err := h.administrative.ListCandidateNotifications(r.Context(), candidateID)
		h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.notifications_listed", notifications, err)
	case http.MethodPost:
		if len(r.URL.Query()) != 0 {
			h.writeError(w, http.StatusBadRequest, "api.error.bad_request", nil)
			return
		}
		var request administrativeGlobalNotificationRequest
		if err := decodeJSON(r, &request); err != nil {
			h.writeError(w, http.StatusBadRequest, "api.error.bad_request", err)
			return
		}
		if !exactCandidateReference(request.CandidateID) {
			h.writeError(w, http.StatusBadRequest, "api.error.bad_request", nil)
			return
		}
		notification, err := h.administrative.CreateCandidateNotification(r.Context(), request.CandidateID, principal, request.candidateRequest())
		h.writeAdministrativeResult(w, http.StatusCreated, "api.candidate.notification_created", notification, err)
	default:
		w.Header().Set("Allow", "GET, POST")
		h.writeError(w, http.StatusMethodNotAllowed, "api.error.method_not_allowed", nil)
	}
}

func (h *Handler) notificationReceiptRequest(
	w http.ResponseWriter,
	r *http.Request,
	_ string,
) (administrativeNotificationReceiptRequest, bool) {
	if !h.requireMethod(w, r, http.MethodPost) {
		return administrativeNotificationReceiptRequest{}, false
	}
	// En el prototipo el solicitante de la transicion tambien declaraba el CSV
	// del envio/lectura. Solo el conector de notificaciones podra producir el
	// recibo que confirme estas transiciones.
	h.writeError(w, http.StatusServiceUnavailable, "api.error.probative_flow_unavailable", errFlujoProbatorioSeguroNoDisponible)
	return administrativeNotificationReceiptRequest{}, false
}

func (h *Handler) handleAuditRoute(w http.ResponseWriter, r *http.Request, principal ports.AuthPrincipal) {
	if !h.requireStaff(w, principal) || !h.requireAdministrative(w) || !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	scope, ok := auditScopeFromQuery(r)
	if !ok {
		h.writeError(w, http.StatusBadRequest, "api.error.bad_request", nil)
		return
	}
	audit, err := h.administrative.ListAuditByScope(r.Context(), scope)
	h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.audit_listed", audit, err)
}

func auditScopeFromQuery(r *http.Request) (string, bool) {
	query := r.URL.Query()
	if candidateID, ok := exactOnlyQueryValue(query, "candidate_id", exactCandidateReference); ok {
		return "candidate:" + candidateID, true
	}
	scope, ok := exactOnlyQueryValue(query, "scope", exactCandidateScope)
	return scope, ok
}

func exactOnlyQueryValue(query url.Values, key string, validate func(string) bool) (string, bool) {
	if len(query) != 1 {
		return "", false
	}
	values, ok := query[key]
	if !ok || len(values) != 1 || !validate(values[0]) {
		return "", false
	}
	return values[0], true
}

func exactCandidateScope(scope string) bool {
	const prefix = "candidate:"
	return strings.HasPrefix(scope, prefix) && exactCandidateReference(strings.TrimPrefix(scope, prefix))
}

func exactCandidateReference(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || len(value) > 256 ||
		!utf8.ValidString(value) || strings.ContainsAny(value, "*/\\?#") {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) || unicode.IsSpace(character) {
			return false
		}
	}
	return true
}

func (h *Handler) requireAdministrative(w http.ResponseWriter) bool {
	if h.administrative != nil {
		return true
	}
	h.writeError(w, http.StatusNotFound, "api.error.not_found", errInvalidRoute)
	return false
}

func (h *Handler) writeAdministrativeResult(w http.ResponseWriter, status int, message string, data any, err error) {
	if err != nil {
		h.writeError(w, administrativeStatusFromError(err), administrativeErrorKey(err), err)
		return
	}
	h.writeJSON(w, status, responseEnvelope{Message: h.t(message), Data: data})
}
