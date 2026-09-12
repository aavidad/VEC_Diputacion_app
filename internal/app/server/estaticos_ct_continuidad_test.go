package server

import (
	"bytes"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestStaticHandlerProduccionSirveContinuidadContratacionTemporal(t *testing.T) {
	const prefijo = "/portal-empleado/modulos/contratacion-temporal/"
	assets := []string{
		"contrato-anotacion-administrativa.js",
		"cliente-http-anotacion-administrativa.js",
		"formulario-anotacion-administrativa.js",
		"contrato-cierre-administrativo.js",
		"cliente-http-cierre-administrativo.js",
		"formulario-cierre-administrativo.js",
	}
	handler := staticHandler(false)
	for _, asset := range assets {
		t.Run(asset, func(t *testing.T) {
			contenido, err := os.ReadFile("../../../web/static" + prefijo + asset)
			if err != nil {
				t.Fatal(err)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, prefijo+asset, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s = %d; se esperaba 200", asset, rec.Code)
			}
			tipo, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
			if err != nil || (tipo != "text/javascript" && tipo != "application/javascript") {
				t.Fatalf("GET %s Content-Type = %q; se esperaba JavaScript", asset, rec.Header().Get("Content-Type"))
			}
			if !bytes.Equal(rec.Body.Bytes(), contenido) {
				t.Fatalf("GET %s no conserva los bytes del fichero estático", asset)
			}
		})
	}
}

func TestStaticHandlerProduccionMantieneDenegadoAssetNoEnumerado(t *testing.T) {
	const ruta = "/portal-empleado/modulos/contratacion-temporal/anotacion-administrativa.test.mjs"
	if _, err := os.Stat("../../../web/static" + ruta); err != nil {
		t.Fatalf("el asset de contraste debe existir: %v", err)
	}
	rec := httptest.NewRecorder()
	staticHandler(false).ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET %s = %d; se esperaba 404", ruta, rec.Code)
	}
}
