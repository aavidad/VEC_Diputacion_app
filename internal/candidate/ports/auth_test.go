package ports

import (
	"errors"
	"strings"
	"testing"
)

func TestAuthCredentialsRequireCanonicalExactValues(t *testing.T) {
	t.Parallel()

	valid := AuthCredentials{
		Mechanism: AuthMechanismClave,
		Subject:   "candidate-1",
		Token:     "header.payload.signature",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid credentials: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*AuthCredentials)
		wantErr error
	}{
		{name: "missing mechanism", mutate: func(value *AuthCredentials) { value.Mechanism = "" }, wantErr: ErrAuthMechanismRequired},
		{name: "unsupported mechanism", mutate: func(value *AuthCredentials) { value.Mechanism = "password" }, wantErr: ErrAuthMechanismUnsupported},
		{name: "non canonical mechanism", mutate: func(value *AuthCredentials) { value.Mechanism = " clave" }, wantErr: ErrAuthMechanismUnsupported},
		{name: "missing subject", mutate: func(value *AuthCredentials) { value.Subject = "" }, wantErr: ErrAuthSubjectRequired},
		{name: "leading subject whitespace", mutate: func(value *AuthCredentials) { value.Subject = " candidate-1" }, wantErr: ErrAuthSubjectInvalid},
		{name: "embedded subject whitespace", mutate: func(value *AuthCredentials) { value.Subject = "candidate 1" }, wantErr: ErrAuthSubjectInvalid},
		{name: "combined subjects", mutate: func(value *AuthCredentials) { value.Subject = "candidate-1,candidate-2" }, wantErr: ErrAuthSubjectInvalid},
		{name: "wildcard subject", mutate: func(value *AuthCredentials) { value.Subject = "*" }, wantErr: ErrAuthSubjectInvalid},
		{name: "invisible subject format", mutate: func(value *AuthCredentials) { value.Subject = "candidate\u200b-1" }, wantErr: ErrAuthSubjectInvalid},
		{name: "missing token", mutate: func(value *AuthCredentials) { value.Token = "" }, wantErr: ErrAuthTokenRequired},
		{name: "leading token whitespace", mutate: func(value *AuthCredentials) { value.Token = " token" }, wantErr: ErrAuthTokenInvalid},
		{name: "embedded token whitespace", mutate: func(value *AuthCredentials) { value.Token = "to ken" }, wantErr: ErrAuthTokenInvalid},
		{name: "combined tokens", mutate: func(value *AuthCredentials) { value.Token = "token-1,token-2" }, wantErr: ErrAuthTokenInvalid},
		{name: "non ascii token", mutate: func(value *AuthCredentials) { value.Token = "tóken" }, wantErr: ErrAuthTokenInvalid},
		{name: "control in token", mutate: func(value *AuthCredentials) { value.Token = "token\n" }, wantErr: ErrAuthTokenInvalid},
		{name: "oversized token", mutate: func(value *AuthCredentials) { value.Token = strings.Repeat("a", 8193) }, wantErr: ErrAuthTokenInvalid},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			credentials := valid
			tt.mutate(&credentials)
			if err := credentials.Validate(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthPrincipalNeverChoosesOneRoleFromAmbiguousRepresentations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		role      AuthRole
		roles     []AuthRole
		want      AuthRole
		wantValid bool
	}{
		{name: "single legacy field", role: AuthRoleCandidate, want: AuthRoleCandidate, wantValid: true},
		{name: "single list field", roles: []AuthRole{AuthRoleValidatorL1}, want: AuthRoleValidatorL1, wantValid: true},
		{name: "exact duplicate", role: AuthRoleValidatorL2, roles: []AuthRole{AuthRoleValidatorL2, AuthRoleValidatorL2}, want: AuthRoleValidatorL2, wantValid: true},
		{name: "absent"},
		{name: "invalid only", role: "unknown"},
		{name: "valid accompanied by invalid", role: AuthRoleCandidate, roles: []AuthRole{"unknown"}},
		{name: "invalid accompanied by valid", role: "unknown", roles: []AuthRole{AuthRoleCandidate}},
		{name: "two distinct roles", role: AuthRoleCandidate, roles: []AuthRole{AuthRoleValidatorL1}},
		{name: "two distinct list roles", roles: []AuthRole{AuthRoleCandidate, AuthRoleValidatorL1}},
		{name: "empty list representation", role: AuthRoleCandidate, roles: []AuthRole{""}},
		{name: "non canonical role", roles: []AuthRole{"candidate "}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			principal := AuthPrincipal{
				Subject:   "identity-1",
				Role:      tt.role,
				Roles:     tt.roles,
				Mechanism: AuthMechanismClave,
			}
			if got := principal.PrimaryRole(); got != tt.want {
				t.Fatalf("PrimaryRole() = %q, want %q", got, tt.want)
			}
			if tt.wantValid {
				if err := principal.Validate(); err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				if roles := principal.AllRoles(); len(roles) != 1 || roles[0] != tt.want {
					t.Fatalf("AllRoles() = %#v, want [%q]", roles, tt.want)
				}
				if !principal.HasRole(tt.want) {
					t.Fatalf("HasRole(%q) = false", tt.want)
				}
				return
			}
			if err := principal.Validate(); !errors.Is(err, ErrAuthRoleInvalid) {
				t.Fatalf("Validate() error = %v, want %v", err, ErrAuthRoleInvalid)
			}
			if roles := principal.AllRoles(); len(roles) != 0 {
				t.Fatalf("AllRoles() rescued authority: %#v", roles)
			}
			for _, role := range []AuthRole{AuthRoleCandidate, AuthRoleValidatorL1, AuthRoleValidatorL2, AuthRoleSystemAdmin} {
				if principal.HasRole(role) {
					t.Fatalf("HasRole(%q) rescued authority", role)
				}
			}
		})
	}
}

