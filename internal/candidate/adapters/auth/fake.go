package auth

import (
	"context"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/candidate/ports"
)

type FakeAuthenticator struct {
	mu         sync.RWMutex
	principals map[authKey]ports.AuthPrincipal
}

type authKey struct {
	mechanism ports.AuthMechanism
	subject   string
	token     string
}

func NewFakeAuthenticator(principals ...ports.AuthPrincipal) (*FakeAuthenticator, error) {
	fake := &FakeAuthenticator{
		principals: make(map[authKey]ports.AuthPrincipal, len(principals)),
	}
	for _, principal := range principals {
		if err := fake.Register(principal, principal.Subject); err != nil {
			return nil, err
		}
	}
	return fake, nil
}

func (f *FakeAuthenticator) Register(principal ports.AuthPrincipal, token string) error {
	if f == nil {
		return ports.ErrAuthenticationFailed
	}
	normalized, err := normalizePrincipal(principal)
	if err != nil {
		return err
	}
	credentials := ports.AuthCredentials{
		Mechanism: normalized.AuthMethod(),
		Subject:   normalized.Subject,
		Token:     strings.TrimSpace(token),
	}
	if err := credentials.Validate(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.principals == nil {
		f.principals = make(map[authKey]ports.AuthPrincipal)
	}
	f.principals[keyFrom(credentials)] = clonePrincipal(normalized)
	return nil
}

func (f *FakeAuthenticator) Authenticate(
	ctx context.Context,
	credentials ports.AuthCredentials,
) (ports.AuthPrincipal, error) {
	if f == nil {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	if err := ctx.Err(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	if err := credentials.Validate(); err != nil {
		return ports.AuthPrincipal{}, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	principal, ok := f.principals[keyFrom(credentials)]
	if !ok {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	return clonePrincipal(principal), nil
}

func keyFrom(credentials ports.AuthCredentials) authKey {
	return authKey{
		mechanism: credentials.Mechanism,
		subject:   strings.TrimSpace(credentials.Subject),
		token:     strings.TrimSpace(credentials.Token),
	}
}

func normalizePrincipal(principal ports.AuthPrincipal) (ports.AuthPrincipal, error) {
	if err := principal.Validate(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	principal.Subject = strings.TrimSpace(principal.Subject)
	principal.DisplayName = strings.TrimSpace(principal.DisplayName)
	principal.Email = strings.TrimSpace(principal.Email)
	principal.Role = principal.PrimaryRole()
	principal.Roles = normalizeRoles(principal.AllRoles())
	principal.Mechanism = principal.AuthMethod()
	principal.Method = principal.AuthMethod()
	return clonePrincipal(principal), nil
}

func normalizeRoles(roles []ports.AuthRole) []ports.AuthRole {
	if len(roles) == 0 {
		return nil
	}
	normalized := make([]ports.AuthRole, 0, len(roles))
	seen := make(map[ports.AuthRole]struct{}, len(roles))
	for _, role := range roles {
		if !role.IsValid() {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		normalized = append(normalized, role)
	}
	return normalized
}

func clonePrincipal(principal ports.AuthPrincipal) ports.AuthPrincipal {
	principal.Roles = cloneRoles(principal.Roles)
	principal.Attributes = cloneMap(principal.Attributes)
	return principal
}

func cloneRoles(roles []ports.AuthRole) []ports.AuthRole {
	if roles == nil {
		return nil
	}
	return append([]ports.AuthRole(nil), roles...)
}

func cloneMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
