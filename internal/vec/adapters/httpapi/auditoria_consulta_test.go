package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

func servirAuditoriaConPermisoExpreso(handler *Handler, rec *httptest.ResponseRecorder, req *http.Request) {
	handler.handleAudit(rec, req, principalConPermisosExpresosPrueba(adminmodule.PermissionAuditRead))
}

func TestAuditoriaHTTPExigeUnaReferenciaCanonicaExacta(t *testing.T) {
	handler := newTestHandler(t)
	referenciaDemasiadoLarga := strings.Repeat("a", maximoBytesReferenciaAuditoria+1)
	casos := []struct {
		nombre string
		ruta   string
	}{
		{nombre: "sin consulta", ruta: "/api/vec/audit"},
		{nombre: "sin referencia", ruta: "/api/vec/audit?limit=1"},
		{nombre: "referencia vacia", ruta: "/api/vec/audit?subject_ref="},
		{nombre: "solo espacio", ruta: "/api/vec/audit?subject_ref=+"},
		{nombre: "espacio inicial", ruta: "/api/vec/audit?subject_ref=%20expediente%3Auno"},
		{nombre: "espacio final", ruta: "/api/vec/audit?subject_ref=expediente%3Auno%20"},
		{nombre: "control", ruta: "/api/vec/audit?subject_ref=expediente%0Auno"},
		{nombre: "comodin", ruta: "/api/vec/audit?subject_ref=%2A"},
		{nombre: "comodin parcial", ruta: "/api/vec/audit?subject_ref=expediente%3A%2A"},
		{nombre: "referencia repetida distinta", ruta: "/api/vec/audit?subject_ref=expediente%3Auno&subject_ref=expediente%3Ados"},
		{nombre: "referencia repetida igual", ruta: "/api/vec/audit?subject_ref=expediente%3Auno&subject_ref=expediente%3Auno"},
		{nombre: "parametro desconocido", ruta: "/api/vec/audit?subject_ref=expediente%3Auno&limit=1"},
		{nombre: "consulta mal formada", ruta: "/api/vec/audit?subject_ref=expediente%3Auno;limit=1"},
		{nombre: "referencia demasiado larga", ruta: "/api/vec/audit?subject_ref=" + referenciaDemasiadoLarga},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rec := httptest.NewRecorder()
			servirAuditoriaConPermisoExpreso(handler, rec, httptest.NewRequest(http.MethodGet, caso.ruta, nil))
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusForbidden, rec.Body.String())
			}
			if rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
			}
			if !strings.Contains(rec.Body.String(), domain.ErrPermissionDenied.Error()) {
				t.Fatalf("respuesta no aplica denegacion uniforme: %s", rec.Body.String())
			}
		})
	}
}

func TestAuditoriaHTTPConsultaSoloLaReferenciaExacta(t *testing.T) {
	store := memory.NewStore()
	service, internal, err := application.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	actor := domain.Principal{
		ID: "actor-auditoria", AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
		Permissions: []string{"prueba.audit.write"},
	}
	for _, referencia := range []string{"expediente:uno", "expediente:dos"} {
		authorized, err := application.NewAuthorizedAuditCommand(application.AuditCommand{
			Principal: actor, Action: "consulta.prueba", ModuleID: "vec.module.prueba",
			SubjectRef: referencia, Result: "ok",
		}, "prueba.audit.write", "")
		if err != nil {
			t.Fatalf("NewAuthorizedAuditCommand(%q) error = %v", referencia, err)
		}
		if _, err := internal.RecordAudit(context.Background(), authorized); err != nil {
			t.Fatalf("RecordAudit(%q) error = %v", referencia, err)
		}
	}
	handler, err := NewHandlerWithOptions(service, HandlerOptions{
		InternalOperations: internal,
		AllowDemoIdentity:  true, DemoIdentityResolver: resolvedorIdentidadPruebas{},
	})
	if err != nil {
		t.Fatalf("NewHandlerWithOptions() error = %v", err)
	}

	rec := httptest.NewRecorder()
	servirAuditoriaConPermisoExpreso(handler, rec, httptest.NewRequest(http.MethodGet, "/api/vec/audit?subject_ref=expediente%3Auno", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	cuerpo := rec.Body.String()
	if !strings.Contains(cuerpo, `"subject_ref":"expediente:uno"`) || strings.Contains(cuerpo, "expediente:dos") {
		t.Fatalf("la consulta no quedo limitada a la referencia exacta: %s", cuerpo)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
}

type almacenAuditoriaDenegada struct {
	*memory.Store
}

func (a *almacenAuditoriaDenegada) ListAudit(context.Context, string) ([]domain.AuditEntry, error) {
	return nil, domain.ErrPermissionDenied
}

func TestAuditoriaHTTPMapeaDenegacionDelCasoDeUsoAForbidden(t *testing.T) {
	store := &almacenAuditoriaDenegada{Store: memory.NewStore()}
	service, internal, err := application.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler, err := NewHandlerWithOptions(service, HandlerOptions{
		InternalOperations: internal,
		AllowDemoIdentity:  true, DemoIdentityResolver: resolvedorIdentidadPruebas{},
	})
	if err != nil {
		t.Fatalf("NewHandlerWithOptions() error = %v", err)
	}

	rec := httptest.NewRecorder()
	servirAuditoriaConPermisoExpreso(handler, rec, httptest.NewRequest(http.MethodGet, "/api/vec/audit?subject_ref=expediente%3Auno", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.ErrPermissionDenied.Error()) {
		t.Fatalf("respuesta = %s, want denegacion uniforme", rec.Body.String())
	}
}

func TestAuditoriaHTTPSoloConServicePermaneceCerrada(t *testing.T) {
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler, err := NewHandlerWithOptions(service, HandlerOptions{
		AllowDemoIdentity: true, DemoIdentityResolver: resolvedorIdentidadPruebas{},
	})
	if err != nil {
		t.Fatalf("NewHandlerWithOptions() error = %v", err)
	}
	rec := httptest.NewRecorder()
	servirAuditoriaConPermisoExpreso(handler, rec, httptest.NewRequest(http.MethodGet, "/api/vec/audit?subject_ref=expediente%3Auno", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestHandlerRechazaOperacionesInternasDeOtraComposicion(t *testing.T) {
	storeA, storeB := memory.NewStore(), memory.NewStore()
	serviceA, _, err := application.NewServiceWithInternalOperations(storeA, storeA, storeA)
	if err != nil {
		t.Fatalf("composicion A error = %v", err)
	}
	_, internalB, err := application.NewServiceWithInternalOperations(storeB, storeB, storeB)
	if err != nil {
		t.Fatalf("composicion B error = %v", err)
	}
	if handler, err := NewHandlerWithOptions(serviceA, HandlerOptions{InternalOperations: internalB}); err == nil || handler != nil {
		t.Fatalf("NewHandlerWithOptions() = (%#v, %v), debe rechazar mezcla", handler, err)
	}
}
