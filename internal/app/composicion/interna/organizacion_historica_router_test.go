package interna

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

func TestOrganizacionHistoricaRouterIndependienteYApagadoPorDefecto(t *testing.T) {
	llamadas := 0
	siguiente := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { llamadas++; w.WriteHeader(204) })
	consulta := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	for _, tc := range []struct {
		consulta http.Handler
		esperado int
	}{{nil, 404}, {consulta, 200}} {
		router, err := nuevoEnrutadorOrganizacionHistorica(siguiente, tc.consulta)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", httpapi.RutaOrganizacionHistoricaPersonal, nil))
		if w.Code != tc.esperado || llamadas != 0 {
			t.Fatalf("consulta: %d, llamadas siguiente=%d", w.Code, llamadas)
		}
	}
	router, _ := nuevoEnrutadorOrganizacionHistorica(siguiente, consulta)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/vec/personal/organizacion-historica/importaciones/publicar", nil))
	if w.Code != 204 || llamadas != 1 {
		t.Fatal("el montaje no debe apropiarse de otras rutas")
	}
}

func TestOrganizacionHistoricaMontajeNoSustituyeDependencias(t *testing.T) {
	if _, ok := montarOrganizacionHistoricaGobernada(context.Background(), "", "", nil, nil, nil, nil, relojGobiernoInterno{}); ok {
		t.Fatal("montaje con dependencias ausentes")
	}
	if err := acreditarPoolOrganizacionHistorica(context.Background(), nil, ""); err == nil {
		t.Fatal("pool ausente acreditado")
	}
}

func TestOrganizacionHistoricaErrorOpcionalConservaRutaBase(t *testing.T) {
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	oh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responderPuenteSeguimiento(w, http.StatusServiceUnavailable)
	})
	router, err := nuevoEnrutadorOrganizacionHistorica(base, oh)
	if err != nil {
		t.Fatal(err)
	}
	for ruta, esperado := range map[string]int{httpapi.RutaOrganizacionHistoricaPersonal: 503, "/api/vec/contratacion-temporal/seguimiento": 200} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", ruta, nil))
		if w.Code != esperado {
			t.Fatalf("%s: %d", ruta, w.Code)
		}
	}
	dir := t.TempDir()
	if materialOrganizacionHistoricaSeleccionado(dir) {
		t.Fatal("ausencia debe permanecer apagada")
	}
	if err := os.WriteFile(filepath.Join(dir, "organizacion_historica_v3.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	c, disponible := montarOrganizacionHistoricaGobernada(context.Background(), dir, "", nil, nil, nil, nil, relojGobiernoInterno{})
	if disponible || !c.seleccionada {
		t.Fatal("material seleccionado inválido debe conservar estado de indisponibilidad")
	}
}
