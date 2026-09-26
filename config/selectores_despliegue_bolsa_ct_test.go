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
		catalogos             []func(*Config)
	}{
		{"portal", func(c *Config, v string) { c.BolsaPortalCandidatoEnabled = v }, Config.BolsaPortalCandidatoDesarrolloActivo,
			ErrConfiguracionBolsaPortalCandidatoSelector, ErrConfiguracionBolsaPortalCandidatoActivacion,
			[]func(*Config){func(c *Config) { c.ReglasEjemplo.BolsaSourcePath = "bolsa.json" }}},
		{"cese", func(c *Config, v string) { c.CTSeguimientoCeseEnabled = v }, Config.CTSeguimientoCeseDesarrolloActivo,
			ErrConfiguracionCTSeguimientoCeseSelector, ErrConfiguracionCTSeguimientoCeseActivacion,
			[]func(*Config){
				func(c *Config) { c.ReglasEjemplo.CTSourcePath = "ct.json" },
				func(c *Config) { c.ReglasEjemplo.CausasCeseSourcePath = " causas.json " },
				func(c *Config) { c.CTAnalisisMotivosSourcePath = "motivos.json" },
			}},
		{"cancelacion", func(c *Config, v string) { c.CTCancelacionEnabled = v }, Config.CTCancelacionDesarrolloActivo,
			ErrConfiguracionCTCancelacionSelector, ErrConfiguracionCTCancelacionActivacion,
			[]func(*Config){
				func(c *Config) { c.ReglasEjemplo.CTSourcePath = "ct.json" },
				func(c *Config) { c.ReglasEjemplo.MotivosCancelacionSourcePath = " motivos_cancelacion.json " },
			}},
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
		// Pedido sin alguno de sus catálogos, el arranque se detiene en lugar
		// de dejar la capacidad sin montar en silencio.
		for _, fijarCatalogo := range caso.catalogos {
			if activo, err := caso.activo(c); activo || !errors.Is(err, caso.errDoble) {
				t.Fatalf("%s: sin todos sus catálogos: %t %v", caso.nombre, activo, err)
			}
			fijarCatalogo(&c)
		}
		if activo, err := caso.activo(c); !activo || err != nil {
			t.Fatalf("%s: con doble llave: %t %v", caso.nombre, activo, err)
		}
	}
}
