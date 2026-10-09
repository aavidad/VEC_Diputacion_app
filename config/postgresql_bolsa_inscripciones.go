package config

import "errors"

// Las lecturas de la persona aspirante y de RRHH usan LOGIN distintos.
// Los actos conservan sus ejecutores nominales.
const (
	EnvBolsaInscripcionesLectorDatabaseURL     = "VEC_BOLSA_INSCRIPCIONES_LECTOR_DATABASE_URL"
	EnvBolsaInscripcionesRRHHLectorDatabaseURL = "VEC_BOLSA_INSCRIPCIONES_RRHH_LECTOR_DATABASE_URL"
)

var (
	ErrBolsaInscripcionesLectorIncompleto = errors.New("config: falta PostgreSQL lector de inscripciones")
	ErrBolsaInscripcionesLectorNoSeparado = errors.New("config: lector de inscripciones comparte LOGIN")
)

func (c Config) DSNBolsaInscripcionesLectorSeparado() (string, error) {
	c = c.Normalize()
	dsn, err := c.BolsaInscripcionesLectorPostgreSQL.DSN()
	if err != nil {
		return "", ErrBolsaInscripcionesLectorIncompleto
	}
	previas := append(c.dsnsPostgreSQLConfigurados(),
		c.ExternoBolsaPostgreSQL.dsn, c.ExternoBolsaFronteraPostgreSQL.dsn,
		c.ExternoBolsaPublicaPostgreSQL.dsn, c.BolsaAuditoriaFronteraPostgreSQL.normalizar().dsn)
	for _, previa := range previas {
		if conexionPostgreSQLComparteLogin(dsn, previa) {
			return "", ErrBolsaInscripcionesLectorNoSeparado
		}
	}
	return dsn, nil
}

// El lector de RRHH no comparte LOGIN con ninguna otra conexión del
// proceso interno ni con el lector de la persona aspirante.
func (c Config) DSNBolsaInscripcionesLectorRRHHSeparado() (string, error) {
	c = c.Normalize()
	rrhh, err := c.BolsaInscripcionesRRHHLectorPostgreSQL.DSN()
	if err != nil {
		return "", ErrBolsaInscripcionesLectorIncompleto
	}
	previas := append(c.dsnsPostgreSQLConfigurados(), c.ExternoBolsaPostgreSQL.dsn,
		c.ExternoBolsaFronteraPostgreSQL.dsn, c.ExternoBolsaPublicaPostgreSQL.dsn,
		c.BolsaInscripcionesLectorPostgreSQL.dsn,
		c.BolsaAuditoriaFronteraPostgreSQL.normalizar().dsn)
	for _, previa := range previas {
		if conexionPostgreSQLComparteLogin(rrhh, previa) {
			return "", ErrBolsaInscripcionesLectorNoSeparado
		}
	}
	return rrhh, nil
}
