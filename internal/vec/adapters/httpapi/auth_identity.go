package httpapi

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
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

type identityPolicy struct {
	allowDemo       bool
	demoResolver    DemoIdentityResolver
	trustHeaders    bool
	trustedProxies  []*net.IPNet
	subjectHeader   string
	rolesHeader     string
	mechanismHeader string
}

func newIdentityPolicy(options HandlerOptions) (identityPolicy, error) {
	policy := identityPolicy{
		allowDemo:       options.AllowDemoIdentity,
		demoResolver:    options.DemoIdentityResolver,
		trustHeaders:    options.TrustIdentityHeaders,
		subjectHeader:   firstNonEmpty(options.IdentitySubjectHeader, "X-VEC-Subject"),
		rolesHeader:     firstNonEmpty(options.IdentityRolesHeader, "X-VEC-Roles"),
		mechanismHeader: firstNonEmpty(options.IdentityMechanismHeader, "X-VEC-Auth-Mechanism"),
	}
	if policy.allowDemo && policy.demoResolver == nil {
		return identityPolicy{}, fmt.Errorf("vec http identity: fake mode requires an explicit credential resolver")
	}
	for _, raw := range options.TrustedProxyCIDRs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(raw))
		if err != nil {
			return identityPolicy{}, fmt.Errorf("vec http identity: trusted proxy %q: %w", raw, err)
		}
		policy.trustedProxies = append(policy.trustedProxies, network)
	}
	if policy.trustHeaders && len(policy.trustedProxies) == 0 {
		return identityPolicy{}, fmt.Errorf("vec http identity: trusted headers require trusted proxy CIDRs")
	}
	return policy, nil
}

func identityFromRequest(r *http.Request, policy identityPolicy) requestIdentity {
	acceptHeaders := policy.acceptHeaders(r)
	identity := requestIdentity{
		attributes: map[string]string{},
	}
	if acceptHeaders {
		var consistente bool
		identity.subject, consistente = consistentHeader(r, policy.subjectHeader, "X-Auth-Subject")
		if !consistente {
			return requestIdentity{attributes: map[string]string{}}
		}
		mecanismo, consistente := consistentHeader(r, policy.mechanismHeader, "X-Auth-Mechanism")
		if !consistente {
			return requestIdentity{attributes: map[string]string{}}
		}
		identity.method = authMethod(mecanismo)
		identity.assurance = assuranceFromHeader(r)
		identity.roles, consistente = rolesFromHeader(r, policy)
		if !consistente {
			return requestIdentity{attributes: map[string]string{}}
		}
	}
	if acceptHeaders && identity.subject == "" {
		var consistente bool
		identity.subject, consistente = consistentHeader(r, "X-Remote-User", "Remote-User")
		if !consistente {
			return requestIdentity{attributes: map[string]string{}}
		}
		if identity.subject != "" && identity.method == domain.AuthMethodDemo {
			identity.method = domain.AuthMethodSSO
		}
	}
	if acceptHeaders {
		var consistente bool
		identity.displayName, consistente = consistentHeader(
			r, "X-VEC-Display-Name", "X-Auth-Display-Name", "X-Auth-Name",
		)
		if !consistente {
			return requestIdentity{attributes: map[string]string{}}
		}
		identity.email, consistente = consistentHeader(
			r, "X-VEC-Email", "X-Auth-Email", "X-Forwarded-Email",
		)
		if !consistente {
			return requestIdentity{attributes: map[string]string{}}
		}
		dni, consistente := consistentHeader(r, "X-VEC-DNI", "X-Auth-DNI", "X-Auth-NIF")
		if !consistente {
			return requestIdentity{attributes: map[string]string{}}
		}
		if dni != "" {
			identity.attributes["dni"] = dni
		}
	}

	if cert, source := certificateFromRequest(r, acceptHeaders); cert != nil {
		if !applyCertificateIdentity(&identity, cert, source) {
			return requestIdentity{attributes: map[string]string{}}
		}
	}
	if identity.displayName == "" {
		identity.displayName = identity.subject
	}
	return identity
}

