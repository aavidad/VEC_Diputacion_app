package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
