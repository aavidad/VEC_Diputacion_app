package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestAPIPublicaBolsaSoloExponeConsultasAnonimas(t *testing.T) {
	api, err := NewAPIPublicaBolsaWithConfig(configuracionAPIPrueba(config.Config{}))
	if err != nil {
		t.Fatalf("NewAPIPublicaBolsaWithConfig() error = %v", err)
	}

	for _, prueba := range []struct {
		ruta      string
		contenido string
	}{
		{ruta: "/api/publico/bolsa/convocatorias", contenido: "vec.bolsa.publico.convocatorias.v1"},
		{ruta: "/api/publico/bolsa/categorias", contenido: `"total":68`},
	} {
		rec := httptest.NewRecorder()
		api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, prueba.ruta, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), prueba.contenido) {
			t.Fatalf("GET anonimo %s = %d %s", prueba.ruta, rec.Code, rec.Body.String())
		}
	}

	for _, ruta := range []string{
		"/api/vec",
		"/api/vec/session",
		"/api/demo",
		"/candidates",
		"/api/publico/bolsa/personas",
	} {
		rec := httptest.NewRecorder()
		api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("la composicion publica expuso %s: estado=%d cuerpo=%s", ruta, rec.Code, rec.Body.String())
		}
	}
}

func TestComposicionPublicaNoCargaCredencialesPersonalNiAlmacenHeredado(t *testing.T) {
	rutaInvalida := t.TempDir()
	cfg := config.Config{
		Address:             "0.0.0.0:0",
		HTTPAllowedCIDRs:    []string{"0.0.0.0/0"},
		AuthMode:            config.AuthModeFake,
		FakeCredentialsPath: filepath.Join(rutaInvalida, "credenciales-inexistentes.json"),
		TrustedProxyCIDRs:   []string{"red-interna-mal-configurada"},
		StorageMode:         config.StorageModeFile,
		DataPath:            rutaInvalida,
		PersonalCatalogPath: rutaInvalida,
	}
	cfg = configurarFuentesProduccionPrueba(t, cfg)
	cfgAPI := configuracionAPIPrueba(cfg)
	if _, err := NewDemoAPIWithConfig(cfgAPI); err == nil {
		t.Fatal("la configuracion de prueba no invalida realmente la composicion interna")
	}

	if api, err := NewAPIPublicaBolsaWithConfig(cfgAPI); api != nil ||
		!errors.Is(err, ErrAutenticacionPublicaNoAdmitida) {
		t.Fatalf("la API publica acepto autenticacion fake: api=%v error=%v", api, err)
	}
	cfgAPI.AuthMode = config.AuthModeDisabled
	api, err := NewAPIPublicaBolsaWithConfig(cfgAPI)
	if err != nil {
		t.Fatalf("la API publica cargo una dependencia privada: %v", err)
	}
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/publico/bolsa/convocatorias", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("consulta publica anonima = %d %s", rec.Code, rec.Body.String())
	}

	cfg.AuthMode = config.AuthModeDisabled
	servidor, err := NewHTTPServerPublicoWithConfig(cfg)
	if servidor != nil || !errors.Is(err, ErrActivacionDesarrolloInvalida) {
		t.Fatalf("servidor publico productivo = (%v, %v)", servidor, err)
	}

	// Sin selectores heredados, la raiz alcanza la validacion de su unica
	// dependencia autoritativa y falla cerrada por ausencia de PostgreSQL.
	cfg.BolsaPublicSourcePath = ""
	cfg.BolsaCategoriesSourcePath = ""
	servidor, err = NewHTTPServerPublicoWithConfig(cfg)
	if servidor != nil || !errors.Is(err, config.ErrConfiguracionPostgreSQLPublicaIncompleta) {
		t.Fatalf("servidor publico sin PostgreSQL = (%v, %v)", servidor, err)
	}
}

func TestAPIPublicaBolsaRechazaHuellaCatalogoNoFijada(t *testing.T) {
	_, err := NewAPIPublicaBolsaWithConfig(configuracionAPIPrueba(config.Config{
		BolsaCategoriesSHA256: strings.Repeat("a", 64),
	}))
	if err == nil || !strings.Contains(err.Error(), "catalogo gobernado de categorias de Bolsa incompatible") {
		t.Fatalf("huella distinta no rechazo la composicion publica: %v", err)
	}
}
