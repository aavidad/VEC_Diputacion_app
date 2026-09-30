package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
)

func atiende(t *testing.T, h http.Handler, ruta string) bool {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code == http.StatusNoContent
}

func filtroCon(portal string) http.Handler {
	return restringirRutasPortalProceso(portal, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

var rutasInternasMuestra = []string{
	"/portal-empleado/",
	"/portal-empleado/contratacion-temporal/",
	"/api/vec/contratacion-temporal/expedientes",
	"/api/vec/bolsa/bolsas",
	"/api/vec/bolsa/llamamientos",
	"/api/vec/usuarios/mis-preferencias",
	"/api/vec/usuarios/mis-correos",
	"/api/vec/session",
	"/api/vec/personal/categories",
	"/candidates",
	"/cartografia/granada.zip",
}

var rutasExternasMuestra = []string{
	"/area-personal",
	"/area-personal/",
	"/area-personal/aplicacion.js",
	"/api/vec/bolsa/mi-bolsa",
	"/api/vec/bolsa/mi-bolsa/contacto",
	"/api/vec/bolsa/mis-solicitudes/borrador",
	"/api/vec/bolsa/area-personal",
	"/api/vec/usuarios/area-personal/mis-correos",
	"/api/vec/usuarios/contacto-propio",
	"/api/vec/personas/mi-perfil/contacto",
}

var rutasPublicasMuestra = []string{
	"/bolsa/", "/verificar/", "/api/publico/bolsa/bolsas", "/comun/tema-vec.css", "/locales/es.json", "/readyz",
}

func TestProcesoCombinadoAtiendeTodasLasRutas(t *testing.T) {
	h := filtroCon("")
	for _, grupo := range [][]string{rutasInternasMuestra, rutasExternasMuestra, rutasPublicasMuestra} {
		for _, ruta := range grupo {
			if !atiende(t, h, ruta) {
				t.Fatalf("el proceso combinado debe atender %s", ruta)
			}
		}
	}
}

func TestProcesoExternoSoloAtiendeSuPortal(t *testing.T) {
	h := filtroCon("externo")
	for _, ruta := range rutasInternasMuestra {
		if atiende(t, h, ruta) {
			t.Fatalf("el proceso externo no debe atender %s", ruta)
		}
	}
	for _, grupo := range [][]string{rutasExternasMuestra, rutasPublicasMuestra} {
		for _, ruta := range grupo {
			if !atiende(t, h, ruta) {
				t.Fatalf("el proceso externo debe atender %s", ruta)
			}
		}
	}
}

func TestProcesoInternoNoAtiendeElAreaPersonal(t *testing.T) {
	h := filtroCon("interno")
	for _, ruta := range rutasExternasMuestra {
		if atiende(t, h, ruta) {
			t.Fatalf("el proceso interno no debe atender %s", ruta)
		}
	}
	for _, grupo := range [][]string{rutasInternasMuestra, rutasPublicasMuestra} {
		for _, ruta := range grupo {
			if !atiende(t, h, ruta) {
				t.Fatalf("el proceso interno debe atender %s", ruta)
			}
		}
	}
}

func TestPortalDeProcesoMalEscritoCierraTodo(t *testing.T) {
	h := filtroCon("Externo")
	for _, ruta := range append(append([]string{}, rutasExternasMuestra...), rutasPublicasMuestra...) {
		if atiende(t, h, ruta) {
			t.Fatalf("un portal no valido no debe atender %s", ruta)
		}
	}
}

// La restricción está montada en la superficie integrada real, después del
// rechazo de rutas no canónicas: una variante escapada o con «..» tampoco
// alcanza la API desde el proceso interno.
func TestSuperficieIntegradaAplicaElPortalDelProceso(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	codigo := func(portal, ruta string) int {
		h := NewHandlerWithConfig(config.Config{PortalProceso: portal, HTTPAllowedCIDRs: []string{"192.0.2.0/24"}}, api)
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if got := codigo("interno", "/api/vec/bolsa/mi-bolsa"); got != http.StatusNotFound {
		t.Fatalf("interno sirvio Mi bolsa: %d", got)
	}
	if got := codigo("interno", "/api/vec/bolsa/bolsas"); got != http.StatusNoContent {
		t.Fatalf("interno no sirvio su API: %d", got)
	}
	if got := codigo("externo", "/api/vec/contratacion-temporal/expedientes"); got != http.StatusNotFound {
		t.Fatalf("externo sirvio la API interna: %d", got)
	}
	if got := codigo("externo", "/api/vec/bolsa/mi-bolsa"); got != http.StatusNoContent {
		t.Fatalf("externo no sirvio Mi bolsa: %d", got)
	}
	if got := codigo("externo", "/api/vec/bolsa/mi-bolsa/../../contratacion-temporal/expedientes"); got != http.StatusNotFound {
		t.Fatalf("externo acepto una ruta no canonica: %d", got)
	}
	if got := codigo("", "/api/vec/bolsa/mi-bolsa"); got != http.StatusNoContent {
		t.Fatalf("el combinado debe seguir sirviendo Mi bolsa: %d", got)
	}
}
