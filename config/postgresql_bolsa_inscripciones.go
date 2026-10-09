package config

import "errors"

// Las lecturas externa propia, de empleado y de RRHH usan LOGIN distintos.
// Los actos conservan sus ejecutores nominales.
const (
	EnvBolsaInscripcionesLectorDatabaseURL         = "VEC_BOLSA_INSCRIPCIONES_LECTOR_DATABASE_URL"
	EnvBolsaInscripcionesEmpleadoLectorDatabaseURL = "VEC_BOLSA_INSCRIPCIONES_EMPLEADO_LECTOR_DATABASE_URL"
	EnvBolsaInscripcionesRRHHLectorDatabaseURL     = "VEC_BOLSA_INSCRIPCIONES_RRHH_LECTOR_DATABASE_URL"
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

func (c Config) DSNBolsaInscripcionesLectoresSeparados() (externo, empleado, rrhh string, err error) {
	c = c.Normalize()
	externo, err = c.DSNBolsaInscripcionesLectorSeparado()
	if err != nil {
		return "", "", "", err
	}
	empleado, err = c.BolsaInscripcionesEmpleadoLectorPostgreSQL.DSN()
	if err != nil {
		return "", "", "", ErrBolsaInscripcionesLectorIncompleto
	}
	rrhh, err = c.BolsaInscripcionesRRHHLectorPostgreSQL.DSN()
	if err != nil {
		return "", "", "", ErrBolsaInscripcionesLectorIncompleto
	}
	previas := append(c.dsnsPostgreSQLConfigurados(), c.ExternoBolsaPostgreSQL.dsn,
		c.ExternoBolsaFronteraPostgreSQL.dsn, c.ExternoBolsaPublicaPostgreSQL.dsn,
		c.BolsaAuditoriaFronteraPostgreSQL.normalizar().dsn)
	for _, lector := range []string{empleado, rrhh} {
		if conexionPostgreSQLComparteLogin(lector, externo) {
			return "", "", "", ErrBolsaInscripcionesLectorNoSeparado
		}
		for _, previa := range previas {
			if conexionPostgreSQLComparteLogin(lector, previa) {
				return "", "", "", ErrBolsaInscripcionesLectorNoSeparado
			}
		}
	}
	if conexionPostgreSQLComparteLogin(empleado, rrhh) {
		return "", "", "", ErrBolsaInscripcionesLectorNoSeparado
	}
	return externo, empleado, rrhh, nil
}

func (c Config) DSNBolsaInscripcionesLectoresInternosSeparados() (empleado, rrhh string, err error) {
	c = c.Normalize()
	empleado, err = c.BolsaInscripcionesEmpleadoLectorPostgreSQL.DSN()
	if err != nil {
		return "", "", ErrBolsaInscripcionesLectorIncompleto
	}
	rrhh, err = c.BolsaInscripcionesRRHHLectorPostgreSQL.DSN()
	if err != nil {
		return "", "", ErrBolsaInscripcionesLectorIncompleto
	}
	previas := append(c.dsnsPostgreSQLConfigurados(), c.ExternoBolsaPostgreSQL.dsn,
		c.ExternoBolsaFronteraPostgreSQL.dsn, c.ExternoBolsaPublicaPostgreSQL.dsn,
		c.BolsaInscripcionesLectorPostgreSQL.dsn,
		c.BolsaAuditoriaFronteraPostgreSQL.normalizar().dsn)
	for _, lector := range []string{empleado, rrhh} {
		for _, previa := range previas {
			if conexionPostgreSQLComparteLogin(lector, previa) {
				return "", "", ErrBolsaInscripcionesLectorNoSeparado
			}
		}
	}
	if conexionPostgreSQLComparteLogin(empleado, rrhh) {
		return "", "", ErrBolsaInscripcionesLectorNoSeparado
	}
	return empleado, rrhh, nil
}
