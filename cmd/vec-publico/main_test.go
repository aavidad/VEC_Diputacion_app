package main

import (
	"errors"
	"testing"

	"vec-diputacion-granada/config"
)

func TestEjecutarRechazaReglasEjemploAntesDeComponer(t *testing.T) {
	casos := map[string]struct {
		dobleLlave bool
		variable   string
		esperado   error
	}{
		"bolsa fuera de desarrollo": {false, config.EnvBolsaReglasSourcePath, config.ErrConfiguracionReglasEjemploFueraDesarrollo},
		"ct fuera de desarrollo":    {false, config.EnvCTReglasSourcePath, config.ErrConfiguracionReglasEjemploFueraDesarrollo},
		"bolsa con doble llave":     {true, config.EnvBolsaReglasSourcePath, config.ErrConfiguracionReglasEjemploSinComposicion},
		"motivos con doble llave":   {true, config.EnvCTAnalisisMotivosSourcePath, config.ErrConfiguracionReglasEjemploSinComposicion},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if caso.dobleLlave {
				t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
				t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
				t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
			} else {
				t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
			}
			t.Setenv(caso.variable, "catalogo.ejemplo.demo.json")
			// El rechazo llega antes de componer y de abrir la escucha.
			if err := ejecutar(); !errors.Is(err, caso.esperado) {
				t.Fatalf("ejecutar() = %v; se esperaba %v", err, caso.esperado)
			}
		})
	}
}
