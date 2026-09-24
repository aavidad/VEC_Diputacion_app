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
)

func TestStaticHandlerProduccionSirveActivosConsumidosF2(t *testing.T) {
	rutas := []string{
		"/comun/tema-vec.css",
		"/comun/tema-vec.js",
		"/portal-empleado/modulos/administracion/vista-apariencia.js",
		"/portal-empleado/portal-i18n-baremacion.js",
		"/portal-empleado/portal-i18n-contratos.js",
		"/portal-empleado/portal-i18n-convocatorias.js",
		"/portal-empleado/modulos/cronos/i18n-permisos.js",
		"/portal-empleado/modulos/dietas/i18n-borradores.js",
		"/portal-empleado/modulos/dietas/i18n-revision.js",
		"/portal-empleado/portal-baremacion.css",
		"/portal-empleado/portal-contratos.css",
		"/portal-empleado/portal-convocatorias.css",
		"/portal-empleado/modulos/cronos/permisos.css",
		"/portal-empleado/modulos/dietas/borradores-propios.css",
		"/bolsa/i18n-publica.js",
		"/area-personal/i18n.js",
	}
	handler := staticHandler(false)
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
					(strings.HasSuffix(ruta, ".js") && tipo != "text/javascript" && tipo != "application/javascript") {
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

func TestStaticHandlerProduccionDeniegaRecursosAjenoF2YMetodosDeEscritura(t *testing.T) {
	handler := staticHandler(false)
	for _, ruta := range []string{
		"/portal-empleado/modulos/seleccion/inscripciones/i18n.js",
		"/portal-empleado/modulos/seleccion/pruebas/i18n.js",
		"/portal-empleado/modulos/seleccion/comunicaciones/i18n.js",
	} {
		t.Run("consumido/"+ruta, func(t *testing.T) {
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
				if err != nil || (tipo != "text/javascript" && tipo != "application/javascript") {
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
	for _, ruta := range []string{
		"/comun/oportunidades/i18n.js",
		"/portal-empleado/modulos/seleccion/inscripciones/inscripciones.test.mjs",
		"/comun/tema-vec.test.mjs",
	} {
		if _, err := os.Stat("../../../web/static" + ruta); err != nil {
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
