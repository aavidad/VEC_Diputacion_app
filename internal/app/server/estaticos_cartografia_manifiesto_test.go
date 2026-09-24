package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
)

func TestManifiestoCartografiaEmpaquetadaNoAbreRutasHTTP(t *testing.T) {
	const prefijo = "produccion.manifest\nstatic/bolsa/index.html\n"
	const zip = "cartografia/granada-base-20260719-z8-z12.zip"
	const indice = "cartografia/granada-base-20260719-z8-z12.json"
	rutas := rutasHTTPDesdeManifiestoWeb([]byte(prefijo + zip + "\n" + indice + "\n"))
	if _, ok := rutas["/bolsa/"]; !ok {
		t.Fatal("el contenido empaquetado ha cerrado la ruta web de Bolsa")
	}
	for _, ruta := range []string{"/" + zip, "/" + indice, "/cartografia/"} {
		if _, ok := rutas[ruta]; ok {
			t.Fatalf("se ha publicado por HTTP el contenido empaquetado %q", ruta)
		}
	}

	for nombre, invalida := range map[string]string{
		"zip adicional":      "cartografia/otro.zip",
		"indice adicional":   "cartografia/otro.json",
		"directorio ajeno":   "documentos/informe.pdf",
		"ruta no canonica":   "cartografia/../cartografia/granada-base-20260719-z8-z12.zip",
		"duplicado ZIP":      zip + "\n" + zip,
		"duplicado indice":   indice + "\n" + indice,
		"nombre por prefijo": zip + ".extra",
	} {
		t.Run(nombre, func(t *testing.T) {
			if rutas := rutasHTTPDesdeManifiestoWeb([]byte(prefijo + invalida + "\n")); len(rutas) != 0 {
				t.Fatalf("manifiesto invalido ha conservado %d rutas HTTP", len(rutas))
			}
		})
	}
}

func TestCartografiaEmpaquetadaDirectaDenegadaYSuperficiesVivas(t *testing.T) {
	for _, prueba := range []struct {
		nombre   string
		handler  http.Handler
		rutaViva string
	}{
		{"integrada", NewHandler(http.NotFoundHandler()), "/bolsa/"},
		{"publica", NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler()), "/bolsa/"},
		{"interna", NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()), "/portal-empleado/"},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			rec := httptest.NewRecorder()
			prueba.handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, prueba.rutaViva, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s = %d; esperado 200", prueba.rutaViva, rec.Code)
			}
			for _, ruta := range []string{
				"/cartografia/granada-base-20260719-z8-z12.zip",
				"/cartografia/granada-base-20260719-z8-z12.json",
			} {
				rec := httptest.NewRecorder()
				prueba.handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
				if rec.Code != http.StatusNotFound {
					t.Fatalf("GET %s = %d; esperado 404", ruta, rec.Code)
				}
			}
		})
	}
}
