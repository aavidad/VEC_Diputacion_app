package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Los catálogos públicos de textos y de traducción se guardan y se revalidan
// (304), sin Pragma, y /locales se sirve comprimido con el mismo compresor de
// los estáticos. La API no hereda nada de esto.
func TestCatalogosTextosSeRevalidanYLocalesSeComprimen(t *testing.T) {
	handler := NewHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for _, ruta := range []string{"/textos/es/portal.json", "/locales/es.json"} {
		rec := httptest.NewRecorder()
		peticion := peticionServidorPrueba(http.MethodGet, ruta, nil)
		peticion.Header.Set("Accept-Encoding", "gzip")
		handler.ServeHTTP(rec, peticion)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s = %d", ruta, rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("%s cache-control = %q", ruta, got)
		}
		if got := rec.Header().Get("Pragma"); got != "" {
			t.Fatalf("%s pragma = %q", ruta, got)
		}
		if rec.Header().Get("Content-Encoding") != "gzip" {
			t.Fatalf("%s sin gzip", ruta)
		}
		lector, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatalf("%s gzip: %v", ruta, err)
		}
		if cuerpo, err := io.ReadAll(lector); err != nil || len(cuerpo) == 0 || cuerpo[0] != '{' {
			t.Fatalf("%s cuerpo no válido: %v", ruta, err)
		}
		ultima := rec.Header().Get("Last-Modified")
		revalidacion := peticionServidorPrueba(http.MethodGet, ruta, nil)
		revalidacion.Header.Set("Accept-Encoding", "gzip")
		revalidacion.Header.Set("If-Modified-Since", ultima)
		rec304 := httptest.NewRecorder()
		handler.ServeHTTP(rec304, revalidacion)
		if ultima == "" || rec304.Code != http.StatusNotModified {
			t.Fatalf("%s revalidación = %d (Last-Modified %q)", ruta, rec304.Code, ultima)
		}
	}
	versionado := httptest.NewRecorder()
	handler.ServeHTTP(versionado, peticionServidorPrueba(http.MethodGet, "/textos/es/portal.json?huella=a37f71229b49d228", nil))
	if versionado.Code != http.StatusOK || versionado.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("catálogo versionado = %d %q", versionado.Code, versionado.Header().Get("Cache-Control"))
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/api/vec/session", nil))
	if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("la API debe seguir sin guardarse: %q %q", rec.Header().Get("Cache-Control"), rec.Header().Get("Pragma"))
	}
}
