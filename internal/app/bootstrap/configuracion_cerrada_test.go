package bootstrap

import (
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestRaicesRechazanSelectoresDesconocidosSinReflejarSuValor(t *testing.T) {
	tipos := []struct {
		nombre  string
		error   error
		alterar func(*config.Config)
	}{
		{nombre: "perfil", error: ErrPerfilEjecucionDesconocido, alterar: func(cfg *config.Config) {
			cfg.ExecutionProfile = "perfil-SECRETO\ninyectado"
		}},
		{nombre: "autenticacion", error: ErrModoAutenticacionDesconocido, alterar: func(cfg *config.Config) {
			cfg.AuthMode = "auth-SECRETO\ninyectado"
		}},
		{nombre: "almacenamiento", error: ErrModoAlmacenamientoDesconocido, alterar: func(cfg *config.Config) {
			cfg.StorageMode = "store-SECRETO\ninyectado"
		}},
	}

	raices := map[string]func(config.Config) error{
		"servidor integrado": func(cfg config.Config) error {
			_, err := NewHTTPServerWithConfig(cfg)
			return err
		},
		"servidor publico": func(cfg config.Config) error {
			_, err := NewHTTPServerPublicoWithConfig(cfg)
			return err
		},
		"API integrada": func(cfg config.Config) error {
			_, err := NewDemoAPIWithConfig(cfg)
			return err
		},
		"API shell VEC": func(cfg config.Config) error {
			_, err := NewVECShellAPIWithConfig(cfg)
			return err
		},
		"API publica": func(cfg config.Config) error {
			_, err := NewAPIPublicaBolsaWithConfig(cfg)
			return err
		},
	}

	for _, tipo := range tipos {
		t.Run(tipo.nombre, func(t *testing.T) {
			base := configurarFuentesProduccionPrueba(t, config.Config{
				ExecutionProfile: config.ExecutionProfileProduction,
				AuthMode:         config.AuthModeDisabled,
				StorageMode:      config.StorageModeLocalDurable,
			})
			tipo.alterar(&base)
			for nombre, construir := range raices {
				t.Run(nombre, func(t *testing.T) {
					err := construir(base)
					if !errors.Is(err, tipo.error) {
						t.Fatalf("error = %v; se esperaba %v", err, tipo.error)
					}
					if strings.Contains(strings.ToLower(err.Error()), "secreto") || strings.Contains(err.Error(), "\n") {
						t.Fatalf("el error reflejo el valor ambiental: %q", err)
					}
				})
			}
		})
	}
}

func TestTodasLasRaicesExportadasFallanCerradasEnProduccion(t *testing.T) {
	base := configurarFuentesProduccionPrueba(t, config.Config{
		ExecutionProfile: config.ExecutionProfileProduction,
		AuthMode:         config.AuthModeDisabled,
		StorageMode:      config.StorageModeLocalDurable,
		DataDir:          t.TempDir(),
	})
	// Las fuentes contienen datos DEMO pero han sido renombradas: el cierre no
	// puede depender de una extension o de una convencion de nombre.
	if strings.Contains(base.BolsaPublicSourcePath, ".demo.json") ||
		strings.Contains(base.BolsaCategoriesSourcePath, ".demo.json") {
		t.Fatal("la prueba no ejercita el intento de evasion por renombrado")
	}

	for nombre, construir := range map[string]func(config.Config) error{
		"servidor integrado": func(cfg config.Config) error {
			_, err := NewHTTPServerWithConfig(cfg)
			return err
		},
		"servidor publico anonimo": func(cfg config.Config) error {
			_, err := NewHTTPServerPublicoWithConfig(cfg)
			return err
		},
		"API integrada embebida": func(cfg config.Config) error {
			_, err := NewDemoAPIWithConfig(cfg)
			return err
		},
		"API shell VEC embebida": func(cfg config.Config) error {
			_, err := NewVECShellAPIWithConfig(cfg)
			return err
		},
		"API publica embebida": func(cfg config.Config) error {
			_, err := NewAPIPublicaBolsaWithConfig(cfg)
			return err
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			esperado := error(ErrComposicionProductivaNoDisponible)
			if nombre == "servidor publico anonimo" {
				// La raiz publica no puede aceptar ni siquiera una copia
				// renombrada de las fuentes JSON de demostracion. Ese cierre se
				// evalua antes de intentar abrir PostgreSQL.
				esperado = ErrActivacionDesarrolloInvalida
			}
			if err := construir(base); !errors.Is(err, esperado) {
				t.Fatalf("raiz productiva aceptada o error incorrecto: %v", err)
			}
		})
	}
}

func TestSelectorGlobalNoAfirmaDurabilidadProductiva(t *testing.T) {
	for _, almacenamiento := range []string{config.StorageModeMemory, config.StorageModeLocalDurable} {
		t.Run(almacenamiento, func(t *testing.T) {
			cfg := configurarFuentesProduccionPrueba(t, config.Config{
				ExecutionProfile: config.ExecutionProfileProduction,
				AuthMode:         config.AuthModeFake,
				StorageMode:      almacenamiento,
			})
			if _, err := NewHTTPServerWithConfig(cfg); !errors.Is(err, ErrComposicionProductivaNoDisponible) {
				t.Fatalf("selector %s eludio el cierre: %v", almacenamiento, err)
			}
		})
	}
}
