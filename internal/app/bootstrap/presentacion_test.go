package bootstrap

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
)

func configuracionPresentacionBootstrap(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		Address:                   "127.0.0.1:0",
		AuthMode:                  config.AuthModeDisabled,
		StorageMode:               config.StorageModeMemory,
		ExecutionProfile:          config.ExecutionProfileRRHHPresentation,
		RRHHPresentationEnabled:   true,
		RRHHPresentationGuardOne:  config.RRHHPresentationGuardOneAcknowledgement,
		RRHHPresentationGuardTwo:  config.RRHHPresentationGuardTwoAcknowledgement,
		HTTPAllowedCIDRs:          []string{"127.0.0.1/32", "::1/128"},
		PersonalCatalogPath:       "memory",
		BolsaPublicSourcePath:     "../../../data/demo/convocatorias_publicas.demo.json",
		BolsaCategoriesSourcePath: "../../../data/catalogos/categorias-profesionales/v1.demo.json",
		BolsaCategoriesCatalogID:  config.DefaultBolsaCategoriesCatalogID,
		BolsaCategoriesVersion:    config.DefaultBolsaCategoriesVersion,
		BolsaCategoriesSHA256:     config.DefaultBolsaCategoriesSHA256,
	}
}

func TestComposicionesNormalesRechazanCualquierSelectorPresentacion(t *testing.T) {
	selectores := []config.Config{
		{ExecutionProfile: config.ExecutionProfileRRHHPresentation},
		{RRHHPresentationEnabled: true},
		{RRHHPresentationGuardOne: config.RRHHPresentationGuardOneAcknowledgement},
		{RRHHPresentationGuardTwo: config.RRHHPresentationGuardTwoAcknowledgement},
	}
	for _, cfg := range selectores {
		for _, constructor := range []func(config.Config) (*http.Server, error){NewHTTPServerWithConfig, NewHTTPServerPublicoWithConfig} {
			if _, err := constructor(cfg); !errors.Is(err, ErrPresentacionRRHHEnComposicionNormal) {
				t.Fatalf("selector no rechazado: %v", err)
			}
		}
	}
}

func TestComposicionPresentacionSirveSoloConsultaPublicaYDatosSinteticos(t *testing.T) {
	servidor, err := NewHTTPServerPresentacionWithConfig(configuracionPresentacionBootstrap(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{"/api/publico/bolsa/convocatorias", "/api/publico/bolsa/categorias"} {
		peticion := httptest.NewRequest(http.MethodGet, ruta, nil)
		peticion.RemoteAddr = "127.0.0.1:50100"
		respuesta := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", ruta, respuesta.Code, respuesta.Body.String())
		}
	}
	for _, ruta := range []string{"/api/vec/session", "/api/demo", "/candidates"} {
		peticion := httptest.NewRequest(http.MethodGet, ruta, nil)
		peticion.RemoteAddr = "127.0.0.1:50100"
		respuesta := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d; se esperaba 404", ruta, respuesta.Code)
		}
	}
}

