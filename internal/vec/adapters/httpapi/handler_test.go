package httpapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

func TestHandlerListsModulesWorkspaceMenuAndRunsModuleActions(t *testing.T) {
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	for _, manifest := range []string{
		personalmodule.Manifest().ID,
		cronosmodule.Manifest().ID,
		dietasmodule.Manifest().ID,
		bolsamodule.Manifest().ID,
	} {
		if manifest == "" {
			t.Fatalf("manifest id is empty")
		}
	}
	for _, module := range []struct {
		name string
		fn   func() error
	}{
		{name: personalmodule.ModuleID, fn: func() error { return service.RegisterModule(context.Background(), personalmodule.Manifest()) }},
		{name: cronosmodule.ModuleID, fn: func() error { return service.RegisterModule(context.Background(), cronosmodule.Manifest()) }},
		{name: dietasmodule.ModuleID, fn: func() error { return service.RegisterModule(context.Background(), dietasmodule.Manifest()) }},
		{name: bolsamodule.ModuleID, fn: func() error { return service.RegisterModule(context.Background(), bolsamodule.Manifest()) }},
	} {
		if err := module.fn(); err != nil {
			t.Fatalf("RegisterModule(%s) error = %v", module.name, err)
		}
	}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	for _, path := range []string{"/api/vec/modules", "/api/vec/workspace", "/api/vec/menu"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d: %s", path, rec.Code, rec.Body.String())
		}
	}
	for _, path := range []string{
		"/api/vec/modules/cronos/action",
		"/api/vec/modules/horarios/action",
		"/api/vec/modules/permisos/action",
		"/api/vec/modules/dietas/action",
		"/api/vec/modules/rutas/action",
		"/api/vec/modules/personal/action",
		"/api/vec/modules/nominas/action",
		"/api/vec/modules/bolsa/action",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusAccepted {
			t.Fatalf("%s status = %d: %s", path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode action response: %v", err)
		}
		if body["data"] == nil {
			t.Fatalf("action response missing data: %#v", body)
		}
	}
}

