package publicatransitoria

import (
	"errors"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/publica"
)

func TestNuevaAPIDirectaRechazaAutenticacion(t *testing.T) {
	for _, modo := range []string{
		config.AuthModeFake,
		config.AuthModeTrustedHeaders,
		config.AuthModeDevelopment,
	} {
		t.Run(modo, func(t *testing.T) {
			api, err := NuevaAPI(config.Config{
				ExecutionProfile: config.ExecutionProfileDevelopment,
				AuthMode:         modo,
			})
			if api != nil || !errors.Is(err, publica.ErrAutenticacionNoAdmitida) {
				t.Fatalf("modo %q = (%v, %v); debe rechazarse", modo, api, err)
			}
		})
	}
}

func TestNuevaAPIProductivaFallaAntesDeAbrirFicheros(t *testing.T) {
	api, err := NuevaAPI(config.Config{
		ExecutionProfile:      config.ExecutionProfileProduction,
		AuthMode:              config.AuthModeDisabled,
		BolsaPublicSourcePath: "/ruta/que/no/existe.json",
	})
	if api != nil || !errors.Is(err, publica.ErrComposicionProductivaNoDisponible) {
		t.Fatalf("API productiva = (%v, %v)", api, err)
	}
}

func TestNuevaAPIRechazaPerfilesNoAutoritativosSinTodasSusGuardas(t *testing.T) {
	pruebas := []struct {
		nombre string
		cfg    config.Config
	}{
		{
			nombre: "presentacion sin guardas",
			cfg: config.Config{
				ExecutionProfile: config.ExecutionProfileRRHHPresentation,
				AuthMode:         config.AuthModeDisabled,
			},
		},
		{
			nombre: "desarrollo sin guarda ni material",
			cfg: config.Config{
				ExecutionProfile: config.ExecutionProfileDevelopment,
				AuthMode:         config.AuthModeDisabled,
			},
		},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			api, err := NuevaAPI(prueba.cfg)
			if api != nil || !errors.Is(err, ErrActivacionNoAutoritativaInvalida) {
				t.Fatalf("perfil sin guardas = (%v, %v)", api, err)
			}
		})
	}
}
