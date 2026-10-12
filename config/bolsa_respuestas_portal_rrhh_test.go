package config

import (
	"errors"
	"strings"
	"testing"
)

func TestDSNRespuestasBolsaNoReutilizaLOGINPrivilegiado(t *testing.T) {
	dsn := func(login string) string {
		return "postgres://" + login + ":secreto-privado@localhost/vec?sslmode=require"
	}
	c := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement,
		ContratacionTemporalPostgreSQL: ConfiguracionPostgreSQLContratacionTemporal{
			dsnGobierno: dsn("gobierno"), dsnRegistroAutorizacion: dsn("registro_v3"),
		},
	}
	if _, err := c.DSNFuenteAutorizacionRespuestasBolsa(); !errors.Is(err, ErrFuenteAutorizacionRespuestasBolsaIncompleta) {
		t.Fatalf("DSN ausente admitido: %v", err)
	}
	const fuente = "postgres://fuente_nominal:secreto-privado@localhost/vec?sslmode=require"
	t.Setenv(EnvBolsaB103FuenteAutorizacionDatabaseURL, fuente)
	if valor, err := c.DSNFuenteAutorizacionRespuestasBolsa(); err != nil || valor != fuente {
		t.Fatalf("DSN nominal rechazado: %t, %v", valor == fuente, err)
	}
	t.Setenv(EnvBolsaB103FuenteAutorizacionDatabaseURL, "postgres://gobierno:otra-clave@localhost/vec?sslmode=require")
	if _, err := c.DSNFuenteAutorizacionRespuestasBolsa(); !errors.Is(err, ErrFuenteAutorizacionRespuestasBolsaIncompleta) || strings.Contains(err.Error(), "otra-clave") {
		t.Fatalf("LOGIN de gobierno reutilizado o secreto expuesto: %v", err)
	}
}
