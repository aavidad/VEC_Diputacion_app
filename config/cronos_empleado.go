package config

import "errors"

// EnvCronosEmpleadoEnabled monta las rutas de Cronos de la persona empleada
// (saldo propio y fichaje remoto). Es un selector de despliegue, no un
// permiso: sólo admite "true" y "false"; la ausencia equivale a apagado y la
// autorización de cada petición sigue siendo del PDP común.
const EnvCronosEmpleadoEnabled = "VEC_CRONOS_EMPLEADO_ENABLED"

// EnvCronosResolucionEnabled añade la resolución de permisos (bandejas de
// jefatura y RRHH) y los avisos de resolución. Exige Cronos de la persona
// empleada activo y, antes del binario, AD3-57 y cronos_v1 000009 instaladas.
const EnvCronosResolucionEnabled = "VEC_CRONOS_RESOLUCION_ENABLED"

var (
	ErrConfiguracionCronosEmpleadoSelector   = errors.New("config: selector de Cronos para la persona empleada invalido")
	ErrConfiguracionCronosEmpleadoActivacion = errors.New("config: Cronos para la persona empleada fuera del perfil de desarrollo")
	ErrConfiguracionCronosResolucionSelector = errors.New("config: selector de la resolución de permisos de Cronos invalido")
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

// CronosResolucionDesarrolloActiva valida el selector de la resolución de
// permisos: sólo "true" o "false", y "true" sólo con Cronos activo.
func (c Config) CronosResolucionDesarrolloActiva() (bool, error) {
	c = c.Normalize()
	switch c.CronosResolucionEnabled {
	case "", "false":
		return false, nil
	case "true":
		activo, err := c.CronosEmpleadoDesarrolloActivo()
		if err != nil {
			return false, err
		}
		if !activo {
			return false, ErrConfiguracionCronosResolucionSelector
		}
		return true, nil
	default:
		return false, ErrConfiguracionCronosResolucionSelector
	}
}