func TestAuthPrincipalNeverChoosesOneAuthenticationMechanism(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mechanism AuthMechanism
		method    AuthMechanism
		want      AuthMechanism
		wantErr   error
	}{
		{name: "single mechanism", mechanism: AuthMechanismClave, want: AuthMechanismClave},
		{name: "single method", method: AuthMechanismDNIe, want: AuthMechanismDNIe},
		{name: "exact duplicate", mechanism: AuthMechanismKerberosAD, method: AuthMechanismKerberosAD, want: AuthMechanismKerberosAD},
		{name: "absent", wantErr: ErrAuthMechanismRequired},
		{name: "invalid", mechanism: "password", wantErr: ErrAuthMechanismUnsupported},
		{name: "valid accompanied by invalid", mechanism: AuthMechanismClave, method: "password", wantErr: ErrAuthMechanismUnsupported},
		{name: "invalid accompanied by valid", mechanism: "password", method: AuthMechanismClave, wantErr: ErrAuthMechanismUnsupported},
		{name: "two distinct mechanisms", mechanism: AuthMechanismClave, method: AuthMechanismDNIe, wantErr: ErrAuthMechanismUnsupported},
		{name: "non canonical mechanism", method: "clave ", wantErr: ErrAuthMechanismUnsupported},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			principal := AuthPrincipal{
				Subject:   "identity-1",
				Role:      AuthRoleCandidate,
				Mechanism: tt.mechanism,
				Method:    tt.method,
			}
			if got := principal.AuthMethod(); got != tt.want {
				t.Fatalf("AuthMethod() = %q, want %q", got, tt.want)
			}
			if tt.wantErr == nil {
				if err := principal.Validate(); err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err := principal.Validate(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthPrincipalRejectsNonCanonicalSubject(t *testing.T) {
	t.Parallel()

	for _, subject := range []string{"", " identity-1", "identity 1", "identity-1,identity-2", "identity\u200b-1"} {
		principal := AuthPrincipal{Subject: subject, Role: AuthRoleCandidate, Mechanism: AuthMechanismClave}
		if err := principal.Validate(); !errors.Is(err, ErrAuthPrincipalInvalid) {
			t.Fatalf("subject %q: Validate() error = %v, want %v", subject, err, ErrAuthPrincipalInvalid)
		}
	}
}
