package config

import (
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// EnvAuditoriaConsultaCatalogoPath permite editar el ejemplo publicado para
// RRHH. Solo se lee cuando la capacidad está activada con doble llave.
const EnvAuditoriaConsultaCatalogoPath = "VEC_AUDITORIA_CONSULTA_CATALOGO_PATH"

const (
	EnvAuditoriaConsultaExpedienteCT              = "VEC_AUDITORIA_CONSULTA_EXPEDIENTE_CT"
	EnvAuditoriaConsultaExpedienteBolsa           = "VEC_AUDITORIA_CONSULTA_EXPEDIENTE_BOLSA"
	EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL = "VEC_RRHH_AUDITORIA_FUENTE_AUTORIZACION_DATABASE_URL"
	EnvRRHHAuditoriaMotivosDatabaseURL            = "VEC_RRHH_AUDITORIA_MOTIVOS_DATABASE_URL"
	RolFuenteAutorizacionAuditoriaDesarrollo      = "vec_autorizacion_fuente"
	RolMotivosAuditoriaDesarrollo                 = "vec_autorizacion_motivos_evaluador"
)

const RutaCatalogoAuditoriaConsultaEjemplo = "data/demo/reglas/auditoria_consulta.ejemplo.demo.json"

var ErrCatalogoAuditoriaConsultaFueraDesarrollo = errors.New("config: catalogo de auditoria fuera de desarrollo")
var ErrExpedientesAuditoriaConsultaInvalidos = errors.New("config: expedientes de auditoria no disponibles")
var ErrFuenteAutorizacionAuditoriaIncompleta = errors.New("config: falta la fuente PostgreSQL de autorizacion de auditoria")
var ErrFuenteAutorizacionAuditoriaNoSeparada = errors.New("config: la fuente PostgreSQL de autorizacion de auditoria comparte LOGIN")
var ErrMotivosAuditoriaIncompletos = errors.New("config: falta el resolutor PostgreSQL de motivos de auditoria")
var ErrMotivosAuditoriaNoSeparados = errors.New("config: el resolutor PostgreSQL de motivos de auditoria comparte LOGIN")

// RutaCatalogoAuditoriaConsultaDesarrollo no activa rutas ni concede permisos.
// Una ruta declarada fuera de desarrollo es un error, incluso si el selector
// de la capacidad está apagado, para evitar que el despliegue la ignore.
func (c Config) RutaCatalogoAuditoriaConsultaDesarrollo() (string, error) {
	ruta, declarada := os.LookupEnv(EnvAuditoriaConsultaCatalogoPath)
	if !c.DevelopmentEnabledByDoubleKey() {
		if declarada {
			return "", ErrCatalogoAuditoriaConsultaFueraDesarrollo
		}
		return "", ErrCatalogoAuditoriaConsultaFueraDesarrollo
	}
	if !declarada {
		return RutaCatalogoAuditoriaConsultaEjemplo, nil
	}
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return "", ErrCatalogoAuditoriaConsultaFueraDesarrollo
	}
	return ruta, nil
}

// ExpedientesAuditoriaConsultaDesarrollo exige dos referencias privadas
// exactas cuando la raíz activa la consulta. No tiene valores por defecto y
// el error no contiene ninguna de las referencias recibidas.
func (c Config) ExpedientesAuditoriaConsultaDesarrollo() (ct, bolsa string, err error) {
	if !c.DevelopmentEnabledByDoubleKey() {
		return "", "", ErrExpedientesAuditoriaConsultaInvalidos
	}
	ct, presenteCT := os.LookupEnv(EnvAuditoriaConsultaExpedienteCT)
	bolsa, presenteBolsa := os.LookupEnv(EnvAuditoriaConsultaExpedienteBolsa)
	if !presenteCT || !presenteBolsa || !ctdomain.ReferenciaOpacaValida(ct) ||
		!ctdomain.ReferenciaOpacaValida(bolsa) || ct == bolsa {
		return "", "", ErrExpedientesAuditoriaConsultaInvalidos
	}
	return ct, bolsa, nil
}

// DSNFuenteAutorizacionAuditoriaDesarrollo reserva un LOGIN miembro únicamente
// de vec_autorizacion_fuente para obtener la instantánea central V3. La raíz
// comprueba esa membresía al abrir el pool. El DSN no puede reutilizar los
// logins de consulta CT, ejecución Bolsa, registro V3 ni motivos; la
// comprobación también incluye cualquier otra conexión VEC configurada.
// Solo se invoca cuando el selector explícito de Auditoría está activo.
func (c Config) DSNFuenteAutorizacionAuditoriaDesarrollo() (string, error) {
	c = c.Normalize()
	if !c.DevelopmentEnabledByDoubleKey() {
		return "", ErrFuenteAutorizacionAuditoriaIncompleta
	}
	dsn := strings.TrimSpace(os.Getenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL))
	if dsn == "" {
		return "", ErrFuenteAutorizacionAuditoriaIncompleta
	}
	configuracion, err := pgconn.ParseConfig(dsn)
	if err != nil || configuracion.User == "" {
		return "", ErrFuenteAutorizacionAuditoriaIncompleta
	}
	if _, _, err := c.ContratacionTemporalPostgreSQL.DSNConsultasRRHHSeparados(); err != nil {
		return "", ErrFuenteAutorizacionAuditoriaIncompleta
	}
	if _, err := c.ContratacionTemporalPostgreSQL.DSNRegistroAutorizacionSeparado(); err != nil {
		return "", ErrFuenteAutorizacionAuditoriaIncompleta
	}
	if _, _, _, err := c.BolsaBorradoresPostgreSQL.DSNSeparados(); err != nil {
		return "", ErrFuenteAutorizacionAuditoriaIncompleta
	}
	for _, previo := range c.dsnsPostgreSQLConfigurados() {
		if conexionPostgreSQLComparteLogin(dsn, previo) {
			return "", ErrFuenteAutorizacionAuditoriaNoSeparada
		}
	}
	if conexionPostgreSQLComparteLogin(dsn, os.Getenv(EnvRRHHAuditoriaMotivosDatabaseURL)) {
		return "", ErrFuenteAutorizacionAuditoriaNoSeparada
	}
	return dsn, nil
}

// DSNMotivosAuditoriaDesarrollo usa el rol evaluador V2 propio. El resolutor
// de motivos RRHH configurado para otras consultas tiene otro rol y no se
// reutiliza para ampliar accidentalmente sus privilegios.
func (c Config) DSNMotivosAuditoriaDesarrollo() (string, error) {
	c = c.Normalize()
	if !c.DevelopmentEnabledByDoubleKey() {
		return "", ErrMotivosAuditoriaIncompletos
	}
	dsn := strings.TrimSpace(os.Getenv(EnvRRHHAuditoriaMotivosDatabaseURL))
	if dsn == "" {
		return "", ErrMotivosAuditoriaIncompletos
	}
	configuracion, err := pgconn.ParseConfig(dsn)
	if err != nil || configuracion.User == "" {
		return "", ErrMotivosAuditoriaIncompletos
	}
	for _, previo := range c.dsnsPostgreSQLConfigurados() {
		if conexionPostgreSQLComparteLogin(dsn, previo) {
			return "", ErrMotivosAuditoriaNoSeparados
		}
	}
	if conexionPostgreSQLComparteLogin(dsn, os.Getenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL)) {
		return "", ErrMotivosAuditoriaNoSeparados
	}
	return dsn, nil
}
