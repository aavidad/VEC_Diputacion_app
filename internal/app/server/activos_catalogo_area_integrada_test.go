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

func TestCatalogoAreaPersonalComunConservaBytesEnSuperficies(t *testing.T) {
	const rutaFuente = "static/textos/es/area-personal.json"
	const rutaHTTP = "/textos/es/area-personal.json"
	contenido, err := os.ReadFile("../../../web/" + rutaFuente)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := os.ReadFile("../../../web/produccion.manifest")
	if err != nil {
		t.Fatal(err)
	}
	cantidad := 0
	for _, ruta := range strings.Fields(string(manifiesto)) {
		if ruta == rutaFuente {
			cantidad++
		}
	}
	if cantidad != 1 {
		t.Fatalf("el catálogo figura %d veces en el manifiesto integrado; esperado 1", cantidad)
	}
	for _, nombre := range []string{"publico.manifest", "interno.manifest"} {
		separado, err := os.ReadFile("../../../web/" + nombre)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(separado, []byte(rutaFuente)) {
			t.Fatalf("%s enumera el catálogo del área personal", nombre)
		}
	}

	integrada := NewHandler(http.NotFoundHandler())
	publica := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	interna := NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler())
	for _, metodo := range []string{http.MethodGet, http.MethodHead} {
		for _, ruta := range []string{rutaHTTP, rutaHTTP + "?v=20260924"} {
			t.Run(metodo+" "+ruta, func(t *testing.T) {
				respuesta := httptest.NewRecorder()
				integrada.ServeHTTP(respuesta, peticionServidorPrueba(metodo, ruta, nil))
				if respuesta.Code != http.StatusOK {
					t.Fatalf("integrada = %d; esperado 200", respuesta.Code)
				}
				tipo, _, err := mime.ParseMediaType(respuesta.Header().Get("Content-Type"))
				if err != nil || tipo != "application/json" {
					t.Fatalf("Content-Type = %q: %v", respuesta.Header().Get("Content-Type"), err)
				}
				if got := respuesta.Header().Get("Content-Length"); got != strconv.Itoa(len(contenido)) {
					t.Fatalf("Content-Length = %q; esperado %d", got, len(contenido))
				}
				if got := respuesta.Header().Get("Cache-Control"); got != "no-cache" {
					t.Fatalf("Cache-Control = %q; esperado no-cache", got)
				}
				if metodo == http.MethodGet && !bytes.Equal(respuesta.Body.Bytes(), contenido) {
					t.Fatal("GET no conserva los bytes del catálogo")
				}
				if metodo == http.MethodHead && respuesta.Body.Len() != 0 {
					t.Fatal("HEAD devolvió cuerpo")
				}
				for _, superficie := range []struct {
					nombre  string
					handler http.Handler
					estado  int
				}{
					{"publica", publica, http.StatusOK},
					{"interna", interna, http.StatusOK},
				} {
					respuesta := httptest.NewRecorder()
					superficie.handler.ServeHTTP(respuesta, peticionServidorPrueba(metodo, ruta, nil))
					if respuesta.Code != superficie.estado {
						t.Fatalf("%s = %d; esperado %d", superficie.nombre, respuesta.Code, superficie.estado)
					}
					if superficie.nombre == "publica" && respuesta.Header().Get("Location") != "" {
						t.Fatalf("redirección pública = %q; esperado vacío", respuesta.Header().Get("Location"))
					}
					if superficie.estado == http.StatusOK && metodo == http.MethodGet && !bytes.Equal(respuesta.Body.Bytes(), contenido) {
						t.Fatalf("%s no conserva los bytes del catálogo común", superficie.nombre)
					}
					if metodo == http.MethodHead && respuesta.Body.Len() != 0 {
						t.Fatalf("%s HEAD devolvió cuerpo", superficie.nombre)
					}
					if respuesta.Header().Get("Cache-Control") != "no-cache" {
						t.Fatalf("%s no conserva no-cache", superficie.nombre)
					}
				}
			})
		}
	}

	respuesta := httptest.NewRecorder()
	integrada.ServeHTTP(respuesta, peticionServidorPrueba(http.MethodGet, "/textos/es/catalogo-inexistente.json", nil))
	if respuesta.Code != http.StatusNotFound {
		t.Fatalf("otro catálogo = %d; esperado 404", respuesta.Code)
	}
}