func (p identityPolicy) acceptHeaders(r *http.Request) bool {
	if !p.trustHeaders || r == nil {
		return false
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if splitHost, _, err := net.SplitHostPort(host); err == nil {
		host = splitHost
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return false
	}
	for _, network := range p.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func assuranceFromHeader(r *http.Request) domain.AuthAssurance {
	if r == nil {
		return ""
	}
	// Una garantia es una afirmacion autoritativa: no se corrige, aproxima ni
	// canoniza en esta frontera. El emisor debe publicar exactamente el valor
	// acordado; cualquier variante queda sin garantia y, por tanto, sin acceso.
	valor, consistente := consistentHeader(r, "X-Auth-Assurance")
	if !consistente {
		return ""
	}
	switch valor {
	case string(domain.AuthAssuranceLow):
		return domain.AuthAssuranceLow
	case string(domain.AuthAssuranceSubstantial):
		return domain.AuthAssuranceSubstantial
	case string(domain.AuthAssuranceHigh):
		return domain.AuthAssuranceHigh
	default:
		return ""
	}
}

func applyCertificateIdentity(identity *requestIdentity, cert *x509.Certificate, source string) bool {
	if identity == nil || cert == nil {
		return false
	}
	certRef := certificateSubjectRef(cert)
	if certRef == "" || (identity.subject != "" && identity.subject != certRef) {
		return false
	}
	dniCertificado := strings.TrimSpace(cert.Subject.SerialNumber)
	if dniAfirmado := strings.TrimSpace(identity.attributes["dni"]); dniAfirmado != "" &&
		dniCertificado != "" && dniAfirmado != dniCertificado {
		return false
	}
	// Hasta que el servicio de identidad fuerte enlace Kerberos y certificado
	// mediante una atestacion unica, esta frontera trata el certificado como un
	// mecanismo independiente y no suma garantias afirmadas por cabecera.
	identity.method = domain.AuthMethodCertificate
	if looksLikeDNIe(cert) {
		identity.method = domain.AuthMethodDNIe
	}
	identity.assurance = domain.AuthAssuranceHigh
	dni := firstNonEmpty(
		dniCertificado,
		identity.attributes["dni"],
	)
	identity.subject = certRef
	identity.displayName = firstNonEmpty(identity.displayName, cert.Subject.CommonName, identity.subject)
	if identity.email == "" && len(cert.EmailAddresses) > 0 {
		identity.email = cert.EmailAddresses[0]
	}
	identity.attributes["auth_source"] = source
	identity.attributes["certificate_ref"] = certRef
	identity.attributes["certificate_issuer"] = cert.Issuer.CommonName
	identity.attributes["certificate_serial"] = certificateSerial(cert)
	identity.attributes["certificate_not_after"] = cert.NotAfter.Format("2006-01-02T15:04:05Z07:00")
	if dni != "" {
		identity.attributes["dni"] = dni
	}
	return true
}

func certificateSubjectRef(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	material := append([]byte(nil), cert.RawIssuer...)
	if len(material) == 0 {
		material = []byte(cert.Issuer.String())
	}
	material = append(material, 0)
	material = append(material, []byte(certificateSerial(cert))...)
	sum := sha256.Sum256(material)
	return "cert:" + hex.EncodeToString(sum[:])
}

func certificateSerial(cert *x509.Certificate) string {
	if cert == nil || cert.SerialNumber == nil {
		return ""
	}
	return cert.SerialNumber.String()
}

func certificateFromRequest(r *http.Request, _ bool) (*x509.Certificate, string) {
	if r != nil {
		if certificado := certificadoClienteTLSVerificado(r.TLS); certificado != nil {
			return certificado, "tls_peer_certificate_verified"
		}
	}
	// Una CIDR confiable no autentica por si sola la afirmacion SUCCESS ni el
	// PEM reenviado. Hasta disponer de mTLS/firma de asercion con audiencia y
	// antirrepeticion, solo el certificado del handshake TLS verificado cuenta.
	return nil, ""
}

func certificadoClienteTLSVerificado(estado *tls.ConnectionState) *x509.Certificate {
	if estado == nil || !estado.HandshakeComplete ||
		(estado.Version != tls.VersionTLS12 && estado.Version != tls.VersionTLS13) ||
		len(estado.PeerCertificates) == 0 || estado.PeerCertificates[0] == nil ||
		len(estado.VerifiedChains) == 0 || len(estado.VerifiedChains[0]) == 0 ||
		estado.VerifiedChains[0][0] == nil {
		return nil
	}
	par := estado.PeerCertificates[0]
	verificado := estado.VerifiedChains[0][0]
	if len(par.Raw) == 0 || !bytes.Equal(par.Raw, verificado.Raw) || !certificadoClienteAdmitido(verificado) {
		return nil
	}
	return verificado
}

func certificadoClienteAdmitido(certificado *x509.Certificate) bool {
	if certificado == nil {
		return false
	}
	if len(certificado.ExtKeyUsage) == 0 {
		return true
	}
	for _, uso := range certificado.ExtKeyUsage {
		if uso == x509.ExtKeyUsageClientAuth || uso == x509.ExtKeyUsageAny {
			return true
		}
	}
	return false
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

func rolesFromHeader(r *http.Request, policy identityPolicy) ([]string, bool) {
	raw, consistente := consistentHeader(r, policy.rolesHeader, "X-Auth-Roles", "X-Auth-Role")
	if !consistente {
		return nil, false
	}
	if raw == "" {
		return nil, true
	}
	// La gramatica de la asercion es deliberadamente unica: roles canonicos
	// separados por coma, sin espacios, punto y coma, duplicados ni entradas
	// vacias. Normalizar aqui ocultaria ambiguedad antes del autorizador.
	if strings.ContainsAny(raw, "; \t\r\n") {
		return nil, false
	}
	fields := strings.Split(raw, ",")
	roles := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, field := range fields {
		if field == "" || field != strings.ToLower(field) || field != strings.TrimSpace(field) {
			return nil, false
		}
		if _, ok := seen[field]; ok {
			return nil, false
		}
		seen[field] = struct{}{}
		roles = append(roles, field)
	}
	return roles, true
}

// consistentHeader falla cerrado si dos alias de una misma asercion llegan
// con valores diferentes. El orden de las cabeceras nunca decide la identidad.
func consistentHeader(r *http.Request, keys ...string) (string, bool) {
	if r == nil {
		return "", true
	}
	seenKeys := make(map[string]struct{}, len(keys))
	value := ""
	for _, key := range keys {
		canonicalKey := http.CanonicalHeaderKey(strings.TrimSpace(key))
		if canonicalKey == "" {
			continue
		}
		if _, exists := seenKeys[canonicalKey]; exists {
			continue
		}
		seenKeys[canonicalKey] = struct{}{}
		valores := r.Header.Values(canonicalKey)
		if len(valores) > 1 {
			return "", false
		}
		if len(valores) == 0 {
			continue
		}
		candidate := valores[0]
		// Ausencia y cabecera presente sin valor no son equivalentes. Una
		// afirmacion explicitamente vacia es una entrada no canonica y no puede
		// combinarse con otro alias para fabricar una identidad completa.
		if candidate == "" {
			return "", false
		}
		if candidate != strings.TrimSpace(candidate) {
			return "", false
		}
		if value != "" && candidate != value {
			return "", false
		}
		value = candidate
	}
	return value, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
