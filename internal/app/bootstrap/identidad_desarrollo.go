package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type identidadCertificadoDesarrollo struct {
	huella    [sha256.Size]byte
	principal vecdomain.Principal
}

type resolvedorIdentidadDesarrollo struct {
	porHuella    map[[sha256.Size]byte]vecdomain.Principal
	porSujeto    map[string]adscripcionCentroDesarrollo
	lectoresRRHH map[[sha256.Size]byte]identidadConsultaRRHHDesarrollo
}

func nuevoResolvedorIdentidadDesarrollo(
	identidades ...identidadCertificadoDesarrollo,
) (*resolvedorIdentidadDesarrollo, error) {
	if len(identidades) == 0 {
		return nil, ErrMaterialDesarrolloInvalido
	}
	porHuella := make(map[[sha256.Size]byte]vecdomain.Principal, len(identidades))
	for _, identidad := range identidades {
		if identidad.principal.Validate() != nil || identidad.huella == ([sha256.Size]byte{}) {
			return nil, ErrMaterialDesarrolloInvalido
		}
		if _, repetida := porHuella[identidad.huella]; repetida {
			return nil, ErrMaterialDesarrolloInvalido
		}
		porHuella[identidad.huella] = clonarPrincipalDesarrollo(identidad.principal)
	}
	return &resolvedorIdentidadDesarrollo{porHuella: porHuella, porSujeto: make(map[string]adscripcionCentroDesarrollo), lectoresRRHH: make(map[[sha256.Size]byte]identidadConsultaRRHHDesarrollo)}, nil
}

func (r *resolvedorIdentidadDesarrollo) registrarLectoresRRHH(lectores []identidadConsultaRRHHDesarrollo) error {
	if r == nil {
		return ErrMaterialDesarrolloInvalido
	}
	for _, lector := range lectores {
		if lector.identidad.principal.Validate() != nil || lector.identidad.huella == ([sha256.Size]byte{}) ||
			lector.identidad.principal.ID == "" || len(lector.identidad.principal.Roles) != 1 ||
			(lector.identidad.principal.Roles[0] != "lector_rrhh" &&
				lector.identidad.principal.Roles[0] != rolTecnicoRRHHContratacionTemporalDesarrollo) {
			return ErrMaterialDesarrolloInvalido
		}
		principal, existe := r.porHuella[lector.identidad.huella]
		if !existe || principal.ID != lector.identidad.principal.ID {
			return ErrMaterialDesarrolloInvalido
		}
		if _, repetido := r.lectoresRRHH[lector.identidad.huella]; repetido {
			return ErrMaterialDesarrolloInvalido
		}
		r.lectoresRRHH[lector.identidad.huella] = lector
	}
	return nil
}

func (r *resolvedorIdentidadDesarrollo) lectorConsultaRRHH(principal vecdomain.Principal) (identidadConsultaRRHHDesarrollo, bool) {
	if r == nil || !principalSinteticoContratacionTemporalDesarrolloValido(principal) || len(principal.Roles) != 1 {
		return identidadConsultaRRHHDesarrollo{}, false
	}
	for huella, lector := range r.lectoresRRHH {
		if r.porHuella[huella].ID == principal.ID && lector.identidad.principal.Attributes["certificate_sha256"] == principal.Attributes["certificate_sha256"] {
			return lector, true
		}
	}
	return identidadConsultaRRHHDesarrollo{}, false
}

func (r *resolvedorIdentidadDesarrollo) lectoresConsultaRRHH() []identidadConsultaRRHHDesarrollo {
	if r == nil {
		return nil
	}
	lectores := make([]identidadConsultaRRHHDesarrollo, 0, len(r.lectoresRRHH))
	for _, lector := range r.lectoresRRHH {
		lectores = append(lectores, lector)
	}
	return lectores
}

func (r *resolvedorIdentidadDesarrollo) adscripcionCentro(subject string) (adscripcionCentroDesarrollo, bool) {
	if r == nil {
		return adscripcionCentroDesarrollo{}, false
	}
	a, ok := r.porSujeto[subject]
	return a, ok
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
	principal, existe := r.porHuella[huella]
	if !existe {
		return vecdomain.Principal{}, ErrMaterialDesarrolloInvalido
	}
	return clonarPrincipalDesarrollo(principal), nil
}

func (r *resolvedorIdentidadDesarrollo) principalConRolUnico(
	rol string,
) (vecdomain.Principal, bool) {
	if r == nil || len(r.porHuella) == 0 || rol == "" {
		return vecdomain.Principal{}, false
	}
	var encontrado vecdomain.Principal
	coincidencias := 0
	for _, principal := range r.porHuella {
		if len(principal.Roles) == 1 && principal.Roles[0] == rol {
			encontrado = principal
			coincidencias++
		}
	}
	if coincidencias != 1 {
		return vecdomain.Principal{}, false
	}
	return clonarPrincipalDesarrollo(encontrado), true
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
