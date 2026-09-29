package application

import (
	"context"
	"testing"
	"time"
)

// Con poco plazo restante no se lanza otro intento: se responde «no
// disponible» antes de que el plazo corte una sentencia a medias.
func TestEsperaCarreraConfirmacionCoberturaRespetaMargenDelPlazo(t *testing.T) {
	t.Parallel()
	corto, cancelarCorto := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancelarCorto()
	if esperarReintentoCarreraConfirmacionCobertura(corto, 1) {
		t.Fatal("reintento lanzado sin margen de plazo")
	}
	largo, cancelarLargo := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelarLargo()
	if !esperarReintentoCarreraConfirmacionCobertura(largo, 1) {
		t.Fatal("reintento denegado con plazo holgado")
	}
	if !esperarReintentoCarreraConfirmacionCobertura(context.Background(), 1) {
		t.Fatal("reintento denegado sin plazo")
	}
}
