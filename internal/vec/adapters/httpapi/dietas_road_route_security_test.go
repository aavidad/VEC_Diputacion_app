package httpapi

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestDietasRoadRouteRejectsNonContractInputBeforeConnector(t *testing.T) {
	var hits atomic.Int32
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[]}`))
	}))
	defer osrm.Close()

	handler := newTestHandlerWithOptions(t, testOSRMOptions(osrm.URL))
	principal := principalRutaDietasPrueba()
	tests := []struct {
		name string
		body string
	}{
		{name: "campo desconocido", body: `{"unknown":true,"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`},
		{name: "segundo valor JSON", body: `{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]} {}`},
		{name: "alternativas negativas", body: `{"alternatives":-1,"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`},
		{name: "demasiadas alternativas", body: `{"alternatives":4,"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := hits.Load()
			req := nuevaPeticionRutaDietas(test.body)
			rec := httptest.NewRecorder()
			handler.handleDietasRoadRoute(rec, req, principal)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
			}
			if hits.Load() != before {
				t.Fatal("una peticion fuera de contrato alcanzo el conector OSRM")
			}
		})
	}
}

func TestDietasRoadRouteRejectsUntrustedConnectorPayload(t *testing.T) {
	principal := principalRutaDietasPrueba()
	tests := []struct {
		name string
		body string
	}{
		{name: "JSON invalido", body: `{`},
		{name: "codigo distinto de Ok", body: `{"code":"NoRoute","routes":[]}`},
		{name: "sin rutas", body: `{"code":"Ok"}`},
		{name: "rutas no es lista", body: `{"code":"Ok","routes":{}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(test.body))
			}))
			defer osrm.Close()
			handler := newTestHandlerWithOptions(t, testOSRMOptions(osrm.URL))
			req := nuevaPeticionRutaDietas(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`)
			rec := httptest.NewRecorder()
			handler.handleDietasRoadRoute(rec, req, principal)
			if rec.Code != http.StatusBadGateway {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func principalRutaDietasPrueba() domain.Principal {
	return domain.Principal{
		ID:            "actor-ruta-dietas-prueba",
		Roles:         []string{"permiso-ruta-dietas-prueba"},
		Permissions:   []string{dietasmodule.PermissionRouteRead},
		AuthMethod:    domain.AuthMethodDemo,
		AuthAssurance: domain.AuthAssuranceHigh,
	}
}
