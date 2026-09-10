package contrataciontemporal

import (
	"context"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultaSeguimientoRutaPrueba struct{ lecturas *int }

func (c consultaSeguimientoRutaPrueba) ConsultarSeguimientoIncorporacionV2(context.Context, string) (ct.VistaSeguimientoIncorporacionV2, error) {
	*c.lecturas++
	return ct.VistaSeguimientoIncorporacionV2{}, ct.ErrConflictoIncorporacionAplicacion
}

func TestConsultaSeguimientoV2PeticionSeparadaYDenegacion(t *testing.T) {
	creadas, lecturas := 0, 0
	h := manejadorConsultaSeguimientoV2{nueva: func(context.Context) (consultaSeguimientoV2, error) {
		creadas++
		return consultaSeguimientoRutaPrueba{&lecturas}, nil
	}}
	ruta := httpinterno.RutaConsultaSeguimientoV2 + "?expediente_ref=expediente:ct:original"
	for _, metodo := range []string{"GET", "GET", "POST"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(metodo, ruta, nil))
		esperado := 409
		if metodo == "POST" {
			esperado = 405
		}
		if w.Code != esperado {
			t.Fatalf("%s respondió %d", metodo, w.Code)
		}
	}
	if creadas != 2 || lecturas != 2 {
		t.Fatalf("petición compartida o escritura alcanzada: %d/%d", creadas, lecturas)
	}
	h.nueva = func(context.Context) (consultaSeguimientoV2, error) {
		return nil, ct.ErrDenegadaIncorporacionAplicacion
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", ruta, nil))
	if w.Code != 403 || lecturas != 2 {
		t.Fatalf("denegación alcanzó al lector: %d/%d", w.Code, lecturas)
	}
}
