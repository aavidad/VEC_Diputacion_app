package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
)

func TestInscripcionCombinadaNoAbreDependenciasApagadaOSeparada(t *testing.T) {
	for _, cfg := range []config.Config{
		{},
		{PortalProceso: config.ValorPortalProcesoInterno, BolsaInscripcionesEnabled: "true"},
		{PortalProceso: config.ValorPortalProcesoExterno, BolsaInscripcionesEnabled: "true"},
	} {
		manejador, inscripcion, cerrar, err := nuevasRutasBolsaPersonalCombinado(cfg, nil, nil)
		if err != nil || manejador != nil || inscripcion != nil || cerrar == nil {
			t.Fatalf("compuso inscripción sin dependencias en proceso %q: %v", cfg.PortalProceso, err)
		}
		cerrar()
	}
}

func TestBolsaCombinadaConservaPersonaCandidatoYCertificadoAlCambiarPerfil(t *testing.T) {
	base := identidadCandidatoBolsaDesarrollo{personaRef: "per_carmen_001", candidatoRef: "can_carmen_001", cuentaRef: "cta_carmen_001", perfilRef: "prf_interno_001"}
	base.identidad.huella[0] = 1
	base.identidad.principal.ID = "actor_carmen_001"
	ext := base
	ext.perfilRef = "prf_externo_001"
	a := &resolvedorIdentidadDesarrollo{candidatoBolsa: &base}
	b := &resolvedorIdentidadDesarrollo{candidatoBolsa: &ext}
	if !mismaIdentidadCandidataBolsa(a, b) {
		t.Fatal("cambiar el perfil cambió la identidad propietaria")
	}
	for _, cambiar := range []func(*identidadCandidatoBolsaDesarrollo){
		func(c *identidadCandidatoBolsaDesarrollo) { c.personaRef = "per_otra_001" },
		func(c *identidadCandidatoBolsaDesarrollo) { c.candidatoRef = "can_otro_001" },
		func(c *identidadCandidatoBolsaDesarrollo) { c.cuentaRef = "cta_otra_001" },
		func(c *identidadCandidatoBolsaDesarrollo) { c.perfilRef = base.perfilRef },
		func(c *identidadCandidatoBolsaDesarrollo) { c.identidad.principal.ID = "actor_otro_001" },
		func(c *identidadCandidatoBolsaDesarrollo) { c.identidad.huella[0] = 2 },
	} {
		ajeno := ext
		cambiar(&ajeno)
		if mismaIdentidadCandidataBolsa(a, &resolvedorIdentidadDesarrollo{candidatoBolsa: &ajeno}) {
			t.Fatal("admitió identidad ajena en el cambio de autoridad")
		}
	}
	if mismaIdentidadCandidataBolsa(nil, b) || mismaIdentidadCandidataBolsa(a, &resolvedorIdentidadDesarrollo{}) {
		t.Fatal("admitió una identidad ausente")
	}
}

func TestBolsaCombinadaMiBolsaYPreferenciasCompartenMontajeExterno(t *testing.T) {
	raiz := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	externa := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	compuesta := componerRaizConBolsaPersonalExterna(raiz, externa, nil)
	for _, ruta := range []string{bolsahttp.RutaMiBolsa, bolsahttp.RutaMiBolsaHistorial, bolsahttp.RutaPreferenciasBolsas} {
		w := httptest.NewRecorder()
		compuesta.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("ruta externa cayó a la interna: %s → %d", ruta, w.Code)
		}
	}
	w := httptest.NewRecorder()
	compuesta.ServeHTTP(w, httptest.NewRequest(http.MethodGet, httpinscripcion.RutaRRHH, nil))
	if w.Code != http.StatusTeapot {
		t.Fatal("la bandeja RRHH cayó al montaje externo")
	}
}

func TestBolsaCombinadaEncendidaExigeAutoridadExterna(t *testing.T) {
	cfg := configInscripcionEncendidaPrueba(t)
	_, _, _, err := nuevasRutasBolsaPersonalCombinado(cfg, nil, nil)
	if !errors.Is(err, errMiBolsaNoDisponible) {
		t.Fatalf("aceptó inscripción sin candidato externo: %v", err)
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
	compuesta := componerRaizConBolsaPersonalExterna(raiz, nil, externo)
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
	componerRaizConBolsaPersonalExterna(raiz, nil, nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, httpinscripcion.RutaPropias, nil))
	if w.Code != http.StatusTeapot {
		t.Fatal("selector apagado registró una ruta externa")
	}
}
