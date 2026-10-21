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
			func(w http.ResponseWriter, _ *http.Request) {
				llamadas++
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"respuesta propia"}`))
			})}},
		AutoridadRutasExactas: autoridadRutasExactasPrueba{},
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaAusente, nil))
	if w.Code != http.StatusNotFound || llamadas != 0 ||
		w.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		w.Header().Get("Cache-Control") != "no-store, no-transform" ||
		w.Header().Get("X-Content-Type-Options") != "nosniff" ||
		w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("ruta ausente mal respondida o delegada: %d llamadas=%d %s", w.Code, llamadas, w.Body)
	}
	var salida struct {
		Error struct {
			Codigo      string `json:"codigo"`
			ClaveI18n   string `json:"clave_i18n"`
			Correlacion string `json:"correlacion_ref"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil ||
		salida.Error.Codigo != "recurso_no_encontrado" ||
		salida.Error.ClaveI18n != "api.vec.ruta_exacta.error.recurso_no_encontrado" ||
		!regexp.MustCompile(`^corr_[0-9a-f]{32}$`).MatchString(salida.Error.Correlacion) {
		t.Fatalf("404 sin catálogo o correlación: %s %v", w.Body, err)
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rutaMontada, nil))
	if w.Code != http.StatusNotFound || llamadas != 1 || w.Body.String() != `{"error":"respuesta propia"}` {
		t.Fatalf("el 404 del manejador montado cambió: %d llamadas=%d %s", w.Code, llamadas, w.Body)
	}
}
