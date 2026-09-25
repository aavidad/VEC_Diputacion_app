package bootstrap

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
)

const (
	rutaRPTRepositorioPrueba        = "../../../data/catalogos/rpt/v1.rpt-2026.json"
	rutaEstructuraRepositorioPrueba = "../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json"
)

// raizRegistradora sustituye a la carcasa: cuenta qué peticiones le llegan.
type raizRegistradora struct{ rutas []string }

func (r *raizRegistradora) ServeHTTP(w http.ResponseWriter, peticion *http.Request) {
	r.rutas = append(r.rutas, peticion.Method+" "+peticion.URL.Path)
	w.WriteHeader(http.StatusTeapot)
}

func componerPersonalPublicoPrueba(t *testing.T, cfg config.Config) (http.Handler, *raizRegistradora, *bytes.Buffer) {
	t.Helper()
	cfg = cfg.Normalize()
	_, categorias, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		t.Fatalf("categorias: %v", err)
	}
	raiz := &raizRegistradora{}
	var registro bytes.Buffer
	return componerRaizConPersonalPublico(raiz, nuevasConsultasPublicasPersonal(cfg, categorias, &registro)), raiz, &registro
}

func pedirPersonalPublico(t *testing.T, h http.Handler, metodo, ruta string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, httptest.NewRequest(metodo, ruta, nil))
	var cuerpo map[string]any
	if respuesta.Body.Len() > 0 && strings.HasPrefix(respuesta.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil {
			t.Fatalf("%s %s: JSON no valido: %v", metodo, ruta, err)
		}
	}
	return respuesta, cuerpo
}

func TestPersonalPublicoSirveFuentesVersionadasDelRepositorio(t *testing.T) {
	h, raiz, registro := componerPersonalPublicoPrueba(t, config.Config{
		RPTCatalogoPath:                rutaRPTRepositorioPrueba,
		PersonalOrganizacionSourcePath: rutaEstructuraRepositorioPrueba,
	})
	if registro.Len() != 0 {
		t.Fatalf("registro inesperado: %q", registro.String())
	}
	respuesta, cuerpo := pedirPersonalPublico(t, h, http.MethodGet, "/api/vec/personal/categories?q=&area=&limit=25&offset=0")
	categorias, _ := cuerpo["data"].(map[string]any)["categories"].(map[string]any)
	if respuesta.Code != http.StatusOK || categorias == nil || categorias["total"].(float64) < 1 {
		t.Fatalf("categorias=%d %s", respuesta.Code, respuesta.Body.String())
	}
	respuesta, cuerpo = pedirPersonalPublico(t, h, http.MethodGet, "/api/vec/personal/rpt-publica?q=&limit=25&offset=0&vista=puestos")
	rpt, _ := cuerpo["data"].(map[string]any)["rpt"].(map[string]any)
	if respuesta.Code != http.StatusOK || rpt == nil || rpt["total"].(float64) < 1 || rpt["vista"] != "puestos" {
		t.Fatalf("rpt=%d %s", respuesta.Code, respuesta.Body.String())
	}
	respuesta, cuerpo = pedirPersonalPublico(t, h, http.MethodGet, "/api/vec/personal/estructura-organizativa-publica")
	if respuesta.Code != http.StatusOK || cuerpo["data"].(map[string]any)["estructura_organizativa"] == nil {
		t.Fatalf("estructura=%d %s", respuesta.Code, respuesta.Body.String())
	}
	for _, cabecera := range []string{"Set-Cookie", "Access-Control-Allow-Origin"} {
		if respuesta.Header().Get(cabecera) != "" {
			t.Fatalf("cabecera %s emitida", cabecera)
		}
	}
	if respuesta.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache=%q", respuesta.Header().Get("Cache-Control"))
	}
	if len(raiz.rutas) != 0 {
		t.Fatalf("la carcasa atendio lecturas publicas: %v", raiz.rutas)
	}
}

func TestPersonalPublicoSoloLecturaYRestoALaCarcasa(t *testing.T) {
	h, raiz, _ := componerPersonalPublicoPrueba(t, config.Config{
		RPTCatalogoPath:                rutaRPTRepositorioPrueba,
		PersonalOrganizacionSourcePath: rutaEstructuraRepositorioPrueba,
	})
	for _, caso := range []struct {
		metodo, ruta string
		estado       int
	}{
		{http.MethodPost, "/api/vec/personal/rpt-publica?q=&limit=1&offset=0", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/api/vec/personal/estructura-organizativa-publica", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/vec/personal/estructura-organizativa-publica?todo=1", http.StatusNotFound},
		{http.MethodGet, "/api/vec/personal/rpt-publica?limit=0&offset=0", http.StatusBadRequest},
		{http.MethodGet, "/api/vec/personal/categories?limit=abc", http.StatusBadRequest},
	} {
		if respuesta, _ := pedirPersonalPublico(t, h, caso.metodo, caso.ruta); respuesta.Code != caso.estado {
			t.Fatalf("%s %s=%d, quiere %d", caso.metodo, caso.ruta, respuesta.Code, caso.estado)
		}
	}
	if len(raiz.rutas) != 0 {
		t.Fatalf("la carcasa atendio rutas publicas: %v", raiz.rutas)
	}
	// La gestión del catálogo y cualquier otra ruta siguen en la carcasa, con
	// sus permisos; la lectura pública no abre escritura.
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodPost, "/api/vec/personal/categories"},
		{http.MethodGet, "/api/vec/personal/categories/auxiliar"},
		{http.MethodGet, "/api/vec/personal/empleados-organismo"},
		{http.MethodGet, "/api/vec/modules"},
	} {
		if respuesta, _ := pedirPersonalPublico(t, h, caso.metodo, caso.ruta); respuesta.Code != http.StatusTeapot {
			t.Fatalf("%s %s no llego a la carcasa: %d", caso.metodo, caso.ruta, respuesta.Code)
		}
	}
	if len(raiz.rutas) != 4 {
		t.Fatalf("carcasa=%v", raiz.rutas)
	}
}

