package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type resolvedorIdentidadDesarrollo struct {
	huellaCertificado [sha256.Size]byte
	principal         vecdomain.Principal
}

func (r *resolvedorIdentidadDesarrollo) ResolveDemoIdentity(
	ctx context.Context,
	peticion *http.Request,
) (vecdomain.Principal, error) {
	if r == nil || peticion == nil || peticion.TLS == nil || ctx.Err() != nil ||
		!peticion.TLS.HandshakeComplete || peticion.TLS.Version != tls.VersionTLS13 ||
		len(peticion.TLS.PeerCertificates) != 1 || len(peticion.TLS.VerifiedChains) != 1 ||
		len(peticion.TLS.VerifiedChains[0]) < 2 || !direccionRemotaLoopback(peticion.RemoteAddr) ||
		cabeceraIdentidadAmbientalPresente(peticion.Header) {
		return vecdomain.Principal{}, ErrMaterialDesarrolloInvalido
	}
	par := peticion.TLS.PeerCertificates[0]
	verificado := peticion.TLS.VerifiedChains[0][0]
	if par == nil || verificado == nil || !bytes.Equal(par.Raw, verificado.Raw) {
		return vecdomain.Principal{}, ErrMaterialDesarrolloInvalido
	}
	huella := sha256.Sum256(verificado.Raw)
	if subtle.ConstantTimeCompare(huella[:], r.huellaCertificado[:]) != 1 {
		return vecdomain.Principal{}, ErrMaterialDesarrolloInvalido
	}
	return clonarPrincipalDesarrollo(r.principal), nil
}

func clonarPrincipalDesarrollo(principal vecdomain.Principal) vecdomain.Principal {
	clon := principal
	clon.Roles = append([]string(nil), principal.Roles...)
	clon.Permissions = append([]string(nil), principal.Permissions...)
	clon.Attributes = make(map[string]string, len(principal.Attributes))
	for clave, valor := range principal.Attributes {
		clon.Attributes[clave] = valor
	}
	return clon
}

func direccionRemotaLoopback(direccion string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(direccion))
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return err == nil && ip != nil && ip.IsLoopback()
}

func cabeceraIdentidadAmbientalPresente(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		minusculas := strings.ToLower(strings.TrimSpace(nombre))
		if strings.HasPrefix(minusculas, "x-vec-") || strings.HasPrefix(minusculas, "x-auth-") ||
			minusculas == "x-remote-user" || minusculas == "remote-user" || minusculas == "forwarded" ||
			strings.HasPrefix(minusculas, "x-forwarded-") {
			return true
		}
	}
	return false
}
