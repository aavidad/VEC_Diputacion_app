package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// B13: la consulta del histórico de contratos llega al manejador de la
// participación, que aplica autorización V3; nunca se sirve desde el cuadro.
func TestBolsasRRHHDesarrolloDelegaContratosSoloEnLectura(t *testing.T) {
	ruta := rutaBolsasRRHHDesarrollo + "/bolsa:01/candidatos/participacion:01/contratos"
	if !rutaBolsasOperacionesRRHHDesarrollo(ruta) {
		t.Fatal("la ruta B13 debe reconocerse como subrecurso protegido de la ficha")
	}
	manejador := manejadorBolsasRRHHPrueba()
	llamadas := 0
	manejador.mutar = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusOK)
	})
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, nil))
	if respuesta.Code != http.StatusOK || llamadas != 1 {
		t.Fatalf("GET B13 no delegado: status=%d llamadas=%d", respuesta.Code, llamadas)
	}
	respuesta = httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodPost, ruta, nil))
	if llamadas != 1 || respuesta.Code == http.StatusOK {
		t.Fatalf("POST B13 no debe delegarse: status=%d llamadas=%d", respuesta.Code, llamadas)
	}
}
