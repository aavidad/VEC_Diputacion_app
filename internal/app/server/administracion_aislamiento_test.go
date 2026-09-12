package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdministracionActivosRealesSeparadosDePortales(t *testing.T) {
	cfg := configuracionAdministracionPrueba()
	api := http.NotFoundHandler()
	admin := NewHandlerAdministracionWithConfig(cfg, api)
	rrhh := NewHandlerInternoWithConfig(cfg, api)
	integrado := NewHandlerWithConfig(cfg, api)
	publico := NewHandlerPublicoWithConfig(cfg, api)
	for _, ruta := range []string{"/administracion/", "/administracion/administracion.js", "/administracion/configuracion-correo.js", "/administracion/configuracion-correo.css", "/administracion/tema.css"} {
		t.Run(ruta, func(t *testing.T) {
			w := httptest.NewRecorder()
			admin.ServeHTTP(w, solicitudAdministracion(http.MethodGet, ruta))
			if w.Code != http.StatusOK || w.Body.Len() == 0 {
				t.Fatalf("activo ADMIN no disponible: estado %d", w.Code)
			}
			for nombre, h := range map[string]http.Handler{"rrhh": rrhh, "integrado": integrado, "publico": publico} {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, solicitudAdministracion(http.MethodGet, ruta))
				if w.Code != http.StatusNotFound {
					t.Errorf("%s expone activo ADMIN: %d", nombre, w.Code)
				}
			}
		})
	}
}
