package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRutaNoMontadaRespondeErrorCatalogadoSinDelegar(t *testing.T) {
	const rutaMontada = "/api/vec/contratacion-temporal/llamamientos/seleccion"
	const rutaAusente = rutaMontada + "/estado"
	llamadas := 0
	h := newTestHandlerWithOptions(t, HandlerOptions{
		RutasExactas: []RutaExacta{{Ruta: rutaMontada, Manejador: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				llamadas++
				if r.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					_, _ = w.Write([]byte(`{"error":"metodo propio"}`))
					return
				}
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"respuesta propia"}`))
			})}},
		AutoridadRutasExactas: autoridadRutasExactasPrueba{},
	})
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodGet, rutaAusente},
		{http.MethodGet, "/api/vec/ruta-inexistente"},
		{http.MethodPost, rutaAusente},
	} {
		w := httptest.NewRecorder()
		for _, cabecera := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Location", "Retry-After"} {
			w.Header().Set(cabecera, "valor_anterior")
		}
		h.ServeHTTP(w, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if w.Code != http.StatusNotFound || llamadas != 0 ||
			w.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
			w.Header().Get("Cache-Control") != "no-store, no-transform" ||
			w.Header().Get("X-Content-Type-Options") != "nosniff" ||
			w.Header().Get("Content-Security-Policy") != "default-src 'none'; base-uri 'none'; frame-ancestors 'none'" ||
			w.Header().Get("Set-Cookie") != "" || w.Header().Get("Access-Control-Allow-Origin") != "" ||
			w.Header().Get("Location") != "" || w.Header().Get("Retry-After") != "" {
			t.Fatalf("ruta ausente mal respondida o delegada: %s %s: %d llamadas=%d %s",
				caso.metodo, caso.ruta, w.Code, llamadas, w.Body)
		}
		var sobre map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil || len(sobre) != 1 {
			t.Fatalf("404 sin sobre único: %s %v", w.Body, err)
		}
		var campos map[string]string
		if err := json.Unmarshal(sobre["error"], &campos); err != nil || len(campos) != 3 ||
			campos["codigo"] != "recurso_no_encontrado" ||
			campos["clave_i18n"] != "api.vec.ruta_exacta.error.recurso_no_encontrado" ||
			!regexp.MustCompile(`^corr_[0-9a-f]{32}$`).MatchString(campos["correlacion_ref"]) {
			t.Fatalf("404 sin catálogo o con detalle adicional: %s %v", w.Body, err)
		}
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaMontada, nil))
	if w.Code != http.StatusNotFound || llamadas != 1 || w.Body.String() != `{"error":"respuesta propia"}` {
		t.Fatalf("el 404 del manejador montado cambió: %d llamadas=%d %s", w.Code, llamadas, w.Body)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, rutaMontada, nil))
	if w.Code != http.StatusMethodNotAllowed || llamadas != 2 || w.Body.String() != `{"error":"metodo propio"}` {
		t.Fatalf("el 405 del manejador montado cambió: %d llamadas=%d %s", w.Code, llamadas, w.Body)
	}
}
