package config

import "errors"

// El selector sólo compone el circuito una vez instaladas sus dependencias.
// La concesión nominal sigue bajo gobierno y nunca nace del selector.
const EnvBolsaInscripcionesEnabled = "VEC_BOLSA_INSCRIPCIONES_ENABLED"

var (
	ErrBolsaInscripcionesSelector   = errors.New("config: selector de inscripciones Bolsa invalido")
	ErrBolsaInscripcionesActivacion = errors.New("config: inscripciones Bolsa fuera de perfil o sin lector")
)

// Cada proceso comprueba sólo las credenciales que necesita su superficie.
// El portal externo nunca requiere ni recibe el DSN de RRHH.
func (c Config) BolsaInscripcionesExternoActivo() (bool, error) {
	c = c.Normalize()
	return selectorDesarrolloActivo(c, c.BolsaInscripcionesEnabled,
		ErrBolsaInscripcionesSelector, ErrBolsaInscripcionesActivacion,
		catalogoRequerido{EnvExternoBolsaInscripcionesLectorDatabaseURL, c.BolsaInscripcionesLectorPostgreSQL.dsn})
}

func (c Config) BolsaInscripcionesInternoActivo() (bool, error) {
	c = c.Normalize()
	return selectorDesarrolloActivo(c, c.BolsaInscripcionesEnabled,
		ErrBolsaInscripcionesSelector, ErrBolsaInscripcionesActivacion,
		catalogoRequerido{EnvBolsaInscripcionesRRHHLectorDatabaseURL, c.BolsaInscripcionesRRHHLectorPostgreSQL.dsn})
}
