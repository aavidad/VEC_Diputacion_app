package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
)

type manejadorSesionNominalPrueba struct {
	peticiones []string
}

func (m *manejadorSesionNominalPrueba) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.peticiones = append(m.peticiones, r.Method+" "+r.URL.RequestURI())
	w.WriteHeader(http.StatusNoContent)
}

func servicioSesionNominalPrueba(t *testing.T) *application.Service {
	t.Helper()
	almacen := memory.NewStore()
	servicio, err := application.NewService(almacen, almacen, almacen)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio
}

func TestSesionNominalDelegaSoloDosRutasExactasAntesDeIdentidadDeCarcasa(t *testing.T) {
	t.Parallel()
	manejador := &manejadorSesionNominalPrueba{}
	h, err := NewHandlerWithOptions(servicioSesionNominalPrueba(t), HandlerOptions{
		ManejadorSesionNominal: manejador,
	})
	if err != nil {
		t.Fatalf("componer manejador nominal: %v", err)
	}
	casos := []struct {
		metodo string
		ruta   string
	}{
		{http.MethodGet, "/api/vec/session?consulta=opaca"},
		{http.MethodPost, "/api/vec/session/start"},
		{http.MethodDelete, "/api/vec/session"},
		{http.MethodPatch, "/api/vec/session/start"},
	}
	for _, caso := range casos {
		respuesta := httptest.NewRecorder()
		peticion := httptest.NewRequest(caso.metodo, caso.ruta, nil)
		peticion.Header.Set("X-VEC-Subject", "aportado-por-cliente")
		h.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusNoContent {
			t.Fatalf("%s %s: estado = %d", caso.metodo, caso.ruta, respuesta.Code)
		}
	}
	if len(manejador.peticiones) != len(casos) {
		t.Fatalf("delegaciones = %d; se esperaban %d", len(manejador.peticiones), len(casos))
	}
	for i, caso := range casos {
		if manejador.peticiones[i] != caso.metodo+" "+caso.ruta {
			t.Fatalf("peticion %d alterada: %q", i, manejador.peticiones[i])
		}
	}
	for _, ruta := range []string{"/api/vec/modules", "/api/vec/session/otra"} {
		respuesta := httptest.NewRecorder()
		peticion := httptest.NewRequest(http.MethodGet, ruta, nil)
		peticion.Header.Set("X-VEC-Subject", "aportado-por-cliente")
		h.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusUnauthorized || len(manejador.peticiones) != len(casos) {
			t.Fatalf("%s: estado=%d delegaciones=%d", ruta, respuesta.Code, len(manejador.peticiones))
		}
	}
}

func TestSesionNominalAusenteConservaEntradaAnterior(t *testing.T) {
	t.Parallel()
	h, err := NewHandlerWithOptions(servicioSesionNominalPrueba(t), HandlerOptions{})
	if err != nil {
		t.Fatalf("componer manejador sin sesión nominal: %v", err)
	}
	for _, ruta := range []string{"/api/vec/session", "/api/vec/session/start"} {
		respuesta := httptest.NewRecorder()
		h.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, nil))
		if respuesta.Code != http.StatusUnauthorized {
			t.Fatalf("%s: estado anterior = %d", ruta, respuesta.Code)
		}
	}
}

func TestSesionNominalRechazaMezclaDeFuentesYNilTipado(t *testing.T) {
	t.Parallel()
	servicio := servicioSesionNominalPrueba(t)
	manejador := &manejadorSesionNominalPrueba{}
	var nulo *manejadorSesionNominalPrueba
	casos := []struct {
		nombre   string
		opciones HandlerOptions
	}{
		{"nil tipado", HandlerOptions{ManejadorSesionNominal: nulo}},
		{"funcion nil", HandlerOptions{ManejadorSesionNominal: http.HandlerFunc(nil)}},
		{"mux", HandlerOptions{ManejadorSesionNominal: http.NewServeMux()}},
		{"demo", HandlerOptions{ManejadorSesionNominal: manejador, AllowDemoIdentity: true}},
		{"resolvedor demo", HandlerOptions{ManejadorSesionNominal: manejador, DemoIdentityResolver: resolvedorIdentidadPruebas{}}},
		{"cabeceras", HandlerOptions{ManejadorSesionNominal: manejador, TrustIdentityHeaders: true}},
	}
	for _, caso := range casos {
		_, err := NewHandlerWithOptions(servicio, caso.opciones)
		if !errors.Is(err, ErrSesionNominalInvalida) {
			t.Fatalf("%s: error = %v", caso.nombre, err)
		}
	}
}

func TestSesionNominalReservaAmbasRutasFrenteAExactasYColecciones(t *testing.T) {
	t.Parallel()
	servicio := servicioSesionNominalPrueba(t)
	manejador := &manejadorSesionNominalPrueba{}
	for _, ruta := range []string{"/api/vec/session", "/api/vec/session/start"} {
		for _, coleccion := range []bool{false, true} {
			opciones := HandlerOptions{AutoridadRutasExactas: autoridadRutasExactasPrueba{}}
			if coleccion {
				opciones.RutasColeccion = []RutaColeccion{{Prefijo: ruta, Manejador: manejador}}
			} else {
				opciones.RutasExactas = []RutaExacta{{Ruta: ruta, Manejador: manejador}}
			}
			_, err := NewHandlerWithOptions(servicio, opciones)
			if !errors.Is(err, ErrRutaExactaInvalida) {
				t.Fatalf("ruta=%s coleccion=%t: error = %v", ruta, coleccion, err)
			}
		}
	}
}
