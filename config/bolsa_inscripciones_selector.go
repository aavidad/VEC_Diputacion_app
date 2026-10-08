package config

import "errors"

// El selector sólo compone el circuito una vez instaladas sus dependencias.
// La concesión nominal sigue bajo gobierno y nunca nace del selector.
const EnvBolsaInscripcionesEnabled = "VEC_BOLSA_INSCRIPCIONES_ENABLED"

var (
	ErrBolsaInscripcionesSelector   = errors.New("config: selector de inscripciones Bolsa invalido")
	ErrBolsaInscripcionesActivacion = errors.New("config: inscripciones Bolsa fuera de perfil o sin lector")
)

func (c Config) BolsaInscripcionesActivo() (bool, error) {
	c = c.Normalize()
	return selectorDesarrolloActivo(c, c.BolsaInscripcionesEnabled,
		ErrBolsaInscripcionesSelector, ErrBolsaInscripcionesActivacion,
		catalogoRequerido{EnvBolsaInscripcionesLectorDatabaseURL, c.BolsaInscripcionesLectorPostgreSQL.dsn},
		catalogoRequerido{EnvBolsaInscripcionesEmpleadoLectorDatabaseURL, c.BolsaInscripcionesEmpleadoLectorPostgreSQL.dsn},
		catalogoRequerido{EnvBolsaInscripcionesRRHHLectorDatabaseURL, c.BolsaInscripcionesRRHHLectorPostgreSQL.dsn})
}
