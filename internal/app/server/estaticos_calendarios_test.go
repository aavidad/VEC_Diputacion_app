package server

import (
	"bytes"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestStaticHandlerSirvePantallaCalendarios(t *testing.T) {
	const prefijo = "/portal-empleado/calendarios/"
	tipos := map[string]string{
		"index.html": "text/html", "calendarios.js": "text/javascript", "i18n.js": "text/javascript", "calendarios.css": "text/css",
	}
	handler := staticHandler(false)
	for asset, esperado := range tipos {
		t.Run(asset, func(t *testing.T) {
			contenido, err := os.ReadFile("../../../web/static" + prefijo + asset)
			if err != nil {
				t.Fatal(err)
			}
			ruta := prefijo + asset
			if asset == "index.html" {
				ruta = prefijo
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
			tipo, _, _ := mime.ParseMediaType(rec.Header().Get("Content-Type"))
			if rec.Code != http.StatusOK || (tipo != esperado && !(esperado == "text/javascript" && tipo == "application/javascript")) {
				t.Fatalf("GET %s = %d %q", asset, rec.Code, tipo)
			}
			if !bytes.Equal(rec.Body.Bytes(), contenido) || rec.Header().Get("Set-Cookie") != "" {
				t.Fatalf("GET %s no conserva los bytes o emite cookies", asset)
			}
		})
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, prefijo+"calendarios.test.mjs", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("las pruebas no se publican: %d", rec.Code)
	}
}
