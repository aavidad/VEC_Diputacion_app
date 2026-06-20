package handler

import (
	"context"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/candidate/ports"
)

type requestAuthenticator interface {
	AuthenticateRequest(context.Context, *http.Request) (ports.AuthPrincipal, error)
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (ports.AuthPrincipal, bool) {
	if authenticator, ok := h.authenticator.(requestAuthenticator); ok {
		principal, err := authenticator.AuthenticateRequest(r.Context(), r)
		if err != nil {
			h.writeError(w, http.StatusUnauthorized, "api.error.unauthorized", nil)
			return ports.AuthPrincipal{}, false
		}
		return principal, true
	}
	credentials := ports.AuthCredentials{
		Mechanism: ports.AuthMechanism(strings.TrimSpace(r.Header.Get("X-Auth-Mechanism"))),
		Subject:   strings.TrimSpace(r.Header.Get("X-Auth-Subject")),
		Token:     authToken(r),
	}
	principal, err := h.authenticator.Authenticate(r.Context(), credentials)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "api.error.unauthorized", nil)
		return ports.AuthPrincipal{}, false
	}
	return principal, true
}

func (h *Handler) requireStaff(w http.ResponseWriter, principal ports.AuthPrincipal) bool {
	if principal.Role == ports.AuthRolePersonalInterno ||
		principal.Role == ports.AuthRoleValidatorL2 ||
		principal.Role == ports.AuthRoleSystemAdmin {
		return true
	}
	h.writeError(w, http.StatusForbidden, "api.error.forbidden", nil)
	return false
}

func (h *Handler) requireCandidate(w http.ResponseWriter, principal ports.AuthPrincipal) bool {
	if principal.Role == ports.AuthRoleCiudadano {
		return true
	}
	h.writeError(w, http.StatusForbidden, "api.error.forbidden", nil)
	return false
}

func (h *Handler) requireCandidateOwner(w http.ResponseWriter, principal ports.AuthPrincipal, candidateID string) bool {
	if !h.requireCandidate(w, principal) {
		return false
	}
	if strings.TrimSpace(principal.Subject) == strings.TrimSpace(candidateID) {
		return true
	}
	h.writeError(w, http.StatusForbidden, "api.error.forbidden", nil)
	return false
}

func authToken(r *http.Request) string {
	if token := strings.TrimSpace(r.Header.Get("X-Auth-Token")); token != "" {
		return token
	}
	return strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
}

func hasAllowedRole(role ports.AuthRole) bool {
	return role == ports.AuthRoleCiudadano || role == ports.AuthRolePersonalInterno ||
		role == ports.AuthRoleValidatorL2 || role == ports.AuthRoleSystemAdmin
}
