package config

import (
	"errors"
	"strings"
	"testing"
)

func TestInscripcionesLectorExigeLoginSeparadoYRedactaDSN(t *testing.T) {
	t.Setenv(EnvBolsaInscripcionesLectorDatabaseURL, "")
	if _, err := Load().DSNBolsaInscripcionesLectorSeparado(); !errors.Is(err, ErrBolsaInscripcionesLectorIncompleto) {
		t.Fatalf("ausente: %v", err)
	}
	secreto := "clave-de-prueba-lector"
	lector := "postgres://vec_bolsa_inscripciones_lector:" + secreto + "@localhost/vec?sslmode=require"
	t.Setenv(EnvBolsaInscripcionesLectorDatabaseURL, lector)
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
	t.Setenv(EnvExternoBolsaDatabaseURL, "postgres://vec_externo_bolsa_desarrollo:otra@localhost/vec?sslmode=require")
	t.Setenv(EnvBolsaInscripcionesEmpleadoLectorDatabaseURL, "postgres://vec_bolsa_inscripciones_empleado_lector:prueba@localhost/vec?sslmode=require")
	t.Setenv(EnvBolsaInscripcionesRRHHLectorDatabaseURL, "postgres://vec_bolsa_inscripciones_rrhh_lector:prueba@localhost/vec?sslmode=require")
	if _, _, _, err := Load().DSNBolsaInscripcionesLectoresSeparados(); err != nil {
		t.Fatalf("tres lectores separados: %v", err)
	}
	t.Setenv(EnvBolsaInscripcionesRRHHLectorDatabaseURL, "postgres://vec_bolsa_inscripciones_empleado_lector:otra@localhost/vec?sslmode=require")
	if _, _, _, err := Load().DSNBolsaInscripcionesLectoresSeparados(); !errors.Is(err, ErrBolsaInscripcionesLectorNoSeparado) {
		t.Fatalf("lector RRHH comparte LOGIN: %v", err)
	}
}

func TestInscripcionesLectoresInternosNoRequierenDSNExterno(t *testing.T) {
	c := Config{}
	c.BolsaInscripcionesEmpleadoLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://empleado@localhost/vec"}
	c.BolsaInscripcionesRRHHLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://rrhh@localhost/vec"}
	if _, _, err := c.DSNBolsaInscripcionesLectoresInternosSeparados(); err != nil {
		t.Fatalf("lectores internos aislados: %v", err)
	}
	c.BolsaInscripcionesRRHHLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://empleado@localhost/vec"}
	if _, _, err := c.DSNBolsaInscripcionesLectoresInternosSeparados(); !errors.Is(err, ErrBolsaInscripcionesLectorNoSeparado) {
		t.Fatalf("LOGIN interno compartido: %v", err)
	}
}
