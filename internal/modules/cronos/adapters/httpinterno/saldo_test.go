package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type resolverSaldoPrueba struct {
	llamadas int
	orden    ports.OrdenConsultaSaldo
	err      error
}

func (r *resolverSaldoPrueba) ResolverConsultaSaldoPropio(_ *http.Request, _ ports.PeriodoSaldo, _, _ string) (ports.OrdenConsultaSaldo, error) {
	r.llamadas++
	return r.orden, r.err
}

type casoSaldoPrueba struct {
	llamadas int
	err      error
}

func (c *casoSaldoPrueba) ConsultarSaldo(_ context.Context, _ ports.OrdenConsultaSaldo, periodo ports.PeriodoSaldo, desde, hasta string) (ports.ConsultaSaldo, error) {
	c.llamadas++
	return ports.ConsultaSaldo{Periodo: ports.PeriodoConsultaSaldo{Tipo: periodo, Desde: desde, Hasta: hasta}, Resumen: ports.ResumenConsultaSaldo{Estado: ports.EstadoSaldoNoDisponible}}, c.err
}

func ordenSaldoPrueba(t *testing.T) ports.OrdenConsultaSaldo {
	t.Helper()
	ref := func(prefijo string) string { return prefijo + strings.Repeat("a", 22) }
	ahora := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: ref("cta_"), Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: ref("vca_"), VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, PersonaRef: ref("per_"), PersonaVersion: 1, PerfilActivoRef: ref("prf_"), PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenConsultaSaldo(actor)
	if err != nil {
		t.Fatal(err)
	}
	return orden
}

func TestSaldoPropioDeniegaIdentidadClienteYOrdenInvalida(t *testing.T) {
	caso := &casoSaldoPrueba{}
	resolver := &resolverSaldoPrueba{}
	h, err := NuevoManejadorSaldoPropio(caso, resolver)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaConsultarSaldoPropio+"?periodo=mes", nil)
	r.Header.Set("X-Empleado", "emp_ajeno")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || caso.llamadas != 0 || resolver.llamadas != 1 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("lectura sin concesion: %d", w.Code)
	}
	resolver.orden = ordenSaldoPrueba(t)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || caso.llamadas != 1 || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("lectura acreditada: %d", w.Code)
	}
}

func TestSaldoPropioDistingueIdentidadDenegacionYDependencia(t *testing.T) {
	for _, tc := range []struct {
		nombre string
		err    error
		estado int
	}{
		{"identidad", ErrAutenticacionCronosRequerida, http.StatusUnauthorized},
		{"denegacion", ErrAccesoCronosDenegado, http.StatusForbidden},
		{"dependencia", ports.ErrDependenciaNoDisponible, http.StatusServiceUnavailable},
		{"fallo privado", errors.New("detalle_sql_privado"), http.StatusServiceUnavailable},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			resolver := &resolverSaldoPrueba{err: tc.err}
			caso := &casoSaldoPrueba{}
			h, _ := NuevoManejadorSaldoPropio(caso, resolver)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaConsultarSaldoPropio+"?periodo=mes", nil))
			if w.Code != tc.estado || caso.llamadas != 0 || strings.Contains(w.Body.String(), "privado") {
				t.Fatalf("resolver expuesto: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestSaldoPropioDistingueErrorDelCasoDeUso(t *testing.T) {
	for _, tc := range []struct {
		err    error
		estado int
	}{
		{ErrAccesoCronosDenegado, http.StatusForbidden},
		{ports.ErrDependenciaNoDisponible, http.StatusServiceUnavailable},
	} {
		resolver := &resolverSaldoPrueba{orden: ordenSaldoPrueba(t)}
		caso := &casoSaldoPrueba{err: tc.err}
		h, _ := NuevoManejadorSaldoPropio(caso, resolver)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaConsultarSaldoPropio+"?periodo=mes", nil))
		if w.Code != tc.estado || caso.llamadas != 1 {
			t.Fatalf("caso de uso: %d", w.Code)
		}
	}
}

func TestSaldoPropioRechazaParametrosNoCanonicosAntesDeResolver(t *testing.T) {
	for _, ruta := range []string{
		RutaConsultarSaldoPropio,
		RutaConsultarSaldoPropio + "?periodo=mes&periodo=hoy",
		RutaConsultarSaldoPropio + "?periodo=rango&desde=2026-02-30&hasta=2026-03-01",
		RutaConsultarSaldoPropio + "?periodo=rango&desde=2026-09-25&hasta=2026-09-24",
		RutaConsultarSaldoPropio + "?periodo=mes&empleado_ref=ajeno",
		RutaConsultarSaldoPropio + "?periodo=hoy&desde=2026-09-24",
	} {
		caso := &casoSaldoPrueba{}
		resolver := &resolverSaldoPrueba{orden: ordenSaldoPrueba(t)}
		h, _ := NuevoManejadorSaldoPropio(caso, resolver)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusBadRequest || resolver.llamadas != 0 || caso.llamadas != 0 {
			t.Fatalf("entrada indebida %q: %d", ruta, w.Code)
		}
	}
}
