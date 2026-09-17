package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRutaColeccionAutorizaLaRutaCompletaYNoCapturaPrefijosParciales(t *testing.T) {
	autoridad := &autoridadRutasExactasEspia{}
	manejador := &manejadorExactoPrueba{}
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		AutoridadRutasExactas: autoridad,
		RutasColeccion: []RutaColeccion{{
			Prefijo: "/api/vec/bolsa/bolsas", Manejador: manejador,
		}},
	})
	ruta := "/api/vec/bolsa/bolsas/bolsa:demo:administrativo/candidatos"
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta+"?limite=50", nil))
	if respuesta.Code != http.StatusNoContent {
		t.Fatalf("status=%d", respuesta.Code)
	}
	if llamadas, recibida := autoridad.estado(); llamadas != 1 || recibida != ruta {
		t.Fatalf("autoridad: llamadas=%d ruta=%q", llamadas, recibida)
	}
	if llamadas, recibida, consulta := manejador.estado(); llamadas != 1 || recibida != ruta || consulta != "limite=50" {
		t.Fatalf("manejador: llamadas=%d ruta=%q consulta=%q", llamadas, recibida, consulta)
	}

	rechazada := httptest.NewRecorder()
	handler.ServeHTTP(rechazada, httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/bolsasx/otra", nil))
	if rechazada.Code != http.StatusNotFound {
		t.Fatalf("un prefijo parcial no debe alcanzar la colección: %d", rechazada.Code)
	}
	if llamadas, _ := autoridad.estado(); llamadas != 1 {
		t.Fatalf("un prefijo parcial consultó la autoridad: %d", llamadas)
	}
}
