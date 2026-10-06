package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/config"
)

func manejadorInternoComprimidoPrueba(api http.Handler) http.Handler {
	if api == nil {
		api = http.NotFoundHandler()
	}
	return NewHandlerInternoWithConfig(config.Config{HTTPAllowedCIDRs: []string{"127.0.0.1/8"}}, api)
}

func pedirComprimidoPrueba(t *testing.T, h http.Handler, metodo, ruta string, cabeceras map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	peticion := peticionServidorPrueba(metodo, ruta, nil)
	for nombre, valor := range cabeceras {
		peticion.Header.Set(nombre, valor)
	}
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, peticion)
	return respuesta
}

func descomprimirPrueba(t *testing.T, cuerpo []byte) []byte {
	t.Helper()
	lector, err := gzip.NewReader(bytes.NewReader(cuerpo))
	if err != nil {
		t.Fatalf("cuerpo gzip ilegible: %v", err)
	}
	plano, err := io.ReadAll(lector)
	if err != nil {
		t.Fatalf("cuerpo gzip ilegible: %v", err)
	}
	return plano
}

func leerEstaticoPrueba(t *testing.T, relativa string) []byte {
	t.Helper()
	contenido, err := os.ReadFile(filepath.Join("..", "..", "..", relativa))
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func TestEstaticosTextoSeSirvenConGzipSiElClienteLoAdmite(t *testing.T) {
	h := manejadorInternoComprimidoPrueba(nil)
	original := leerEstaticoPrueba(t, "web/static/portal-empleado/portal.js")
	ruta := "/portal-empleado/portal.js?v=prueba"

	sin := pedirComprimidoPrueba(t, h, http.MethodGet, ruta, nil)
	if sin.Code != http.StatusOK || sin.Header().Get("Content-Encoding") != "" || !bytes.Equal(sin.Body.Bytes(), original) {
		t.Fatalf("sin Accept-Encoding = %d %q", sin.Code, sin.Header().Get("Content-Encoding"))
	}
	if !strings.Contains(sin.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatalf("la variante sin comprimir debe declarar Vary: %q", sin.Header().Get("Vary"))
	}

	con := pedirComprimidoPrueba(t, h, http.MethodGet, ruta, map[string]string{"Accept-Encoding": "gzip, deflate, br"})
	if con.Code != http.StatusOK || con.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("con gzip = %d %q", con.Code, con.Header().Get("Content-Encoding"))
	}
	if !bytes.Equal(descomprimirPrueba(t, con.Body.Bytes()), original) {
		t.Fatal("la variante gzip no conserva los bytes del fichero")
	}
	if !strings.Contains(con.Header().Get("Vary"), "Accept-Encoding") ||
		con.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" ||
		!strings.HasPrefix(con.Header().Get("Content-Type"), "text/javascript") ||
		con.Header().Get("Content-Security-Policy") == "" || con.Header().Get("ETag") == "" {
		t.Fatalf("cabeceras gzip incompletas: %v", con.Header())
	}
	t.Logf("portal.js: %d B sin comprimir, %d B con gzip", sin.Body.Len(), con.Body.Len())
	if con.Body.Len() >= len(original)/2 {
		t.Fatalf("gzip no reduce portal.js: %d de %d", con.Body.Len(), len(original))
	}

	revalidada := pedirComprimidoPrueba(t, h, http.MethodGet, ruta, map[string]string{
		"Accept-Encoding": "gzip", "If-None-Match": con.Header().Get("ETag"),
	})
	if revalidada.Code != http.StatusNotModified || revalidada.Body.Len() != 0 {
		t.Fatalf("If-None-Match = %d con %d B", revalidada.Code, revalidada.Body.Len())
	}

	indice := pedirComprimidoPrueba(t, h, http.MethodGet, "/portal-empleado/", map[string]string{"Accept-Encoding": "gzip"})
	if indice.Code != http.StatusOK || indice.Header().Get("Content-Encoding") != "gzip" ||
		indice.Header().Get("Cache-Control") != "no-store" ||
		!strings.HasPrefix(indice.Header().Get("Content-Type"), "text/html") ||
		!bytes.Equal(descomprimirPrueba(t, indice.Body.Bytes()), leerEstaticoPrueba(t, "web/static/portal-empleado/index.html")) {
		t.Fatalf("índice del portal = %d %v", indice.Code, indice.Header())
	}

	cabeza := pedirComprimidoPrueba(t, h, http.MethodHead, ruta, map[string]string{"Accept-Encoding": "gzip"})
	if cabeza.Code != http.StatusOK || cabeza.Body.Len() != 0 || cabeza.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("HEAD = %d %d %q", cabeza.Code, cabeza.Body.Len(), cabeza.Header().Get("Content-Encoding"))
	}
}

