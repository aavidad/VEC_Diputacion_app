package httpapi

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/url"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
)

type requestIdentity struct {
	subject     string
	displayName string
	email       string
	method      domain.AuthMethod
	assurance   domain.AuthAssurance
	roles       []string
	attributes  map[string]string
}

func identityFromRequest(r *http.Request) requestIdentity {
	identity := requestIdentity{
		subject:    firstHeader(r, "X-Auth-Subject"),
		method:     authMethod(strings.TrimSpace(r.Header.Get("X-Auth-Mechanism"))),
		assurance:  assuranceFromHeader(r),
		roles:      rolesFromHeader(r),
		attributes: map[string]string{},
	}
	if identity.subject == "" {
		identity.subject = firstHeader(r, "X-Remote-User", "Remote-User")
		if identity.subject != "" && identity.method == domain.AuthMethodDemo {
			identity.method = domain.AuthMethodSSO
		}
	}
	if identity.displayName = firstHeader(r, "X-Auth-Display-Name", "X-Auth-Name"); identity.displayName == "" {
		identity.displayName = identity.subject
	}
	identity.email = firstHeader(r, "X-Auth-Email", "X-Forwarded-Email")
	if dni := firstHeader(r, "X-Auth-DNI", "X-Auth-NIF"); dni != "" {
		identity.attributes["dni"] = dni
		if identity.subject == "" {
			identity.subject = dni
		}
	}

	if cert, source := certificateFromRequest(r); cert != nil {
		applyCertificateIdentity(&identity, cert, source)
	}
	if identity.subject == "" {
		identity.subject = "staff"
		identity.displayName = "staff"
		identity.method = domain.AuthMethodDemo
		identity.assurance = domain.AuthAssuranceHigh
	}
	if identity.displayName == "" {
		identity.displayName = identity.subject
	}
	if len(identity.roles) == 0 {
		identity.roles = defaultRolesForIdentity(identity)
	}
	return identity
}

func assuranceFromHeader(r *http.Request) domain.AuthAssurance {
	switch strings.TrimSpace(strings.ToLower(r.Header.Get("X-Auth-Assurance"))) {
	case string(domain.AuthAssuranceLow):
		return domain.AuthAssuranceLow
	case string(domain.AuthAssuranceSubstantial):
		return domain.AuthAssuranceSubstantial
	case string(domain.AuthAssuranceHigh):
		return domain.AuthAssuranceHigh
	default:
		return domain.AuthAssuranceSubstantial
	}
}

func applyCertificateIdentity(identity *requestIdentity, cert *x509.Certificate, source string) {
	if identity == nil || cert == nil {
		return
	}
	if identity.method == domain.AuthMethodDemo {
		identity.method = domain.AuthMethodCertificate
		if looksLikeDNIe(cert) {
			identity.method = domain.AuthMethodDNIe
		}
	}
	identity.assurance = domain.AuthAssuranceHigh
	dni := firstNonEmpty(
		identity.attributes["dni"],
		strings.TrimSpace(cert.Subject.SerialNumber),
		firstNonEmpty(cert.Subject.CommonName, certificateSerial(cert)),
	)
	if headerSubject := strings.TrimSpace(identity.subject); headerSubject != "" {
		identity.attributes["external_subject"] = headerSubject
	}
	identity.subject = dni
	identity.displayName = firstNonEmpty(identity.displayName, cert.Subject.CommonName, identity.subject)
	if identity.email == "" && len(cert.EmailAddresses) > 0 {
		identity.email = cert.EmailAddresses[0]
	}
	identity.attributes["auth_source"] = source
	identity.attributes["certificate_subject"] = cert.Subject.String()
	identity.attributes["certificate_issuer"] = cert.Issuer.String()
	identity.attributes["certificate_serial"] = certificateSerial(cert)
	identity.attributes["certificate_not_after"] = cert.NotAfter.Format("2006-01-02T15:04:05Z07:00")
	if strings.TrimSpace(cert.Subject.SerialNumber) != "" {
		identity.attributes["dni"] = strings.TrimSpace(cert.Subject.SerialNumber)
	}
}

func certificateSerial(cert *x509.Certificate) string {
	if cert == nil || cert.SerialNumber == nil {
		return ""
	}
	return cert.SerialNumber.String()
}

func certificateFromRequest(r *http.Request) (*x509.Certificate, string) {
	if r != nil && r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		return r.TLS.PeerCertificates[0], "tls_peer_certificate"
	}
	verify := strings.ToUpper(strings.TrimSpace(r.Header.Get("X-SSL-Client-Verify")))
	raw := firstHeader(r, "X-SSL-Client-Cert", "X-Client-Cert", "X-Forwarded-Tls-Client-Cert")
	if raw == "" || (verify != "" && verify != "SUCCESS") {
		return nil, ""
	}
	cert, ok := parseCertificateHeader(raw)
	if !ok {
		return nil, ""
	}
	return cert, "trusted_proxy_header"
}

func parseCertificateHeader(raw string) (*x509.Certificate, bool) {
	raw = strings.TrimSpace(raw)
	if decoded, err := url.QueryUnescape(raw); err == nil {
		raw = decoded
	}
	raw = strings.ReplaceAll(raw, `\n`, "\n")
	raw = strings.ReplaceAll(raw, "\t", "")
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, false
	}
	return cert, true
}

func looksLikeDNIe(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}
	haystack := strings.ToLower(cert.Issuer.String() + " " + cert.Subject.String())
	return strings.Contains(haystack, "dnie") ||
		strings.Contains(haystack, "dni") ||
		strings.Contains(haystack, "direccion general de la policia")
}

func rolesFromHeader(r *http.Request) []string {
	raw := firstHeader(r, "X-Auth-Roles", "X-Auth-Role", "X-Vec-Roles")
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	roles := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, field := range fields {
		role := strings.TrimSpace(strings.ToLower(field))
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	return roles
}

func defaultRolesForIdentity(identity requestIdentity) []string {
	switch {
	case identity.subject == "candidate":
		return []string{"ciudadano"}
	case identity.subject == "staff" || identity.method == domain.AuthMethodDemo:
		return []string{"administrador", "tecnico_rrhh"}
	default:
		return []string{"personal_interno"}
	}
}

func firstHeader(r *http.Request, keys ...string) string {
	if r == nil {
		return ""
	}
	for _, key := range keys {
		if value := strings.TrimSpace(r.Header.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
