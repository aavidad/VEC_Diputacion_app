package domain

import (
	"errors"
	"testing"
)

func TestPoliticaAvisosValidaLimitesDeLaBase(t *testing.T) {
	validas := []PoliticaAvisosBolsa{
		{},
		{ContinuadoConfigurado: true, ContinuadoMeses: 36, ContinuadoAntelacionDias: 30},
		{ContinuadoConfigurado: true, ContinuadoMeses: 36, ContinuadoAntelacionDias: 0, EncadenamientoUmbralMeses: 18, EncadenamientoVentanaMeses: 24},
		{PrestaServiciosModo: ModoPrestaServiciosExcluir, PrestaServiciosSituaciones: []string{SituacionTrabajando, SituacionPendienteIncorporacion}},
	}
	for i, p := range validas {
		if err := p.Validar(); err != nil {
			t.Fatalf("válida %d rechazada: %v", i, err)
		}
	}
	invalidas := map[string]PoliticaAvisosBolsa{
		"plazo sin marcar":         {ContinuadoMeses: 36},
		"plazo cero":               {ContinuadoConfigurado: true},
		"antelación negativa":      {ContinuadoConfigurado: true, ContinuadoMeses: 36, ContinuadoAntelacionDias: -1},
		"umbral sin ventana":       {EncadenamientoUmbralMeses: 18},
		"umbral mayor que ventana": {EncadenamientoUmbralMeses: 30, EncadenamientoVentanaMeses: 24},
		"modo sin situaciones":     {PrestaServiciosModo: ModoPrestaServiciosAviso},
		"situaciones sin modo":     {PrestaServiciosSituaciones: []string{SituacionTrabajando}},
		"modo desconocido":         {PrestaServiciosModo: "bloquear", PrestaServiciosSituaciones: []string{SituacionTrabajando}},
		"situación disponible":     {PrestaServiciosModo: ModoPrestaServiciosAviso, PrestaServiciosSituaciones: []string{SituacionDisponible}},
		"situación repetida":       {PrestaServiciosModo: ModoPrestaServiciosAviso, PrestaServiciosSituaciones: []string{SituacionTrabajando, SituacionTrabajando}},
		"situación desconocida":    {PrestaServiciosModo: ModoPrestaServiciosAviso, PrestaServiciosSituaciones: []string{"contratado"}},
	}
	for nombre, p := range invalidas {
		if err := p.Validar(); !errors.Is(err, ErrPoliticaAvisosInvalida) {
			t.Fatalf("%s aceptada: %v", nombre, err)
		}
	}
}
