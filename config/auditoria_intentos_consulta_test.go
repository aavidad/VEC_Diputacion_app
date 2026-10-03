package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestConexionIntentosConsultaExigeOrigenYLoginSeparado(t *testing.T) {
	c := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	for _, nombre := range []string{EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL, EnvRRHHAuditoriaMotivosDatabaseURL, EnvRRHHAuditoriaFronteraDatabaseURL} {
		t.Setenv(nombre, "")
	}
	t.Setenv(EnvAuditoriaIntentosDatabaseURL, "postgres://login_intentos@localhost/vec?sslmode=require")
	t.Setenv(EnvAuditoriaIntentosProceso, "vec-rrhh")
	t.Setenv(EnvAuditoriaIntentosCanal, "interna_corporativa")
	t.Setenv(EnvAuditoriaIntentosPlazo, "2s")
	dsn, proceso, canal, plazo, err := c.ConexionIntentosAuditoriaConsulta()
	if err != nil || dsn == "" || proceso != "vec-rrhh" || canal != "interna_corporativa" || plazo != 2*time.Second {
		t.Fatal("configuración explícita no admitida")
	}
	for _, caso := range []struct{ nombre, valor string }{
		{EnvAuditoriaIntentosDatabaseURL, ""},
		{EnvAuditoriaIntentosDatabaseURL, "postgres://%malformado"},
		{EnvAuditoriaIntentosProceso, ""},
		{EnvAuditoriaIntentosProceso, "proceso con datos"},
		{EnvAuditoriaIntentosCanal, "administracion_privilegiada"},
		{EnvAuditoriaIntentosPlazo, "0s"},
		{EnvAuditoriaIntentosPlazo, "11s"},
		{EnvAuditoriaIntentosPlazo, "plazo no válido"},
	} {
		t.Run(caso.nombre+":"+caso.valor, func(t *testing.T) {
			t.Setenv(caso.nombre, caso.valor)
			dsn, proceso, canal, plazo, err := c.ConexionIntentosAuditoriaConsulta()
			if dsn != "" || proceso != "" || canal != "" || plazo != 0 || !errors.Is(err, ErrConexionIntentosAuditoria) ||
				(caso.valor != "" && strings.Contains(err.Error(), caso.valor)) {
				t.Fatal("configuración incompleta admitida o reflejada")
			}
		})
	}
	for _, nombre := range []string{EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL, EnvRRHHAuditoriaMotivosDatabaseURL, EnvRRHHAuditoriaFronteraDatabaseURL} {
		t.Run("separación:"+nombre, func(t *testing.T) {
			t.Setenv(nombre, "host=localhost user=login_intentos dbname=otra sslmode=require")
			if _, _, _, _, err := c.ConexionIntentosAuditoriaConsulta(); !errors.Is(err, ErrConexionIntentosAuditoria) {
				t.Fatal("LOGIN compartido admitido con otro formato de conexión")
			}
		})
	}
	c.ContratacionTemporalPostgreSQL.dsnEjecucion = "postgres://login_intentos@localhost/negocio?sslmode=require"
	if _, _, _, _, err := c.ConexionIntentosAuditoriaConsulta(); !errors.Is(err, ErrConexionIntentosAuditoria) {
		t.Fatal("LOGIN de negocio admitido")
	}
	if _, _, _, _, err := (Config{ExecutionProfile: ExecutionProfileProduction}).ConexionIntentosAuditoriaConsulta(); !errors.Is(err, ErrConexionIntentosAuditoria) {
		t.Fatal("consulta de desarrollo admitida fuera de su composición")
	}
}
