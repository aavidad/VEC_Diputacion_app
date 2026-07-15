package auth

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/candidate/ports"
)

func TestFakeAuthenticatorAuthenticatesRegisteredPrincipalsByRole(t *testing.T) {
	t.Parallel()

	authenticator, err := NewFakeAuthenticator()
	if err != nil {
		t.Fatalf("NewFakeAuthenticator() error = %v", err)
	}
	for _, registro := range []struct {
		principal ports.AuthPrincipal
		token     string
	}{
		{principal: ports.AuthPrincipal{
			Subject:   "candidate",
			Role:      ports.AuthRoleCiudadano,
			Mechanism: ports.AuthMechanismClave,
			Attributes: map[string]string{
				"source": "test",
			},
		}, token: "candidate"},
		{principal: ports.AuthPrincipal{
			Subject:   "staff",
			Role:      ports.AuthRolePersonalInterno,
			Mechanism: ports.AuthMechanismKerberosAD,
		}, token: "staff"},
	} {
		if err := authenticator.Register(registro.principal, registro.token); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	tests := []struct {
		name        string
		credentials ports.AuthCredentials
		wantRole    ports.AuthRole
	}{
		{
			name: "citizen",
			credentials: ports.AuthCredentials{
				Mechanism: ports.AuthMechanismClave,
				Subject:   "candidate",
				Token:     "candidate",
			},
			wantRole: ports.AuthRoleCiudadano,
		},
		{
			name: "internal staff",
			credentials: ports.AuthCredentials{
				Mechanism: ports.AuthMechanismKerberosAD,
				Subject:   "staff",
				Token:     "staff",
			},
			wantRole: ports.AuthRolePersonalInterno,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			principal, err := authenticator.Authenticate(context.Background(), tt.credentials)
			if err != nil {
				t.Fatalf("Authenticate() error = %v", err)
			}
			if principal.Role != tt.wantRole {
				t.Fatalf("Authenticate() role = %q, want %q", principal.Role, tt.wantRole)
			}
			if principal.Subject != tt.credentials.Subject {
				t.Fatalf("Authenticate() subject = %q, want %q", principal.Subject, tt.credentials.Subject)
			}
		})
	}
}

