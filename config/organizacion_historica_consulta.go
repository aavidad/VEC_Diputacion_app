package config

import "errors"

const (
	EnvOrganizacionHistoricaConsultaEnabled = "VEC_ORGANIZACION_HISTORICA_CONSULTA_ENABLED"
	EnvOrganizacionHistoricaMaterialDir     = "VEC_ORGANIZACION_HISTORICA_MATERIAL_DIR"
)

var ErrConfiguracionOrganizacionHistoricaConsulta = errors.New("config: consulta de organizacion historica no disponible")

// OrganizacionHistoricaConsultaDesarrolloActivo sólo selecciona el montaje.
// La consulta sigue dependiendo de sesión, concesión V3 y material privado.
func (c Config) OrganizacionHistoricaConsultaDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	switch c.OrganizacionHistoricaConsultaEnabled {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() || c.OrganizacionHistoricaMaterialDir == "" {
			return false, ErrConfiguracionOrganizacionHistoricaConsulta
		}
		return true, nil
	default:
		return false, ErrConfiguracionOrganizacionHistoricaConsulta
	}
}
