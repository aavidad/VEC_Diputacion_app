package auth

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestTrustedHeadersAuthenticatorAuthenticatesConfiguredHeaders(t *testing.T) {
	t.Parallel()

	authenticator := newTestTrustedHeadersAuthenticator(t)
	req := httptest.NewRequest("GET", "/api/portal", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set(config.DefaultTrustedHeaderSubject, "candidate-1")
	req.Header.Set(config.DefaultTrustedHeaderRoles, "candidate,validator_l1")
	req.Header.Set(config.DefaultTrustedHeaderMechanism, string(ports.AuthMechanismClave))

	principal, err := authenticator.AuthenticateRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("AuthenticateRequest() error = %v", err)
	}
	if principal.Subject != "candidate-1" || principal.Role != ports.AuthRoleCandidate {
		t.Fatalf("principal = %+v", principal)
	}
	if !principal.HasRole(ports.AuthRoleValidatorL1) {
		t.Fatalf("principal roles = %#v", principal.Roles)
	}
}

func TestTrustedHeadersAuthenticatorAcceptsCanonicalVECRoles(t *testing.T) {
	t.Parallel()

	authenticator := newTestTrustedHeadersAuthenticator(t)
	for _, role := range []ports.AuthRole{
		ports.AuthRoleCandidate,
		ports.AuthRoleValidatorL1,
		ports.AuthRoleValidatorL2,
		ports.AuthRoleSystemAdmin,
	} {
		role := role
		t.Run(string(role), func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/api/portal", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			req.Header.Set(config.DefaultTrustedHeaderSubject, "vec-user")
			req.Header.Set(config.DefaultTrustedHeaderRoles, string(role))
			req.Header.Set(config.DefaultTrustedHeaderMechanism, string(ports.AuthMechanismKerberosAD))

			principal, err := authenticator.AuthenticateRequest(context.Background(), req)
			if err != nil {
				t.Fatalf("AuthenticateRequest() error = %v", err)
			}
			if principal.Role != role {
				t.Fatalf("principal role = %q, want %q", principal.Role, role)
			}
		})
	}
}

func TestTrustedHeadersAuthenticatorRejectsUntrustedOrIncompleteRequest(t *testing.T) {
	t.Parallel()

	authenticator := newTestTrustedHeadersAuthenticator(t)
	tests := []struct {
		name       string
		remoteAddr string
		subject    string
		roles      string
		mechanism  string
		wantErr    error
	}{
		{
			name:       "untrusted proxy",
			remoteAddr: "10.10.10.10:12345",
			subject:    "staff",
			roles:      string(ports.AuthRoleValidatorL1),
			mechanism:  string(ports.AuthMechanismKerberosAD),
			wantErr:    ports.ErrAuthenticationFailed,
		},
		{
			name:       "missing subject",
			remoteAddr: "127.0.0.1:12345",
			roles:      string(ports.AuthRoleValidatorL1),
			mechanism:  string(ports.AuthMechanismKerberosAD),
			wantErr:    ports.ErrAuthPrincipalInvalid,
		},
		{
			name:       "missing roles",
			remoteAddr: "127.0.0.1:12345",
			subject:    "staff",
			mechanism:  string(ports.AuthMechanismKerberosAD),
			wantErr:    ports.ErrAuthRoleInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/api/portal", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set(config.DefaultTrustedHeaderSubject, tt.subject)
			req.Header.Set(config.DefaultTrustedHeaderRoles, tt.roles)
			req.Header.Set(config.DefaultTrustedHeaderMechanism, tt.mechanism)

			_, err := authenticator.AuthenticateRequest(context.Background(), req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("AuthenticateRequest() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func newTestTrustedHeadersAuthenticator(t *testing.T) *TrustedHeadersAuthenticator {
	t.Helper()

	authenticator, err := NewTrustedHeadersAuthenticator(config.Config{})
	if err != nil {
		t.Fatalf("NewTrustedHeadersAuthenticator() error = %v", err)
	}
	return authenticator
}
