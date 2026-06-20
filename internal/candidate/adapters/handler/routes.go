package handler

import (
	"errors"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/candidate/ports"
)

var errInvalidRoute = errors.New("invalid route")

func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request, principal ports.AuthPrincipal) {
	path := apiPath(r.URL.Path)
	switch {
	case path == "/":
		h.handleAPIRoot(w, r)
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

func (h *Handler) handleAPIRoot(w http.ResponseWriter, r *http.Request) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	h.writeJSON(w, http.StatusOK, responseEnvelope{
		Data: map[string][]string{"routes": apiRootRoutes()},
	})
}

func apiRootRoutes() []string {
	routes := append([]string(nil), apiRoutes()...)
	routes = append(routes,
		"/api/modules/bolsa",
		"/api/modules/bolsa/manifest",
		"/api/modules/bolsa/healthz",
	)
	return routes
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
	if len(parts) != 3 || parts[0] != "candidates" || parts[1] == "" {
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
		var request administrativeClaimRequest
		if err := decodeJSON(r, &request); err != nil {
			h.writeError(w, http.StatusBadRequest, "api.error.bad_request", err)
			return
		}
		claim, err := h.administrative.PresentCandidateClaim(r.Context(), candidateID, principal, request)
		h.writeAdministrativeResult(w, http.StatusCreated, "api.candidate.claim_presented", claim, err)
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
		notifications, err := h.administrative.ListCandidateNotifications(r.Context(), r.URL.Query().Get("candidate_id"))
		h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.notifications_listed", notifications, err)
	case http.MethodPost:
		var request administrativeGlobalNotificationRequest
		if err := decodeJSON(r, &request); err != nil {
			h.writeError(w, http.StatusBadRequest, "api.error.bad_request", err)
			return
		}
		candidateID := defaultString(request.CandidateID, defaultString(r.URL.Query().Get("candidate_id"), r.URL.Query().Get("candidate")))
		notification, err := h.administrative.CreateCandidateNotification(r.Context(), candidateID, principal, request.candidateRequest())
		h.writeAdministrativeResult(w, http.StatusCreated, "api.candidate.notification_created", notification, err)
	default:
		w.Header().Set("Allow", "GET, POST")
		h.writeError(w, http.StatusMethodNotAllowed, "api.error.method_not_allowed", nil)
	}
}

func (h *Handler) notificationReceiptRequest(
	w http.ResponseWriter,
	r *http.Request,
	notificationID string,
) (administrativeNotificationReceiptRequest, bool) {
	if !h.requireMethod(w, r, http.MethodPost) {
		return administrativeNotificationReceiptRequest{}, false
	}
	var request administrativeNotificationReceiptRequest
	if err := decodeJSON(r, &request); err != nil {
		h.writeError(w, http.StatusBadRequest, "api.error.bad_request", err)
		return administrativeNotificationReceiptRequest{}, false
	}
	request.NotificationID = notificationID
	return request, true
}

func (h *Handler) handleAuditRoute(w http.ResponseWriter, r *http.Request, principal ports.AuthPrincipal) {
	if !h.requireStaff(w, principal) || !h.requireAdministrative(w) || !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	audit, err := h.administrative.ListAuditByScope(r.Context(), auditScopeFromQuery(r))
	h.writeAdministrativeResult(w, http.StatusOK, "api.candidate.audit_listed", audit, err)
}

func auditScopeFromQuery(r *http.Request) string {
	query := r.URL.Query()
	candidateID := strings.TrimSpace(query.Get("candidate_id"))
	if candidateID != "" {
		return "candidate:" + candidateID
	}
	return strings.TrimSpace(query.Get("scope"))
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
