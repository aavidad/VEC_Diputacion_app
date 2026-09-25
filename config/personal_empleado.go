package config

import "errors"

// EnvPersonalEmpleadoEnabled monta la ficha propia de la persona empleada
// («mis datos» de Personal). Es un selector de despliegue, no un permiso:
// solo admite "true" y "false"; la ausencia equivale a apagado y cada
// consulta sigue necesitando su concesión V3 del PDP común.
const EnvPersonalEmpleadoEnabled = "VEC_PERSONAL_EMPLEADO_ENABLED"

var (
	ErrConfiguracionPersonalEmpleadoSelector   = errors.New("config: selector de la ficha propia de Personal invalido")
	ErrConfiguracionPersonalEmpleadoActivacion = errors.New("config: ficha propia de Personal fuera del perfil de desarrollo")
)

// PersonalEmpleadoDesarrolloActivo valida la activación sin abrir ficheros
// ni conexiones. Un selector mal escrito o fuera de la doble llave falla
// cerrado.
func (c Config) PersonalEmpleadoDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	switch c.PersonalEmpleadoEnabled {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() {
			return false, ErrConfiguracionPersonalEmpleadoActivacion
		}
		return true, nil
	default:
		return false, ErrConfiguracionPersonalEmpleadoSelector
	}
}
