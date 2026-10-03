package config

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
)

const (
	EnvAuditoriaIntentosDatabaseURL = "VEC_AUDITORIA_INTENTOS_DATABASE_URL"
	EnvAuditoriaIntentosProceso     = "VEC_AUDITORIA_INTENTOS_PROCESO"
	EnvAuditoriaIntentosCanal       = "VEC_AUDITORIA_INTENTOS_CANAL"
	EnvAuditoriaIntentosPlazo       = "VEC_AUDITORIA_INTENTOS_PLAZO"
)

var ErrConexionIntentosAuditoria = errors.New("config: registrador nominal de consultas no disponible")
var patronProcesoIntentosAuditoria = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)

// ConexionIntentosAuditoriaConsulta exige el LOGIN segregado de AD169 y su
// origen explícito. Solo se invoca al montar la consulta RRHH de desarrollo;
// no configura permisos ni publica identidad. Los errores no reflejan valores.
func (c Config) ConexionIntentosAuditoriaConsulta() (dsn, proceso, canal string, plazo time.Duration, err error) {
	fallo := func() (string, string, string, time.Duration, error) {
		return "", "", "", 0, ErrConexionIntentosAuditoria
	}
	if !c.DevelopmentEnabledByDoubleKey() {
		return fallo()
	}
	dsn = strings.TrimSpace(os.Getenv(EnvAuditoriaIntentosDatabaseURL))
	conexion, e := pgconn.ParseConfig(dsn)
	if dsn == "" || e != nil || conexion.User == "" {
		return fallo()
	}
	proceso = os.Getenv(EnvAuditoriaIntentosProceso)
	canal = os.Getenv(EnvAuditoriaIntentosCanal)
	plazo, e = time.ParseDuration(os.Getenv(EnvAuditoriaIntentosPlazo))
	if !patronProcesoIntentosAuditoria.MatchString(proceso) ||
		canal != string(domain.SuperficieAutenticacionInternaCorporativaV1) ||
		e != nil || plazo <= 0 || plazo > 10*time.Second {
		return fallo()
	}
	for _, previo := range c.dsnsPostgreSQLConfigurados() {
		if conexionPostgreSQLComparteLogin(dsn, previo) {
			return fallo()
		}
	}
	for _, nombre := range []string{EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL, EnvRRHHAuditoriaMotivosDatabaseURL, EnvRRHHAuditoriaFronteraDatabaseURL} {
		if conexionPostgreSQLComparteLogin(dsn, os.Getenv(nombre)) {
			return fallo()
		}
	}
	return dsn, proceso, canal, plazo, nil
}
