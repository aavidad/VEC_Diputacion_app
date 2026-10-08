package config

import "errors"

// EnvBolsaInscripcionesLectorDatabaseURL usa un LOGIN exclusivo para las
// lecturas de inscripción. Los actos conservan sus ejecutores nominales.
const EnvBolsaInscripcionesLectorDatabaseURL = "VEC_BOLSA_INSCRIPCIONES_LECTOR_DATABASE_URL"

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
