package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestSuperficiePublicaSirveLandingAccesoSinIdentidad(t *testing.T) {
	handler := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	for _, prueba := range []struct {
		metodo string
		ruta   string
		texto  string
	}{
		{http.MethodGet, "/", "Acceso a VEC"},
		{http.MethodHead, "/", ""},
		{http.MethodGet, "/acceso/", "Acceso a VEC"},
		{http.MethodGet, "/acceso/acceso.css?v=1", ".acceso-principal"},
		{http.MethodGet, "/acceso/acceso-i18n.js?v=1", "CATALOGOS_EMPAQUETADOS"},
	} {
		t.Run(prueba.metodo+" "+prueba.ruta, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(prueba.metodo, prueba.ruta, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("%s %s = %d; se esperaba 200", prueba.metodo, prueba.ruta, rec.Code)
			}
			if prueba.texto != "" && !strings.Contains(rec.Body.String(), prueba.texto) {
				t.Fatalf("%s no contiene %q", prueba.ruta, prueba.texto)
			}
			if prueba.metodo == http.MethodGet && prueba.ruta == "/" {
				landing := rec.Body.String()
				for _, contenido := range []string{
					"Cl@ve", "Certificado digital", "DNIe", "Acceso pendiente de configuración",
					"/bolsa/#ayuda-publica",
				} {
					if !strings.Contains(landing, contenido) {
						t.Errorf("landing no contiene %q", contenido)
					}
				}
				if strings.Contains(landing, "auth.vec.dipgra.cloud") ||
					strings.Contains(landing, "href=\"/acceso/inicio/") ||
					strings.Count(landing, "disabled aria-describedby=\"estado-acceso\"") != 3 {
					t.Error("landing ofrece un acceso autenticado activo o no deshabilita los tres métodos")
				}
			}
		})
	}
}

func TestSuperficiePublicaReservaIniciosAccesoSinSesion(t *testing.T) {
	handler := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	for _, metodo := range []string{"clave", "certificado", "dnie"} {
		for _, verbo := range []string{http.MethodGet, http.MethodHead} {
			t.Run(verbo+" "+metodo, func(t *testing.T) {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, peticionServidorPrueba(verbo, "/acceso/inicio/"+metodo, nil))
				if rec.Code != http.StatusServiceUnavailable {
					t.Fatalf("%s %s = %d; se esperaba 503", verbo, metodo, rec.Code)
				}
				if rec.Header().Get("Set-Cookie") != "" || rec.Header().Get("Location") != "" {
					t.Fatalf("%s emitio estado de sesion o redireccion", metodo)
				}
				if verbo == http.MethodGet && !strings.Contains(rec.Body.String(), "acceso no configurado") {
					t.Fatalf("%s no declara el estado generico", metodo)
				}
			})
		}
	}
}

func TestSuperficiePublicaLandingMantieneFronteraYSoloLectura(t *testing.T) {
	handler := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	for _, ruta := range []string{"/portal-empleado/", "/api/vec/consulta", "/area-personal/"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d; se esperaba 404", ruta, rec.Code)
		}
	}
	for _, ruta := range []string{"/", "/acceso/", "/acceso/inicio/clave"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodPost, ruta, strings.NewReader("x")))
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("POST %s = %d Allow=%q; se esperaba 405", ruta, rec.Code, rec.Header().Get("Allow"))
		}
	}
	for _, cabecera := range []string{"Cookie", "Authorization"} {
		rec := httptest.NewRecorder()
		req := peticionServidorPrueba(http.MethodGet, "/", nil)
		req.Header[cabecera] = []string{""}
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s en landing = %d; se esperaba 400", cabecera, rec.Code)
		}
	}
}

func TestSuperficiePublicaExponeCatalogoEspanolLandingAcceso(t *testing.T) {
	handler := NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/acceso/locales/es.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("catalogo acceso = %d; se esperaba 200", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("tipo catalogo = %q; se esperaba JSON", rec.Header().Get("Content-Type"))
	}
	var catalogo map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &catalogo); err != nil {
		t.Fatalf("catalogo acceso no es JSON: %v", err)
	}
	for _, clave := range []string{
		"acceso.titulo_pagina", "acceso.estado.titulo", "acceso.metodo.clave.titulo",
		"acceso.metodo.certificado.titulo", "acceso.metodo.dnie.titulo", "acceso.metodo.pendiente",
		"acceso.enlaces.bolsa",
	} {
		if strings.TrimSpace(catalogo[clave]) == "" {
			t.Errorf("catalogo sin traduccion por defecto para %q", clave)
		}
	}
	landing := httptest.NewRecorder()
	handler.ServeHTTP(landing, peticionServidorPrueba(http.MethodGet, "/", nil))
	for _, marca := range []string{
		"lang=\"es\"", "data-i18n-catalogo=\"/acceso/locales/es.json\"",
		"data-i18n=\"acceso.titulo\"", "data-i18n=\"acceso.metodo.pendiente\"",
		"data-i18n-atributo=\"aria-label:acceso.enlaces.etiqueta\"",
		"src=\"/acceso/acceso-i18n.js\"",
	} {
		if !strings.Contains(landing.Body.String(), marca) {
			t.Errorf("landing no declara extension i18n %q", marca)
		}
	}
}
