package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

type manejadorSesionNominalPrueba struct {
	peticiones []string
	auditorias []string
	fallo      error
	estado     int
}

func (m *manejadorSesionNominalPrueba) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.peticiones = append(m.peticiones, r.Method+" "+r.URL.RequestURI())
	if m.estado == 0 {
		m.estado = http.StatusNoContent
	}
	w.WriteHeader(m.estado)
}

func (m *manejadorSesionNominalPrueba) AuditarRechazoSesionNoCanonica(_ context.Context, ruta string) error {
	m.auditorias = append(m.auditorias, ruta)
	return m.fallo
}

func servicioSesionNominalPrueba(t *testing.T) *application.Service {
	t.Helper()
	almacen := memory.NewStore()
	servicio, err := application.NewService(almacen, almacen, almacen)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func TestSesionNominalContratoHTTP(t *testing.T) {
	servicio := servicioSesionNominalPrueba(t)
	manejador := &manejadorSesionNominalPrueba{}
	h, err := NewHandlerWithOptions(servicio, HandlerOptions{ManejadorSesionNominal: manejador})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		metodo, ruta string
	}{
		{http.MethodGet, "/api/vec/session"},
		{http.MethodPost, "/api/vec/session/start"},
	} {
		respuesta := httptest.NewRecorder()
		peticion := httptest.NewRequest(caso.metodo, caso.ruta, nil)
		h.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusNoContent || respuesta.Header().Get("Cache-Control") != "no-store" ||
			respuesta.Header().Get("Set-Cookie") != "" || len(manejador.peticiones) == 0 ||
			manejador.peticiones[len(manejador.peticiones)-1] != caso.metodo+" "+caso.ruta {
			t.Fatalf("%s %s: estado=%d delegaciones=%v", caso.metodo, caso.ruta, respuesta.Code, manejador.peticiones)
		}
	}
	for _, caso := range []struct {
		metodo, ruta, auditada, cuerpo string
		estado                         int
	}{
		{http.MethodGet, "/api/vec/session/start", "/api/vec/session/start", "", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/vec/session", "/api/vec/session", "", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/vec/%73ession", "/api/vec/session", "", http.StatusBadRequest},
		{http.MethodPost, "/api/vec/%73ession/start", "/api/vec/session/start", "", http.StatusBadRequest},
		{http.MethodGet, "/api/vec/session?token=ajeno", "/api/vec/session", "", http.StatusBadRequest},
		{http.MethodPost, "/api/vec/session/start", "/api/vec/session/start", `{"actor":"inyectado","perfil":"ajeno","nonce":"cliente"}`, http.StatusBadRequest},
	} {
		respuesta := httptest.NewRecorder()
		var cuerpo io.Reader
		if caso.cuerpo != "" {
			cuerpo = strings.NewReader(caso.cuerpo)
		}
		peticion := httptest.NewRequest(caso.metodo, caso.ruta, cuerpo)
		peticion.Header.Set("Authorization", "Bearer ajeno")
		peticion.Header.Set("X-VEC-Perfil", "inyectado")
		peticion.Header.Set("Cookie", "sesion=ajena")
		h.ServeHTTP(respuesta, peticion)
		if respuesta.Code != caso.estado || respuesta.Header().Get("Cache-Control") != "no-store" ||
			respuesta.Header().Get("Set-Cookie") != "" ||
			len(manejador.peticiones) != 2 ||
			len(manejador.auditorias) == 0 ||
			manejador.auditorias[len(manejador.auditorias)-1] != caso.auditada ||
			strings.Contains(respuesta.Body.String(), "token=ajeno") {
			t.Fatalf("%s %s: estado=%d delegaciones=%v auditorias=%v", caso.metodo, caso.ruta,
				respuesta.Code, manejador.peticiones, manejador.auditorias)
		}
	}
	// Las credenciales extrañas de una URL canónica llegan sólo al manejador
	// nominal; la carcasa nunca intenta resolverlas como identidad demo.
	manejador.estado = http.StatusUnauthorized
	respuesta := httptest.NewRecorder()
	peticion := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
	peticion.Header.Set("Authorization", "Bearer ajeno")
	peticion.Header.Set("X-VEC-Perfil", "inyectado")
	peticion.Header.Set("Cookie", "sesion=ajena")
	h.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusUnauthorized || respuesta.Header().Get("Set-Cookie") != "" || len(manejador.peticiones) != 3 {
		t.Fatalf("rechazo nominal alterado: %d, delegaciones=%d", respuesta.Code, len(manejador.peticiones))
	}
	for _, ruta := range vecRoutes() {
		if ruta == "/api/vec/session/start" {
			t.Fatal("inicio nominal anunciado sin activar manejador")
		}
	}
}

func TestSesionNominalConstructorYFalloAuditoria(t *testing.T) {
	servicio := servicioSesionNominalPrueba(t)
	manejador := &manejadorSesionNominalPrueba{}
	var nulo *manejadorSesionNominalPrueba
	for nombre, opciones := range map[string]HandlerOptions{
		"nil tipado":       {ManejadorSesionNominal: nulo},
		"demo":             {ManejadorSesionNominal: manejador, AllowDemoIdentity: true},
		"resolvedor demo":  {ManejadorSesionNominal: manejador, DemoIdentityResolver: resolvedorIdentidadPruebas{}},
		"cabeceras":        {ManejadorSesionNominal: manejador, TrustIdentityHeaders: true},
		"proxy confiado":   {ManejadorSesionNominal: manejador, TrustedProxyCIDRs: []string{"127.0.0.1/32"}},
		"perfil inyectado": {ManejadorSesionNominal: manejador, IdentityRolesHeader: "X-VEC-Perfil"},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := NewHandlerWithOptions(servicio, opciones); !errors.Is(err, ErrSesionNominalInvalida) {
				t.Fatalf("mezcla admitida: %v", err)
			}
		})
	}
	for _, ruta := range []string{"/api/vec/session", "/api/vec/session/start"} {
		for _, coleccion := range []bool{false, true} {
			opciones := HandlerOptions{AutoridadRutasExactas: autoridadRutasExactasPrueba{}}
			if coleccion {
				opciones.RutasColeccion = []RutaColeccion{{Prefijo: ruta, Manejador: manejador}}
			} else {
				opciones.RutasExactas = []RutaExacta{{Ruta: ruta, Manejador: manejador}}
			}
			if _, err := NewHandlerWithOptions(servicio, opciones); !errors.Is(err, ErrRutaExactaInvalida) {
				t.Fatalf("colisión %s admitida: %v", ruta, err)
			}
		}
	}
	manejador.fallo = domain.ErrPermissionDenied
	h, err := NewHandlerWithOptions(servicio, HandlerOptions{ManejadorSesionNominal: manejador})
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/api/vec/%73ession", nil))
	if respuesta.Code != http.StatusServiceUnavailable || len(manejador.peticiones) != 0 ||
		!strings.Contains(respuesta.Body.String(), "auditoria_no_disponible") {
		t.Fatalf("auditoría caída: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	legacy, err := NewHandlerWithOptions(servicio, HandlerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{"/api/vec/session", "/api/vec/session/start"} {
		respuesta := httptest.NewRecorder()
		legacy.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, nil))
		if respuesta.Code != http.StatusUnauthorized {
			t.Fatalf("ruta heredada %s cambió: %d", ruta, respuesta.Code)
		}
	}
}
