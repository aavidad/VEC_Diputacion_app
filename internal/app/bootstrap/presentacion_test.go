package bootstrap

import (
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
		func(c *config.Config) { c.AuthMode = config.AuthModeFake },
		func(c *config.Config) { c.StorageMode = config.StorageModeFile },
	}
	for indice, mutar := range mutaciones {
		cfg := base
		mutar(&cfg)
		if _, err := NewHTTPServerPresentacionWithConfig(cfg); !errors.Is(err, ErrComposicionPresentacionRRHHInvalida) {
			t.Errorf("mutacion %d no rechazada: %v", indice, err)
		}
	}
}
