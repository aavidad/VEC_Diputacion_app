package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestPWARecursosPorFrontera(t *testing.T) {
	api := http.NotFoundHandler()
	configuracion := config.Config{HTTPAllowedCIDRs: []string{"127.0.0.1/8"}}
	casos := []struct {
		nombre   string
		handler  http.Handler
		rutas    []string
		cerradas []string
	}{
		{
			nombre: "interno", handler: NewHandlerInternoWithConfig(configuracion, api),
			rutas:    []string{"/portal-empleado/", "/portal-empleado/sw.js", "/portal-empleado/cache-publica-v1.json", "/pwa/instalar.js", "/pwa/icons/vec-192.png", "/textos/es/pwa-portal-empleado.json"},
			cerradas: []string{"/area-personal/sw.js", "/area-personal/cache-publica-v1.json", "/administracion-perfiles/sw.js", "/administracion-perfiles/cache-publica-v1.json", "/textos/es/pwa-admin.json", "/pwa/manifiestos.test.mjs"},
		},
		{
			nombre: "externo", handler: NewHandlerWithConfig(config.Config{PortalProceso: "externo", HTTPAllowedCIDRs: []string{"127.0.0.1/8"}}, api),
			rutas:    []string{"/area-personal/", "/area-personal/sw.js", "/area-personal/cache-publica-v1.json", "/pwa/instalar.js", "/pwa/icons/vec-512.png", "/textos/en/pwa-area-personal.json"},
			cerradas: []string{"/portal-empleado/sw.js", "/portal-empleado/cache-publica-v1.json", "/administracion-perfiles/sw.js", "/administracion-perfiles/cache-publica-v1.json", "/textos/es/pwa-admin.json", "/pwa/sw-public-assets.test.mjs"},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			for _, ruta := range caso.rutas {
				rec := httptest.NewRecorder()
				caso.handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
				if rec.Code != http.StatusOK {
					t.Errorf("GET %s = %d; esperado 200", ruta, rec.Code)
				}
				if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "worker-src 'self'; manifest-src 'self'") {
					t.Errorf("GET %s sin CSP PWA acotada", ruta)
				}
			}
			for _, ruta := range caso.cerradas {
				rec := httptest.NewRecorder()
				caso.handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
				if rec.Code != http.StatusNotFound {
					t.Errorf("GET %s = %d; esperado 404", ruta, rec.Code)
				}
			}
		})
	}
}

func TestPWAIconoVersionadoYManifiestoSinCacheHTTP(t *testing.T) {
	handler := NewHandlerInternoWithConfig(config.Config{HTTPAllowedCIDRs: []string{"127.0.0.1/8"}}, http.NotFoundHandler())
	for _, caso := range []struct{ ruta, cache string }{
		{"/pwa/icons/vec-192.png?v=20261002-pwa-v1", "public, max-age=31536000, immutable"},
		{"/pwa/icons/vec-192.png", "no-cache"},
		{"/textos/es/pwa-portal-empleado.json?v=20261002-pwa-v1", "no-cache"},
		{"/portal-empleado/cache-publica-v1.json?v=20261002-pwa-v4", "no-store"},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, caso.ruta, nil))
		if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != caso.cache {
			t.Errorf("GET %s = %d, cache = %q; esperado 200 y %q", caso.ruta, rec.Code, rec.Header().Get("Cache-Control"), caso.cache)
		}
	}
}
