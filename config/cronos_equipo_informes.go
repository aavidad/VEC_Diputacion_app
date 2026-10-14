package config

import (
	"errors"
	"path/filepath"
	"strings"
)

const (
	EnvCronosEquipoInformesEnabled    = "VEC_CRONOS_EQUIPO_INFORMES_ENABLED"
	EnvCronosEquipoInformesConfigFile = "VEC_CRONOS_EQUIPO_INFORMES_CONFIG_FILE"
	RutaPrivadaCronosEquipoInformes   = "identidad/cronos-equipo-informes.json"
)

var ErrConfiguracionCronosEquipoInformes = errors.New("config: equipo e informes de Cronos no disponibles")

// El selector y el archivo solo deciden si se componen las rutas. El perfil
// vigente, los campos y el equipo se revalidan por autorización V3 y SQL.
func (c Config) CronosEquipoInformesDesarrolloActivo() (bool, error) {
	selector := strings.TrimSpace(c.CronosEquipoInformesEnabled)
	switch selector {
	case "", "false":
		return false, nil
	case "true":
		activo, err := c.CronosEmpleadoDesarrolloActivo()
		if err != nil || !activo || !c.DevelopmentEnabledByDoubleKey() {
			return false, ErrConfiguracionCronosEquipoInformes
		}
		ruta := c.CronosEquipoInformesConfigFile
		if ruta != RutaPrivadaCronosEquipoInformes || filepath.IsAbs(ruta) ||
			filepath.Clean(ruta) != ruta || c.DevelopmentMaterialDir == "" {
			return false, ErrConfiguracionCronosEquipoInformes
		}
		return true, nil
	default:
		return false, ErrConfiguracionCronosEquipoInformes
	}
}
