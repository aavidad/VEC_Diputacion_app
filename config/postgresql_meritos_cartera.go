package config

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	EnvExternoMeritosOperacionDatabaseURL            = "VEC_EXTERNO_MERITOS_OPERACION_DATABASE_URL"
	EnvExternoMeritosCarteraDatabaseURL              = "VEC_EXTERNO_MERITOS_CARTERA_DATABASE_URL"
	EnvExternoMeritosFronteraDatabaseURL             = "VEC_EXTERNO_MERITOS_FRONTERA_DATABASE_URL"
	EnvExternoMeritosRegistroAutorizacionDatabaseURL = "VEC_EXTERNO_MERITOS_REGISTRO_AUTORIZACION_DATABASE_URL"
	EnvExternoMeritosMotivosAutorizacionDatabaseURL  = "VEC_EXTERNO_MERITOS_MOTIVOS_AUTORIZACION_DATABASE_URL"
	EnvExternoMeritosFuenteAutorizacionDatabaseURL   = "VEC_EXTERNO_MERITOS_FUENTE_AUTORIZACION_DATABASE_URL"
	EnvExternoMeritosContextoDatabaseURL             = "VEC_EXTERNO_MERITOS_CONTEXTO_DATABASE_URL"
)

var (
	ErrMeritosCarteraPostgreSQLIncompleto = errors.New("config: falta conexion nominal de cartera de meritos")
	ErrMeritosCarteraPostgreSQLNoSeparado = errors.New("config: login de cartera de meritos no separado")
)

// DSNMeritosCarteraSeparados exige LOGIN distintos para datos, frontera,
// fuente, registro, motivos V3 y contexto Usuarios externo CA15.
// El montaje convierte cualquier error en 503 del módulo; no detiene el portal.
func (c Config) DSNMeritosCarteraSeparados() (datos, frontera, fuente, registro, motivos, contexto string, err error) {
	c = c.Normalize()
	configuraciones := [...]ConfiguracionPostgreSQLExterna{
		c.ExternoMeritosCarteraPostgreSQL, c.ExternoMeritosFronteraPostgreSQL,
		c.ExternoMeritosFuenteAutorizacionPostgreSQL,
		c.ExternoMeritosRegistroAutorizacionPostgreSQL, c.ExternoMeritosMotivosAutorizacionPostgreSQL,
		c.ExternoMeritosContextoPostgreSQL,
	}
	esperados := [...]string{
		"vec_externo_meritos_cartera", "vec_externo_meritos_frontera",
		"vec_externo_meritos_fuente_autorizacion",
		"vec_externo_meritos_registro_autorizacion", "vec_externo_meritos_motivos_autorizacion",
		"vec_externo_meritos_contexto",
	}
	var dsns [6]string
	for i, cfg := range configuraciones {
		dsn, e := cfg.DSN()
		if e != nil {
			return "", "", "", "", "", "", ErrMeritosCarteraPostgreSQLIncompleto
		}
		configuracion, parseErr := pgconn.ParseConfig(dsn)
		if parseErr != nil || configuracion.User == "" {
			return "", "", "", "", "", "", ErrMeritosCarteraPostgreSQLIncompleto
		}
		if configuracion.User != esperados[i] {
			return "", "", "", "", "", "", fmt.Errorf("%w: clave=login_m4_%d esperado=%s actual=%s",
				ErrMeritosCarteraPostgreSQLNoSeparado, i, esperados[i], configuracion.User)
		}
		for _, anterior := range dsns[:i] {
			if conexionPostgreSQLComparteLogin(dsn, anterior) {
				return "", "", "", "", "", "", ErrMeritosCarteraPostgreSQLNoSeparado
			}
		}
		dsns[i] = dsn
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
		for _, dsn := range dsns {
			if conexionPostgreSQLComparteLogin(dsn, previa) {
				return "", "", "", "", "", "", ErrMeritosCarteraPostgreSQLNoSeparado
			}
		}
	}
	return dsns[0], dsns[1], dsns[2], dsns[3], dsns[4], dsns[5], nil
}
