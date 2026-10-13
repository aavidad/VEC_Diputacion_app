package meritoshttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCarteraSinConfiguracionConservaRutasCerradas(t *testing.T) {
	h := NuevoHandlerCarteraNoDisponible()
	for _, caso := range []struct {
		metodo, ruta string
		esperado     int
	}{
		{http.MethodGet, RutaCartera + "?filtro=todos&lang=es", http.StatusServiceUnavailable},
		{http.MethodPost, RutaRetiradasCartera, http.StatusServiceUnavailable},
		{http.MethodPut, RutaCartera, http.StatusMethodNotAllowed},
		{http.MethodGet, RutaCartera + "/otro", http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if w.Code != caso.esperado || !strings.Contains(w.Body.String(), `"codigo"`) ||
			w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("ruta cerrada %s %s: estado=%d cuerpo=%s", caso.metodo, caso.ruta, w.Code, w.Body.String())
		}
	}
}
