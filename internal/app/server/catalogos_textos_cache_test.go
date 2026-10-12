package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
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
	// El índice de idiomas anuncia la huella real; solo esa fija un año.
	indice := httptest.NewRecorder()
	handler.ServeHTTP(indice, peticionServidorPrueba(http.MethodGet, "/textos/idiomas.json", nil))
	anunciada := indice.Header().Get("X-VEC-Huella-Textos")
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(anunciada) || anunciada != huellaCatalogosTextos() {
		t.Fatalf("idiomas.json anuncia %q, huella del servidor %q", anunciada, huellaCatalogosTextos())
	}
	otro := httptest.NewRecorder()
	handler.ServeHTTP(otro, peticionServidorPrueba(http.MethodGet, "/textos/es/portal.json", nil))
	if otro.Header().Get("X-VEC-Huella-Textos") != "" {
		t.Fatal("solo el índice anuncia la huella")
	}
	if _, err := calcularHuellaCatalogosTextos(os.DirFS(filepath.Join(t.TempDir(), "sin-textos"))); err == nil {
		t.Fatal("sin catálogos legibles no debe haber huella")
	}
	for huella, esperada := range map[string]string{
		anunciada:          "public, max-age=31536000, immutable",
		"0000000000000000": "no-cache",
	} {
		versionado := httptest.NewRecorder()
		handler.ServeHTTP(versionado, peticionServidorPrueba(http.MethodGet, "/textos/es/portal.json?huella="+huella, nil))
		if versionado.Code != http.StatusOK || versionado.Header().Get("Cache-Control") != esperada {
			t.Fatalf("huella %s = %d %q", huella, versionado.Code, versionado.Header().Get("Cache-Control"))
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/api/vec/session", nil))
	if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("la API debe seguir sin guardarse: %q %q", rec.Header().Get("Cache-Control"), rec.Header().Get("Pragma"))
	}
}
