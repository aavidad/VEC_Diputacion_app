package config

import (
	"errors"
	"testing"
)

func TestInscripcionesApagadasPorDefectoYSelectorInvalido(t *testing.T) {
	var c Config
	for _, activo := range []func() (bool, error){c.BolsaInscripcionesExternoActivo, c.BolsaInscripcionesInternoActivo} {
		if a, err := activo(); a || err != nil {
			t.Fatalf("sin selector debe quedar apagado: %v %v", a, err)
		}
	}
	c.BolsaInscripcionesEnabled = "otro"
	if activo, err := c.BolsaInscripcionesExternoActivo(); activo || !errors.Is(err, ErrBolsaInscripcionesSelector) {
		t.Fatalf("selector invalido: %v %v", activo, err)
	}
	c.BolsaInscripcionesEnabled = "true"
	if activo, err := c.BolsaInscripcionesExternoActivo(); activo || err == nil {
		t.Fatalf("encendido fuera del perfil de desarrollo: %v %v", activo, err)
	}
}

func TestInscripcionesActivanCadaSuperficieSinCredencialesAjenas(t *testing.T) {
	c := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement, BolsaInscripcionesEnabled: "true"}
	c.BolsaInscripcionesLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://externo@localhost/vec"}
	if activo, err := c.BolsaInscripcionesExternoActivo(); !activo || err != nil {
		t.Fatalf("externo sin DSN interno: %v %v", activo, err)
	}
	if activo, err := c.BolsaInscripcionesInternoActivo(); activo || !errors.Is(err, ErrBolsaInscripcionesActivacion) {
		t.Fatalf("interno sin su DSN: %v %v", activo, err)
	}
	c.BolsaInscripcionesLectorPostgreSQL = ConfiguracionPostgreSQLExterna{}
	c.BolsaInscripcionesRRHHLectorPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: "postgres://rrhh@localhost/vec"}
	if activo, err := c.BolsaInscripcionesInternoActivo(); !activo || err != nil {
		t.Fatalf("interno sin DSN externo: %v %v", activo, err)
	}
	if activo, err := c.BolsaInscripcionesExternoActivo(); activo || !errors.Is(err, ErrBolsaInscripcionesActivacion) {
		t.Fatalf("externo sin su DSN: %v %v", activo, err)
	}
}
