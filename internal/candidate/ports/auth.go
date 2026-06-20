package ports

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrAuthMechanismRequired    = errors.New("auth mechanism is required")
	ErrAuthMechanismUnsupported = errors.New("auth mechanism is unsupported")
	ErrAuthSubjectRequired      = errors.New("auth subject is required")
	ErrAuthTokenRequired        = errors.New("auth token is required")
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
	case !c.Mechanism.IsValid():
		if strings.TrimSpace(string(c.Mechanism)) == "" {
			return ErrAuthMechanismRequired
		}
		return ErrAuthMechanismUnsupported
	case strings.TrimSpace(c.Subject) == "":
		return ErrAuthSubjectRequired
	case strings.TrimSpace(c.Token) == "":
		return ErrAuthTokenRequired
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
	case strings.TrimSpace(p.Subject) == "":
		return ErrAuthPrincipalInvalid
	case !p.PrimaryRole().IsValid():
		return ErrAuthRoleInvalid
	case !p.AuthMethod().IsValid():
		return ErrAuthMechanismUnsupported
	default:
		return nil
	}
}

func (p AuthPrincipal) PrimaryRole() AuthRole {
	if p.Role.IsValid() {
		return p.Role
	}
	for _, role := range p.Roles {
		if role.IsValid() {
			return role
		}
	}
	return ""
}

func (p AuthPrincipal) AuthMethod() AuthMechanism {
	if p.Method.IsValid() {
		return p.Method
	}
	return p.Mechanism
}

func (p AuthPrincipal) AllRoles() []AuthRole {
	roles := make([]AuthRole, 0, len(p.Roles)+1)
	seen := make(map[AuthRole]struct{}, len(p.Roles)+1)
	add := func(role AuthRole) {
		if !role.IsValid() {
			return
		}
		if _, ok := seen[role]; ok {
			return
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	add(p.Role)
	for _, role := range p.Roles {
		add(role)
	}
	return roles
}

func (p AuthPrincipal) HasRole(role AuthRole) bool {
	for _, current := range p.AllRoles() {
		if current == role {
			return true
		}
	}
	return false
}

type Authenticator interface {
	Authenticate(context.Context, AuthCredentials) (Identity, error)
}
