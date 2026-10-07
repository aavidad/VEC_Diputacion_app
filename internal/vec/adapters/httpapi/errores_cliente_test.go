package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type emisorErroresClientePrueba struct {
	solicitudes []domain.SolicitudIncidenciaTecnica
}

func (e *emisorErroresClientePrueba) Emitir(s domain.SolicitudIncidenciaTecnica) {
	e.solicitudes = append(e.solicitudes, s)
}

const cuerpoErrorClientePrueba = `{"pantalla":"portal_empleado","codigo":"CLIENTE_FALLO_NO_CLASIFICADO","correlacion":"0123456789abcdef0123456789abcdef"}`

func peticionErrorClientePrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "http://vec.local/api/vec/observabilidad/errores-cliente", strings.NewReader(cuerpo))
	r.Header.Set("Origin", "http://vec.local")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestErroresClienteRutaInternaEmiteSoloCatalogoTecnico(t *testing.T) {
	emisor := &emisorErroresClientePrueba{}
	h := newTestHandlerWithOptions(t, HandlerOptions{EmisorIncidenciasTecnicas: emisor})
	r := peticionErrorClientePrueba(cuerpoErrorClientePrueba)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("POST = %d; esperado 204", w.Code)
	}
	if len(emisor.solicitudes) != 1 || emisor.solicitudes[0].Codigo != domain.IncidenciaClienteFalloNoClasificado ||
		emisor.solicitudes[0].Componente != domain.ComponenteIncidenciaPortalWeb ||
		emisor.solicitudes[0].Etapa != domain.EtapaIncidenciaEjecucion {
		t.Fatalf("incidencias = %#v", emisor.solicitudes)
	}
	if w.Body.Len() != 0 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("respuesta no minimizada: cuerpo=%d cache=%q", w.Body.Len(), w.Header().Get("Cache-Control"))
	}
}

func TestErroresClienteRechazaFronterasYJSONNoCerrado(t *testing.T) {
	casos := []struct {
		nombre  string
		cuerpo  string
		cambiar func(*http.Request)
		estado  int
	}{
		{"sin origen", cuerpoErrorClientePrueba, func(r *http.Request) { r.Header.Del("Origin") }, 403},
		{"origen ajeno", cuerpoErrorClientePrueba, func(r *http.Request) { r.Header.Set("Origin", "http://ajeno.local") }, 403},
		{"sitio ajeno", cuerpoErrorClientePrueba, func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, 403},
		{"tipo ajeno", cuerpoErrorClientePrueba, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, 415},
		{"query", cuerpoErrorClientePrueba, func(r *http.Request) { r.URL.RawQuery = "persona=1" }, 400},
		{"texto libre", strings.Replace(cuerpoErrorClientePrueba, `}`, `,"mensaje":"DNI 12345678Z"}`, 1), nil, 400},
		{"clave duplicada", strings.Replace(cuerpoErrorClientePrueba, `"codigo":`, `"codigo":"MODULO_WEB_NO_CARGADO","codigo":`, 1), nil, 400},
		{"codigo ajeno", strings.Replace(cuerpoErrorClientePrueba, "CLIENTE_FALLO_NO_CLASIFICADO", "DNI_12345678Z", 1), nil, 400},
		{"pantalla ajena", strings.Replace(cuerpoErrorClientePrueba, "portal_empleado", "/portal-empleado/persona/123", 1), nil, 400},
		{"correlacion ajena", strings.Replace(cuerpoErrorClientePrueba, "0123456789abcdef0123456789abcdef", "persona:123", 1), nil, 400},
		{"cuerpo grande", strings.Repeat("a", limiteCuerpoErrorCliente+1), nil, 413},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			emisor := &emisorErroresClientePrueba{}
			h := &Handler{emisorIncidencias: emisor}
			r := peticionErrorClientePrueba(tc.cuerpo)
			if tc.cambiar != nil {
				tc.cambiar(r)
			}
			w := httptest.NewRecorder()
			h.atenderErroresCliente(w, r, principalConPermisosExpresosPrueba("vec.session.read"))
			if w.Code != tc.estado || len(emisor.solicitudes) != 0 {
				t.Fatalf("status=%d incidencias=%d; esperado %d, 0", w.Code, len(emisor.solicitudes), tc.estado)
			}
		})
	}
}

func TestErroresClienteRequierePermisoYEmisorReal(t *testing.T) {
	r := peticionErrorClientePrueba(cuerpoErrorClientePrueba)
	emisor := &emisorErroresClientePrueba{}
	for _, caso := range []struct {
		h         *Handler
		principal domain.Principal
		estado    int
	}{
		{&Handler{emisorIncidencias: emisor}, principalConPermisosExpresosPrueba(), 403},
		{&Handler{}, principalConPermisosExpresosPrueba("vec.session.read"), 503},
	} {
		w := httptest.NewRecorder()
		caso.h.atenderErroresCliente(w, r.Clone(r.Context()), caso.principal)
		if w.Code != caso.estado || len(emisor.solicitudes) != 0 {
			t.Fatalf("status=%d incidencias=%d; esperado %d, 0", w.Code, len(emisor.solicitudes), caso.estado)
		}
	}
}

func TestLimiteErroresClienteEsAcotadoYReinicia(t *testing.T) {
	l := &limiteErroresCliente{}
	inicio := time.Date(2026, time.October, 7, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 120; i++ {
		if !l.permitir(inicio) {
			t.Fatalf("rechazado antes del cupo: %d", i)
		}
	}
	if l.permitir(inicio) {
		t.Fatal("aceptó por encima del cupo")
	}
	if !l.permitir(inicio.Add(time.Minute)) {
		t.Fatal("no reinició al minuto")
	}
}
