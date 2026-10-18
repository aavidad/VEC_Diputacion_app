package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func TestVersionWebPropagaHojaHastaEntradaYCache(t *testing.T) {
	fuente := fstest.MapFS{
		"portal/index.html": {Data: []byte(`<script src="./portal.js?v=VEC_WEB_VERSION"></script>`)},
		"portal/portal.js":  {Data: []byte(`import "./hoja.js?v=VEC_WEB_VERSION";` + strings.Repeat(" ", 500))},
		"portal/hoja.js":    {Data: []byte(`export const valor = 1;`)},
		"portal/cache.json": {Data: []byte(`{"version":"VEC_WEB_VERSION","propios":["/portal/portal.js?v=VEC_WEB_VERSION"]}`)},
	}
	rutas := map[string]struct{}{"/portal/": {}, "/portal/index.html": {}, "/portal/portal.js": {}, "/portal/hoja.js": {}, "/portal/cache.json": {}}
	primera, err := calcularVersionWeb(fuente, rutas)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := calcularVersionWeb(fuente, rutas)
	if err != nil || primera.version != segunda.version {
		t.Fatalf("versión inestable: %v", err)
	}
	fuente["portal/hoja.js"].Data = []byte(`export const valor = 2;`)
	nueva, err := calcularVersionWeb(fuente, rutas)
	if err != nil || primera.version == nueva.version {
		t.Fatalf("la hoja debe renovar sus padres: %v", err)
	}
	for nombre, contenido := range nueva.textos {
		if bytes.Contains(contenido, []byte(marcaVersionWeb)) || !bytes.Contains(contenido, []byte(nueva.version)) {
			t.Fatalf("versión incoherente en %s", nombre)
		}
	}
	for _, caso := range []struct {
		metodo, ruta, codificacion string
		estado                     int
	}{
		{"GET", "/portal/", "", 200}, {"HEAD", "/portal/portal.js", "", 200}, {"GET", "/portal/portal.js", "gzip", 200},
	} {
		r := httptest.NewRequest(caso.metodo, caso.ruta, nil)
		r.Header.Set("Accept-Encoding", caso.codificacion)
		w := httptest.NewRecorder()
		if !nueva.servir(w, r) || w.Code != caso.estado {
			t.Fatalf("no se sirvió %s: %d", caso.ruta, w.Code)
		}
		if caso.metodo == "HEAD" && w.Body.Len() != 0 {
			t.Fatal("HEAD con cuerpo")
		}
		if caso.codificacion == "gzip" && !bytes.Equal(descomprimir(t, w.Body.Bytes()), nueva.textos[caso.ruta]) {
			t.Fatal("gzip diferente")
		}
	}
	r := httptest.NewRequest("GET", "/portal/portal.js", nil)
	r.Header.Set("If-None-Match", `W/"`+primera.version+`"`)
	w := httptest.NewRecorder()
	nueva.servir(w, r)
	if w.Code != http.StatusOK {
		t.Fatal("la versión anterior no puede validar los bytes nuevos")
	}
	r.Header.Set("If-None-Match", w.Header().Get("ETag"))
	w = httptest.NewRecorder()
	nueva.servir(w, r)
	if w.Code != http.StatusNotModified {
		t.Fatal("ETag vigente no revalida")
	}
	delete(fuente, "portal/hoja.js")
	if _, err := calcularVersionWeb(fuente, rutas); err == nil {
		t.Fatal("falta una hoja declarada")
	}
}

func leerFuenteWebVersionada(ruta string) ([]byte, error) {
	b, err := os.ReadFile(ruta)
	return bytes.ReplaceAll(b, []byte(marcaVersionWeb), []byte(versionWeb().version)), err
}

func TestVersionWebNoPublicaMarcasNiFijaVersionInventada(t *testing.T) {
	handler := staticHandler()
	for ruta := range versionWeb().textos {
		for _, version := range []string{versionWeb().version, "anterior"} {
			r := httptest.NewRequest("GET", ruta+"?v="+version, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != 200 || strings.Contains(w.Body.String(), marcaVersionWeb) {
				t.Fatalf("marca sin resolver: %s, estado %d", ruta, w.Code)
			}
			if strings.HasSuffix(ruta, ".js") || strings.HasSuffix(ruta, ".css") {
				immutable := strings.Contains(w.Header().Get("Cache-Control"), "immutable")
				if immutable != (version == versionWeb().version) {
					t.Fatalf("caché de %s con versión %s", ruta, version)
				}
			}
		}
	}
}
