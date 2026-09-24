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

func TestCatalogoVerificarSeSirveSoloEnLaSuperficiePublica(t *testing.T) {
	contenido, err := os.ReadFile("../../../web/static/verificar/i18n.js")
	if err != nil {
		t.Fatal(err)
	}
	const ruta = "/verificar/i18n.js"
	versionada := ruta + "?v=20260924-cotejo-espera-i18n-v1"
	publico := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())

	for _, superficie := range []struct {
		nombre  string
		handler http.Handler
	}{
		{nombre: "publica", handler: publico},
		{nombre: "produccion", handler: NewHandler(http.NotFoundHandler())},
	} {
		for _, caso := range []struct {
			metodo string
			ruta   string
			cache  string
		}{
			{http.MethodGet, ruta, "no-cache"},
			{http.MethodHead, ruta, "no-cache"},
			{http.MethodGet, versionada, "public, max-age=31536000, immutable"},
			{http.MethodHead, versionada, "public, max-age=31536000, immutable"},
		} {
			t.Run(superficie.nombre+" "+caso.metodo+" "+caso.ruta, func(t *testing.T) {
				rec := httptest.NewRecorder()
				superficie.handler.ServeHTTP(rec, peticionServidorPrueba(caso.metodo, caso.ruta, nil))
				if rec.Code != http.StatusOK {
					t.Fatalf("estado = %d; se esperaba 200", rec.Code)
				}
				tipo, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
				if err != nil || (tipo != "text/javascript" && tipo != "application/javascript") {
					t.Fatalf("Content-Type = %q; se esperaba JavaScript", rec.Header().Get("Content-Type"))
				}
				if got := rec.Header().Get("Cache-Control"); got != caso.cache {
					t.Fatalf("Cache-Control = %q; se esperaba %q", got, caso.cache)
				}
				if got := rec.Header().Get("Content-Length"); got != strconv.Itoa(len(contenido)) {
					t.Fatalf("Content-Length = %q; se esperaba %d", got, len(contenido))
				}
				if rec.Header().Get("Set-Cookie") != "" {
					t.Fatal("el recurso emitio una cookie")
				}
				if caso.metodo == http.MethodHead {
					if rec.Body.Len() != 0 {
						t.Fatal("HEAD devolvio cuerpo")
					}
				} else if !bytes.Equal(rec.Body.Bytes(), contenido) {
					t.Fatal("GET no devolvio los bytes exactos del catalogo")
				}
			})
		}
	}

	for _, caso := range []struct {
		metodo string
		ruta   string
		estado int
	}{
		{http.MethodGet, "/verificar/i18n.test.mjs", http.StatusNotFound},
		{http.MethodGet, ruta + ".map", http.StatusNotFound},
		{http.MethodPost, ruta, http.StatusMethodNotAllowed},
	} {
		rec := httptest.NewRecorder()
		publico.ServeHTTP(rec, peticionServidorPrueba(caso.metodo, caso.ruta, strings.NewReader("x")))
		if rec.Code != caso.estado {
			t.Errorf("%s %s = %d; se esperaba %d", caso.metodo, caso.ruta, rec.Code, caso.estado)
		}
		if caso.metodo == http.MethodPost && rec.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("POST Allow = %q; se esperaba GET, HEAD", rec.Header().Get("Allow"))
		}
		if rec.Header().Get("Set-Cookie") != "" {
			t.Error("la respuesta emitio una cookie")
		}
	}

	rec := httptest.NewRecorder()
	NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()).
		ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, versionada, nil))
	if rec.Code != http.StatusNotFound || bytes.Equal(rec.Body.Bytes(), contenido) {
		t.Fatalf("la superficie interna expuso el catalogo: estado=%d", rec.Code)
	}
	if rec.Header().Get("Set-Cookie") != "" {
		t.Fatal("la superficie interna emitio una cookie")
	}
}
