package server

import (
	"bytes"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestActivosCompartidosTemaSoloSirveRutasExplicitasEnLectura(t *testing.T) {
	mux := http.NewServeMux()
	registrarActivosCompartidos(mux, staticFileServer())

	for _, recurso := range []struct {
		ruta string
		tipo string
	}{
		{ruta: "/comun/tema-vec.css", tipo: "text/css"},
		{ruta: "/comun/tema-vec.js", tipo: "javascript"},
		{ruta: "/comun/iconos-vec.js", tipo: "javascript"},
	} {
		esperado, err := os.ReadFile("../../../web/static" + recurso.ruta)
		if err != nil {
			t.Fatal(err)
		}
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			t.Run(metodo+recurso.ruta, func(t *testing.T) {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, peticionServidorPrueba(metodo, recurso.ruta, nil))
				if rec.Code != http.StatusOK {
					t.Fatalf("estado = %d, esperado 200", rec.Code)
				}
				if got := rec.Header().Get("Content-Type"); !strings.Contains(got, recurso.tipo) {
					t.Fatalf("Content-Type = %q, falta %q", got, recurso.tipo)
				}
				if metodo == http.MethodGet && !bytes.Equal(rec.Body.Bytes(), esperado) {
					t.Fatal("el cuerpo no coincide con el recurso en web/static")
				}
				if metodo == http.MethodHead && rec.Body.Len() != 0 {
					t.Fatalf("HEAD devolvió %d bytes de cuerpo", rec.Body.Len())
				}
			})
		}
		for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions} {
			t.Run(metodo+recurso.ruta, func(t *testing.T) {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, peticionServidorPrueba(metodo, recurso.ruta, nil))
				if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
					t.Fatalf("estado = %d, Allow = %q", rec.Code, rec.Header().Get("Allow"))
				}
			})
		}
	}

	for _, ruta := range []string{"/comun/", "/comun/tema-vec.test.mjs"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s = %d, esperado 404", ruta, rec.Code)
		}
	}
}

func TestHandlerPublicoSirveTemaComunCSSExacto(t *testing.T) {
	esperado, err := os.ReadFile("../../../web/static/comun/tema-vec.css")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	for _, caso := range []struct {
		metodo, ruta, cache string
	}{
		{http.MethodGet, "/comun/tema-vec.css", "no-cache"},
		{http.MethodHead, "/comun/tema-vec.css", "no-cache"},
		{http.MethodGet, "/comun/tema-vec.css?v=f2-publico", "public, max-age=31536000, immutable"},
		{http.MethodHead, "/comun/tema-vec.css?v=f2-publico", "public, max-age=31536000, immutable"},
	} {
		t.Run(caso.metodo+" "+caso.ruta, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(caso.metodo, caso.ruta, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("estado = %d, esperado 200", rec.Code)
			}
			tipo, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
			if err != nil || tipo != "text/css" {
				t.Fatalf("Content-Type = %q, error = %v", rec.Header().Get("Content-Type"), err)
			}
			if got := rec.Header().Get("Cache-Control"); got != caso.cache {
				t.Fatalf("Cache-Control = %q, esperado %q", got, caso.cache)
			}
			if got := rec.Header().Get("Content-Length"); got != strconv.Itoa(len(esperado)) {
				t.Fatalf("Content-Length = %q, esperado %d", got, len(esperado))
			}
			if caso.metodo == http.MethodGet && !bytes.Equal(rec.Body.Bytes(), esperado) {
				t.Fatal("GET no conserva los bytes del CSS")
			}
			if caso.metodo == http.MethodHead && rec.Body.Len() != 0 {
				t.Fatal("HEAD devolvió cuerpo")
			}
		})
	}
	for _, caso := range []struct {
		metodo, ruta string
		estado       int
	}{
		{http.MethodGet, "/comun/", http.StatusNotFound},
		{http.MethodPost, "/comun/tema-vec.css", http.StatusMethodNotAllowed},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(caso.metodo, caso.ruta, nil))
		if rec.Code != caso.estado {
			t.Fatalf("%s %s = %d, esperado %d", caso.metodo, caso.ruta, rec.Code, caso.estado)
		}
		if caso.estado == http.StatusMethodNotAllowed && rec.Header().Get("Allow") != "GET, HEAD" {
			t.Fatalf("Allow = %q", rec.Header().Get("Allow"))
		}
	}
}
