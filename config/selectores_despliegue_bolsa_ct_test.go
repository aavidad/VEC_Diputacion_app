package config

import (
	"errors"
	"testing"
)

func TestSelectoresPortalCandidatoYSeguimientoCese(t *testing.T) {
	casos := []struct {
		nombre                string
		fijar                 func(*Config, string)
		activo                func(Config) (bool, error)
		errSelector, errDoble error
	}{
		{"portal", func(c *Config, v string) { c.BolsaPortalCandidatoEnabled = v }, Config.BolsaPortalCandidatoDesarrolloActivo,
			ErrConfiguracionBolsaPortalCandidatoSelector, ErrConfiguracionBolsaPortalCandidatoActivacion},
		{"cese", func(c *Config, v string) { c.CTSeguimientoCeseEnabled = v }, Config.CTSeguimientoCeseDesarrolloActivo,
			ErrConfiguracionCTSeguimientoCeseSelector, ErrConfiguracionCTSeguimientoCeseActivacion},
	}
	for _, caso := range casos {
		for _, valor := range []string{"", "false", " false "} {
			var c Config
			caso.fijar(&c, valor)
			if activo, err := caso.activo(c); activo || err != nil {
				t.Fatalf("%s %q: apagado por defecto: %t %v", caso.nombre, valor, activo, err)
			}
		}
		var c Config
		caso.fijar(&c, "si")
		if _, err := caso.activo(c); !errors.Is(err, caso.errSelector) {
			t.Fatalf("%s: selector inválido: %v", caso.nombre, err)
		}
		caso.fijar(&c, "true")
		if _, err := caso.activo(c); !errors.Is(err, caso.errDoble) {
			t.Fatalf("%s: sin doble llave: %v", caso.nombre, err)
		}
		c.ExecutionProfile, c.AuthMode, c.DevelopmentGuard = ExecutionProfileDevelopment, AuthModeDevelopment, DevelopmentGuardAcknowledgement
		if activo, err := caso.activo(c); !activo || err != nil {
			t.Fatalf("%s: con doble llave: %t %v", caso.nombre, activo, err)
		}
	}
}
