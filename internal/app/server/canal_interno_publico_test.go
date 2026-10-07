package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestCanalInternoSoloSirvePaquetePublicoExacto(t *testing.T) {
	for _, superficie := range []struct {
		nombre string
		http.Handler
	}{
		{"publica", NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())},
		{"integrada", NewHandler(http.NotFoundHandler())},
		{"integrada_externa", NewHandlerWithConfig(config.Config{PortalProceso: config.ValorPortalProcesoExterno}, http.NotFoundHandler())},
	} {
		t.Run(superficie.nombre, func(t *testing.T) {
			for _, caso := range []struct {
				ruta string
				tipo string
			}{
				{"/canal-interno/", "text/html"},
				{"/canal-interno/canal-interno.css", "text/css"},
				{"/canal-interno/canal-interno.js", "text/javascript"},
				{"/canal-interno/destinos.json", "application/json"},
				{"/textos/es/canal-interno.json", "application/json"},
				{"/textos/en/canal-interno.json", "application/json"},
				{"/textos/es/canal-interno-error.json", "application/json"},
				{"/textos/en/canal-interno-error.json", "application/json"},
			} {
				for _, metodo := range []string{http.MethodGet, http.MethodHead} {
					rec := httptest.NewRecorder()
					superficie.ServeHTTP(rec, peticionServidorPrueba(metodo, caso.ruta, nil))
					if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), caso.tipo) {
						t.Fatalf("%s %s = %d, %q", metodo, caso.ruta, rec.Code, rec.Header().Get("Content-Type"))
					}
					if rec.Header().Get("Set-Cookie") != "" || metodo == http.MethodHead && rec.Body.Len() != 0 {
						t.Fatalf("%s %s: cookie o cuerpo HEAD", metodo, caso.ruta)
					}
				}
			}
			for _, caso := range []struct {
				metodo string
				ruta   string
				estado int
			}{
				{http.MethodGet, "/canal-interno", http.StatusMovedPermanently},
				{http.MethodPost, "/canal-interno/", http.StatusMethodNotAllowed},
				{http.MethodGet, "/canal-interno/canal-interno.test.mjs", http.StatusNotFound},
				{http.MethodGet, "/canal-interno/../portal-empleado/", http.StatusNotFound},
				{http.MethodGet, "/canal-interno/formulario", http.StatusNotFound},
			} {
				rec := httptest.NewRecorder()
				superficie.ServeHTTP(rec, peticionServidorPrueba(caso.metodo, caso.ruta, nil))
				if rec.Code != caso.estado {
					t.Fatalf("%s %s = %d; esperado %d", caso.metodo, caso.ruta, rec.Code, caso.estado)
				}
			}
		})
	}
}

func TestCanalInternoNoSeMontaEnSuperficieInterna(t *testing.T) {
	handler := NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/canal-interno/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET interno = %d; esperado 404", rec.Code)
	}
}
