package config

import (
	"errors"
	"strings"
	"testing"
)

func TestDSNAutoridadesAutorizacionRRHHExigeDosLoginNominales(t *testing.T) {
	activo := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement,
		ContratacionTemporalPostgreSQL: ConfiguracionPostgreSQLContratacionTemporal{
			dsnEjecucion: "postgres://ejecutor:secreto@localhost/vec?sslmode=require"}}
	const fuente = "postgres://fuente:secreto@localhost/vec?sslmode=require"
	const motivos = "postgres://motivos:secreto@localhost/vec?sslmode=require"
	t.Setenv(EnvAutorizacionFuenteDatabaseURL, fuente)
	t.Setenv(EnvAutorizacionMotivosEvaluadorDatabaseURL, motivos)
	t.Setenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL, "")
	t.Setenv(EnvRRHHAuditoriaMotivosDatabaseURL, "")
	if a, b, err := activo.DSNAutoridadesAutorizacionRRHH(); err != nil || a != fuente || b != motivos {
		t.Fatalf("dos LOGIN distintos rechazados: %v", err)
	}
	for _, caso := range []struct {
		nombre, a, b string
		esperado     error
	}{
		{"sin_fuente", "", motivos, ErrAutoridadesAutorizacionRRHHIncompletas},
		{"sin_motivos", fuente, "", ErrAutoridadesAutorizacionRRHHIncompletas},
		{"dsn_invalido", "postgres://mal%zz", motivos, ErrAutoridadesAutorizacionRRHHIncompletas},
		{"mismo_login_otra_base", fuente, "postgres://fuente:otra@localhost/otra?sslmode=require", ErrAutoridadesAutorizacionRRHHNoSeparadas},
		{"ejecutor_reutilizado", fuente, "postgres://ejecutor:otra@localhost/otra?sslmode=require", ErrAutoridadesAutorizacionRRHHNoSeparadas},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Setenv(EnvAutorizacionFuenteDatabaseURL, caso.a)
			t.Setenv(EnvAutorizacionMotivosEvaluadorDatabaseURL, caso.b)
			_, _, err := activo.DSNAutoridadesAutorizacionRRHH()
			if !errors.Is(err, caso.esperado) || strings.Contains(err.Error(), "secreto") || strings.Contains(err.Error(), "otra") {
				t.Fatalf("fallo sin redacción: %v", err)
			}
		})
	}
	if _, _, err := (Config{ExecutionProfile: ExecutionProfileProduction}).DSNAutoridadesAutorizacionRRHH(); !errors.Is(err, ErrAutoridadesAutorizacionRRHHIncompletas) {
		t.Fatalf("fuera de doble llave = %v", err)
	}
	t.Setenv(EnvAutorizacionFuenteDatabaseURL, fuente)
	t.Setenv(EnvAutorizacionMotivosEvaluadorDatabaseURL, motivos)
	t.Setenv(EnvRRHHAuditoriaFuenteAutorizacionDatabaseURL, "postgres://fuente:otra@localhost/otra?sslmode=require")
	if _, _, err := activo.DSNAutoridadesAutorizacionRRHH(); !errors.Is(err, ErrAutoridadesAutorizacionRRHHNoSeparadas) {
		t.Fatalf("LOGIN de Auditoría reutilizado sin pool compartido = %v", err)
	}
}
