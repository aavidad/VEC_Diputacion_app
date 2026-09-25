package config

import "errors"

// EnvCronosEmpleadoEnabled monta las rutas de Cronos de la persona empleada
// (saldo propio y fichaje remoto). Es un selector de despliegue, no un
// permiso: sólo admite "true" y "false"; la ausencia equivale a apagado y la
// autorización de cada petición sigue siendo del PDP común.
const EnvCronosEmpleadoEnabled = "VEC_CRONOS_EMPLEADO_ENABLED"

var (
	ErrConfiguracionCronosEmpleadoSelector   = errors.New("config: selector de Cronos para la persona empleada invalido")
	ErrConfiguracionCronosEmpleadoActivacion = errors.New("config: Cronos para la persona empleada fuera del perfil de desarrollo")
)

// CronosEmpleadoDesarrolloActivo valida la activación sin abrir ficheros ni
// conexiones. Un selector mal escrito o fuera de la doble llave falla cerrado.
func (c Config) CronosEmpleadoDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	switch c.CronosEmpleadoEnabled {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() {
			return false, ErrConfiguracionCronosEmpleadoActivacion
		}
		return true, nil
	default:
		return false, ErrConfiguracionCronosEmpleadoSelector
	}
}
