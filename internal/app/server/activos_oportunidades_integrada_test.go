package server

import (
	"bytes"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"vec-diputacion-granada/config"
)

func TestActivosOportunidadesSoloEnSuperficieIntegrada(t *testing.T) {
	rutas := []struct {
		ruta string
		tipo string
	}{
		{"/comun/oportunidades/vista.js", "text/javascript"},
		{"/comun/oportunidades/i18n.js", "text/javascript"},
		{"/comun/oportunidades/oportunidades.css", "text/css"},
	}
	integrada := NewHandlerWithConfig(config.Config{}, http.NotFoundHandler())
	publica := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	interna := NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler())

	for _, archivo := range rutas {
		esperado, err := os.ReadFile("../../../web/static" + archivo.ruta)
		if err != nil {
			t.Fatal(err)
		}
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			for _, version := range []string{"", "?v=b15"} {
				ruta := archivo.ruta + version
				t.Run(metodo+" "+ruta, func(t *testing.T) {
					rec := httptest.NewRecorder()
					integrada.ServeHTTP(rec, peticionServidorPrueba(metodo, ruta, nil))
					if rec.Code != http.StatusOK {
						t.Fatalf("estado = %d, esperado 200", rec.Code)
					}
					tipo, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
					if err != nil || tipo != archivo.tipo {
						t.Fatalf("Content-Type = %q, esperado %q: %v", rec.Header().Get("Content-Type"), archivo.tipo, err)
					}
					if got := rec.Header().Get("Content-Length"); got != strconv.Itoa(len(esperado)) {
						t.Fatalf("Content-Length = %q, esperado %d", got, len(esperado))
					}
					cache := "no-cache"
					if version != "" {
						cache = "public, max-age=31536000, immutable"
					}
					if got := rec.Header().Get("Cache-Control"); got != cache {
						t.Fatalf("Cache-Control = %q, esperado %q", got, cache)
					}
					if metodo == http.MethodGet && !bytes.Equal(rec.Body.Bytes(), esperado) {
						t.Fatal("GET no conserva los bytes del archivo")
					}
					if metodo == http.MethodHead && rec.Body.Len() != 0 {
						t.Fatal("HEAD devolvió cuerpo")
					}
				})
			}
		}
		for _, superficie := range []struct {
			nombre  string
			handler http.Handler
		}{
			{"publica", publica},
			{"interna", interna},
		} {
			for _, metodo := range []string{http.MethodGet, http.MethodHead} {
				t.Run(superficie.nombre+" "+metodo+" "+archivo.ruta, func(t *testing.T) {
					rec := httptest.NewRecorder()
					superficie.handler.ServeHTTP(rec, peticionServidorPrueba(metodo, archivo.ruta, nil))
					if rec.Code != http.StatusNotFound {
						t.Fatalf("estado = %d, esperado 404", rec.Code)
					}
				})
			}
		}
		for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
			rec := httptest.NewRecorder()
			integrada.ServeHTTP(rec, peticionServidorPrueba(metodo, archivo.ruta, nil))
			if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
				t.Errorf("%s %s = %d, Allow = %q; esperado 405 y GET, HEAD", metodo, archivo.ruta, rec.Code, rec.Header().Get("Allow"))
			}
		}
	}

	for _, ruta := range []string{
		"/comun/oportunidades/",
		"/comun/oportunidades/vista.test.mjs",
		"/comun/oportunidades/otro.js",
	} {
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			rec := httptest.NewRecorder()
			integrada.ServeHTTP(rec, peticionServidorPrueba(metodo, ruta, nil))
			if rec.Code != http.StatusNotFound {
				t.Errorf("%s %s = %d, esperado 404", metodo, ruta, rec.Code)
			}
		}
	}
}
