package config

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	EnvExternoMeritosCarteraDatabaseURL              = "VEC_EXTERNO_MERITOS_CARTERA_DATABASE_URL"
	EnvExternoMeritosFronteraDatabaseURL             = "VEC_EXTERNO_MERITOS_FRONTERA_DATABASE_URL"
	EnvExternoMeritosRegistroAutorizacionDatabaseURL = "VEC_EXTERNO_MERITOS_REGISTRO_AUTORIZACION_DATABASE_URL"
	EnvExternoMeritosMotivosAutorizacionDatabaseURL  = "VEC_EXTERNO_MERITOS_MOTIVOS_AUTORIZACION_DATABASE_URL"
)

var (
	ErrMeritosCarteraPostgreSQLIncompleto = errors.New("config: falta conexion nominal de cartera de meritos")
	ErrMeritosCarteraPostgreSQLNoSeparado = errors.New("config: login de cartera de meritos no separado")
)

// DSNMeritosCarteraSeparados exige LOGIN distintos para datos, frontera y
// las dos operaciones V3; ninguna credencial se reutiliza de otro módulo.
// El montaje convierte cualquier error en 503 del módulo; no detiene el portal.
func (c Config) DSNMeritosCarteraSeparados() (datos, frontera, registro, motivos string, err error) {
	c = c.Normalize()
	configuraciones := [...]ConfiguracionPostgreSQLExterna{
		c.ExternoMeritosCarteraPostgreSQL, c.ExternoMeritosFronteraPostgreSQL,
		c.ExternoMeritosRegistroAutorizacionPostgreSQL, c.ExternoMeritosMotivosAutorizacionPostgreSQL,
	}
	esperados := [...]string{
		"vec_externo_meritos_cartera", "vec_externo_meritos_frontera",
		"vec_externo_meritos_registro_autorizacion", "vec_externo_meritos_motivos_autorizacion",
	}
	var dsns [4]string
	for i, cfg := range configuraciones {
		dsn, e := cfg.DSN()
		if e != nil {
			return "", "", "", "", ErrMeritosCarteraPostgreSQLIncompleto
		}
		configuracion, parseErr := pgconn.ParseConfig(dsn)
		if parseErr != nil || configuracion.User == "" {
			return "", "", "", "", ErrMeritosCarteraPostgreSQLIncompleto
		}
		if configuracion.User != esperados[i] {
			return "", "", "", "", fmt.Errorf("%w: clave=login_m4_%d esperado=%s actual=%s",
				ErrMeritosCarteraPostgreSQLNoSeparado, i, esperados[i], configuracion.User)
		}
		for _, anterior := range dsns[:i] {
			if conexionPostgreSQLComparteLogin(dsn, anterior) {
				return "", "", "", "", ErrMeritosCarteraPostgreSQLNoSeparado
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
				return "", "", "", "", ErrMeritosCarteraPostgreSQLNoSeparado
			}
		}
	}
	return dsns[0], dsns[1], dsns[2], dsns[3], nil
}
