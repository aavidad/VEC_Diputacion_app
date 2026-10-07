package config

import "errors"

// EnvOrganizacionHistoricaGobiernoEnabled selecciona la publicación de la
// clave de consulta histórica en el gobierno V3 existente de vec-server.
// No concede permisos ni habilita los consumidores de Personal B2.
const EnvOrganizacionHistoricaGobiernoEnabled = "VEC_ORGANIZACION_HISTORICA_GOBIERNO_ENABLED"

var (
	ErrConfiguracionOrganizacionHistoricaGobiernoSelector   = errors.New("config: selector de gobierno de organizacion historica invalido")
	ErrConfiguracionOrganizacionHistoricaGobiernoActivacion = errors.New("config: gobierno de organizacion historica fuera del perfil de desarrollo")
)

// OrganizacionHistoricaGobiernoDesarrolloActivo exige selección explícita y
// las llaves de desarrollo. El error pertenece a esta capacidad opcional.
func (c Config) OrganizacionHistoricaGobiernoDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	switch c.OrganizacionHistoricaGobiernoEnabled {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() {
			return false, ErrConfiguracionOrganizacionHistoricaGobiernoActivacion
		}
		return true, nil
	default:
		return false, ErrConfiguracionOrganizacionHistoricaGobiernoSelector
	}
}
