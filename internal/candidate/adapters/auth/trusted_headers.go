package auth

import (
	"context"
	"net"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

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
	if a == nil || r == nil || ctx == nil {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	if err := ctx.Err(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	if !a.isTrustedRemote(r.RemoteAddr) {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	mechanismValue, ok := singleCanonicalHeaderValue(r.Header.Values(a.mechanismHeader))
	if !ok {
		return ports.AuthPrincipal{}, ports.ErrAuthMechanismRequired
	}
	mechanism := ports.AuthMechanism(mechanismValue)
	role, ok := parseSingleRole(r.Header.Values(a.rolesHeader))
	// Este adaptador heredado no dispone del PDP por recurso necesario para
	// combinar perfiles. Hasta que se migre al nucleo RBAC+ABAC solo acepta un
	// perfil canonico exacto; elegir uno por orden ampliaria autoridad.
	if !ok {
		return ports.AuthPrincipal{}, ports.ErrAuthRoleInvalid
	}
	subject, ok := singleCanonicalHeaderValue(r.Header.Values(a.subjectHeader))
	if !ok || !canonicalSubject(subject) {
		return ports.AuthPrincipal{}, ports.ErrAuthPrincipalInvalid
	}
	principal := ports.AuthPrincipal{
		Subject:   subject,
		Role:      role,
		Roles:     []ports.AuthRole{role},
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
	_ ports.AuthCredentials,
) (ports.AuthPrincipal, error) {
	if a == nil || ctx == nil {
		return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
	}
	if err := ctx.Err(); err != nil {
		return ports.AuthPrincipal{}, err
	}
	// Sin la peticion no se puede probar que las aserciones procedan de un
	// proxy permitido. Aceptarlas desde AuthCredentials convertiria un mapa
	// controlado por el llamador en una fuente de autoridad.
	return ports.AuthPrincipal{}, ports.ErrAuthenticationFailed
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

func parseSingleRole(values []string) (ports.AuthRole, bool) {
	value, ok := singleCanonicalHeaderValue(values)
	if !ok || strings.ContainsAny(value, ",;") || strings.ContainsFunc(value, unicode.IsSpace) {
		return "", false
	}
	role := ports.AuthRole(value)
	return role, role.IsValid()
}

func singleCanonicalHeaderValue(values []string) (string, bool) {
	if len(values) != 1 || values[0] == "" || values[0] != strings.TrimSpace(values[0]) ||
		strings.ContainsFunc(values[0], unicode.IsControl) {
		return "", false
	}
	return values[0], true
}

func canonicalSubject(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || len(value) > 512 ||
		!utf8.ValidString(value) || strings.ContainsAny(value, "*,;") {
		return false
	}
	return !strings.ContainsFunc(value, func(character rune) bool {
		return unicode.IsControl(character) || unicode.IsSpace(character) || unicode.Is(unicode.Cf, character)
	})
}
