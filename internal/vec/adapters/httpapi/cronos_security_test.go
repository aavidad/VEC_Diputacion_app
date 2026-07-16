package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
)

func servirCronosCerradoConPermisosPreliminares(handler *Handler, rec *httptest.ResponseRecorder, req *http.Request) {
	principal := principalConPermisosExpresosPrueba(
		"vec.workspace.read",
		cronosmodule.PermissionTimeRead,
		cronosmodule.PermissionTimeManage,
		cronosmodule.PermissionLeaveRead,
		cronosmodule.PermissionLeaveManage,
	)
	switch vecPath(req.URL.Path) {
	case "/workspace":
		handler.handleWorkspace(rec, req, principal)
	case "/cronos/timecards":
		handler.handleCronosTimecards(rec, req, principal)
	case "/cronos/leave-requests":
		handler.handleCronosLeaveRequests(rec, req, principal)
	default:
		handler.writeError(rec, http.StatusNotFound, "ruta Cronos de prueba no soportada")
	}
}

func TestCronosLegacyPermisoPreliminarNoConcedeDatosNiOperaciones(t *testing.T) {
	handler := newTestHandler(t)
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/vec/workspace"},
		{method: http.MethodGet, path: "/api/vec/cronos/timecards?employee_id=EMP-0031"},
		{method: http.MethodPost, path: "/api/vec/cronos/timecards", body: `{"employee_id":"EMP-0031"}`},
		{method: http.MethodGet, path: "/api/vec/cronos/leave-requests?employee_id=EMP-0031"},
		{method: http.MethodPost, path: "/api/vec/cronos/leave-requests", body: `{"employee_id":"EMP-0031"}`},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		servirCronosCerradoConPermisosPreliminares(handler, rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s status = %d: %s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("%s %s Cache-Control = %q", tc.method, tc.path, got)
		}
		body := rec.Body.String()
		for _, forbidden := range []string{"EMP-0031", "Asunto propio", "Consulta medica", "Certificado automatico"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("%s %s leaked %q: %s", tc.method, tc.path, forbidden, body)
			}
		}
	}
}

func TestCronosLegacySinPermisoPositivoDeniega(t *testing.T) {
	handler := newTestHandler(t)
	for _, path := range []string{
		"/api/vec/cronos/timecards",
		"/api/vec/cronos/leave-requests",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Auth-Subject", "candidate")
		req.Header.Set("X-Auth-Mechanism", "clave")
		req.Header.Set("X-Auth-Roles", "ciudadano")
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("GET %s status = %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}
