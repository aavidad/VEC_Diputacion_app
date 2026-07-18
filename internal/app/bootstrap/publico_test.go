package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestAPIPublicaBolsaSoloExponeConsultasAnonimas(t *testing.T) {
	api, err := NewAPIPublicaBolsaWithConfig(config.Config{})
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

func TestComposicionPublicaIgnoraCredencialesPersonalYAlmacenHeredado(t *testing.T) {
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
	if _, err := NewDemoAPIWithConfig(cfg); err == nil {
		t.Fatal("la configuracion de prueba no invalida realmente la composicion interna")
	}

	api, err := NewAPIPublicaBolsaWithConfig(cfg)
	if err != nil {
		t.Fatalf("la API publica cargo una dependencia privada: %v", err)
	}
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/publico/bolsa/convocatorias", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("consulta publica anonima = %d %s", rec.Code, rec.Body.String())
	}

	servidor, err := NewHTTPServerPublicoWithConfig(cfg)
	if err != nil {
		t.Fatalf("el servidor publico cargo una dependencia privada: %v", err)
	}
	for _, prueba := range []struct {
		ruta   string
		estado int
	}{
		{ruta: "/api/publico/bolsa/categorias", estado: http.StatusOK},
		{ruta: "/api/vec", estado: http.StatusNotFound},
		{ruta: "/api/demo", estado: http.StatusNotFound},
	} {
		rec = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, prueba.ruta, nil)
		req.RemoteAddr = "203.0.113.8:12345"
		servidor.Handler.ServeHTTP(rec, req)
		if rec.Code != prueba.estado {
			t.Fatalf("GET %s = %d, esperado %d: %s", prueba.ruta, rec.Code, prueba.estado, rec.Body.String())
		}
	}
}

func TestAPIPublicaBolsaRechazaHuellaCatalogoNoFijada(t *testing.T) {
	_, err := NewAPIPublicaBolsaWithConfig(config.Config{
		BolsaCategoriesSHA256: strings.Repeat("a", 64),
	})
	if err == nil || !strings.Contains(err.Error(), "catalogo gobernado de categorias de Bolsa incompatible") {
		t.Fatalf("huella distinta no rechazo la composicion publica: %v", err)
	}
}
