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
	req.Header.Set(config.DefaultTrustedHeaderRoles, "candidate")
	req.Header.Set(config.DefaultTrustedHeaderMechanism, string(ports.AuthMechanismClave))

	principal, err := authenticator.AuthenticateRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("AuthenticateRequest() error = %v", err)
	}
	if principal.Subject != "candidate-1" || principal.Role != ports.AuthRoleCandidate {
		t.Fatalf("principal = %+v", principal)
	}
	if len(principal.AllRoles()) != 1 {
		t.Fatalf("principal roles = %#v", principal.Roles)
	}
}

func TestTrustedHeadersAuthenticatorRejectsAmbiguousRoleSet(t *testing.T) {
	t.Parallel()

	authenticator := newTestTrustedHeadersAuthenticator(t)
	for _, roles := range []string{
		"candidate,validator_l1",
		"candidate,candidate",
		"candidate,unknown",
		"candidate unknown",
		" candidate",
		"candidate ",
	} {
		req := httptest.NewRequest("GET", "/api/portal", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set(config.DefaultTrustedHeaderSubject, "candidate-1")
		req.Header.Set(config.DefaultTrustedHeaderRoles, roles)
		req.Header.Set(config.DefaultTrustedHeaderMechanism, string(ports.AuthMechanismClave))

		_, err := authenticator.AuthenticateRequest(context.Background(), req)
		if !errors.Is(err, ports.ErrAuthRoleInvalid) {
			t.Fatalf("roles %q: AuthenticateRequest() error = %v, want %v", roles, err, ports.ErrAuthRoleInvalid)
		}
	}
}

func TestTrustedHeadersAuthenticatorRejectsUnpublishedRoleAliases(t *testing.T) {
	t.Parallel()

	authenticator := newTestTrustedHeadersAuthenticator(t)
	for _, role := range []string{"ciudadano", "personal_interno", "administrador", "admin", "Candidate"} {
		req := httptest.NewRequest("GET", "/api/portal", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set(config.DefaultTrustedHeaderSubject, "identity-1")
		req.Header.Set(config.DefaultTrustedHeaderRoles, role)
		req.Header.Set(config.DefaultTrustedHeaderMechanism, string(ports.AuthMechanismClave))

		if _, err := authenticator.AuthenticateRequest(context.Background(), req); !errors.Is(err, ports.ErrAuthRoleInvalid) {
			t.Fatalf("role alias %q: error = %v, want %v", role, err, ports.ErrAuthRoleInvalid)
		}
	}
}

func TestTrustedHeadersAuthenticatorRejectsInvalidOrAmbiguousMechanism(t *testing.T) {
	t.Parallel()

	authenticator := newTestTrustedHeadersAuthenticator(t)
	for _, mechanisms := range [][]string{
		{"password"},
		{"kerberos"},
		{"ad"},
		{"certificado"},
		{" clave"},
		{string(ports.AuthMechanismClave), string(ports.AuthMechanismDNIe)},
		{string(ports.AuthMechanismClave), string(ports.AuthMechanismClave)},
	} {
		req := httptest.NewRequest("GET", "/api/portal", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set(config.DefaultTrustedHeaderSubject, "candidate-1")
		req.Header.Set(config.DefaultTrustedHeaderRoles, string(ports.AuthRoleCandidate))
		req.Header.Del(config.DefaultTrustedHeaderMechanism)
		for _, mechanism := range mechanisms {
			req.Header.Add(config.DefaultTrustedHeaderMechanism, mechanism)
		}

		if _, err := authenticator.AuthenticateRequest(context.Background(), req); err == nil {
			t.Fatalf("mechanisms %#v were accepted", mechanisms)
		}
	}
}

func TestTrustedHeadersAuthenticatorRejectsRepeatedAuthorityHeaders(t *testing.T) {
	t.Parallel()

	authenticator := newTestTrustedHeadersAuthenticator(t)
	for _, header := range []string{
		config.DefaultTrustedHeaderSubject,
		config.DefaultTrustedHeaderRoles,
		config.DefaultTrustedHeaderMechanism,
	} {
		req := httptest.NewRequest("GET", "/api/portal", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set(config.DefaultTrustedHeaderSubject, "candidate-1")
		req.Header.Set(config.DefaultTrustedHeaderRoles, string(ports.AuthRoleCandidate))
		req.Header.Set(config.DefaultTrustedHeaderMechanism, string(ports.AuthMechanismClave))
		req.Header.Add(header, req.Header.Get(header))

		if _, err := authenticator.AuthenticateRequest(context.Background(), req); err == nil {
			t.Fatalf("la cabecera autoritativa repetida %s fue aceptada", header)
		}
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
		{
			name:       "non canonical subject",
			remoteAddr: "127.0.0.1:12345",
			subject:    " candidate-1",
			roles:      string(ports.AuthRoleCandidate),
			mechanism:  string(ports.AuthMechanismClave),
			wantErr:    ports.ErrAuthPrincipalInvalid,
		},
		{
			name:       "ambiguous combined subject",
			remoteAddr: "127.0.0.1:12345",
			subject:    "candidate-1,candidate-2",
			roles:      string(ports.AuthRoleCandidate),
			mechanism:  string(ports.AuthMechanismClave),
			wantErr:    ports.ErrAuthPrincipalInvalid,
		},
		{
			name:       "invisible subject format",
			remoteAddr: "127.0.0.1:12345",
			subject:    "candidate\u200b-1",
			roles:      string(ports.AuthRoleCandidate),
			mechanism:  string(ports.AuthMechanismClave),
			wantErr:    ports.ErrAuthPrincipalInvalid,
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

func TestTrustedHeadersAuthenticatorNeverTrustsDetachedAssertions(t *testing.T) {
	t.Parallel()

	authenticator := newTestTrustedHeadersAuthenticator(t)
	_, err := authenticator.Authenticate(context.Background(), ports.AuthCredentials{
		Mechanism: ports.AuthMechanismClave,
		Subject:   "candidate-1",
		Token:     "token",
		Assertions: map[string]string{
			"roles": string(ports.AuthRoleSystemAdmin),
		},
	})
	if !errors.Is(err, ports.ErrAuthenticationFailed) {
		t.Fatalf("Authenticate() error = %v, want %v", err, ports.ErrAuthenticationFailed)
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
