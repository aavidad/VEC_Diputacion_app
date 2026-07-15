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
		if err != nil || principal.Validate() != nil {
			h.writeError(w, http.StatusUnauthorized, "api.error.unauthorized", nil)
			return ports.AuthPrincipal{}, false
		}
		return principal, true
	}
	mechanism, mechanismOK := singleAuthorityHeader(r, "X-Auth-Mechanism")
	subject, subjectOK := singleAuthorityHeader(r, "X-Auth-Subject")
	token, tokenOK := authToken(r)
	if !mechanismOK || !subjectOK || !tokenOK {
		h.writeError(w, http.StatusUnauthorized, "api.error.unauthorized", nil)
		return ports.AuthPrincipal{}, false
	}
	credentials := ports.AuthCredentials{
		Mechanism: ports.AuthMechanism(mechanism),
		Subject:   subject,
		Token:     token,
	}
	if err := credentials.Validate(); err != nil {
		h.writeError(w, http.StatusUnauthorized, "api.error.unauthorized", nil)
		return ports.AuthPrincipal{}, false
	}
	principal, err := h.authenticator.Authenticate(r.Context(), credentials)
	if err != nil || principal.Validate() != nil {
		h.writeError(w, http.StatusUnauthorized, "api.error.unauthorized", nil)
		return ports.AuthPrincipal{}, false
	}
	return principal, true
}

func (h *Handler) requireStaff(w http.ResponseWriter, principal ports.AuthPrincipal) bool {
	// Solo perfiles funcionales de seleccion. El administrador tecnico no
	// hereda acceso a candidatos, expedientes, baremos ni notificaciones.
	if principal.Role == ports.AuthRolePersonalInterno ||
		principal.Role == ports.AuthRoleValidatorL2 {
		return true
	}
	h.writeError(w, http.StatusForbidden, "api.error.forbidden", nil)
	return false
}

func (h *Handler) requireTechnicalAdmin(w http.ResponseWriter, principal ports.AuthPrincipal) bool {
	if principal.Role == ports.AuthRoleSystemAdmin {
		return true
	}
	h.writeError(w, http.StatusForbidden, "api.error.forbidden", nil)
	return false
}

func (h *Handler) requireInternalMetadata(w http.ResponseWriter, principal ports.AuthPrincipal) bool {
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
	if exactCandidateReference(principal.Subject) && exactCandidateReference(candidateID) &&
		principal.Subject == candidateID {
		return true
	}
	h.writeError(w, http.StatusForbidden, "api.error.forbidden", nil)
	return false
}

func singleAuthorityHeader(r *http.Request, name string) (string, bool) {
	values := r.Header.Values(name)
	if len(values) == 0 {
		return "", true
	}
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func authToken(r *http.Request) (string, bool) {
	direct := r.Header.Values("X-Auth-Token")
	bearer := r.Header.Values("Authorization")
	if len(direct) > 0 && len(bearer) > 0 {
		return "", false
	}
	if len(direct) > 0 {
		if len(direct) != 1 {
			return "", false
		}
		return direct[0], true
	}
	if len(bearer) == 0 {
		return "", true
	}
	if len(bearer) != 1 || !strings.HasPrefix(bearer[0], "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(bearer[0], "Bearer "), true
}

func hasAllowedRole(role ports.AuthRole) bool {
	return role == ports.AuthRoleCiudadano || role == ports.AuthRolePersonalInterno ||
		role == ports.AuthRoleValidatorL2 || role == ports.AuthRoleSystemAdmin
}
