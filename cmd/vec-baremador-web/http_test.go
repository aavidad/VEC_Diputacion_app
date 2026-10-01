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