func TestComposicionPresentacionPersonalConcedeSoloCategoriasSinteticas(t *testing.T) {
	cfg := configuracionPresentacionBootstrap(t)
	servidor, err := NewHTTPServerPresentacionPersonalWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	peticion := func(metodo, ruta string) *http.Request {
		r := httptest.NewRequest(metodo, ruta, nil)
		r.RemoteAddr = "127.0.0.1:50100"
		return r
	}

	respuesta := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(respuesta, peticion(http.MethodGet, "/api/vec/personal/categories?q=administrativo&area=administracion_general&limit=1&offset=1"))
	if respuesta.Code != http.StatusOK || respuesta.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("GET categorías = %d Cache-Control=%q %s", respuesta.Code, respuesta.Header().Get("Cache-Control"), respuesta.Body.String())
	}
	var sobre struct {
		Data struct {
			Categories struct {
				Items []struct {
					Clave string `json:"clave"`
				} `json:"items"`
				Total  int `json:"total"`
				Limit  int `json:"limit"`
				Offset int `json:"offset"`
				Fuente struct {
					Demostracion bool `json:"demostracion"`
				} `json:"fuente"`
			} `json:"categories"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respuesta.Body).Decode(&sobre); err != nil || !sobre.Data.Categories.Fuente.Demostracion ||
		sobre.Data.Categories.Limit != 1 || sobre.Data.Categories.Offset != 1 ||
		sobre.Data.Categories.Total != 2 || len(sobre.Data.Categories.Items) != 1 ||
		sobre.Data.Categories.Items[0].Clave != "auxiliar-administrativo" {
		t.Fatalf("contrato de categorias no valido: %#v error=%v", sobre, err)
	}

	for _, ruta := range []string{
		"/api/vec", "/api/vec/session", "/api/vec/personal/categories/administrativo",
		"/api/vec/personal/rpt/positions", "/api/vec/personal/rpt/stats", "/api/vec/personal/catalogs",
	} {
		rec := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(rec, peticion(http.MethodGet, ruta))
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d; se esperaba 404", ruta, rec.Code)
		}
	}
	for _, cabecera := range []string{"Cookie", "Authorization", "Proxy-Authorization", "X-VEC-Subject", "X-Forwarded-For"} {
		rec := httptest.NewRecorder()
		r := peticion(http.MethodGet, "/api/vec/personal/categories")
		r.Header.Set(cabecera, "valor")
		servidor.Handler.ServeHTTP(rec, r)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET categorías con %s = %d; se esperaba 400", cabecera, rec.Code)
		}
	}
	recPOST := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(recPOST, peticion(http.MethodPost, "/api/vec/personal/categories"))
	if recPOST.Code != http.StatusMethodNotAllowed || recPOST.Header().Get("Allow") != "GET, HEAD" {
		t.Errorf("POST categorías = %d Allow=%q", recPOST.Code, recPOST.Header().Get("Allow"))
	}
	recHEAD := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(recHEAD, peticion(http.MethodHead, "/api/vec/personal/categories?area=administracion_general&limit=1&offset=0"))
	if recHEAD.Code != http.StatusOK || recHEAD.Body.Len() != 0 {
		t.Errorf("HEAD categorías = %d cuerpo=%d", recHEAD.Code, recHEAD.Body.Len())
	}

	ordinario, err := NewHTTPServerPresentacionWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	recOrdinario := httptest.NewRecorder()
	ordinario.Handler.ServeHTTP(recOrdinario, peticion(http.MethodGet, "/api/vec/personal/categories"))
	if recOrdinario.Code != http.StatusNotFound {
		t.Errorf("constructor ordinario publicó categorías: %d", recOrdinario.Code)
	}
}

func TestComposicionPresentacionPersonalRPTExigeFuenteYSoloConcedeRutaExacta(t *testing.T) {
	cfg := configuracionPresentacionBootstrap(t)
	if _, err := NewHTTPServerPresentacionPersonalRPTWithConfig(cfg); !errors.Is(err, ErrComposicionPresentacionRRHHInvalida) {
		t.Fatalf("arranque sin fuente RPT = %v", err)
	}
	cfg.RPTCatalogoPath = "../../../data/catalogos/rpt/v1.rpt-2026.json"
	cfg.PersonalOrganizacionSourcePath = "../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json"
	servidor, err := NewHTTPServerPresentacionPersonalRPTWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	peticion := func(metodo, ruta string) *http.Request {
		r := httptest.NewRequest(metodo, ruta, nil)
		r.RemoteAddr = "127.0.0.1:50100"
		return r
	}
	for _, caso := range []struct {
		metodo, ruta string
		want         int
	}{{http.MethodGet, "/api/vec/personal/rpt-publica?q=administrativo&limit=1&offset=0", http.StatusOK}, {http.MethodHead, "/api/vec/personal/rpt-publica?q=&limit=1&offset=0", http.StatusOK}, {http.MethodGet, "/api/vec/personal/estructura-organizativa-publica", http.StatusOK}, {http.MethodHead, "/api/vec/personal/estructura-organizativa-publica", http.StatusOK}, {http.MethodGet, "/api/vec/personal/estructura-organizativa-publica/extra", http.StatusNotFound}, {http.MethodGet, "/api/vec/personal/rpt-publica/administrativo", http.StatusNotFound}, {http.MethodPost, "/api/vec/personal/rpt-publica", http.StatusMethodNotAllowed}, {http.MethodGet, "/api/vec/session", http.StatusNotFound}} {
		rec := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(rec, peticion(caso.metodo, caso.ruta))
		if rec.Code != caso.want {
			t.Errorf("%s %s = %d: %s", caso.metodo, caso.ruta, rec.Code, rec.Body.String())
		}
	}
}

func TestCargaAmbientalPresentacionComponeTrasNormalizacionesRepetidas(t *testing.T) {
	t.Setenv(config.EnvAddress, "127.0.0.1:0")
	t.Setenv(config.EnvHTTPAllowedCIDRs, "127.0.0.1/32,::1/128")
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileRRHHPresentation)
	t.Setenv(config.EnvRRHHPresentationEnabled, "true")
	t.Setenv(config.EnvRRHHPresentationGuardOne, config.RRHHPresentationGuardOneAcknowledgement)
	t.Setenv(config.EnvRRHHPresentationGuardTwo, config.RRHHPresentationGuardTwoAcknowledgement)
	t.Setenv(config.EnvAuthMode, config.AuthModeDisabled)
	t.Setenv(config.EnvStorageMode, config.StorageModeMemory)
	t.Setenv(config.EnvPersonalCatalogPath, "memory")
	t.Setenv(config.EnvBolsaPublicSourcePath, "../../../data/demo/convocatorias_publicas.demo.json")
	t.Setenv(config.EnvBolsaCategoriesSourcePath, "../../../data/catalogos/categorias-profesionales/v1.demo.json")
	servidor, err := NewHTTPServerPresentacionWithConfig(config.Load())
	if err != nil {
		t.Fatalf("Load -> bootstrap: %v", err)
	}
	if servidor.Addr != "127.0.0.1:0" {
		t.Fatalf("listener = %q", servidor.Addr)
	}
}

func TestComposicionPresentacionRechazaDatosNoMarcadosYConectores(t *testing.T) {
	base := configuracionPresentacionBootstrap(t)
	mutaciones := []func(*config.Config){
		func(c *config.Config) { c.BolsaPublicSourcePath = "/datos/convocatorias.json" },
		func(c *config.Config) { c.BolsaCategoriesSourcePath = "/datos/categorias.json" },
		func(c *config.Config) { c.PersonalCatalogPath = "/datos/personal.json" },
		func(c *config.Config) { c.OSRMBaseURL = "http://127.0.0.1:5000" },
		func(c *config.Config) { c.OSRMGraphVersion = "grafo-osm-granada-v1" },
		func(c *config.Config) { c.AuthMode = config.AuthModeFake },
		func(c *config.Config) { c.StorageMode = config.StorageModeFile },
		func(c *config.Config) { c.FirmaVerificacionEnabled = "true" },
		func(c *config.Config) { c.FirmaVerificacionEnabled = " true " },
		func(c *config.Config) { c.FirmaVerificacionEnabled = "si" },
		// La presentación no tiene la doble llave de desarrollo: cualquier
		// catálogo de ejemplo declarado impide arrancar.
		func(c *config.Config) { c.ReglasEjemplo.BolsaSourcePath = "bolsa_reglas.ejemplo.demo.json" },
		func(c *config.Config) { c.ReglasEjemplo.CTSourcePath = "ct_reglas.ejemplo.demo.json" },
		func(c *config.Config) { c.CTAnalisisMotivosSourcePath = "motivos.demo.json" },
	}
	for indice, mutar := range mutaciones {
		cfg := base
		mutar(&cfg)
		if _, err := NewHTTPServerPresentacionWithConfig(cfg); !errors.Is(err, ErrComposicionPresentacionRRHHInvalida) {
			t.Errorf("mutacion %d no rechazada: %v", indice, err)
		}
		if _, err := NewHTTPServerPresentacionPersonalWithConfig(cfg); !errors.Is(err, ErrComposicionPresentacionRRHHInvalida) {
			t.Errorf("Personal: mutacion %d no rechazada: %v", indice, err)
		}
		if _, err := NewHTTPServerPresentacionPersonalRPTWithConfig(cfg); !errors.Is(err, ErrComposicionPresentacionRRHHInvalida) {
			t.Errorf("RPT: mutacion %d no rechazada: %v", indice, err)
		}
	}
	apagada := base
	apagada.FirmaVerificacionEnabled = "false"
	if _, err := NewHTTPServerPresentacionWithConfig(apagada); err != nil {
		t.Fatalf("firma apagada explicitamente rechazada: %v", err)
	}
}
