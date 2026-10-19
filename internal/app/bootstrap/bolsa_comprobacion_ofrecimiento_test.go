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
	h := nuevoManejadorBolsasRRHHDesarrollo(nil)
	h.mutar = m
	base := "/api/vec/bolsa/bolsas/bolsa:uno/candidatos/participacion:uno/"
	for _, c := range []struct {
		operacion string
		estado    int
	}{{"comprobaciones-ofrecimiento", http.StatusServiceUnavailable}, {"situacion", http.StatusNoContent}} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, base+c.operacion, strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Idempotency-Key", "comprobacion:misma-operacion")
		h.ServeHTTP(w, r)
		if w.Code != c.estado {
			t.Fatalf("%s: %d, %s", c.operacion, w.Code, w.Body)
		}
		if c.estado == http.StatusServiceUnavailable && !strings.Contains(w.Body.String(), "comprobacion_ofrecimiento_no_habilitada") {
			t.Fatal("la dependencia ausente no tiene código de catálogo")
		}
	}
	r := httptest.NewRequest(http.MethodPost, base+"comprobaciones-ofrecimiento-otra", strings.NewReader(`{}`))
	r.Header.Set("Idempotency-Key", "comprobacion:misma-operacion")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("se admitió la cabecera de otra operación: %d", w.Code)
	}
}

func TestComprobacionSinGobiernoNoPublicaMaterial(t *testing.T) {
	if proveedorComprobacionOfrecimiento(context.Background(), nil,
		materialAtestacionContratacionTemporalDesarrollo{}, relojContratacionTemporalDesarrollo{}) != nil {
		t.Fatal("se habilitó la comprobación sin gobierno")
	}
}
