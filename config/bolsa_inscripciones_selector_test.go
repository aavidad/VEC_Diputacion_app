package config

import (
	"errors"
	"testing"
)

func TestInscripcionesRequiereSelectorYDSNLector(t *testing.T) {
	var c Config
	if activo, err := c.BolsaInscripcionesActivo(); activo || err != nil {
		t.Fatalf("sin selector: %v %v", activo, err)
	}
	c.BolsaInscripcionesEnabled = "otro"
	if activo, err := c.BolsaInscripcionesActivo(); activo || !errors.Is(err, ErrBolsaInscripcionesSelector) {
		t.Fatalf("selector invalido: %v %v", activo, err)
	}
	c.BolsaInscripcionesEnabled = "true"
	if activo, err := c.BolsaInscripcionesActivo(); activo || err == nil {
		t.Fatalf("sin lector: %v %v", activo, err)
	}
	c.ExecutionProfile, c.AuthMode, c.DevelopmentGuard = ExecutionProfileDevelopment, AuthModeDevelopment, DevelopmentGuardAcknowledgement
	c.BolsaInscripcionesLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://externo@localhost/vec"}
	if activo, err := c.BolsaInscripcionesActivo(); activo || err == nil {
		t.Fatalf("sin empleado/RRHH: %v %v", activo, err)
	}
	c.BolsaInscripcionesEmpleadoLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://empleado@localhost/vec"}
	if activo, err := c.BolsaInscripcionesActivo(); activo || err == nil {
		t.Fatalf("sin RRHH: %v %v", activo, err)
	}
	c.BolsaInscripcionesRRHHLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://rrhh@localhost/vec"}
	if activo, err := c.BolsaInscripcionesActivo(); !activo || err != nil {
		t.Fatalf("tres lectores: %v %v", activo, err)
	}
}
