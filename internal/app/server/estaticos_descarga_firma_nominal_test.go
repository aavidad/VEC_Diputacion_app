package server

import (
	"bytes"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"vec-diputacion-granada/config"
)

func TestActivoDescargaFirmaNominalServidoSoloPorRutaExacta(t *testing.T) {
	const ruta = "/portal-empleado/modulos/contratacion-temporal/cliente-http-descarga-firma-v2.js"
	const rutaAjena = "/portal-empleado/modulos/contratacion-temporal/cliente-http-descarga-firma-v2.test.mjs"
	contenido, err := os.ReadFile("../../../web/static" + ruta)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("../../../web/static" + rutaAjena); err != nil {
		t.Fatalf("el activo ajeno debe existir para comprobar la lista positiva: %v", err)
	}
	api := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("un activo estático no debe alcanzar la API")
	})
	for nombre, handler := range map[string]http.Handler{
		"integrada": NewHandler(api),
		"interna":   NewHandlerInternoWithConfig(config.Config{}, api),
	} {
		t.Run(nombre, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta+"?v=20261014-firma-pdf-v1", nil))
			if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), contenido) {
				t.Fatalf("GET del módulo nominal = %d, bytes exactos = %v", rec.Code, bytes.Equal(rec.Body.Bytes(), contenido))
			}
			tipo, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
			if err != nil || (tipo != "text/javascript" && tipo != "application/javascript") {
				t.Fatalf("tipo de módulo nominal inválido: %q", rec.Header().Get("Content-Type"))
			}
			rec = httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, rutaAjena, nil))
			if rec.Code != http.StatusNotFound || rec.Header().Get("Location") != "" {
				t.Fatalf("activo ajeno = %d, redirección = %q; se esperaba 404", rec.Code, rec.Header().Get("Location"))
			}
		})
	}
}
