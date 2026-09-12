package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
)

func configuracionAdministracionPrueba() config.Config {
	return config.Config{
		Address:          "127.0.0.1:8443",
		HTTPAllowedCIDRs: []string{"127.0.0.1/32"},
		TransporteAdministracion: config.ConfiguracionTransporteAdministracion{
			Address:          "127.0.0.1:9443",
			HTTPAllowedCIDRs: []string{"127.0.0.1/32"},
			AllowedOrigins:   []string{"https://admin.test"},
			TLSCertFile:      "/estado/admin/cert.pem",
			TLSKeyFile:       "/estado/admin/key.pem",
			TLSClientCAFile:  "/estado/admin/ca.pem",
		},
	}
}

func solicitudAdministracion(metodo, ruta string) *http.Request {
	r := httptest.NewRequest(metodo, ruta, nil)
	r.RemoteAddr = "127.0.0.1:45321"
	r.Host = "admin.test"
	return r
}

func TestSuperficieAdministracionListaPositivaYAPIExacta(t *testing.T) {
	llamadas := 0
	handler := NewHandlerAdministracionWithConfig(configuracionAdministracionPrueba(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas++
		if r.URL.Path != rutaConfiguracionCorreoAdministracion {
			t.Errorf("ruta API inesperada: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	casos := []struct {
		ruta   string
		estado int
	}{
		{rutaConfiguracionCorreoAdministracion, http.StatusNoContent},
		{"/portal-empleado/portal-i18n.js", http.StatusOK},
		{"/api/vec/administracion/configuracion-correo/", http.StatusNotFound},
		{"/api/vec/administracion/configuracion-correo?x=1", http.StatusNotFound},
		{"/api/vec/contratacion-temporal/solicitudes", http.StatusNotFound},
		{"/portal-empleado/", http.StatusNotFound},
		{"/bolsa/", http.StatusNotFound},
		{"/administracion/ajeno.js", http.StatusNotFound},
	}
	for _, caso := range casos {
		t.Run(caso.ruta, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, solicitudAdministracion(http.MethodGet, caso.ruta))
			if respuesta.Code != caso.estado {
				t.Fatalf("%s = %d; se esperaba %d", caso.ruta, respuesta.Code, caso.estado)
			}
		})
	}
	if llamadas != 1 {
		t.Fatalf("API ADMIN recibio %d llamadas; se esperaba una", llamadas)
	}
}

func TestSuperficieAdministracionExigeOrigenParaCambioYNoAceptaFallback(t *testing.T) {
	llamadas := 0
	handler := NewHandlerAdministracionWithConfig(configuracionAdministracionPrueba(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusNoContent)
	}))
	for nombre, origen := range map[string]string{
		"ausente":    "",
		"ajeno":      "https://rrhh.test",
		"host ajeno": "https://admin.test",
	} {
		t.Run(nombre, func(t *testing.T) {
			r := solicitudAdministracion(http.MethodPut, rutaConfiguracionCorreoAdministracion)
			if nombre == "host ajeno" {
				r.Host = "otro.test"
			}
			if origen != "" {
				r.Header.Set("Origin", origen)
			}
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, r)
			if respuesta.Code != http.StatusForbidden {
				t.Fatalf("origen %q = %d; se esperaba 403", origen, respuesta.Code)
			}
		})
	}
	r := solicitudAdministracion(http.MethodPut, rutaConfiguracionCorreoAdministracion)
	r.Header.Set("Origin", "https://admin.test")
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, r)
	if respuesta.Code != http.StatusNoContent || llamadas != 1 {
		t.Fatalf("origen ADMIN autorizado: estado=%d llamadas=%d", respuesta.Code, llamadas)
	}

	sinAdmin := configuracionAdministracionPrueba()
	sinAdmin.TransporteAdministracion = config.ConfiguracionTransporteAdministracion{}
	respuesta = httptest.NewRecorder()
	NewHandlerAdministracionWithConfig(sinAdmin, http.NotFoundHandler()).ServeHTTP(respuesta, solicitudAdministracion(http.MethodGet, rutaConfiguracionCorreoAdministracion))
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("ausencia ADMIN = %d; se esperaba 503", respuesta.Code)
	}
}
