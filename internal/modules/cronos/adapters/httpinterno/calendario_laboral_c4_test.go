package httpinterno

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
)

func TestCalendarioC4HTTPRechazaFormaAntesDeIdentidad(t *testing.T) {
	resolver := &resolverSaldoPrueba{}
	m := &AdaptadorCalendarioLaboralC4{resolver: resolver, consultar: func(*http.Request, application.ConsultaCalendarioLaboralC4) (application.ResultadoCalendarioLaboralC4, error) {
		t.Fatal("consulta no permitida")
		return application.ResultadoCalendarioLaboralC4{}, nil
	}}
	for _, tc := range []struct {
		metodo, ruta string
		esperado     int
	}{
		{http.MethodPost, RutaCalendarioLaboralPropioC4 + "?fecha=2026-06-15", http.StatusMethodNotAllowed},
		{http.MethodGet, RutaCalendarioLaboralPropioC4 + "?fecha=2026-02-30", http.StatusBadRequest},
		{http.MethodGet, RutaCalendarioLaboralPropioC4 + "?fecha=2026-06-15&empleado=otro", http.StatusBadRequest},
		{http.MethodGet, RutaCalendarioLaboralPropioC4 + "?fecha=2026-06-15&fecha=2026-06-16", http.StatusBadRequest},
		{http.MethodGet, "/api/interna/cronos/calendario/otro?fecha=2026-06-15", http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(tc.metodo, tc.ruta, nil))
		if w.Code != tc.esperado || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s: %d", tc.metodo, tc.ruta, w.Code)
		}
	}
	if resolver.llamadas != 0 {
		t.Fatalf("identidad invocada %d veces", resolver.llamadas)
	}
}

func TestCalendarioC4HTTPEntregaSoloFechaConActorServidor(t *testing.T) {
	resolver := &resolverSaldoPrueba{orden: ordenSaldoPrueba(t)}
	m := &AdaptadorCalendarioLaboralC4{resolver: resolver, consultar: func(_ *http.Request, q application.ConsultaCalendarioLaboralC4) (application.ResultadoCalendarioLaboralC4, error) {
		if q.Fecha != "2026-06-15" {
			t.Fatalf("fecha=%q", q.Fecha)
		}
		return application.ResultadoCalendarioLaboralC4{Fecha: q.Fecha, Estado: application.EstadoCalendarioC4NoDeterminable}, nil
	}}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCalendarioLaboralPropioC4+"?fecha=2026-06-15", nil))
	if w.Code != http.StatusOK || resolver.llamadas != 1 || !strings.Contains(w.Body.String(), `"no_determinable"`) {
		t.Fatalf("estado=%d, identidad=%d, cuerpo=%s", w.Code, resolver.llamadas, w.Body.String())
	}
	m.consultar = func(*http.Request, application.ConsultaCalendarioLaboralC4) (application.ResultadoCalendarioLaboralC4, error) {
		return application.ResultadoCalendarioLaboralC4{}, errors.New("fuente caída")
	}
	w = httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCalendarioLaboralPropioC4+"?fecha=2026-06-15", nil))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "fuente caída") {
		t.Fatalf("error externo expuesto: %d %s", w.Code, w.Body.String())
	}
	m.consultar = func(*http.Request, application.ConsultaCalendarioLaboralC4) (application.ResultadoCalendarioLaboralC4, error) {
		return application.ResultadoCalendarioLaboralC4{}, personaldomain.ErrUnidadTrabajoCronosDenegada
	}
	w = httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCalendarioLaboralPropioC4+"?fecha=2026-06-15", nil))
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), `"acceso_denegado"`) {
		t.Fatalf("denegación=%d %s", w.Code, w.Body.String())
	}
}
