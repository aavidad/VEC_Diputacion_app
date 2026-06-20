package auth

import (
	"context"
	"net"
	"net/http"
	"strings"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/candidate/ports"
)

type TrustedHeadersAuthenticator struct {
	subjectHeader   string
	rolesHeader     string
	mechanismHeader string
	trustedProxies  []*net.IPNet
}

func NewTrustedHeadersAuthenticator(cfg config.Config) (*TrustedHeadersAuthenticator, error) {
	cfg = cfg.Normalize()
	proxies, err := parseTrustedProxyCIDRs(cfg.TrustedProxyCIDRs)
	if err != nil {
		return nil, err
	}
	return &TrustedHeadersAuthenticator{
		subjectHeader:   cfg.TrustedHeaderSubject,
		rolesHeader:     cfg.TrustedHeaderRoles,
		mechanismHeader: cfg.TrustedHeaderMechanism,
		trustedProxies:  proxies,
	}, nil
}

func (a *TrustedHeadersAuthenticator) AuthenticateRequest(
	ctx context.Context,
	r *http.Request,
) (ports.AuthPrincipal, error) {
	if a == nil || r == nil {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	if err := ctx.Err(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	if !a.isTrustedRemote(r.RemoteAddr) {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	mechanism := ports.AuthMechanism(strings.TrimSpace(r.Header.Get(a.mechanismHeader)))
	if strings.TrimSpace(string(mechanism)) == "" {
		return ports.AuthPrincipal{}, ports.ErrAuthMechanismRequired
	}
	roles := parseRoles(r.Header.Values(a.rolesHeader))
	if len(roles) == 0 {
		return ports.AuthPrincipal{}, ports.ErrAuthRoleInvalid
	}
	principal := ports.AuthPrincipal{
		Subject:   strings.TrimSpace(r.Header.Get(a.subjectHeader)),
		Role:      roles[0],
		Roles:     roles,
		Mechanism: mechanism,
		Method:    mechanism,
		Attributes: map[string]string{
			"auth.mode":         config.AuthModeTrustedHeaders,
			"trusted.subject":   a.subjectHeader,
			"trusted.roles":     a.rolesHeader,
			"trusted.mechanism": a.mechanismHeader,
		},
	}
	return normalizePrincipal(principal)
}

func (a *TrustedHeadersAuthenticator) Authenticate(
	ctx context.Context,
	credentials ports.AuthCredentials,
) (ports.AuthPrincipal, error) {
	if a == nil {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	if err := ctx.Err(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	if strings.TrimSpace(string(credentials.Mechanism)) == "" {
		return ports.AuthPrincipal{}, ports.ErrAuthMechanismRequired
	}
	roles := parseRoles([]string{credentials.Assertions["roles"]})
	if len(roles) == 0 {
		return ports.AuthPrincipal{}, ports.ErrAuthRoleInvalid
	}
	return normalizePrincipal(ports.AuthPrincipal{
		Subject:   strings.TrimSpace(credentials.Subject),
		Role:      roles[0],
		Roles:     roles,
		Mechanism: credentials.Mechanism,
		Method:    credentials.Mechanism,
	})
}

func (a *TrustedHeadersAuthenticator) isTrustedRemote(remoteAddr string) bool {
	if len(a.trustedProxies) == 0 {
		return false
	}
	host := strings.TrimSpace(remoteAddr)
	if splitHost, _, err := net.SplitHostPort(host); err == nil {
		host = splitHost
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, network := range a.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseTrustedProxyCIDRs(values []string) ([]*net.IPNet, error) {
	proxies := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, network)
	}
	return proxies, nil
}

func parseRoles(values []string) []ports.AuthRole {
	roles := make([]ports.AuthRole, 0, len(values))
	seen := make(map[ports.AuthRole]struct{})
	for _, value := range values {
		for _, part := range strings.FieldsFunc(value, roleSeparator) {
			role := ports.AuthRole(strings.TrimSpace(part))
			if !role.IsValid() {
				continue
			}
			if _, ok := seen[role]; ok {
				continue
			}
			seen[role] = struct{}{}
			roles = append(roles, role)
		}
	}
	return roles
}

func roleSeparator(r rune) bool {
	return r == ',' || r == ';' || r == ' '
}