func TestFakeAuthenticatorRejectsInvalidCredentialsAndPrincipals(t *testing.T) {
	t.Parallel()

	authenticator, err := NewFakeAuthenticator()
	if err != nil {
		t.Fatalf("NewFakeAuthenticator() error = %v", err)
	}
	err = authenticator.Register(ports.AuthPrincipal{
		Subject:   "candidate",
		Role:      ports.AuthRoleCiudadano,
		Mechanism: ports.AuthMechanismClave,
	}, "candidate")
	if err != nil {
		t.Fatalf("NewFakeAuthenticator() error = %v", err)
	}

	tests := []struct {
		name        string
		credentials ports.AuthCredentials
		wantErr     error
	}{
		{
			name: "wrong token",
			credentials: ports.AuthCredentials{
				Mechanism: ports.AuthMechanismClave,
				Subject:   "candidate",
				Token:     "wrong",
			},
			wantErr: ports.ErrAuthenticationFailed,
		},
		{
			name: "wrong role mechanism",
			credentials: ports.AuthCredentials{
				Mechanism: ports.AuthMechanismKerberosAD,
				Subject:   "candidate",
				Token:     "candidate",
			},
			wantErr: ports.ErrAuthenticationFailed,
		},
		{
			name: "missing token",
			credentials: ports.AuthCredentials{
				Mechanism: ports.AuthMechanismClave,
				Subject:   "candidate",
			},
			wantErr: ports.ErrAuthTokenRequired,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := authenticator.Authenticate(context.Background(), tt.credentials)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Authenticate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	err = authenticator.Register(ports.AuthPrincipal{
		Subject:   "reviewer",
		Role:      ports.AuthRole("auditor"),
		Mechanism: ports.AuthMechanismClave,
	}, "reviewer")
	if !errors.Is(err, ports.ErrAuthRoleInvalid) {
		t.Fatalf("Register() error = %v, want %v", err, ports.ErrAuthRoleInvalid)
	}
}

func TestFakeAuthenticatorNeverNormalizesIdentityOrTokenToCreateAMatch(t *testing.T) {
	t.Parallel()

	validPrincipal := ports.AuthPrincipal{
		Subject:   "candidate",
		Role:      ports.AuthRoleCiudadano,
		Mechanism: ports.AuthMechanismClave,
	}

	registerTests := []struct {
		name      string
		principal ports.AuthPrincipal
		token     string
	}{
		{name: "token with surrounding whitespace", principal: validPrincipal, token: " citizen-token "},
		{name: "subject with surrounding whitespace", principal: ports.AuthPrincipal{Subject: " candidate", Role: ports.AuthRoleCandidate, Mechanism: ports.AuthMechanismClave}, token: "citizen-token"},
		{name: "two roles", principal: ports.AuthPrincipal{Subject: "candidate", Role: ports.AuthRoleCandidate, Roles: []ports.AuthRole{ports.AuthRoleValidatorL1}, Mechanism: ports.AuthMechanismClave}, token: "citizen-token"},
		{name: "valid and invalid role", principal: ports.AuthPrincipal{Subject: "candidate", Role: ports.AuthRoleCandidate, Roles: []ports.AuthRole{"unknown"}, Mechanism: ports.AuthMechanismClave}, token: "citizen-token"},
		{name: "two mechanisms", principal: ports.AuthPrincipal{Subject: "candidate", Role: ports.AuthRoleCandidate, Mechanism: ports.AuthMechanismClave, Method: ports.AuthMechanismDNIe}, token: "citizen-token"},
	}
	for _, tt := range registerTests {
		authenticator, err := NewFakeAuthenticator()
		if err != nil {
			t.Fatalf("NewFakeAuthenticator() error = %v", err)
		}
		if err := authenticator.Register(tt.principal, tt.token); err == nil {
			t.Fatalf("%s: Register() accepted non canonical or ambiguous identity", tt.name)
		}
	}

	authenticator, err := NewFakeAuthenticator()
	if err != nil {
		t.Fatalf("NewFakeAuthenticator() error = %v", err)
	}
	if err := authenticator.Register(validPrincipal, "citizen-token"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	for _, credentials := range []ports.AuthCredentials{
		{Mechanism: ports.AuthMechanismClave, Subject: " candidate", Token: "citizen-token"},
		{Mechanism: ports.AuthMechanismClave, Subject: "candidate", Token: " citizen-token"},
		{Mechanism: ports.AuthMechanismClave, Subject: "candidate", Token: "citizen-token "},
	} {
		if _, err := authenticator.Authenticate(context.Background(), credentials); err == nil {
			t.Fatalf("Authenticate() accepted non canonical credentials: %+v", credentials)
		}
	}
}

func TestFakeAuthenticatorAcceptsOnlyExactDuplicateAuthorityRepresentation(t *testing.T) {
	t.Parallel()

	authenticator, err := NewFakeAuthenticator()
	if err != nil {
		t.Fatalf("NewFakeAuthenticator() error = %v", err)
	}
	principal := ports.AuthPrincipal{
		Subject:   "candidate",
		Role:      ports.AuthRoleCandidate,
		Roles:     []ports.AuthRole{ports.AuthRoleCandidate, ports.AuthRoleCandidate},
		Mechanism: ports.AuthMechanismClave,
		Method:    ports.AuthMechanismClave,
	}
	if err := authenticator.Register(principal, "citizen-token"); err != nil {
		t.Fatalf("Register() exact duplicate error = %v", err)
	}
	got, err := authenticator.Authenticate(context.Background(), ports.AuthCredentials{
		Mechanism: ports.AuthMechanismClave,
		Subject:   "candidate",
		Token:     "citizen-token",
	})
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if got.Role != ports.AuthRoleCandidate || len(got.Roles) != 1 || got.Roles[0] != ports.AuthRoleCandidate ||
		got.Mechanism != ports.AuthMechanismClave || got.Method != ports.AuthMechanismClave {
		t.Fatalf("principal was not reduced to one exact authority: %+v", got)
	}
}

func TestFakeAuthenticatorNeverOverwritesAnExistingCredentialAuthority(t *testing.T) {
	t.Parallel()

	authenticator, err := NewFakeAuthenticator()
	if err != nil {
		t.Fatalf("NewFakeAuthenticator() error = %v", err)
	}
	if err := authenticator.Register(ports.AuthPrincipal{
		Subject: "identity-1", Role: ports.AuthRoleCandidate, Mechanism: ports.AuthMechanismClave,
	}, "token-1"); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := authenticator.Register(ports.AuthPrincipal{
		Subject: "identity-1", Role: ports.AuthRoleSystemAdmin, Mechanism: ports.AuthMechanismClave,
	}, "token-1"); !errors.Is(err, ports.ErrAuthenticationFailed) {
		t.Fatalf("second Register() error = %v, want %v", err, ports.ErrAuthenticationFailed)
	}
	principal, err := authenticator.Authenticate(context.Background(), ports.AuthCredentials{
		Mechanism: ports.AuthMechanismClave,
		Subject:   "identity-1",
		Token:     "token-1",
	})
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.Role != ports.AuthRoleCandidate {
		t.Fatalf("credential authority was overwritten: %+v", principal)
	}
}
