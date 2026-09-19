package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpcatalogo "vec-diputacion-granada/internal/modules/dietas/adapters/httpcatalogo"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestDietasRouteCatalogExigePermisoYDelega(t *testing.T) {
	handler := newTestHandlerWithOptions(t, HandlerOptions{ManejadorCatalogoRutaDietas: httpcatalogo.NuevoManejador()})
	for _, caso := range []struct {
		nombre, metodo string
		principal      domain.Principal
		estado         int
	}{
		{"sin permiso", http.MethodGet, domain.Principal{ID: "sin-permiso", Roles: []string{"x"}}, http.StatusForbidden},
		{"metodo", http.MethodPost, principalRutaDietasPrueba(), http.StatusMethodNotAllowed},
		{"ok", http.MethodGet, principalRutaDietasPrueba(), http.StatusOK},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.handleDietasRouteCatalog(rec, httptest.NewRequest(caso.metodo, "/api/vec/dietas/route-catalog", nil), caso.principal)
			if rec.Code != caso.estado {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
