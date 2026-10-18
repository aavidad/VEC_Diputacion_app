package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComprobacionOpcionalConservaLasOtrasOperaciones(t *testing.T) {
	m := &manejadorParticipacionBolsaDesarrollo{
		comprobacion: montarComprobacionOfrecimiento(nil, nil, nil),
		situacion: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	base := "/api/vec/bolsa/bolsas/bolsa:uno/candidatos/participacion:uno/"
	for _, c := range []struct {
		operacion string
		estado    int
	}{{"comprobaciones-ofrecimiento", http.StatusServiceUnavailable}, {"situacion", http.StatusNoContent}} {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base+c.operacion, nil))
		if w.Code != c.estado {
			t.Fatalf("%s: %d, %s", c.operacion, w.Code, w.Body)
		}
		if c.estado == http.StatusServiceUnavailable && !strings.Contains(w.Body.String(), "comprobacion_ofrecimiento_no_habilitada") {
			t.Fatal("la dependencia ausente no tiene código de catálogo")
		}
	}
}

func TestComprobacionSinGobiernoNoPublicaMaterial(t *testing.T) {
	if proveedorComprobacionOfrecimiento(context.Background(), nil,
		materialAtestacionContratacionTemporalDesarrollo{}, relojContratacionTemporalDesarrollo{}) != nil {
		t.Fatal("se habilitó la comprobación sin gobierno")
	}
}
