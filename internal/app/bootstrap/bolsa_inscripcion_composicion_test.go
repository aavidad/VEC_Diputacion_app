package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

func configInscripcionEncendidaPrueba(t *testing.T) config.Config {
	t.Helper()
	t.Setenv(config.EnvBolsaInscripcionesLectorDatabaseURL, "postgres://externo@localhost/vec")
	t.Setenv(config.EnvBolsaInscripcionesRRHHLectorDatabaseURL, "postgres://rrhh@localhost/vec")
	c := config.Load()
	c.ExecutionProfile, c.AuthMode, c.DevelopmentGuard = config.ExecutionProfileDevelopment, config.AuthModeDevelopment, config.DevelopmentGuardAcknowledgement
	c.BolsaInscripcionesEnabled = "true"
	return c
}

// Con el selector apagado (valor por defecto) ninguna raíz abre conexiones ni
// registra rutas de inscripción, aunque falten todas sus dependencias.
func TestInscripcionApagadaNoRegistraRutasEnNingunaRaiz(t *testing.T) {
	externo, cerrar, err := nuevaInscripcionPortalExterno(context.Background(), config.Config{}, nil, nil)
	if err != nil || externo != nil || cerrar == nil {
		t.Fatalf("portal externo apagado compuso inscripción: %v", err)
	}
	cerrar()
	interno, err := nuevaInscripcionRRHHDesarrollo(config.Config{}, nil, nil, materialInscripcionRRHHDesarrollo{})
	if err != nil || interno != nil {
		t.Fatalf("vec-server apagado compuso inscripción: %v", err)
	}
	raiz := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	compuesta := componerRaizConInscripcionRRHH(raiz, interno)
	for _, ruta := range []string{httpinscripcion.RutaRRHH, httpinscripcion.RutaRRHH + "/convocatorias"} {
		w := httptest.NewRecorder()
		compuesta.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusTeapot {
			t.Fatalf("ruta de inscripción registrada con el selector apagado: %s → %d", ruta, w.Code)
		}
	}
	s, err := seleccionMaterialCTDesarrolloDesdeConfig(config.Config{})
	if err != nil || s.inscripcionBolsa {
		t.Fatalf("material de inscripción pedido con el selector apagado: %v", err)
	}
	for _, d := range descriptoresMaterialSeleccionadosCTDesarrollo(s) {
		for _, propio := range descriptoresMaterialInscripcionRRHHDesarrollo() {
			if d.Audiencia == propio.Audiencia {
				t.Fatalf("audiencia de inscripción publicada apagada: %s", d.Audiencia)
			}
		}
	}
}

func TestInscripcionEncendidaSinDependenciasNoArranca(t *testing.T) {
	invalido := config.Config{BolsaInscripcionesEnabled: "quizas"}
	if _, _, err := nuevaInscripcionPortalExterno(context.Background(), invalido, nil, nil); !errors.Is(err, errMontajeInscripcionBolsa) {
		t.Fatalf("selector inválido en el portal externo: %v", err)
	}
	if _, err := nuevaInscripcionRRHHDesarrollo(invalido, nil, nil, materialInscripcionRRHHDesarrollo{}); !errors.Is(err, errMontajeInscripcionBolsa) {
		t.Fatalf("selector inválido en vec-server: %v", err)
	}
	cfg := configInscripcionEncendidaPrueba(t)
	if _, _, err := nuevaInscripcionPortalExterno(context.Background(), cfg, nil, nil); !errors.Is(err, errMontajeInscripcionBolsa) {
		t.Fatalf("portal externo sin sesión ni preflight: %v", err)
	}
	if _, err := nuevaInscripcionRRHHDesarrollo(cfg, nil, nil, materialInscripcionRRHHDesarrollo{}); !errors.Is(err, errMontajeInscripcionBolsa) {
		t.Fatalf("vec-server sin identidad ni material: %v", err)
	}
	if _, _, err := nuevasCapacidadesPersonalesPortalExterno(cfg, nil, nil); !errors.Is(err, errMontajeInscripcionBolsa) {
		t.Fatalf("portal externo con inscripción y sin Área personal: %v", err)
	}
	s, err := seleccionMaterialCTDesarrolloDesdeConfig(cfg)
	if err != nil || !s.inscripcionBolsa {
		t.Fatalf("encendido sin material de RRHH: %v", err)
	}
}

func TestInscripcionRaizInternaSoloDespachaRRHH(t *testing.T) {
	raiz := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	preparador := &preparadorConteoPrueba{}
	manejador, err := httpinscripcion.NuevoInterno(preparador, servicioInscripcionNuloPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	compuesta := componerRaizConInscripcionRRHH(raiz, &inscripcionRRHHDesarrollo{manejador: manejador, cerrar: func() {}})
	w := httptest.NewRecorder()
	compuesta.ServeHTTP(w, httptest.NewRequest(http.MethodGet, httpinscripcion.RutaRRHH+"/motivos?decision=admitir", nil))
	if preparador.llamadas != 1 {
		t.Fatalf("la bandeja RRHH no llegó a la inscripción: %d", w.Code)
	}
	for _, ruta := range []string{httpinscripcion.RutaPropias, httpinscripcion.RutaAbiertas, "/api/vec/bolsa/mi-bolsa"} {
		w = httptest.NewRecorder()
		compuesta.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusTeapot {
			t.Fatalf("vec-server desvió %s a la inscripción: %d", ruta, w.Code)
		}
	}
}

type preparadorConteoPrueba struct{ llamadas int }

func (p *preparadorConteoPrueba) PrepararLecturaAspirante(*http.Request, string, string, inscripcion.Filtro, string) (inscripcion.Actor, error) {
	p.llamadas++
	return inscripcion.Actor{}, inscripcion.ErrAccesoDenegado
}
func (p *preparadorConteoPrueba) PrepararLecturaRRHH(*http.Request, string, string, inscripcion.Filtro, string) (inscripcion.Actor, error) {
	p.llamadas++
	return inscripcion.Actor{}, inscripcion.ErrAccesoDenegado
}
func (p *preparadorConteoPrueba) PrepararPresentacion(*http.Request, inscripcion.Presentacion) (inscripcion.Actor, error) {
	p.llamadas++
	return inscripcion.Actor{}, inscripcion.ErrAccesoDenegado
}
func (p *preparadorConteoPrueba) PrepararDecision(*http.Request, inscripcion.Decision) (inscripcion.Actor, error) {
	p.llamadas++
	return inscripcion.Actor{}, inscripcion.ErrAccesoDenegado
}
func (p *preparadorConteoPrueba) PrepararIncorporacion(*http.Request, inscripcion.Incorporacion) (inscripcion.Actor, error) {
	p.llamadas++
	return inscripcion.Actor{}, inscripcion.ErrAccesoDenegado
}

// servicioInscripcionNuloPrueba no debe alcanzarse: el preparador deniega antes.
type servicioInscripcionNuloPrueba struct{ httpinscripcion.Aplicacion }
