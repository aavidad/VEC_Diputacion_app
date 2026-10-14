package config

import "errors"

const EnvCronosSaldoEfectosC3Enabled = "VEC_CRONOS_SALDO_EFECTOS_C3_ENABLED"

var ErrConfiguracionSaldoEfectosC3 = errors.New("config: saldo con efectos de permisos no disponible")

func CronosSaldoEfectosC3DesarrolloActivo(c Config, selector string) (bool, error) {
	switch selector {
	case "", "false":
		return false, nil
	case "true":
		activo, err := c.CronosEmpleadoDesarrolloActivo()
		if err != nil || !activo {
			return false, ErrConfiguracionSaldoEfectosC3
		}
		return true, nil
	default:
		return false, ErrConfiguracionSaldoEfectosC3
	}
}
