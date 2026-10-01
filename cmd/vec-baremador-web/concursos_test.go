package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
)

func TestConcursosHTTPUsaMotorYEntradaLigada(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	w := request(h, "GET", "/api/provision/v1/configuracion-local", "", nil)
	var catalogo struct {
		Ejemplos []simulacion.Ejemplo `json:"ejemplos"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &catalogo) != nil || len(catalogo.Ejemplos) == 0 {
		t.Fatal("catálogo Concursos no disponible")
	}
	for _, e := range catalogo.Ejemplos {
		solicitud := map[string]any{"ejemplo_ref": e.Referencia, "configuracion": e.Configuracion}
		b, err := json.Marshal(solicitud)
		if err != nil {
			t.Fatal(err)
		}
		w = request(h, "POST", "/api/provision/v1/simulaciones", string(b), nil)
		r, err := application.Simular(e.Configuracion, e.Entrada)
		if err != nil {
			t.Fatal(err)
		}
		esperado, err := json.Marshal(simulacion.Sobre(r))
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || !bytes.Equal(bytes.TrimSpace(w.Body.Bytes()), esperado) {
			t.Fatalf("HTTP difiere del motor: %d %s", w.Code, w.Body.String())
		}
		solicitud["entrada"] = e.Entrada
		b, _ = json.Marshal(solicitud)
		if w = request(h, "POST", "/api/provision/v1/simulaciones", string(b), nil); w.Code != 400 {
			t.Fatalf("datos de personas admitidos: %d", w.Code)
		}
	}
}

func TestConcursosComparteFronteraDeOrigenYCarga(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	for _, caso := range []struct {
		nombre, cuerpo string
		estado         int
		alterar        func(*http.Request)
	}{
		{"origen", `{}`, 403, func(r *http.Request) { r.Header.Del("Origin") }},
		{"host", `{}`, 403, func(r *http.Request) { r.Host = "evil.example" }},
		{"consulta", `{}`, 400, func(r *http.Request) { r.URL.RawQuery = "ejemplo_ref=x" }},
		{"duplicada", `{"ejemplo_ref":"x","ejemplo_ref":"y","configuracion":{}}`, 400, nil},
		{"concatenada", `{} {}`, 400, nil},
		{"metodo", `{}`, 405, func(r *http.Request) { r.Method = "GET" }},
		{"activacion", `{}`, 404, func(r *http.Request) { r.URL.Path = "/api/provision/v1/activar" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			w := request(h, "POST", "/api/provision/v1/simulaciones", caso.cuerpo, caso.alterar)
			if w.Code != caso.estado {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
}
