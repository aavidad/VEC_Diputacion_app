package config

import "errors"

// EnvPersonalB2GobiernoEnabled hace que el publicador único de gobierno V3 de
// desarrollo (vec-server) publique también las ocho claves de consumo del
// registro B2 de Personal, que después lee vec-interno. Es un selector de
// despliegue, no un permiso: sólo admite "true" y "false"; la ausencia equivale
// a apagado y la autorización de cada petición sigue siendo del PDP común.
const EnvPersonalB2GobiernoEnabled = "VEC_PERSONAL_B2_GOBIERNO_ENABLED"

var (
	ErrConfiguracionPersonalB2GobiernoSelector   = errors.New("config: selector de gobierno de Personal B2 invalido")
	ErrConfiguracionPersonalB2GobiernoActivacion = errors.New("config: gobierno de Personal B2 fuera del perfil de desarrollo")
)

// PersonalB2GobiernoDesarrolloActivo valida la activación sin abrir ficheros
// ni conexiones. Un selector mal escrito o fuera de la doble llave falla
// cerrado.
func (c Config) PersonalB2GobiernoDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	switch c.PersonalB2GobiernoEnabled {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() {
			return false, ErrConfiguracionPersonalB2GobiernoActivacion
		}
		return true, nil
	default:
		return false, ErrConfiguracionPersonalB2GobiernoSelector
	}
}
