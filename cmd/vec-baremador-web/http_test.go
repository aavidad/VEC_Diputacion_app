package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/simuladorlocal"
)

const hostPrueba = "127.0.0.1:43219"

func request(h http.Handler, method, path, body string, alterar func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://"+hostPrueba+path, strings.NewReader(body))
	r.Header.Set("Origin", "http://"+hostPrueba)
	r.Header.Set("Content-Type", "application/json")
	if alterar != nil {
		alterar(r)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestHTTPMismoResultadoQueMotor(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	w := request(h, "GET", "/ejemplos", "", nil)
	var catalogo struct {
		Ejemplos []simuladorlocal.Ejemplo `json:"ejemplos"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &catalogo) != nil || len(catalogo.Ejemplos) != 2 {
		t.Fatal("catálogo no disponible")
	}
	for _, e := range catalogo.Ejemplos {
		s := simuladorlocal.Solicitud{Modo: e.Modo, EjemploRef: e.Referencia, Reglas: e.Reglas}
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		w = request(h, "POST", "/simular", string(b), nil)
		esperado, err := (simuladorlocal.Motor{}).Simular(s)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), esperado) {
			t.Fatalf("resultado HTTP distinto: %d %s", w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" || w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("frontera de almacenamiento/origen")
		}
	}
}

func TestHTTPRechazaOrigenMetodoYCarga(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	for _, c := range []struct {
		nombre, method, path, body string
		status                     int
		alterar                    func(*http.Request)
	}{
		{"host", "GET", "/ejemplos", "", 403, func(r *http.Request) { r.Host = "evil.example" }},
		{"origin_ausente", "POST", "/simular", "{}", 403, func(r *http.Request) { r.Header.Del("Origin") }},
		{"origin_ajeno", "POST", "/simular", "{}", 403, func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") }},
		{"origin_multiple", "POST", "/simular", "{}", 403, func(r *http.Request) { r.Header.Add("Origin", "http://"+hostPrueba) }},
		{"cookie", "GET", "/ejemplos", "", 403, func(r *http.Request) { r.Header.Set("Cookie", "session=x") }},
		{"autorizacion", "GET", "/ejemplos", "", 403, func(r *http.Request) { r.Header.Set("Authorization", "Bearer x") }},
		{"tipo", "POST", "/simular", "{}", 415, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }},
		{"compresion", "POST", "/simular", "{}", 415, func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }},
		{"metodo", "GET", "/simular", "", 405, nil},
		{"ruta", "POST", "/activar", "{}", 404, nil},
		{"archivos", "GET", "/../../go.mod", "", 404, nil},
		{"unknown", "POST", "/simular", `{"reglas":{},"entrada":{}}`, 400, nil},
		{"bytes", "POST", "/simular", strings.Repeat("x", simuladorlocal.MaximoBytes+1), 413, nil},
		{"reglas", "POST", "/simular", `{"modo":"experiencia","ejemplo_ref":"experiencia_sintetica_v1","reglas":{}}`, 422, nil},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			w := request(h, c.method, c.path, c.body, c.alterar)
			if w.Code != c.status {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestRecursosCerradosYSinEscape(t *testing.T) {
	dir := t.TempDir()
	for _, r := range recursos {
		p := filepath.Join(dir, r)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("asset"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "textos/idiomas.json"), []byte(`{"idiomas":[{"codigo":"zz"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "textos/zz"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "textos/zz/baremo-bolsa.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "textos/zz/baremo-concursos.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "textos/zz/provision.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	restos := `{"esquema":"catalogo_sintetico.v1","version":1}`
	if err := os.WriteFile(filepath.Join(dir, "catalogos/baremo-restos-v1.json"), []byte(restos), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "textos/zz/seleccion.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	assets, err := cargarRecursos(dir)
	if err != nil {
		t.Fatal(err)
	}
	w := request(nuevoHandler(hostPrueba, assets), "GET", entradaWeb, "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	w = request(nuevoHandler(hostPrueba, assets), "GET", entradaSeleccion, "", nil)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("entrada del ensayo de selección no disponible")
	}
	if _, ok := assets["/textos/zz/baremo-bolsa.json"]; !ok {
		t.Fatal("idioma del índice omitido")
	}
	w = request(nuevoHandler(hostPrueba, assets), "GET", "/catalogos/baremo-jornada-v1.json", "", nil)
	if w.Code != 200 || w.Body.String() != "asset" || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatal("catalogo tecnico de jornada no disponible")
	}
	w = request(nuevoHandler(hostPrueba, assets), "GET", "/catalogos/baremo-restos-v1.json", "", nil)
	if w.Code != 200 || w.Body.String() != restos || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatal("catalogo tecnico de restos no disponible")
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("catalogo de restos altera la frontera de almacenamiento")
	}
	if request(nuevoHandler(hostPrueba, assets), "POST", "/catalogos/baremo-restos-v1.json", "{}", nil).Code != 405 {
		t.Fatal("catalogo tecnico de restos admite escritura")
	}
	if request(nuevoHandler(hostPrueba, assets), "GET", "/catalogos/baremo-restos-v2.json", "", nil).Code != 404 {
		t.Fatal("version ajena del catalogo disponible")
	}
	if request(nuevoHandler(hostPrueba, assets), "GET", "/catalogos/no-declarado.json", "", nil).Code != 404 {
		t.Fatal("catalogo ajeno a la lista positiva disponible")
	}
	rutaRestos := filepath.Join(dir, "catalogos/baremo-restos-v1.json")
	if err := os.Remove(rutaRestos); err != nil {
		t.Fatal(err)
	}
	sinRestos, err := cargarRecursos(dir)
	if err != nil {
		t.Fatal("ausencia del catalogo tecnico impide abrir las capacidades existentes", err)
	}
	h := nuevoHandler(hostPrueba, sinRestos)
	if request(h, "GET", entradaWeb, "", nil).Code != 200 || request(h, "GET", "/catalogos/baremo-restos-v1.json", "", nil).Code != 404 {
		t.Fatal("ausencia del catalogo tecnico altera la entrada o publica contenido")
	}
	for _, contenido := range []string{"", strings.Repeat("x", 1024*1024+1)} {
		if err := os.WriteFile(rutaRestos, []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := cargarRecursos(dir); err == nil {
			t.Fatal("catalogo tecnico invalido tratado como ausente")
		}
	}
	if err := os.Remove(rutaRestos); err != nil {
		t.Fatal(err)
	}
	exterior := filepath.Join(t.TempDir(), "restos.json")
	if err := os.WriteFile(exterior, []byte(restos), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exterior, rutaRestos); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarRecursos(dir); err == nil {
		t.Fatal("escape del catalogo por enlace tratado como ausencia")
	}
	if err := os.Remove(rutaRestos); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rutaRestos, []byte(restos), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "textos/idiomas.json"), []byte(`{"idiomas":[{"codigo":"../../etc"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarRecursos(dir); err == nil {
		t.Fatal("ruta desde código de idioma admitida")
	}
	if err := os.WriteFile(filepath.Join(dir, "textos/idiomas.json"), []byte(`{"idiomas":[{"codigo":"zz"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, recursos[0])); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(dir, recursos[0])); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarRecursos(dir); err == nil {
		t.Fatal("escape por enlace admitido")
	}
}
