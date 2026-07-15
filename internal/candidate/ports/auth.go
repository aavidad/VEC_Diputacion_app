package ports

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrAuthMechanismRequired    = errors.New("auth mechanism is required")
	ErrAuthMechanismUnsupported = errors.New("auth mechanism is unsupported")
	ErrAuthSubjectRequired      = errors.New("auth subject is required")
	ErrAuthSubjectInvalid       = errors.New("auth subject is invalid")
	ErrAuthTokenRequired        = errors.New("auth token is required")
	ErrAuthTokenInvalid         = errors.New("auth token is invalid")
	ErrAuthRoleInvalid          = errors.New("auth role is invalid")
	ErrAuthPrincipalInvalid     = errors.New("auth principal is invalid")
	ErrAuthenticationFailed     = errors.New("authentication failed")
)

type AuthMechanism string

const (
	AuthMechanismKerberosAD AuthMechanism = "kerberos_ad"
	AuthMechanismDNIe       AuthMechanism = "dnie"
	AuthMechanismClave      AuthMechanism = "clave"
)

func (m AuthMechanism) IsValid() bool {
	switch m {
	case AuthMechanismKerberosAD, AuthMechanismDNIe, AuthMechanismClave:
		return true
	default:
		return false
	}
}

type AuthRole string

const (
	AuthRoleCandidate   AuthRole = "candidate"
	AuthRoleValidatorL1 AuthRole = "validator_l1"
	AuthRoleValidatorL2 AuthRole = "validator_l2"
	AuthRoleSystemAdmin AuthRole = "system_admin"

	AuthRoleCiudadano       AuthRole = AuthRoleCandidate
	AuthRolePersonalInterno AuthRole = AuthRoleValidatorL1
)

func (r AuthRole) IsValid() bool {
	switch r {
	case AuthRoleCandidate, AuthRoleValidatorL1, AuthRoleValidatorL2, AuthRoleSystemAdmin:
		return true
	default:
		return false
	}
}

type AuthCredentials struct {
	Mechanism  AuthMechanism
	Subject    string
	Token      string
	Assertions map[string]string
}

func (c AuthCredentials) Validate() error {
	switch {
	case c.Mechanism == "":
		return ErrAuthMechanismRequired
	case !c.Mechanism.IsValid():
		return ErrAuthMechanismUnsupported
	case c.Subject == "":
		return ErrAuthSubjectRequired
	case !canonicalAuthSubject(c.Subject):
		return ErrAuthSubjectInvalid
	case c.Token == "":
		return ErrAuthTokenRequired
	case !canonicalAuthToken(c.Token):
		return ErrAuthTokenInvalid
	default:
		return nil
	}
}

type AuthPrincipal struct {
	Subject     string
	DisplayName string
	Email       string
	Role        AuthRole
	Roles       []AuthRole
	Mechanism   AuthMechanism
	Method      AuthMechanism
	Attributes  map[string]string
}

type Identity = AuthPrincipal

func (p AuthPrincipal) Validate() error {
	switch {
	case !canonicalAuthSubject(p.Subject):
		return ErrAuthPrincipalInvalid
	case !p.PrimaryRole().IsValid():
		return ErrAuthRoleInvalid
	case p.Mechanism == "" && p.Method == "":
		return ErrAuthMechanismRequired
	case !p.AuthMethod().IsValid():
		return ErrAuthMechanismUnsupported
	default:
		return nil
	}
}

func (p AuthPrincipal) PrimaryRole() AuthRole {
	var resolved AuthRole
	present := false
	add := func(role AuthRole) bool {
		if !role.IsValid() {
			return false
		}
		if !present {
			resolved = role
			present = true
			return true
		}
		return resolved == role
	}
	if p.Role != "" && !add(p.Role) {
		return ""
	}
	for _, role := range p.Roles {
		if !add(role) {
			return ""
		}
	}
	if !present {
		return ""
	}
	return resolved
}

func (p AuthPrincipal) AuthMethod() AuthMechanism {
	var resolved AuthMechanism
	present := false
	for _, mechanism := range []AuthMechanism{p.Mechanism, p.Method} {
		if mechanism == "" {
			continue
		}
		if !mechanism.IsValid() {
			return ""
		}
		if !present {
			resolved = mechanism
			present = true
			continue
		}
		if resolved != mechanism {
			return ""
		}
	}
	return resolved
}

func (p AuthPrincipal) AllRoles() []AuthRole {
	role := p.PrimaryRole()
	if !role.IsValid() {
		return nil
	}
	return []AuthRole{role}
}

func (p AuthPrincipal) HasRole(role AuthRole) bool {
	return role.IsValid() && p.PrimaryRole() == role
}

func canonicalAuthSubject(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || len(value) > 512 ||
		!utf8.ValidString(value) || strings.ContainsAny(value, "*,;") {
		return false
	}
	return !strings.ContainsFunc(value, func(character rune) bool {
		return unicode.IsControl(character) || unicode.IsSpace(character) || unicode.Is(unicode.Cf, character)
	})
}

func canonicalAuthToken(value string) bool {
	if value == "" || len(value) > 8192 || strings.ContainsAny(value, ",;") {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x21 || value[index] > 0x7e {
			return false
		}
	}
	return true
}

type Authenticator interface {
	Authenticate(context.Context, AuthCredentials) (Identity, error)
}
