package bootstrap

import (
	"errors"
	"testing"
)

func TestCronosNoPreparaRutasSinAutoridades(t *testing.T) {
	preparados, err := PrepararManejadoresCronos(DependenciasManejadoresCronos{})
	if !errors.Is(err, ErrManejadoresCronosNoDisponibles) || preparados.SaldoPropio != nil || preparados.MarcajeRemoto != nil || preparados.RecuperacionRemota != nil {
		t.Fatalf("composicion Cronos sin autoridades: %v", err)
	}
}