func TestEstaticosSinGzipSiSeRechazaOPideRango(t *testing.T) {
	h := manejadorInternoComprimidoPrueba(nil)
	original := leerEstaticoPrueba(t, "web/static/portal-empleado/portal.js")
	for _, cabeceras := range []map[string]string{
		{"Accept-Encoding": "gzip;q=0, identity"},
		{"Accept-Encoding": "br"},
		{"Accept-Encoding": "gzip", "Range": "bytes=0-9"},
	} {
		respuesta := pedirComprimidoPrueba(t, h, http.MethodGet, "/portal-empleado/portal.js?v=prueba", cabeceras)
		if respuesta.Header().Get("Content-Encoding") != "" {
			t.Fatalf("%v comprimió la respuesta", cabeceras)
		}
		if cabeceras["Range"] != "" {
			if respuesta.Code != http.StatusPartialContent || !bytes.Equal(respuesta.Body.Bytes(), original[:10]) {
				t.Fatalf("Range = %d", respuesta.Code)
			}
			continue
		}
		if respuesta.Code != http.StatusOK || !bytes.Equal(respuesta.Body.Bytes(), original) {
			t.Fatalf("%v = %d", cabeceras, respuesta.Code)
		}
	}
}

func TestCatalogosTextosSeRevalidanYSeComprimen(t *testing.T) {
	h := manejadorInternoComprimidoPrueba(nil)
	original := leerEstaticoPrueba(t, "web/static/textos/es/area-personal.json")
	con := pedirComprimidoPrueba(t, h, http.MethodGet, "/textos/es/area-personal.json", map[string]string{"Accept-Encoding": "gzip"})
	if con.Code != http.StatusOK || con.Header().Get("Content-Encoding") != "gzip" ||
		con.Header().Get("Cache-Control") != "no-cache" || con.Header().Get("Last-Modified") == "" ||
		!strings.HasPrefix(con.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("catálogo = %d %v", con.Code, con.Header())
	}
	if !bytes.Equal(descomprimirPrueba(t, con.Body.Bytes()), original) {
		t.Fatal("el catálogo gzip no conserva sus bytes")
	}
	t.Logf("textos/es/area-personal.json: %d B sin comprimir, %d B con gzip", len(original), con.Body.Len())
	for _, cabeceras := range []map[string]string{
		{"Accept-Encoding": "gzip", "If-None-Match": con.Header().Get("ETag")},
		{"If-Modified-Since": con.Header().Get("Last-Modified")},
	} {
		if r := pedirComprimidoPrueba(t, h, http.MethodGet, "/textos/es/area-personal.json", cabeceras); r.Code != http.StatusNotModified {
			t.Fatalf("revalidación %v = %d", cabeceras, r.Code)
		}
	}

	locale := pedirComprimidoPrueba(t, h, http.MethodGet, "/locales/es.json", map[string]string{"Accept-Encoding": "gzip"})
	if locale.Code != http.StatusOK || locale.Header().Get("Content-Encoding") != "gzip" || locale.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("locale = %d %v", locale.Code, locale.Header())
	}
	if !bytes.Equal(descomprimirPrueba(t, locale.Body.Bytes()), leerEstaticoPrueba(t, "locales/es.json")) {
		t.Fatal("el locale gzip no conserva sus bytes")
	}
}