func TestSessionBuildsPrincipalFromDNIeCertificate(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{
		{
			SerialNumber: big.NewInt(123456),
			Subject: pkix.Name{
				CommonName:   "Alberto Avidad",
				SerialNumber: "12345678Z",
			},
			Issuer: pkix.Name{CommonName: "DNIe Direccion General de la Policia"},
		},
	}}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/vec/session status = %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`"id":"12345678Z"`, `"display_name":"Alberto Avidad"`, `"auth_method":"dnie"`, `"auth_assurance":"alto"`, `"dni":"12345678Z"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("session body missing %s: %s", want, body)
		}
	}
	if strings.Contains(body, "vec.roles.manage") {
		t.Fatalf("certificate without configured role should not get administrator permissions: %s", body)
	}
}

func TestSessionUsesConfiguredRolesForExternalIdentity(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
	req.Header.Set("X-Auth-Subject", "rrhh-001")
	req.Header.Set("X-Auth-Mechanism", "dnie")
	req.Header.Set("X-Auth-Roles", "tecnico_rrhh")
	req.Header.Set("X-Auth-Assurance", "alto")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/vec/session status = %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`"id":"rrhh-001"`, `"auth_method":"dnie"`, `"tecnico_rrhh"`, personalmodule.PermissionEmployeeManage} {
		if !strings.Contains(body, want) {
			t.Fatalf("session body missing %s: %s", want, body)
		}
	}
}

func TestPersonalRPTCatalogEndpoints(t *testing.T) {
	handler := newTestHandler(t)
	for _, path := range []string{
		"/api/vec/personal/rpt/positions?q=administrativo",
		"/api/vec/personal/rpt/positions/8",
		"/api/vec/personal/rpt/stats",
		"/api/vec/personal/categories?q=trabajador",
		"/api/vec/personal/catalogs",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Auth-Subject", "rrhh-001")
		req.Header.Set("X-Auth-Mechanism", "dnie")
		req.Header.Set("X-Auth-Roles", "tecnico_rrhh")
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d: %s", path, rec.Code, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/vec/personal/rpt/positions/999", strings.NewReader(`{"name":"Puesto prueba","dot":1,"group":"A1","provision":"C","state":"Vigente"}`))
	req.Header.Set("X-Auth-Subject", "rrhh-001")
	req.Header.Set("X-Auth-Mechanism", "dnie")
	req.Header.Set("X-Auth-Roles", "tecnico_rrhh")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/vec/personal/rpt/positions/999 status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, `"code":"999"`) || !strings.Contains(body, "personal.rpt.position.upsert") {
		t.Fatalf("PUT body missing position/audit receipt: %s", body)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/vec/personal/rpt/positions/999", nil)
	req.Header.Set("X-Auth-Subject", "rrhh-001")
	req.Header.Set("X-Auth-Mechanism", "dnie")
	req.Header.Set("X-Auth-Roles", "tecnico_rrhh")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/vec/personal/rpt/positions/999 status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "personal.rpt.position.delete") {
		t.Fatalf("DELETE body missing audit receipt: %s", body)
	}
}

func TestPersonalCategoryCRUDEndpoints(t *testing.T) {
	handler := newTestHandler(t)
	headers := func(req *http.Request) {
		req.Header.Set("X-Auth-Subject", "rrhh-001")
		req.Header.Set("X-Auth-Mechanism", "dnie")
		req.Header.Set("X-Auth-Roles", "tecnico_rrhh")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vec/personal/categories", strings.NewReader(`{"slug":"nueva-categoria","name":"Nueva Categoria","area":"administracion_especial","source":"test","state":"vigente"}`))
	headers(req)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/vec/personal/categories status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/vec/personal/categories/nueva-categoria", strings.NewReader(`{"name":"Nueva Categoria Editada","area":"administracion_general","source":"test","state":"vigente"}`))
	headers(req)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Nueva Categoria Editada") {
		t.Fatalf("PUT category status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories/nueva-categoria", nil)
	headers(req)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "nueva-categoria") {
		t.Fatalf("GET category status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/vec/personal/categories/nueva-categoria", nil)
	headers(req)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "personal.category.delete") {
		t.Fatalf("DELETE category status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPersonalRPTCatalogRequiresStaffPermissions(t *testing.T) {
	handler := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/vec/personal/rpt/positions", nil)
	req.Header.Set("X-Auth-Subject", "candidate")
	req.Header.Set("X-Auth-Mechanism", "clave")
	req.Header.Set("X-Auth-Roles", "ciudadano")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("candidate GET RPT status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRegistersCronosTimecard(t *testing.T) {
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err := service.RegisterModule(context.Background(), cronosmodule.Manifest()); err != nil {
		t.Fatalf("RegisterModule(cronos) error = %v", err)
	}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	payload := []byte(`{
		"employee_id":"EMP-0999",
		"date":"2026-06-19",
		"age":64,
		"profile_id":"H-FLEX-ADM",
		"punches":[
			{"at":"08:00","kind":"entrada","channel":"web"},
			{"at":"13:30","kind":"salida","channel":"web"}
		]
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vec/cronos/timecards", bytes.NewReader(payload))
	req.Header.Set("X-Auth-Subject", "staff")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/vec/cronos/timecards status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "EMP-0999") || !strings.Contains(body, "05:30") {
		t.Fatalf("POST body missing persisted result: %s", body)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/cronos/timecards", nil)
	req.Header.Set("X-Auth-Subject", "staff")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/vec/cronos/timecards status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "EMP-0999") {
		t.Fatalf("GET body missing persisted workday: %s", body)
	}
}

