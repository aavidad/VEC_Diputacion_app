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

func TestStaticHandlerProduccionSirveActivosConsumidosF2(t *testing.T) {
	rutas := []string{
		"/comun/tema-vec.css",
		"/comun/tema-vec.js",
		"/comun/iconos-vec.js",
		"/comun/idioma.js",
		"/comun/textos.js",
		"/comun/http.js",
		"/comun/registro-errores.js",
		"/comun/correos-propios.js",
		"/comun/correos-propios.css",
		"/comun/imagen-propia.js",
		"/comun/imagen-propia.css",
		"/textos/idiomas.json",
		"/textos/es/cronos.json",
		"/textos/en/cronos.json",
		"/comun/oportunidades/vista.js",
		"/comun/oportunidades/i18n.js",
		"/comun/oportunidades/oportunidades.css",
		"/textos/es/portal-ayuda.json",
		"/textos/en/portal-ayuda.json",
		"/portal-empleado/portal-panel-interno-i18n.js",
		"/portal-empleado/portal-i18n-contratos.js",
		"/portal-empleado/modulos/cronos/i18n-permisos.js",
		"/portal-empleado/modulos/dietas/i18n-borradores.js",
		"/portal-empleado/modulos/dietas/i18n-revision.js",
		"/portal-empleado/portal-contratos.css",
		"/portal-empleado/modulos/cronos/permisos.css",
		"/portal-empleado/modulos/dietas/borradores-propios.css",
		"/bolsa/i18n-publica.js",
		"/area-personal/i18n.js",
	}
	handler := staticHandler()
	for _, ruta := range rutas {
		t.Run(ruta, func(t *testing.T) {
			esperado, err := os.ReadFile("../../../web/static" + ruta)
			if err != nil {
				t.Fatal(err)
			}
			for _, metodo := range []string{http.MethodGet, http.MethodHead} {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, peticionServidorPrueba(metodo, ruta, nil))
				if rec.Code != http.StatusOK {
					t.Fatalf("%s %s = %d; esperado 200", metodo, ruta, rec.Code)
				}
				tipo, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
				if err != nil || (strings.HasSuffix(ruta, ".css") && tipo != "text/css") ||
					(strings.HasSuffix(ruta, ".js") && tipo != "text/javascript" && tipo != "application/javascript") ||
					(strings.HasSuffix(ruta, ".json") && tipo != "application/json") {
					t.Fatalf("%s %s Content-Type = %q", metodo, ruta, rec.Header().Get("Content-Type"))
				}
				if largo := rec.Header().Get("Content-Length"); largo != strconv.Itoa(len(esperado)) {
					t.Fatalf("%s %s Content-Length = %q; esperado %d", metodo, ruta, largo, len(esperado))
				}
				if metodo == http.MethodGet && !bytes.Equal(rec.Body.Bytes(), esperado) {
					t.Fatalf("GET %s no conserva los bytes del fichero", ruta)
				}
				if metodo == http.MethodHead && rec.Body.Len() != 0 {
					t.Fatalf("HEAD %s devolvió cuerpo", ruta)
				}
			}
		})
	}
}

func TestActivosHTTPComunesSoloRutasExactasEnPortalInterno(t *testing.T) {
	handler := NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler())
	for _, ruta := range []string{"/comun/http.js", "/comun/registro-errores.js"} {
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(metodo, ruta, nil))
			if rec.Code != http.StatusOK {
				t.Errorf("%s %s = %d; esperado 200", metodo, ruta, rec.Code)
			}
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodPost, ruta, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s = %d; esperado 405", ruta, rec.Code)
		}
	}
	for _, ruta := range []string{"/comun/http.test.mjs", "/comun/http.js/ajena", "/comun/registro-errores.test.mjs"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d; esperado 404", ruta, rec.Code)
		}
	}
}

func TestStaticHandlerProduccionDeniegaRecursosAjenoF2YMetodosDeEscritura(t *testing.T) {
	handler := staticHandler()
	for _, ruta := range []string{
		"/comun/oportunidades/vista.test.mjs",
		"/comun/tema-vec.test.mjs",
		"/portal-empleado/modulos/administracion/vista-apariencia.js",
		"/portal-empleado/modulos/seleccion/inscripciones/i18n.js",
		"/portal-empleado/modulos/seleccion/pruebas/i18n.js",
		"/portal-empleado/modulos/seleccion/comunicaciones/i18n.js",
		"/portal-empleado/portal-i18n-baremacion.js",
		"/portal-empleado/portal-i18n-convocatorias.js",
		"/portal-empleado/portal-baremacion.css",
		"/portal-empleado/portal-convocatorias.css",
		"/portal-empleado/portal-vistas-baremacion.js",
		"/portal-empleado/portal-vistas-convocatorias.js",
	} {
		if _, err := os.Stat("../../../web/static" + ruta); err != nil && !strings.Contains(ruta, "/modulos/seleccion/") {
			t.Fatalf("el recurso de contraste %s debe existir: %v", ruta, err)
		}
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(metodo, ruta, nil))
			if rec.Code != http.StatusNotFound {
				t.Errorf("%s %s = %d; esperado 404", metodo, ruta, rec.Code)
			}
		}
	}
	for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(metodo, "/comun/tema-vec.css", nil))
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("%s = %d, Allow = %q; esperado 405 y GET, HEAD", metodo, rec.Code, rec.Header().Get("Allow"))
		}
	}
}