func TestRespuestasAPINoSeComprimen(t *testing.T) {
	cuerpo := strings.Repeat(`{"dato":"valor"}`, 500)
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, cuerpo)
	})
	h := manejadorInternoComprimidoPrueba(api)
	respuesta := pedirComprimidoPrueba(t, h, http.MethodGet, "/api/vec/session.json", map[string]string{"Accept-Encoding": "gzip"})
	if respuesta.Header().Get("Content-Encoding") != "" || respuesta.Header().Get("Cache-Control") != "no-store" ||
		respuesta.Body.String() != cuerpo {
		t.Fatalf("la API se comprimió o perdió no-store: %v", respuesta.Header())
	}
}

func TestRutaEstaticaFueraDelManifiestoSigueCerradaConGzip(t *testing.T) {
	h := manejadorInternoComprimidoPrueba(nil)
	for _, ruta := range []string{
		"/portal-empleado/no-existe.js",
		"/portal-empleado/../../go.mod",
		"/textos/es/catalogo-inexistente.json",
	} {
		respuesta := pedirComprimidoPrueba(t, h, http.MethodGet, ruta, map[string]string{"Accept-Encoding": "gzip"})
		if respuesta.Code == http.StatusOK || respuesta.Header().Get("Content-Encoding") != "" {
			t.Fatalf("%s = %d %q", ruta, respuesta.Code, respuesta.Header().Get("Content-Encoding"))
		}
	}
}

func TestAceptaGzip(t *testing.T) {
	for valor, esperado := range map[string]bool{
		"":                    false,
		"gzip":                true,
		"GZIP;q=0.5":          true,
		"deflate, gzip;q=1.0": true,
		"*":                   true,
		"gzip;q=0":            false,
		"gzip; q=0.000":       false,
		"br, identity":        false,
		"x-gzip":              false,
	} {
		cabeceras := http.Header{}
		if valor != "" {
			cabeceras.Set("Accept-Encoding", valor)
		}
		if aceptaGzip(cabeceras) != esperado {
			t.Errorf("aceptaGzip(%q) = %v", valor, !esperado)
		}
	}
}

func TestCacheComprimidosCuentaUnaVezConPeticionesConcurrentes(t *testing.T) {
	dir := t.TempDir()
	contenido := []byte(strings.Repeat("const dato = 'valor';\n", 4000))
	if err := os.WriteFile(filepath.Join(dir, "app.js"), contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	cache := &cacheComprimidos{entrada: map[string]entradaComprimida{}}
	var grupo sync.WaitGroup
	for i := 0; i < 64; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			if entrada, ok := cache.obtener(dir, "/app.js"); !ok || entrada.cuerpo == nil {
				t.Error("sin variante comprimida")
			}
		}()
	}
	grupo.Wait()
	entrada := cache.entrada[dir+"\x00/app.js"]
	if len(cache.entrada) != 1 || cache.bytes != len(entrada.cuerpo) {
		t.Fatalf("contabilidad = %d B con %d entradas; guardado %d B", cache.bytes, len(cache.entrada), len(entrada.cuerpo))
	}
	// Una versión nueva del fichero sustituye a la anterior sin acumular.
	if err := os.WriteFile(filepath.Join(dir, "app.js"), append(contenido, "// v2\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	nueva, ok := cache.obtener(dir, "/app.js")
	if !ok || cache.bytes != len(nueva.cuerpo) || nueva.etiqueta == entrada.etiqueta {
		t.Fatalf("tras cambiar el fichero: %d B, guardado %d B", cache.bytes, len(nueva.cuerpo))
	}
}
