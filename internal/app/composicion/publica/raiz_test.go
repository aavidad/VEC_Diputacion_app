package publica

import (
	"errors"
	"testing"

	"vec-diputacion-granada/config"
)

func TestNuevoServidorExigeAutenticacionDeshabilitada(t *testing.T) {
	for _, modo := range []string{
		config.AuthModeFake,
		config.AuthModeTrustedHeaders,
		config.AuthModeDevelopment,
	} {
		t.Run(modo, func(t *testing.T) {
			servidor, err := NuevoServidor(Configuracion{
				PerfilEjecucion:         config.ExecutionProfileProduction,
				AutenticacionSolicitada: modo,
			})
			if servidor != nil || !errors.Is(err, ErrAutenticacionNoAdmitida) {
				t.Fatalf("modo %q = (%v, %v); debe rechazarse", modo, servidor, err)
			}
		})
	}
}

func TestNuevoServidorPermaneceCerradoSinFuenteAutoritativa(t *testing.T) {
	servidor, err := NuevoServidor(Configuracion{
		PerfilEjecucion:         config.ExecutionProfileProduction,
		AutenticacionSolicitada: config.AuthModeDisabled,
	})
	if servidor != nil || !errors.Is(err, ErrComposicionProductivaNoDisponible) {
		t.Fatalf("raiz productiva = (%v, %v)", servidor, err)
	}
}

func TestNuevoServidorRechazaSelectoresNoProductivos(t *testing.T) {
	pruebas := []struct {
		nombre string
		cfg    Configuracion
		error  error
	}{
		{
			nombre: "desarrollo",
			cfg: Configuracion{
				PerfilEjecucion:         config.ExecutionProfileDevelopment,
				AutenticacionSolicitada: config.AuthModeDisabled,
			},
			error: ErrActivacionDesarrolloInvalida,
		},
		{
			nombre: "presentacion",
			cfg: Configuracion{
				PerfilEjecucion:         config.ExecutionProfileRRHHPresentation,
				AutenticacionSolicitada: config.AuthModeDisabled,
			},
			error: ErrSelectoresPresentacionProhibidos,
		},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			servidor, err := NuevoServidor(prueba.cfg)
			if servidor != nil || !errors.Is(err, prueba.error) {
				t.Fatalf("resultado = (%v, %v); se esperaba %v", servidor, err, prueba.error)
			}
		})
	}
}
