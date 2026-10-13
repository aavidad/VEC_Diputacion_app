package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/config"
	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
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

func TestBolsaCombinadaSeleccionExternaSinSustitucion(t *testing.T) {
	for _, caso := range []struct {
		nombre, dsn, directorio, inscripcion string
		externa                              bool
	}{
		{"sin externo", "", "", "", false},
		{"externo apagado", "", "", "false", false},
		{"selección completa", "postgres://externo@localhost/vec", "/material-externo", "false", true},
		{"sólo DSN", "postgres://externo@localhost/vec", "", "false", true},
		{"sólo material", "", "/material-externo", "false", true},
		{"sólo inscripción", "", "", "true", true},
		{"selector inválido", "", "", "incorrecto", true},
		{"DSN inválido", "incorrecto", "", "false", true},
		{"material relativo", "", "material-relativo", "false", true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Setenv(config.EnvExternoBolsaDatabaseURL, caso.dsn)
			cfg := config.Config{ExternoBolsaPostgreSQL: config.Load().ExternoBolsaPostgreSQL,
				ExternoMaterialDir: caso.directorio, BolsaInscripcionesEnabled: caso.inscripcion}
			if seleccionada := miBolsaExternaSeleccionadaEnCombinado(cfg); seleccionada != caso.externa {
				t.Fatalf("selección externa: esperado %t, recibido %t", caso.externa, seleccionada)
			}
			for _, identidad := range []*resolvedorIdentidadDesarrollo{nil, {candidatoBolsa: &identidadCandidatoBolsaDesarrollo{}}} {
				bolsa, inscripcion, cerrar, err := nuevasRutasBolsaPersonalCombinado(cfg, identidad, nil)
				if (err != nil) != caso.externa || bolsa != nil || inscripcion != nil || cerrar == nil {
					t.Fatalf("montaje sin autoridad: error esperado %t, recibido %v", caso.externa, err)
				}
				cerrar()
			}
		})
	}
}

func TestBolsaCombinadaSeleccionComparteMaterialHZ36(t *testing.T) {
	t.Setenv(config.EnvBolsaLlamamientosDatabaseURL, "postgres://bolsa@localhost/vec")
	t.Setenv(config.EnvExternoBolsaDatabaseURL, "postgres://externo@localhost/vec")
	cfg := config.Config{
		ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard:               config.DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir:         t.TempDir(),
		ContratacionTemporalPostgreSQL: config.Load().ContratacionTemporalPostgreSQL,
	}
	directorio := filepath.Join(cfg.DevelopmentMaterialDir, "identidad")
	if err := os.Mkdir(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directorio, "bolsa-candidato.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, externa := range []bool{false, true} {
		if externa {
			cfg.ExternoBolsaPostgreSQL = config.Load().ExternoBolsaPostgreSQL
			cfg.ExternoMaterialDir = "/material-externo"
		}
		seleccion, err := seleccionMaterialCTDesarrolloDesdeConfig(cfg)
		if err != nil || seleccion.miBolsa == externa {
			t.Fatalf("material Mi Bolsa: interno esperado %t, recibido %t, error %v", !externa, seleccion.miBolsa, err)
		}
		for _, descriptor := range []descriptorMaterialConsumidorV3Desarrollo{descriptorMaterialMiBolsaDesarrollo(), descriptorMaterialHistorialMiBolsaDesarrollo()} {
			coincidencias := 0
			for _, material := range descriptoresMaterialSeleccionadosCTDesarrollo(seleccion) {
				if material.Audiencia == descriptor.Audiencia {
					coincidencias++
				}
			}
			esperadas := 0
			if !externa {
				esperadas = 1
			}
			if coincidencias != esperadas {
				t.Fatalf("audiencia %s: esperado %d, recibido %d", descriptor.Audiencia, esperadas, coincidencias)
			}
		}
	}
}

func TestBolsaCombinadaConservaPersonaCandidatoYCertificadoAlCambiarPerfil(t *testing.T) {
	base := identidadCandidatoBolsaDesarrollo{personaRef: "per_carmen_001", candidatoRef: "can_carmen_001", cuentaRef: "cta_carmen_001", perfilRef: "prf_interno_001"}
	base.identidad.huella[0] = 1
	base.identidad.principal.ID = "actor_carmen_001"
	ext := base
	ext.perfilRef = "prf_externo_001"
	ext.cuentaRef = "cta_externa_001"
	a := &resolvedorIdentidadDesarrollo{candidatoBolsa: &base}
	b := &resolvedorIdentidadDesarrollo{candidatoBolsa: &ext}
	if !mismaIdentidadCandidataBolsa(a, b) {
		t.Fatal("perfil y cuenta externos propios cambiaron la identidad propietaria")
	}
	for _, cambiar := range []func(*identidadCandidatoBolsaDesarrollo){
		func(c *identidadCandidatoBolsaDesarrollo) { c.personaRef = "per_otra_001" },
		func(c *identidadCandidatoBolsaDesarrollo) { c.candidatoRef = "can_otro_001" },
		func(c *identidadCandidatoBolsaDesarrollo) { c.cuentaRef = base.cuentaRef },
		func(c *identidadCandidatoBolsaDesarrollo) { c.cuentaRef = "" },
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
	for _, ruta := range []string{bolsahttp.RutaMiBolsa, bolsahttp.RutaMiBolsaHistorial, bolsahttp.RutaPreferenciasBolsas,
		meritoshttp.RutaCartera, meritoshttp.RutaRetiradasCartera, meritoshttp.RutaDeclaracionesCartera} {
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
	w = httptest.NewRecorder()
	compuesta.ServeHTTP(w, httptest.NewRequest(http.MethodGet, meritoshttp.RutaCartera+"-ajena", nil))
	if w.Code != http.StatusTeapot {
		t.Fatal("una ruta distinta de cartera cayó al montaje externo")
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
