package auth

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/candidate/ports"
)

func TestFakeAuthenticatorAuthenticatesRegisteredPrincipalsByRole(t *testing.T) {
	t.Parallel()

	authenticator, err := NewFakeAuthenticator(
		ports.AuthPrincipal{
			Subject:   "candidate",
			Role:      ports.AuthRoleCiudadano,
			Mechanism: ports.AuthMechanismClave,
			Attributes: map[string]string{
				"source": "test",
			},
		},
		ports.AuthPrincipal{
			Subject:   "staff",
			Role:      ports.AuthRolePersonalInterno,
			Mechanism: ports.AuthMechanismKerberosAD,
		},
	)
	if err != nil {
		t.Fatalf("NewFakeAuthenticator() error = %v", err)
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

	authenticator, err := NewFakeAuthenticator(ports.AuthPrincipal{
		Subject:   "candidate",
		Role:      ports.AuthRoleCiudadano,
		Mechanism: ports.AuthMechanismClave,
	})
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

func TestFakeAuthenticatorTrimsRegisteredTokensForSmokeAuth(t *testing.T) {
	t.Parallel()

	authenticator, err := NewFakeAuthenticator()
	if err != nil {
		t.Fatalf("NewFakeAuthenticator() error = %v", err)
	}
	err = authenticator.Register(ports.AuthPrincipal{
		Subject:   "candidate",
		Role:      ports.AuthRoleCiudadano,
		Mechanism: ports.AuthMechanismClave,
	}, " citizen-token ")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	principal, err := authenticator.Authenticate(context.Background(), ports.AuthCredentials{
		Mechanism: ports.AuthMechanismClave,
		Subject:   "candidate",
		Token:     "citizen-token",
	})
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.Role != ports.AuthRoleCiudadano {
		t.Fatalf("Authenticate() role = %q", principal.Role)
	}
}
