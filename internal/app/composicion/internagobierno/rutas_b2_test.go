package internagobierno

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

func TestRutaInternaGobernadaPersonalB2EsListaPositiva(t *testing.T) {
	for _, ruta := range []string{
		"/api/vec/personal/empleados", "/api/vec/personal/hechos",
		"/api/vec/personal/vacantes", "/api/vec/personal/empleados/emp_0123456789abcdefghijkl",
	} {
		if !RutaInternaGobernada(ruta) {
			t.Fatalf("ruta %q rechazada", ruta)
		}
	}
	for _, ruta := range []string{
		"/api/vec/personal/empleados/", "/api/vec/personal/empleados/emp_corta",
		"/api/vec/personal/empleados/emp_0123456789abcdefghijkl/relaciones",
		"/api/vec/personal/vacantes/otra", "/api/vec/personal/hechos/otra",
		"/api/vec/admin", "/api/vec/personal/empleados%2Femp_0123456789abcdefghijkl",
	} {
		if RutaInternaGobernada(ruta) {
			t.Fatalf("ruta %q aceptada", ruta)
		}
	}
}

func TestAutoridadRutaPersonalB2DeniegaSinSelloF1(t *testing.T) {
	fuente, err := NuevaFuenteF1(configuracionFuentePrueba())
	if err != nil {
		t.Fatal(err)
	}
	a, err := NuevaAutoridadRutaSeguimiento(fuente)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AutorizarRutaExacta(context.Background(), "/api/vec/personal/vacantes"); !errors.Is(err, httpapi.ErrAccesoRutaExactaDenegado) {
		t.Fatalf("ruta B2 sin sello F1: %v", err)
	}
}
