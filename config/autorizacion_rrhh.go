package config

import (
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	EnvAutorizacionFuenteDatabaseURL           = "VEC_AUTORIZACION_FUENTE_DATABASE_URL"
	EnvAutorizacionMotivosEvaluadorDatabaseURL = "VEC_AUTORIZACION_MOTIVOS_EVALUADOR_DATABASE_URL"
	RolAutorizacionFuenteRRHH                  = "vec_autorizacion_fuente"
	RolAutorizacionMotivosEvaluadorRRHH        = "vec_autorizacion_motivos_evaluador"
)

var ErrAutoridadesAutorizacionRRHHIncompletas = errors.New("config: faltan autoridades PostgreSQL de autorizacion RRHH")
var ErrAutoridadesAutorizacionRRHHNoSeparadas = errors.New("config: autoridades PostgreSQL de autorizacion RRHH comparten LOGIN")

// DSNAutoridadesAutorizacionRRHH reserva un LOGIN de solo lectura para la
// instantánea V3 y otro evaluador de motivos. No depende del selector de
// Auditoría ni reutiliza gobierno, registro de decisiones o ejecutor CT.
// La raíz los abre y comprueba sus membresías; aquí no se conserva el DSN.
func (c Config) DSNAutoridadesAutorizacionRRHH() (fuente, motivos string, err error) {
	c = c.Normalize()
	if !c.DevelopmentEnabledByDoubleKey() {
		return "", "", ErrAutoridadesAutorizacionRRHHIncompletas
	}
	fuente = strings.TrimSpace(os.Getenv(EnvAutorizacionFuenteDatabaseURL))
	motivos = strings.TrimSpace(os.Getenv(EnvAutorizacionMotivosEvaluadorDatabaseURL))
	if fuente == "" || motivos == "" {
		return "", "", ErrAutoridadesAutorizacionRRHHIncompletas
	}
	for _, dsn := range []string{fuente, motivos} {
		parsed, parseErr := pgconn.ParseConfig(dsn)
		if parseErr != nil || parsed == nil || parsed.User == "" {
			return "", "", ErrAutoridadesAutorizacionRRHHIncompletas
		}
	}
	if conexionPostgreSQLComparteLogin(fuente, motivos) {
		return "", "", ErrAutoridadesAutorizacionRRHHNoSeparadas
	}
	for _, previo := range c.dsnsPostgreSQLConfigurados() {
		if conexionPostgreSQLComparteLogin(fuente, previo) || conexionPostgreSQLComparteLogin(motivos, previo) {
			return "", "", ErrAutoridadesAutorizacionRRHHNoSeparadas
		}
	}
	// Auditoría conserva su contrato y ciclo de vida propios. Mientras ambos
	// montajes no compartan explícitamente el mismo pool, un LOGIN no puede
	// aparecer en las dos configuraciones por accidente.
	for _, previo := range []string{os.Getenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL),
		os.Getenv(EnvRRHHAuditoriaMotivosDatabaseURL)} {
		if conexionPostgreSQLComparteLogin(fuente, previo) || conexionPostgreSQLComparteLogin(motivos, previo) {
			return "", "", ErrAutoridadesAutorizacionRRHHNoSeparadas
		}
	}
	return fuente, motivos, nil
}
