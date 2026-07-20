package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	"vec-diputacion-granada/internal/vec/application"
)

func TestPersonalCategoriesUsaCatalogoGobernadoCompletoYNoExponeProcedenciaInterna(t *testing.T) {
	handler := newTestHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories?limit=500", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET categorias status = %d: %s", rec.Code, rec.Body.String())
	}
	cuerpo := rec.Body.String()
	for _, esperado := range []string{`"total":68`, `"catalogo_id":"categorias-profesionales"`, `"demostracion":true`, `"slug":"administrativo"`, `"name":"Técnico de Administración General"`} {
		if !strings.Contains(cuerpo, esperado) {
			t.Fatalf("catalogo gobernado no contiene %s: %s", esperado, cuerpo)
		}
	}
	for _, prohibido := range []string{"source_path", "creado_por", "publicado_por", "aprobacion_ref", "motivo_publicacion"} {
		if strings.Contains(cuerpo, prohibido) {
			t.Fatalf("catalogo expone %q: %s", prohibido, cuerpo)
		}
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories?q=tecnico&area=administracion_general&limit=1&offset=1", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"total":2`) || !strings.Contains(rec.Body.String(), `"offset":1`) {
		t.Fatalf("filtro/paginacion categorias status = %d: %s", rec.Code, rec.Body.String())
	}
	for area, total := range map[string]int{"administracion_general": 5, "administracion_especial": 60, "organismos_dependientes": 3} {
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories?area="+area+"&limit=500", nil)
		servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), fmt.Sprintf(`"total":%d`, total)) {
			t.Fatalf("area %s status = %d: %s", area, rec.Code, rec.Body.String())
		}
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/personal/rpt/stats", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"categories":68`) ||
		!strings.Contains(rec.Body.String(), `"administracion_general":5`) ||
		!strings.Contains(rec.Body.String(), `"administracion_especial":60`) ||
		!strings.Contains(rec.Body.String(), `"organismos_dependientes":3`) {
		t.Fatalf("estadisticas no proceden del catalogo gobernado: %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories/medico", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"Médico"`) {
		t.Fatalf("GET detalle categoria status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPersonalCategoriesRechazaMutacionDirectaAntesDeLeerCuerpoYSinAuditoriaAceptada(t *testing.T) {
	handler := newTestHandler(t)
	casos := []struct {
		metodo string
		ruta   string
		cuerpo string
	}{
		{metodo: http.MethodPost, ruta: "/api/vec/personal/categories", cuerpo: `{json-invalido`},
		{metodo: http.MethodPut, ruta: "/api/vec/personal/categories/administrativo", cuerpo: `{json-invalido`},
		{metodo: http.MethodDelete, ruta: "/api/vec/personal/categories/administrativo"},
	}
	for _, caso := range casos {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(caso.metodo, caso.ruta, strings.NewReader(caso.cuerpo))
		servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionManage)
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "catalogo_gobernado_requiere_borrador") {
			t.Fatalf("%s %s = %d: %s", caso.metodo, caso.ruta, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s Cache-Control = %q", caso.metodo, caso.ruta, rec.Header().Get("Cache-Control"))
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/vec/personal/categories/administrativo", strings.NewReader(`{}`))
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("sin permiso debe responder 403 antes que 409: %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories/administrativo", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"Administrativo"`) {
		t.Fatalf("la categoria publicada fue alterada: %d: %s", rec.Code, rec.Body.String())
	}
	consultaAuditoria, err := application.NewAuditQuery(
		principalConPermisosExpresosPrueba(adminmodule.PermissionAuditRead), "administrativo",
	)
	if err != nil {
		t.Fatal(err)
	}
	auditoria, err := handler.internal.Audit(context.Background(), consultaAuditoria)
	if err != nil || len(auditoria) != 0 {
		t.Fatalf("las mutaciones rechazadas generaron auditoria aceptada: %#v, %v", auditoria, err)
	}
}

func TestPersonalCategoriesFallaCerradoSinAutoridadYRechazaFiltrosAmbiguos(t *testing.T) {
	handler := newTestHandler(t)
	for _, ruta := range []string{
		"/api/vec/personal/categories?q=a&q=b",
		"/api/vec/personal/categories?desconocido=1",
		"/api/vec/personal/categories?limit=no-numero",
		"/api/vec/personal/categories?area=_invalida",
	} {
		rec := httptest.NewRecorder()
		servirCatalogoPersonalConPermisosExpresos(
			handler, rec, httptest.NewRequest(http.MethodGet, ruta, nil), personalmodule.PermissionPositionRead,
		)
		if rec.Code != http.StatusBadRequest || rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("filtro %s = %d, cache=%q: %s", ruta, rec.Code, rec.Header().Get("Cache-Control"), rec.Body.String())
		}
	}

	handler.categoriasProfesionales = nil
	for _, ruta := range []string{"/api/vec/personal/categories", "/api/vec/personal/rpt/stats"} {
		rec := httptest.NewRecorder()
		servirCatalogoPersonalConPermisosExpresos(
			handler, rec, httptest.NewRequest(http.MethodGet, ruta, nil), personalmodule.PermissionPositionRead,
		)
		if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Cache-Control") != "no-store" ||
			!strings.Contains(rec.Body.String(), "catalogo_categorias_profesionales_no_disponible") {
			t.Fatalf("sin autoridad %s = %d, cache=%q: %s", ruta, rec.Code, rec.Header().Get("Cache-Control"), rec.Body.String())
		}
	}
}
