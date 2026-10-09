package config

import (
	"errors"
	"strings"
	"testing"
)

func TestInscripcionesLectorExigeLoginSeparadoYRedactaDSN(t *testing.T) {
	t.Setenv(EnvExternoBolsaInscripcionesLectorDatabaseURL, "")
	if _, err := Load().DSNBolsaInscripcionesLectorSeparado(); !errors.Is(err, ErrBolsaInscripcionesLectorIncompleto) {
		t.Fatalf("ausente: %v", err)
	}
	secreto := "clave-de-prueba-lector"
	lector := "postgres://vec_bolsa_inscripciones_lector:" + secreto + "@localhost/vec?sslmode=require"
	t.Setenv(EnvExternoBolsaInscripcionesLectorDatabaseURL, lector)
	t.Setenv(EnvExternoBolsaDatabaseURL, "postgres://vec_externo_bolsa_desarrollo:otra@localhost/vec?sslmode=require")
	c := Load()
	dsn, err := c.DSNBolsaInscripcionesLectorSeparado()
	if err != nil || dsn != lector {
		t.Fatalf("lector: %v", err)
	}
	if strings.Contains(c.BolsaInscripcionesLectorPostgreSQL.String(), secreto) {
		t.Fatal("DSN expuesto")
	}
	t.Setenv(EnvExternoBolsaDatabaseURL, "postgres://vec_bolsa_inscripciones_lector:otra@localhost/vec?sslmode=require")
	if _, err := Load().DSNBolsaInscripcionesLectorSeparado(); !errors.Is(err, ErrBolsaInscripcionesLectorNoSeparado) {
		t.Fatalf("login compartido: %v", err)
	}
}

func TestInscripcionesLectorRRHHNoComparteLogin(t *testing.T) {
	c := Config{}
	if _, err := c.DSNBolsaInscripcionesLectorRRHHSeparado(); !errors.Is(err, ErrBolsaInscripcionesLectorIncompleto) {
		t.Fatalf("ausente: %v", err)
	}
	c.BolsaInscripcionesRRHHLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://rrhh@localhost/vec"}
	if _, err := c.DSNBolsaInscripcionesLectorRRHHSeparado(); err != nil {
		t.Fatalf("lector RRHH aislado: %v", err)
	}
	c.BolsaInscripcionesLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://rrhh@localhost/vec"}
	if _, err := c.DSNBolsaInscripcionesLectorRRHHSeparado(); !errors.Is(err, ErrBolsaInscripcionesLectorNoSeparado) {
		t.Fatalf("LOGIN compartido con el lector de aspirante: %v", err)
	}
}
