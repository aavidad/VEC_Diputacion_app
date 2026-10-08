package config

import (
	"errors"
	"regexp"
	"strings"
)

// EnvPortalModulosVisibles limita los módulos que el shell registra en
// /api/vec/modules y, por tanto, los que el portal muestra. Es un selector de
// despliegue, no un permiso: no concede acceso y cada ruta sigue autorizándose
// por su cuenta. Vacío equivale a mostrar todos los módulos compuestos. El
// valor es una lista separada por comas de claves de módulo («bolsa»,
// «contratacion_temporal»…), sin el prefijo «vec.module.».
const EnvPortalModulosVisibles = "VEC_PORTAL_MODULOS_VISIBLES"

// ErrConfiguracionPortalModulos falla cerrado ante una lista mal escrita.
var ErrConfiguracionPortalModulos = errors.New("config: lista de modulos visibles del portal invalida")

var patronClaveModuloPortal = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

// PortalModulosVisibles devuelve nil cuando no hay lista (todos visibles) o el
// conjunto exacto de claves admitidas. Una clave repetida o con formato no
// válido invalida toda la lista.
func (c Config) PortalModulosVisibles() (map[string]bool, error) {
	valor := strings.TrimSpace(c.PortalModulosVisiblesLista)
	if valor == "" {
		return nil, nil
	}
	claves := strings.Split(valor, ",")
	if len(claves) > 64 {
		return nil, ErrConfiguracionPortalModulos
	}
	visibles := make(map[string]bool, len(claves))
	for _, clave := range claves {
		clave = strings.TrimSpace(clave)
		if !patronClaveModuloPortal.MatchString(clave) || visibles[clave] {
			return nil, ErrConfiguracionPortalModulos
		}
		visibles[clave] = true
	}
	return visibles, nil
}