func TestHandlerRequestsCronosLeave(t *testing.T) {
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err := service.RegisterModule(context.Background(), cronosmodule.Manifest()); err != nil {
		t.Fatalf("RegisterModule(cronos) error = %v", err)
	}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	payload := []byte(`{
		"employee_id":"EMP-0031",
		"policy_id":"asuntos_propios",
		"from":"2026-06-26",
		"to":"2026-06-26",
		"amount":1,
		"reason":"Asunto propio"
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vec/cronos/leave-requests", bytes.NewReader(payload))
	req.Header.Set("X-Auth-Subject", "staff")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/vec/cronos/leave-requests status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "cronos.permiso.request") || !strings.Contains(body, "asuntos_propios") {
		t.Fatalf("POST body missing request receipt: %s", body)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/cronos/leave-requests", nil)
	req.Header.Set("X-Auth-Subject", "staff")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/vec/cronos/leave-requests status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "Asunto propio") || !strings.Contains(body, `"remaining":"2"`) {
		t.Fatalf("GET body missing persisted leave or updated balance: %s", body)
	}
}

func TestWorkspaceSnapshotIncludesProfessionalDemoPayload(t *testing.T) {
	snapshot := workspaceSnapshot(nil)
	for _, key := range []string{
		"generated_at",
		"modules",
		"kpis",
		"operational_records",
		"screen_catalog",
		"payroll_run",
		"expense_policy",
		"action_catalog",
		"flow_states",
		"access_roles",
		"role_assignments",
		"rpt_catalog",
		"rpt_contract_types",
		"rpt_position_samples",
		"professional_categories",
		"professional_category_aliases",
		"bolsa_category_rules",
	} {
		if snapshot[key] == nil {
			t.Fatalf("workspace snapshot missing %q", key)
		}
	}

	screenCatalog, ok := snapshot["screen_catalog"].([]map[string]any)
	if !ok || len(screenCatalog) == 0 {
		t.Fatalf("screen_catalog = %#v, want non-empty []map[string]any", snapshot["screen_catalog"])
	}
	var hasPayrollScreen bool
	for _, screen := range screenCatalog {
		if screen["id"] == "nominas.cierre" {
			hasPayrollScreen = true
			fields, ok := screen["fields"].([]map[string]any)
			if !ok || len(fields) == 0 {
				t.Fatalf("nominas.cierre fields = %#v, want non-empty field catalog", screen["fields"])
			}
		}
	}
	if !hasPayrollScreen {
		t.Fatalf("screen_catalog missing nominas.cierre: %#v", screenCatalog)
	}

	payrollRun, ok := snapshot["payroll_run"].(map[string]any)
	if !ok || payrollRun["period"] != "2026-06" || payrollRun["state"] == "" {
		t.Fatalf("payroll_run = %#v, want demo payroll period and state", snapshot["payroll_run"])
	}
	expensePolicy, ok := snapshot["expense_policy"].(map[string]any)
	if !ok || expensePolicy["demo_notice"] == "" {
		t.Fatalf("expense_policy = %#v, want demo policy metadata", snapshot["expense_policy"])
	}

	actionCatalog, ok := snapshot["action_catalog"].(map[string][]map[string]any)
	if !ok {
		t.Fatalf("action_catalog = %#v, want map[string][]map[string]any", snapshot["action_catalog"])
	}
	for _, module := range []string{"personal", "nominas", "cronos", "horarios", "permisos", "dietas", "rutas", "bolsa"} {
		if len(actionCatalog[module]) == 0 {
			t.Fatalf("action_catalog missing actions for %s: %#v", module, actionCatalog)
		}
	}

	flowStates, ok := snapshot["flow_states"].([]map[string]any)
	if !ok || len(flowStates) == 0 {
		t.Fatalf("flow_states = %#v, want non-empty []map[string]any", snapshot["flow_states"])
	}
	var hasPayrollHandoff bool
	for _, state := range flowStates {
		if state["id"] == "lista_para_nomina" {
			hasPayrollHandoff = true
		}
	}
	if !hasPayrollHandoff {
		t.Fatalf("flow_states missing lista_para_nomina: %#v", flowStates)
	}

	accessRoles, ok := snapshot["access_roles"].([]map[string]any)
	if !ok || len(accessRoles) < 5 {
		t.Fatalf("access_roles = %#v, want role matrix for VEC", snapshot["access_roles"])
	}
	var hasAdmin, hasRRHH, hasJefeServicio bool
	for _, role := range accessRoles {
		switch role["id"] {
		case "administrador":
			hasAdmin = true
		case "tecnico_rrhh":
			hasRRHH = true
		case "jefe_servicio":
			hasJefeServicio = true
		}
	}
	if !hasAdmin || !hasRRHH || !hasJefeServicio {
		t.Fatalf("access_roles missing expected public-sector roles: %#v", accessRoles)
	}

	rptTypes, ok := snapshot["rpt_contract_types"].([]map[string]any)
	if !ok || len(rptTypes) == 0 {
		t.Fatalf("rpt_contract_types = %#v, want RPT/personnel catalog", snapshot["rpt_contract_types"])
	}
	var hasFuncionario, hasProvision bool
	for _, item := range rptTypes {
		if item["code"] == "funcionario_carrera" {
			hasFuncionario = true
		}
		if item["catalog"] == "forma_provision" && item["code"] == "L" {
			hasProvision = true
		}
	}
	if !hasFuncionario || !hasProvision {
		t.Fatalf("rpt_contract_types missing regime/provision entries: %#v", rptTypes)
	}

	categories, ok := snapshot["professional_categories"].([]map[string]any)
	if !ok || len(categories) < 50 {
		t.Fatalf("professional_categories = %#v, want Diputacion category master from Bolsa/OPES", snapshot["professional_categories"])
	}
	var hasAdministrativo, hasTrabajadorSocial bool
	for _, category := range categories {
		switch category["slug"] {
		case "administrativo":
			hasAdministrativo = true
		case "trabajador-social":
			hasTrabajadorSocial = true
		}
	}
	if !hasAdministrativo || !hasTrabajadorSocial {
		t.Fatalf("professional_categories missing expected categories: %#v", categories)
	}
}
