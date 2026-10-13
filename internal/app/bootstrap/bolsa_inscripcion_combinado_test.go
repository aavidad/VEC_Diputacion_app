package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
)

func TestInscripcionCombinadaNoAbreDependenciasApagadaOSeparada(t *testing.T) {
	for _, cfg := range []config.Config{
		{},
		{PortalProceso: config.ValorPortalProcesoInterno, BolsaInscripcionesEnabled: "true"},
		{PortalProceso: config.ValorPortalProcesoExterno, BolsaInscripcionesEnabled: "true"},
	} {
		manejador, cerrar, err := nuevaInscripcionCombinadaDesarrollo(cfg, nil)
		if err != nil || manejador != nil || cerrar == nil {
			t.Fatalf("compuso inscripción sin dependencias en proceso %q: %v", cfg.PortalProceso, err)
		}
		cerrar()
	}
}

func TestInscripcionCombinadaEncendidaExigeSesionYPreflightExternos(t *testing.T) {
	cfg := configInscripcionEncendidaPrueba(t)
	_, _, err := nuevaInscripcionCombinadaDesarrollo(cfg, nil)
	if !errors.Is(err, errMontajeInscripcionBolsa) {
		t.Fatalf("aceptó inscripción sin sesión externa: %v", err)
	}
	_, _, err = nuevaInscripcionCombinadaDesarrollo(cfg, &composicionPreferenciasUsuarios{
		interna: &autoridadPreferenciasUsuariosDesarrollo{},
	})
	if !errors.Is(err, errMontajeInscripcionBolsa) {
		t.Fatalf("usó la sesión interna para el aspirante: %v", err)
	}
	cfg.ExternoPreflightV3DatabaseURL = ""
	_, _, err = nuevaInscripcionCombinadaDesarrollo(cfg, &composicionPreferenciasUsuarios{
		externa: &autoridadPreferenciasUsuariosDesarrollo{},
	})
	if !errors.Is(err, ErrMaterialV3PortalExternoInvalido) {
		t.Fatalf("aceptó inscripción sin preflight externo: %v", err)
	}
}

func TestInscripcionCombinadaDespachaSoloRutasAspirante(t *testing.T) {
	raiz := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	preparador := &preparadorConteoPrueba{}
	externo, err := httpinscripcion.NuevoExterno(preparador, servicioInscripcionNuloPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	compuesta := componerRaizConInscripcionExterna(raiz, externo)
	for _, ruta := range []string{
		httpinscripcion.RutaAbiertas,
		httpinscripcion.RutaPropias,
	} {
		w := httptest.NewRecorder()
		compuesta.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s no llegó a la autoridad externa: %d", ruta, w.Code)
		}
	}
	if preparador.llamadas != 2 {
		t.Fatalf("lecturas propias no preparadas: %d", preparador.llamadas)
	}
	for _, ruta := range []string{
		httpinscripcion.RutaRRHH,
		httpinscripcion.RutaRRHH + "/motivos",
		"/api/vec/bolsa/mi-bolsa",
		httpinscripcion.RutaPropias + "-ajenas",
		"/api/vec/usuarios/mis-preferencias",
	} {
		w := httptest.NewRecorder()
		compuesta.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusTeapot {
			t.Fatalf("ruta ajena desviada al aspirante: %s → %d", ruta, w.Code)
		}
	}
	if preparador.llamadas != 2 {
		t.Fatal("una ruta ajena alcanzó la autoridad externa")
	}
	w := httptest.NewRecorder()
	componerRaizConInscripcionExterna(raiz, nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, httpinscripcion.RutaPropias, nil))
	if w.Code != http.StatusTeapot {
		t.Fatal("selector apagado registró una ruta externa")
	}
}
