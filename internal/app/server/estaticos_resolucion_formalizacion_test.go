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

// Comprueba el servido real, no sólo la presencia del fichero: los imports
// pueden existir en Git y seguir bloqueados por el manifiesto de staticHandler.
func TestModulosResolucionFormalizacionServidos(t *testing.T) {
	const prefijo = "/portal-empleado/modulos/contratacion-temporal/"
	modulos := []string{
		"cliente-http-resolucion-formalizacion.js",
		"contrato-resolucion-formalizacion.js",
		"formulario-resolucion-formalizacion.js",
	}
	api := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("la consulta estática no debe alcanzar la API")
	})
	for nombre, handler := range map[string]http.Handler{
		"integrada": NewHandler(api),
		"interna":   NewHandlerInternoWithConfig(config.Config{}, api),
	} {
		for _, modulo := range modulos {
			t.Run(nombre+"/"+modulo, func(t *testing.T) {
				original, err := os.ReadFile("../../../web/static" + prefijo + modulo)
				if err != nil {
					t.Fatal(err)
				}
				for _, metodo := range []string{http.MethodGet, http.MethodHead} {
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, peticionServidorPrueba(metodo, prefijo+modulo, nil))
					if rec.Code != http.StatusOK {
						t.Fatalf("%s: estado %d; se esperaba 200", metodo, rec.Code)
					}
					tipo, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
					if err != nil || (tipo != "text/javascript" && tipo != "application/javascript") {
						t.Fatalf("%s: tipo JavaScript no válido: %q", metodo, tipo)
					}
					if metodo == http.MethodGet && !bytes.Equal(rec.Body.Bytes(), original) {
						t.Fatal("el cuerpo servido no coincide con el módulo original")
					}
					if metodo == http.MethodHead && rec.Body.Len() != 0 {
						t.Fatal("HEAD devolvió un cuerpo")
					}
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodPost, prefijo+modulo, nil))
				if rec.Code != http.StatusMethodNotAllowed {
					t.Fatalf("POST: estado %d; se esperaba 405", rec.Code)
				}
			})
		}
	}
	publico := NewHandlerPublicoWithConfig(config.Config{}, api)
	for _, modulo := range modulos {
		rec := httptest.NewRecorder()
		publico.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, prefijo+modulo, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("la superficie pública expuso %s: %d", modulo, rec.Code)
		}
	}
}
