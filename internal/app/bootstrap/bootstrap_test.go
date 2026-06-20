package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestNewHTTPServerWithConfigComposesDemoAPI(t *testing.T) {
	srv, err := NewHTTPServerWithConfig(config.Config{
		Address:           "127.0.0.1:0",
		APIBasePath:       "/api",
		ReadHeaderTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewHTTPServerWithConfig() error = %v", err)
	}
	if srv.Addr != "127.0.0.1:0" || srv.ReadHeaderTimeout != time.Second {
		t.Fatalf("server config = %s/%s", srv.Addr, srv.ReadHeaderTimeout)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/demo", nil)
	req.Header.Set("X-Auth-Mechanism", string(ports.AuthMechanismKerberosAD))
	req.Header.Set("X-Auth-Subject", "staff")
	req.Header.Set("Authorization", "Bearer staff-token")
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/api/demo status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "demo-convocatoria") {
		t.Fatalf("/api/demo body missing convocatoria: %s", body)
	}
}

func TestNewHTTPServerExposesUnifiedVECShellModules(t *testing.T) {
	srv, err := NewHTTPServerWithConfig(config.Config{
		Address:           "127.0.0.1:0",
		APIBasePath:       "/api",
		ReadHeaderTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewHTTPServerWithConfig() error = %v", err)
	}

	for _, tc := range []struct {
		method string
		path   string
		want   string
		status int
	}{
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.personal", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.cronos", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.dietas", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.bolsa", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/workspace", want: "Certificado automatico para meritos Bolsa", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/workspace", want: "Granada - Motril", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/menu", want: "personal.nominas", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/menu", want: "cronos.fichajes", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/menu", want: "dietas.kilometraje", status: http.StatusOK},
		{method: http.MethodPost, path: "/api/vec/modules/cronos/action", want: "cronos.jornada.justificacion.review", status: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/vec/modules/horarios/action", want: "cronos.jornada.justificacion.review", status: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/vec/modules/permisos/action", want: "cronos.permiso.vacacion.review", status: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/vec/modules/dietas/action", want: "dietas.comision.review", status: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/vec/modules/rutas/action", want: "dietas.ruta.km.review", status: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/vec/modules/personal/action", want: "personal.certificado.servicios.issue", status: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/vec/modules/nominas/action", want: "personal.nomina.incidencia.review", status: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/vec/modules/bolsa/action", want: "bolsa.demo.integration", status: http.StatusAccepted},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("X-Auth-Mechanism", string(ports.AuthMechanismKerberosAD))
		req.Header.Set("X-Auth-Subject", "staff")
		req.Header.Set("Authorization", "Bearer staff-token")
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("%s %s status = %d, want %d: %s", tc.method, tc.path, rec.Code, tc.status, rec.Body.String())
		}
		if body := rec.Body.String(); !strings.Contains(body, tc.want) {
			t.Fatalf("%s %s body missing %q: %s", tc.method, tc.path, tc.want, body)
		}
	}
}
