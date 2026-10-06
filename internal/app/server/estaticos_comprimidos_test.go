package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

func leerEstatico(t *testing.T, ruta string) []byte {
	t.Helper()
	contenido, err := os.ReadFile(filepath.Join(directorioEstaticos(), filepath.FromSlash(ruta)))
	if err != nil {
		t.Fatalf("leer %s: %v", ruta, err)
	}
	return contenido
}

func pedirPublico(t *testing.T, handler http.Handler, ruta string, cabeceras map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	peticion := peticionServidorPrueba(http.MethodGet, ruta, nil)
	for clave, valor := range cabeceras {
		peticion.Header.Set(clave, valor)
	}
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	return respuesta
}

func descomprimir(t *testing.T, cuerpo []byte) []byte {
	t.Helper()
	lector, err := gzip.NewReader(strings.NewReader(string(cuerpo)))
	if err != nil {
		t.Fatalf("gzip no válido: %v", err)
	}
	contenido, err := io.ReadAll(lector)
	if err != nil {
		t.Fatalf("gzip incompleto: %v", err)
	}
	return contenido
}

func TestEstaticosSeSirvenComprimidosSiElNavegadorLoAcepta(t *testing.T) {
	handler := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	original := leerEstatico(t, "bolsa/bolsa.js")
	ruta := "/bolsa/bolsa.js?v=prueba"

	comprimida := pedirPublico(t, handler, ruta, map[string]string{"Accept-Encoding": "br, gzip;q=0.8"})
	if comprimida.Code != http.StatusOK || comprimida.Header().Get("Content-Encoding") != "gzip" ||
		!strings.Contains(comprimida.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatalf("respuesta comprimida inesperada: %d %v", comprimida.Code, comprimida.Header())
	}
	if comprimida.Body.Len() >= len(original) || string(descomprimir(t, comprimida.Body.Bytes())) != string(original) {
		t.Fatal("el contenido comprimido no reproduce el fichero o no ahorra bytes")
	}
	if comprimida.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" ||
		!strings.HasPrefix(comprimida.Header().Get("Content-Type"), "text/javascript") ||
		comprimida.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("la compresión no debe cambiar la política de caché, el tipo ni las cabeceras de seguridad: %v", comprimida.Header())
	}

	for _, cabeceras := range []map[string]string{
		{},
		{"Accept-Encoding": "gzip;q=0"},
		{"Accept-Encoding": "identity"},
		{"Accept-Encoding": "gzip", "Range": "bytes=0-99"},
	} {
		plana := pedirPublico(t, handler, ruta, cabeceras)
		if plana.Header().Get("Content-Encoding") != "" || !strings.Contains(plana.Header().Get("Vary"), "Accept-Encoding") {
			t.Fatalf("%v: no debía comprimirse: %v", cabeceras, plana.Header())
		}
		if cabeceras["Range"] == "" && (plana.Code != http.StatusOK || plana.Body.String() != string(original)) {
			t.Fatalf("%v: contenido sin comprimir distinto: %d", cabeceras, plana.Code)
		}
	}

	condicional := pedirPublico(t, handler, ruta, map[string]string{
		"Accept-Encoding":   "gzip",
		"If-Modified-Since": comprimida.Header().Get("Last-Modified"),
	})
	if condicional.Code != http.StatusNotModified || condicional.Body.Len() != 0 {
		t.Fatalf("la revalidación debe responder 304 sin cuerpo: %d", condicional.Code)
	}
}

func TestEstaticosComprimidosRespetanLaListaPositivaYElIndice(t *testing.T) {
	handler := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	indice := pedirPublico(t, handler, "/bolsa/", map[string]string{"Accept-Encoding": "gzip"})
	if indice.Code != http.StatusOK || indice.Header().Get("Content-Encoding") != "gzip" ||
		indice.Header().Get("Cache-Control") != "no-store" ||
		string(descomprimir(t, indice.Body.Bytes())) != string(leerEstatico(t, "bolsa/index.html")) {
		t.Fatalf("índice de /bolsa/ inesperado: %d %v", indice.Code, indice.Header())
	}
	for _, ruta := range []string{"/bolsa/no-existe.js", "/portal-empleado/portal.js", "/bolsa/../portal-empleado/portal.js"} {
		if respuesta := pedirPublico(t, handler, ruta, map[string]string{"Accept-Encoding": "gzip"}); respuesta.Code != http.StatusNotFound &&
			respuesta.Code != http.StatusBadRequest && respuesta.Code != http.StatusMovedPermanently {
			t.Fatalf("%s: la compresión no debe abrir rutas fuera de la lista positiva: %d", ruta, respuesta.Code)
		}
	}
}

func TestCacheEstaticosComprimidosSeRenuevaYSeAcota(t *testing.T) {
	directorio := t.TempDir()
	fichero := filepath.Join(directorio, "a.js")
	if err := os.WriteFile(fichero, []byte(strings.Repeat("const a = 1;\n", 200)), 0o600); err != nil {
		t.Fatal(err)
	}
	cache := &cacheEstaticosComprimidos{}
	primera, ok := cache.obtener(directorio, "/a.js")
	if !ok || string(descomprimir(t, primera.gzip)) != strings.Repeat("const a = 1;\n", 200) {
		t.Fatal("primera compresión inesperada")
	}
	if segunda, ok := cache.obtener(directorio, "/a.js"); !ok || segunda != primera {
		t.Fatal("un fichero sin cambios debe salir de la caché")
	}
	nuevo := strings.Repeat("const b = 22;\n", 200)
	if err := os.WriteFile(fichero, []byte(nuevo), 0o600); err != nil {
		t.Fatal(err)
	}
	futuro := time.Now().Add(time.Minute)
	if err := os.Chtimes(fichero, futuro, futuro); err != nil {
		t.Fatal(err)
	}
	if renovada, ok := cache.obtener(directorio, "/a.js"); !ok || string(descomprimir(t, renovada.gzip)) != nuevo {
		t.Fatal("un fichero modificado debe volver a comprimirse")
	}
	if cache.ocupado.Load() <= 0 || cache.ocupado.Load() > int64(len(nuevo)) {
		t.Fatalf("contabilidad de memoria inesperada: %d", cache.ocupado.Load())
	}

	llena := &cacheEstaticosComprimidos{}
	llena.ocupado.Store(presupuestoCacheEstaticosComprimidos)
	if _, ok := llena.obtener(directorio, "/a.js"); ok || llena.ocupado.Load() != presupuestoCacheEstaticosComprimidos {
		t.Fatal("sin presupuesto debe servirse sin comprimir y sin crecer")
	}
	if _, ok := cache.obtener(directorio, "/../a.js"); !ok {
		t.Fatal("la ruta limpia debe resolverse dentro del directorio")
	}
	if _, ok := cache.obtener(directorio, "/no-existe.js"); ok {
		t.Fatal("un fichero inexistente no se comprime")
	}
}
