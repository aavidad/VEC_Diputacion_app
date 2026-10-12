package config

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	EnvExternoMeritosCarteraDatabaseURL  = "VEC_EXTERNO_MERITOS_CARTERA_DATABASE_URL"
	EnvExternoMeritosFronteraDatabaseURL = "VEC_EXTERNO_MERITOS_FRONTERA_DATABASE_URL"
)

var (
	ErrMeritosCarteraPostgreSQLIncompleto = errors.New("config: falta conexion nominal de cartera de meritos")
	ErrMeritosCarteraPostgreSQLNoSeparado = errors.New("config: login de cartera de meritos no separado")
)

// DSNMeritosCarteraSeparados exige LOGIN distintos para datos y denegaciones.
// El montaje convierte cualquier error en 503 del módulo; no detiene el portal.
func (c Config) DSNMeritosCarteraSeparados() (datos, frontera string, err error) {
	c = c.Normalize()
	datos, err = c.ExternoMeritosCarteraPostgreSQL.DSN()
	if err != nil {
		return "", "", ErrMeritosCarteraPostgreSQLIncompleto
	}
	frontera, err = c.ExternoMeritosFronteraPostgreSQL.DSN()
	if err != nil {
		return "", "", ErrMeritosCarteraPostgreSQLIncompleto
	}
	for _, dsn := range []string{datos, frontera} {
		configuracion, parseErr := pgconn.ParseConfig(dsn)
		if parseErr != nil || configuracion.User == "" {
			return "", "", ErrMeritosCarteraPostgreSQLIncompleto
		}
	}
	if conexionPostgreSQLComparteLogin(datos, frontera) {
		return "", "", ErrMeritosCarteraPostgreSQLNoSeparado
	}
	previas := append(c.dsnsPostgreSQLConfigurados(),
		c.ExternoPreflightV3DatabaseURL, c.ExternoBolsaPublicaPostgreSQL.dsn,
		c.ExternoBolsaPostgreSQL.dsn, c.ExternoBolsaFronteraPostgreSQL.dsn,
		c.BolsaInscripcionesLectorPostgreSQL.dsn, c.BolsaInscripcionesRRHHLectorPostgreSQL.dsn,
		c.ExternoCalendariosPostgreSQL.dsn, c.ExternoAutorizacionFuentePostgreSQL.dsn,
		c.ExternoAutorizacionRegistroPostgreSQL.dsn, c.ExternoAutorizacionMotivosPostgreSQL.dsn,
		c.ExternoIdentidadRegistroPostgreSQL.dsn, c.ExternoIdentidadRevalidacionPostgreSQL.dsn,
		c.ExternoContextoPostgreSQL.dsn)
	for _, previa := range previas {
		if conexionPostgreSQLComparteLogin(datos, previa) || conexionPostgreSQLComparteLogin(frontera, previa) {
			return "", "", ErrMeritosCarteraPostgreSQLNoSeparado
		}
	}
	return datos, frontera, nil
}
