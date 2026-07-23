package domain

import (
	"errors"
	"testing"
	"time"
)

func TestRegistrarViaCoberturaExigeResultadoRCHabilitante(t *testing.T) {
	pruebas := []struct {
		nombre       string
		resultado    ResultadoValidacionRC
		esperaAvance bool
	}{
		{"validada", RCValidada, true},
		{"no requerida", RCNoRequerida, true},
		{"rechazada", RCRechazada, false},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			expediente := expedienteValido(t)
			analisis := analisisValido()
			if prueba.resultado != RCValidada {
				prepararRCNegativa(&analisis.ValidacionRC, prueba.resultado)
			}
			conAnalisis, err := expediente.RegistrarAnalisis(
				1,
				analisis,
				actuacion("analisis.validado", "gestion_bolsa", instanteBase.Add(time.Minute)),
			)
			if err != nil {
				t.Fatalf("registrar resultado %s: %v", prueba.resultado, err)
			}

			_, err = conAnalisis.RegistrarViaCobertura(
				2,
				decisionValida(),
				actuacion("cobertura.decidida", "asignacion_unidad", instanteBase.Add(2*time.Minute)),
			)
			if prueba.esperaAvance && err != nil {
				t.Fatalf("%s debía habilitar cobertura: %v", prueba.resultado, err)
			}
			if !prueba.esperaAvance && !errors.Is(err, ErrTransicionInvalida) {
				t.Fatalf("%s no debía habilitar cobertura: %v", prueba.resultado, err)
			}
		})
	}
}

func TestRegistrarAnalisisOrdenaValidacionRCAntesDeActuacion(t *testing.T) {
	instanteActuacion := instanteBase.Add(time.Minute)

	t.Run("igualdad admitida", func(t *testing.T) {
		analisis := analisisValido()
		analisis.ValidacionRC.ValidadaEn = instanteActuacion
		if _, err := expedienteValido(t).RegistrarAnalisis(
			1,
			analisis,
			actuacion("analisis.validado", "gestion_bolsa", instanteActuacion),
		); err != nil {
			t.Fatalf("validación simultánea a la actuación: %v", err)
		}
	})

	t.Run("validación futura rechazada", func(t *testing.T) {
		analisis := analisisValido()
		analisis.ValidacionRC.ValidadaEn = instanteActuacion.Add(time.Microsecond)
		if _, err := expedienteValido(t).RegistrarAnalisis(
			1,
			analisis,
			actuacion("analisis.validado", "gestion_bolsa", instanteActuacion),
		); !errors.Is(err, ErrTransicionInvalida) {
			t.Fatalf("validación posterior a la actuación: %v", err)
		}
	})
}
