package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"vec-diputacion-granada/config"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// El portal (web/static/portal-empleado/portal-catalogo-modulos.js) valida cada
// manifiesto de /api/vec/modules y descarta el que no cumple. Un manifiesto
// mal formado dejó una vez Inicio sin módulos; este contrato lo impide en Go.
var (
	contratoWebID       = regexp.MustCompile(`^vec\.module\.[a-z][a-z0-9_.-]{1,79}$`)
	contratoWebClave    = regexp.MustCompile(`^[a-z][a-z0-9_.-]+$`)
	contratoWebVersion  = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	contratoWebRutaBase = regexp.MustCompile(`^/modules/[a-z][a-z0-9/_-]*$`)
	contratoWebRutaMenu = regexp.MustCompile(`^/(?:[a-z0-9/_-]+)$`)
)

func TestCatalogoModulosCumpleContratoDelPortal(t *testing.T) {
	srv := nuevoServidorFakeAisladoPrueba(t, config.Config{Address: "127.0.0.1:0", APIBasePath: "/api", PersonalCatalogPath: "memory"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/vec/modules", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer "+tokenFakePruebas)
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("catálogo = %d", rec.Code)
	}
	var catalogo struct {
		Data struct {
			Modules []vecdomain.ModuleManifest `json:"modules"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &catalogo); err != nil || len(catalogo.Data.Modules) == 0 {
		t.Fatalf("catálogo no decodificable o vacío: %v", err)
	}
	var traducciones map[string]any
	contenido, err := os.ReadFile("../../../locales/es.json")
	if err != nil || json.Unmarshal(contenido, &traducciones) != nil {
		t.Fatalf("locales/es.json no legible: %v", err)
	}
	for _, m := range catalogo.Data.Modules {
		if !contratoWebID.MatchString(m.ID) || !contratoWebClave.MatchString(m.NameKey) ||
			!contratoWebClave.MatchString(m.DescriptionKey) || !contratoWebVersion.MatchString(m.Version) ||
			!contratoWebClave.MatchString(m.Group) || !contratoWebRutaBase.MatchString(m.BasePath) {
			t.Errorf("%s no cumple el contrato del portal: %+v", m.ID, m)
		}
		for _, clave := range []string{m.NameKey, m.DescriptionKey} {
			if texto, ok := traducciones[clave].(string); !ok || texto == "" {
				t.Errorf("%s: falta la traducción %q en locales/es.json", m.ID, clave)
			}
		}
		if len(m.Permissions) == 0 {
			t.Errorf("%s: el portal exige al menos un permiso", m.ID)
		}
		for _, entrada := range m.Menu {
			if !contratoWebRutaMenu.MatchString(entrada.Path) {
				t.Errorf("%s: ruta de menú %q no cumple el contrato del portal", m.ID, entrada.Path)
			}
		}
	}
}
