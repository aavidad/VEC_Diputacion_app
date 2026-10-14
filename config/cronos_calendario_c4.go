package config

import "errors"

const EnvCronosCalendarioC4Enabled = "VEC_CRONOS_CALENDARIO_C4_ENABLED"

var ErrConfiguracionCronosCalendarioC4 = errors.New("config: calendario laboral de Cronos no disponible")

// La opción se lee en el arranque del componente. Un valor ausente lo deja
// apagado; un valor inválido cierra sólo la capacidad C4.
func CronosCalendarioC4DesarrolloActivo(c Config, selector string) (bool, error) {
	switch selector {
	case "", "false":
		return false, nil
	case "true":
		activo, err := c.CronosEmpleadoDesarrolloActivo()
		if err != nil || !activo {
			return false, ErrConfiguracionCronosCalendarioC4
		}
		return true, nil
	default:
		return false, ErrConfiguracionCronosCalendarioC4
	}
}
