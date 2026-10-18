package httppersonal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorPreferenciasPrueba struct{ llamadas int }

func (p *preparadorPreferenciasPrueba) PrepararMiBolsa(*http.Request) (mibolsa.Orden, error) {
	p.llamadas++
	return mibolsa.Orden{}, nil
}

type ordenadorPreferenciasPrueba struct {
	llamadas int
	orden    []string
	version  uint64
	clave    string
	err      error
}

type lectorPreferenciasPrueba struct {
	llamadas int
	err      error
}

func (l *lectorPreferenciasPrueba) ConsultarPreferencias(context.Context, mibolsa.Orden) (bolsa.EstadoPreferenciasBolsas, error) {
	l.llamadas++
	return bolsa.EstadoPreferenciasBolsas{Version: 0, OrdenGuardado: []string{}, BolsasActuales: []string{"bolsa:propia"},
		BolsasNuevas: []string{"bolsa:propia"}, BolsasRetiradas: []string{}, RevisionRequerida: true}, l.err
}

func (o *ordenadorPreferenciasPrueba) Ordenar(_ context.Context, _ mibolsa.Orden,
	orden []string, version uint64, clave string,
) (bolsa.ReciboPreferenciasBolsas, error) {
	o.llamadas++
	o.orden, o.version, o.clave = orden, version, clave
	return bolsa.ReciboPreferenciasBolsas{ReciboRef: "recibo:preferencias-bolsas:prueba",
		Version: version + 1, Orden: orden, RegistradoEn: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)}, o.err
}

func peticionPreferenciasPrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPut, RutaPreferenciasBolsas, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestPreferenciasHTTPRechazaDuplicadosAntesDeAutenticar(t *testing.T) {
	p, o := new(preparadorPreferenciasPrueba), new(ordenadorPreferenciasPrueba)
	h, err := NuevoHandlerPreferenciasBolsas(p, o)
	if err != nil {
		t.Fatal(err)
	}
	for _, cuerpo := range []string{
		`{"orden":["bolsa:a"],"orden":["bolsa:b"],"version_esperada":0,"clave_idempotencia":"clave-0001"}`,
		`{"orden":["bolsa:a"],"version_esperada":0,"clave_idempotencia":"clave-0001","candidato_ref":"can_ajeno"}`,
		`{"orden":["bolsa:a"],"version_esperada":null,"clave_idempotencia":"clave-0001"}`,
		`{"orden":["bolsa:a"],"version_esperada":0,"clave_idempotencia":null}`,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPreferenciasPrueba(cuerpo))
		if w.Code != http.StatusBadRequest || p.llamadas != 0 || o.llamadas != 0 {
			t.Fatalf("cuerpo no autorizado pasó frontera: %d %d %d", w.Code, p.llamadas, o.llamadas)
		}
	}
}

func TestPreferenciasHTTPEntregaReciboYConservaErrorDeAutorizacion(t *testing.T) {
	p, o := new(preparadorPreferenciasPrueba), new(ordenadorPreferenciasPrueba)
	h, err := NuevoHandlerPreferenciasBolsas(p, o)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"orden":["bolsa:b","bolsa:a"],"version_esperada":2,"clave_idempotencia":"clave-0002"}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPreferenciasPrueba(cuerpo))
	if w.Code != http.StatusCreated || p.llamadas != 1 || o.llamadas != 1 ||
		o.version != 2 || o.clave != "clave-0002" || len(o.orden) != 2 ||
		!strings.Contains(w.Body.String(), `"version":3`) {
		t.Fatalf("recibo HTTP incorrecto: %d %s", w.Code, w.Body.String())
	}
	o.err = errors.Join(dominiovec.ErrAutorizacionDenegada, bolsa.ErrOrdenPreferenciasInvalido)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPreferenciasPrueba(cuerpo))
	if w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), "orden_invalido") {
		t.Fatalf("denegación filtró validación: %d %s", w.Code, w.Body.String())
	}
}

func TestPreferenciasGETExigeLecturaYNoAceptaSelectoresDeIdentidad(t *testing.T) {
	p, o, l := new(preparadorPreferenciasPrueba), new(ordenadorPreferenciasPrueba), new(lectorPreferenciasPrueba)
	h, err := NuevoHandlerPreferenciasBolsasConLectura(p, o, l)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{RutaPreferenciasBolsas + "?candidato_ref=can_ajeno", RutaPreferenciasBolsas + "?", RutaPreferenciasBolsas + "?bolsa_ref=otra"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusBadRequest || p.llamadas != 0 || l.llamadas != 0 {
			t.Fatalf("selector alcanzó identidad: %d", w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPreferenciasBolsas, nil))
	if w.Code != http.StatusOK || p.llamadas != 1 || l.llamadas != 1 || o.llamadas != 0 ||
		!strings.Contains(w.Body.String(), `"bolsas_nuevas":["bolsa:propia"]`) || strings.Contains(w.Body.String(), "can_") {
		t.Fatalf("GET propio incorrecto: %d %s", w.Code, w.Body.String())
	}
	l.err = dominiovec.ErrAutorizacionDenegada
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPreferenciasBolsas, nil))
	if w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), "bolsa:propia") {
		t.Fatalf("GET revocado filtró estado: %d %s", w.Code, w.Body.String())
	}
	// El montaje con GET sigue utilizando el PUT V3 existente.
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPreferenciasPrueba(`{"orden":["bolsa:propia"],"version_esperada":0,"clave_idempotencia":"clave-replay-1"}`))
	if w.Code != http.StatusCreated || o.llamadas != 1 || l.llamadas != 2 {
		t.Fatalf("PUT cambió tras montar GET: %d", w.Code)
	}
}

func TestPreferenciasNoDisponibleConservaRutaSinResolverIdentidad(t *testing.T) {
	h := NuevoHandlerPreferenciasBolsasNoDisponible()
	for _, metodo := range []string{http.MethodGet, http.MethodPut} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(metodo, RutaPreferenciasBolsas, nil))
		if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "servicio_no_disponible") {
			t.Fatalf("ruta incompleta: %s %d %s", metodo, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, RutaPreferenciasBolsas, nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("método adicional publicado: %d", w.Code)
	}
}
