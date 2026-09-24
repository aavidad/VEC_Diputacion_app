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

func TestActivosContactoPropioSoloEnSuperficieIntegrada(t *testing.T) {
	rutas := []string{
		"static/area-personal/contacto-propio.js",
		"static/area-personal/i18n-contacto-propio.js",
	}
	manifiesto, err := os.ReadFile("../../../web/produccion.manifest")
	if err != nil {
		t.Fatal(err)
	}
	lineas := strings.Split(strings.TrimSpace(string(manifiesto)), "\n")
	for _, ruta := range rutas {
		cantidad := 0
		for _, linea := range lineas {
			if linea == ruta {
				cantidad++
			}
		}
		if cantidad != 1 {
			t.Fatalf("%s figura %d veces en el manifiesto integrado; esperado 1", ruta, cantidad)
		}
	}
	for _, nombre := range []string{"publico.manifest", "interno.manifest"} {
		contenido, err := os.ReadFile("../../../web/" + nombre)
		if err != nil {
			t.Fatal(err)
		}
		for _, linea := range strings.Split(string(contenido), "\n") {
			if strings.HasPrefix(linea, "static/area-personal/") {
				t.Fatalf("%s enumera el área personal: %s", nombre, linea)
			}
		}
	}

	integrada := NewHandlerWithConfig(config.Config{}, http.NotFoundHandler())
	publica := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	interna := NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler())
	for _, rutaFuente := range rutas {
		esperado, err := os.ReadFile("../../../web/" + rutaFuente)
		if err != nil {
			t.Fatal(err)
		}
		ruta := "/" + strings.TrimPrefix(rutaFuente, "static/")
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			for _, version := range []string{"", "?v=contacto-f2"} {
				t.Run(metodo+" "+ruta+version, func(t *testing.T) {
					respuesta := httptest.NewRecorder()
					integrada.ServeHTTP(respuesta, peticionServidorPrueba(metodo, ruta+version, nil))
					if respuesta.Code != http.StatusOK {
						t.Fatalf("estado = %d, esperado 200", respuesta.Code)
					}
					tipo, _, err := mime.ParseMediaType(respuesta.Header().Get("Content-Type"))
					if err != nil || tipo != "text/javascript" {
						t.Fatalf("Content-Type = %q: %v", respuesta.Header().Get("Content-Type"), err)
					}
					if got := respuesta.Header().Get("Content-Length"); got != strconv.Itoa(len(esperado)) {
						t.Fatalf("Content-Length = %q, esperado %d", got, len(esperado))
					}
					cache := "no-cache"
					if version != "" {
						cache = "public, max-age=31536000, immutable"
					}
					if got := respuesta.Header().Get("Cache-Control"); got != cache {
						t.Fatalf("Cache-Control = %q, esperado %q", got, cache)
					}
					if metodo == http.MethodGet && !bytes.Equal(respuesta.Body.Bytes(), esperado) {
						t.Fatal("GET no conserva los bytes del archivo")
					}
					if metodo == http.MethodHead && respuesta.Body.Len() != 0 {
						t.Fatal("HEAD devolvió cuerpo")
					}
				})
				for _, superficie := range []struct {
					nombre  string
					handler http.Handler
					estado  int
				}{
					{"publica", publica, http.StatusSeeOther},
					{"interna", interna, http.StatusNotFound},
				} {
					t.Run(superficie.nombre+" "+metodo+" "+ruta+version, func(t *testing.T) {
						respuesta := httptest.NewRecorder()
						superficie.handler.ServeHTTP(respuesta, peticionServidorPrueba(metodo, ruta+version, nil))
						if respuesta.Code != superficie.estado {
							t.Fatalf("estado = %d, esperado %d", respuesta.Code, superficie.estado)
						}
						if superficie.nombre == "publica" && respuesta.Header().Get("Location") != "/" {
							t.Fatalf("Location = %q, esperado /", respuesta.Header().Get("Location"))
						}
						if bytes.Equal(respuesta.Body.Bytes(), esperado) {
							t.Fatal("la superficie separada sirvió los bytes del activo")
						}
					})
				}
			}
		}
	}
}
