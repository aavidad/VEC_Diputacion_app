package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
)

func TestRutasPersonalesEfimerasRetiradasDelGuardian(t *testing.T) {
	for _, ruta := range []string{
		"/api/vec/bolsa/area-personal",
		"/api/vec/bolsa/mi-disponibilidad",
	} {
		if esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, ruta, nil)) {
			t.Fatalf("la ruta antigua %s sigue admitida por el guardián", ruta)
		}
	}
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, bolsapersonal.RutaMiBolsa, nil)) {
		t.Fatal("la consulta propia autorizada perdió su ruta")
	}
}
