package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Regresión del 23/09/2026: la ruta de estadísticas se registró en el
// manejador pero no en la lista de rutas que revalida la identidad mTLS, y la
// frontera la denegaba con 401 aunque el mismo certificado abría el cuadro de
// bolsas. Las rutas del manejador RRHH de Bolsa deben declararse juntas.
func TestFronteraAdmiteTodasLasRutasDelManejadorRRHHDeBolsa(t *testing.T) {
	for _, ruta := range []string{
		rutaBolsasRRHHDesarrollo,
		rutaEstadisticasBolsaRRHHDesarrollo,
		rutaAvisosBolsaRRHHDesarrollo,
	} {
		if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, ruta, nil)) {
			t.Errorf("la frontera de desarrollo no revalida %s: respondería 401", ruta)
		}
	}
}
