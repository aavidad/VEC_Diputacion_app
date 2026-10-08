package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

type manejadorSesionNominalPrueba struct {
	peticiones     []string
	auditorias     []string
	falloAuditoria error
}

func (m *manejadorSesionNominalPrueba) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.peticiones = append(m.peticiones, r.Method+" "+r.URL.RequestURI())
	w.WriteHeader(http.StatusNoContent)
}

func (m *manejadorSesionNominalPrueba) AuditarRechazoSesionNoCanonica(_ context.Context, rutaCanonica string) error {
	m.auditorias = append(m.auditorias, rutaCanonica)
	return m.falloAuditoria
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
		{http.MethodGet, "/api/vec/session"},
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
	if len(manejador.auditorias) != 0 {
		t.Fatalf("las rutas canónicas provocaron %d auditorías de rechazo", len(manejador.auditorias))
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

func TestSesionNominalURLNoCanonicaAuditaSinDelegar(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre       string
		ruta         string
		rutaAuditada string
		alterar      func(*http.Request)
	}{
		{"sesión escapada", "/api/vec/%73ession", "/api/vec/session", nil},
		{"inicio escapado", "/api/vec/%73ession/start", "/api/vec/session/start", nil},
		{"opaque", "/api/vec/session", "/api/vec/session", func(r *http.Request) { r.URL.Opaque = "hostil" }},
		{"force query", "/api/vec/session/start", "/api/vec/session/start", func(r *http.Request) { r.URL.ForceQuery = true }},
		{"query", "/api/vec/session?secreto=cliente", "/api/vec/session", nil},
		{"URI absoluta", "https://extra.invalid/api/vec/session/start", "/api/vec/session/start", nil},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador := &manejadorSesionNominalPrueba{}
			h, err := NewHandlerWithOptions(servicioSesionNominalPrueba(t), HandlerOptions{ManejadorSesionNominal: manejador})
			if err != nil {
				t.Fatal(err)
			}
			peticion := httptest.NewRequest(http.MethodGet, caso.ruta, nil)
			if caso.alterar != nil {
				caso.alterar(peticion)
			}
			respuesta := httptest.NewRecorder()
			h.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest ||
				!strings.Contains(respuesta.Body.String(), "solicitud_invalida") ||
				respuesta.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("%s: estado=%d cuerpo=%s", caso.nombre, respuesta.Code, respuesta.Body.String())
			}
			if len(manejador.peticiones) != 0 || len(manejador.auditorias) != 1 ||
				manejador.auditorias[0] != caso.rutaAuditada ||
				strings.Contains(respuesta.Body.String(), "secreto=cliente") ||
				strings.Contains(respuesta.Body.String(), "extra.invalid") {
				t.Fatalf("%s: delegadas=%v auditadas=%v", caso.nombre, manejador.peticiones, manejador.auditorias)
			}
		})
	}
}

func TestSesionNominalAuditoriaNoCanonicaCaidaFallaCerrada(t *testing.T) {
	t.Parallel()
	manejador := &manejadorSesionNominalPrueba{falloAuditoria: errors.New("clave interna secreta")}
	emisor := &emisorContextoPrueba{}
	h, err := NewHandlerWithOptions(servicioSesionNominalPrueba(t), HandlerOptions{
		ManejadorSesionNominal: manejador, EmisorIncidenciasTecnicas: emisor,
	})
	if err != nil {
		t.Fatal(err)
	}
	peticion, declarada := peticionMarcada(http.MethodGet, "/api/vec/%73ession?secreto=cliente")
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusServiceUnavailable ||
		strings.TrimSpace(respuesta.Body.String()) != `{"error":"auditoria_no_disponible"}` ||
		strings.Contains(respuesta.Body.String(), "secreto") || !declarada() ||
		len(manejador.peticiones) != 0 || len(manejador.auditorias) != 1 ||
		manejador.auditorias[0] != "/api/vec/session" {
		t.Fatalf("estado=%d cuerpo=%s delegadas=%v auditadas=%v", respuesta.Code,
			respuesta.Body.String(), manejador.peticiones, manejador.auditorias)
	}
	if incidencia := emisor.unica(t); incidencia.Codigo != domain.IncidenciaAuditoriaNoRegistrada {
		t.Fatalf("incidencia = %v", incidencia)
	}
	manejador.falloAuditoria = domain.ErrPermissionDenied
	respuestaPermiso := httptest.NewRecorder()
	h.ServeHTTP(respuestaPermiso, httptest.NewRequest(http.MethodGet, "/api/vec/%73ession", nil))
	if respuestaPermiso.Code != http.StatusServiceUnavailable || len(manejador.peticiones) != 0 {
		t.Fatalf("error de auditoría de permiso: estado=%d delegadas=%v", respuestaPermiso.Code, manejador.peticiones)
	}
}

func TestSesionNominalURLNoCanonicaCanceladaNuncaDelega(t *testing.T) {
	t.Parallel()
	manejador := &manejadorSesionNominalPrueba{}
	h, err := NewHandlerWithOptions(servicioSesionNominalPrueba(t), HandlerOptions{ManejadorSesionNominal: manejador})
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodPost, "/api/vec/session/start?dato=cliente", nil)
	ctx, cancelar := context.WithCancel(peticion.Context())
	cancelar()
	peticion = peticion.WithContext(ctx)
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusServiceUnavailable || len(manejador.peticiones) != 0 ||
		len(manejador.auditorias) != 1 || manejador.auditorias[0] != "/api/vec/session/start" {
		t.Fatalf("estado=%d delegadas=%v auditadas=%v", respuesta.Code, manejador.peticiones, manejador.auditorias)
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