func TestPersonalPublicoSinFuenteFallaCerradoSinCaerEnLaCarcasa(t *testing.T) {
	directorio := t.TempDir()
	alterada := filepath.Join(directorio, "rpt-alterada.json")
	if err := os.WriteFile(alterada, []byte(`{"esquema":"vec.catalogo.rpt.v1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		cfg    config.Config
		motivo string
	}{
		{"sin_configurar", config.Config{}, "fuente no configurada"},
		{"huella_distinta", config.Config{RPTCatalogoPath: alterada, PersonalOrganizacionSourcePath: alterada}, "fuente no valida"},
		{"ruta_inexistente", config.Config{RPTCatalogoPath: filepath.Join(directorio, "no"), PersonalOrganizacionSourcePath: filepath.Join(directorio, "no")}, "fuente no valida"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			h, raiz, registro := componerPersonalPublicoPrueba(t, caso.cfg)
			for ruta, codigo := range map[string]string{
				"/api/vec/personal/rpt-publica?q=&limit=1&offset=0": "rpt_publica_no_disponible",
				"/api/vec/personal/estructura-organizativa-publica": "estructura_organizativa_publica_no_disponible",
			} {
				respuesta, cuerpo := pedirPersonalPublico(t, h, http.MethodGet, ruta)
				if respuesta.Code != http.StatusServiceUnavailable || cuerpo["error"] != codigo ||
					respuesta.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("%s=%d %s", ruta, respuesta.Code, respuesta.Body.String())
				}
			}
			if len(raiz.rutas) != 0 {
				t.Fatalf("una ruta sin fuente cayo en la carcasa: %v", raiz.rutas)
			}
			texto := registro.String()
			if strings.Count(texto, caso.motivo) != 2 || strings.Contains(texto, directorio) {
				t.Fatalf("registro=%q", texto)
			}
			// El catálogo profesional es obligatorio en la raíz y sigue servido.
			if respuesta, _ := pedirPersonalPublico(t, h, http.MethodGet, "/api/vec/personal/categories?q=&area=&limit=1&offset=0"); respuesta.Code != http.StatusOK {
				t.Fatalf("categorias=%d", respuesta.Code)
			}
		})
	}
}

// La superficie integrada que monta vec-server sirve las consultas y los
// módulos web que las pintan, con sus barreras comunes delante.
func TestPersonalPublicoAtraviesaLaSuperficieIntegrada(t *testing.T) {
	cfg := config.Config{
		RPTCatalogoPath:                rutaRPTRepositorioPrueba,
		PersonalOrganizacionSourcePath: rutaEstructuraRepositorioPrueba,
	}.Normalize()
	_, categorias, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := server.NewHandlerWithConfig(cfg, componerRaizConPersonalPublico(http.NotFoundHandler(), nuevasConsultasPublicasPersonal(cfg, categorias, nil)))
	pedir := func(ruta string, cabeceras map[string]string) *httptest.ResponseRecorder {
		peticion := httptest.NewRequest(http.MethodGet, ruta, nil)
		peticion.RemoteAddr = "127.0.0.1:50000"
		for clave, valor := range cabeceras {
			peticion.Header.Set(clave, valor)
		}
		respuesta := httptest.NewRecorder()
		h.ServeHTTP(respuesta, peticion)
		return respuesta
	}
	for _, ruta := range []string{
		"/api/vec/personal/categories?q=&area=&limit=1&offset=0",
		"/api/vec/personal/rpt-publica?q=&limit=1&offset=0",
		"/api/vec/personal/estructura-organizativa-publica",
		"/portal-empleado/modulos/personal/vista-rpt-publica.js?v=20260925-personal-real-v1",
		"/portal-empleado/modulos/personal/vista-estructura-organizativa-publica.js?v=20260925-personal-real-v1",
		"/portal-empleado/modulos/personal/cliente-http-rpt-publica.js",
		"/portal-empleado/modulos/personal/cliente-http-estructura-organizativa-publica.js",
	} {
		if respuesta := pedir(ruta, nil); respuesta.Code != http.StatusOK || respuesta.Header().Get("Set-Cookie") != "" {
			t.Fatalf("%s=%d", ruta, respuesta.Code)
		}
	}
	// Las barreras comunes siguen delante: sin cookies ni Authorization.
	if respuesta := pedir("/api/vec/personal/rpt-publica?q=&limit=1&offset=0", map[string]string{"Cookie": "sesion=1"}); respuesta.Code == http.StatusOK {
		t.Fatal("la consulta publica acepto una cookie")
	}
}

func TestPersonalPublicoSinCatalogoProfesionalResponde503(t *testing.T) {
	raiz := &raizRegistradora{}
	var registro bytes.Buffer
	h := componerRaizConPersonalPublico(raiz, nuevasConsultasPublicasPersonal(config.Config{}.Normalize(), nil, &registro))
	respuesta, cuerpo := pedirPersonalPublico(t, h, http.MethodGet, "/api/vec/personal/categories?q=&area=&limit=1&offset=0")
	if respuesta.Code != http.StatusServiceUnavailable || cuerpo["error"] != "catalogo_categorias_profesionales_no_disponible" || len(raiz.rutas) != 0 {
		t.Fatalf("categorias=%d %s carcasa=%v", respuesta.Code, respuesta.Body.String(), raiz.rutas)
	}
	if !strings.Contains(registro.String(), "categorias") {
		t.Fatalf("registro=%q", registro.String())
	}
}
