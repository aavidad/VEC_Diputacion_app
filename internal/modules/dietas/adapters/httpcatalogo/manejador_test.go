package httpcatalogo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManejadorSirveCatalogoSemillaNoLiquidable(t *testing.T) {
	rec := httptest.NewRecorder()
	NuevoManejador().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/vec/dietas/route-catalog", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}
	var body struct {
		ProvinceRoutePoints []map[string]any `json:"province_route_points"`
		ProvinceRouteMatrix map[string]any   `json:"province_route_matrix"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.ProvinceRoutePoints) < 2 {
		t.Fatalf("puntos = %d", len(body.ProvinceRoutePoints))
	}
	if body.ProvinceRouteMatrix["import_required_before_liquidation"] != true {
		t.Fatalf("catalogo no marca importacion pendiente: %#v", body.ProvinceRouteMatrix)
	}
	if body.ProvinceRouteMatrix["state"] != "pendiente_importacion_completa" {
		t.Fatalf("estado = %#v", body.ProvinceRouteMatrix["state"])
	}
}

func TestManejadorRechazaMetodoYConsulta(t *testing.T) {
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/vec/dietas/route-catalog", nil),
		httptest.NewRequest(http.MethodGet, "/api/vec/dietas/route-catalog?x=1", nil),
	} {
		rec := httptest.NewRecorder()
		NuevoManejador().ServeHTTP(rec, request)
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
	}
}
