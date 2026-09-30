package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"

	"vec-diputacion-granada/config"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
)

// La configuración privada selecciona cuentas certificadas ya conocidas por
// el resolvedor. Esta comprobación ocurre antes de construir CT y de publicar
// las audiencias V3. El montaje repite el cotejo al abrir sus pools.
func validarIdentidadesPreferenciasAntesDeCT(cfg config.Config, resolvedor vechttp.DemoIdentityResolver) error {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil {
		return err
	}
	if !activo {
		return nil
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil {
		return errComposicionUsuariosPreferencias
	}
	interna, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	if _, err = cuentasPreferenciasAcreditadas(identidad, interna); err != nil {
		return err
	}
	// El proceso interno separado no tiene la configuración del Área personal.
	if !superficieExternaUsuariosEnProceso(cfg) {
		return nil
	}
	externa, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionExternaPersonalV1)
	if err != nil || !configuracionesPreferenciasSeparadas(interna, externa) {
		return errComposicionUsuariosPreferencias
	}
	if _, err = cuentasPreferenciasAcreditadas(identidad, externa); err != nil {
		return err
	}
	return nil
}

func cuentasPreferenciasAcreditadas(identidad *resolvedorIdentidadDesarrollo, c configuracionUsuariosPreferenciasDesarrollo) (map[string]cuentaUsuariosPreferenciasDesarrollo, error) {
	if identidad == nil || (c.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 && c.Superficie != core.SuperficieAutenticacionExternaPersonalV1) {
		return nil, errComposicionUsuariosPreferencias
	}
	cuentas := make(map[string]cuentaUsuariosPreferenciasDesarrollo, len(c.Cuentas))
	for _, cuenta := range c.Cuentas {
		b, err := hex.DecodeString(cuenta.CertificadoSHA256)
		if err != nil || len(b) != sha256.Size || hex.EncodeToString(b) != cuenta.CertificadoSHA256 || cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" {
			return nil, errComposicionUsuariosPreferencias
		}
		if _, repetida := cuentas[cuenta.CertificadoSHA256]; repetida {
			return nil, errComposicionUsuariosPreferencias
		}
		var digest [sha256.Size]byte
		copy(digest[:], b)
		principal, existe := identidad.porHuella[digest]
		if !existe || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh ||
			principal.Attributes["certificate_sha256"] != cuenta.CertificadoSHA256 || !principalParaSuperficieUsuariosPreferenciasValido(identidad, principal, string(c.Superficie)) {
			return nil, errComposicionUsuariosPreferencias
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	return cuentas, nil
}
