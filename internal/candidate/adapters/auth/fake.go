package auth

import (
	"context"
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

// NewFakeAuthenticator crea un doble vacio para pruebas unitarias. No deriva
// nunca un token del sujeto ni precarga identidades; la composicion fake real
// usa exclusivamente el fichero seguro de bootstrap.
func NewFakeAuthenticator() (*FakeAuthenticator, error) {
	return &FakeAuthenticator{principals: make(map[authKey]ports.AuthPrincipal)}, nil
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
		Token:     token,
	}
	if err := credentials.Validate(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.principals == nil {
		f.principals = make(map[authKey]ports.AuthPrincipal)
	}
	key := keyFrom(credentials)
	if _, exists := f.principals[key]; exists {
		return ports.ErrAuthenticationFailed
	}
	f.principals[key] = clonePrincipal(normalized)
	return nil
}

func (f *FakeAuthenticator) Authenticate(
	ctx context.Context,
	credentials ports.AuthCredentials,
) (ports.AuthPrincipal, error) {
	if f == nil || ctx == nil {
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
	if err := ctx.Err(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	principal, ok := f.principals[keyFrom(credentials)]
	if !ok {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	return clonePrincipal(principal), nil
}

func keyFrom(credentials ports.AuthCredentials) authKey {
	return authKey{
		mechanism: credentials.Mechanism,
		subject:   credentials.Subject,
		token:     credentials.Token,
	}
}

func normalizePrincipal(principal ports.AuthPrincipal) (ports.AuthPrincipal, error) {
	if err := principal.Validate(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	principal.Role = principal.PrimaryRole()
	principal.Roles = principal.AllRoles()
	principal.Mechanism = principal.AuthMethod()
	principal.Method = principal.AuthMethod()
	return clonePrincipal(principal), nil
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
