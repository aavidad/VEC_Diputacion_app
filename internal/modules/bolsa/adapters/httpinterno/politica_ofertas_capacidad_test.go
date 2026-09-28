package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorCapacidadPoliticaPrueba struct {
	bolsa    string
	llamadas int
	permiso  bool
	err      error
}

func (p *preparadorCapacidadPoliticaPrueba) ComprobarCapacidadPublicarPoliticaOfertas(_ context.Context, bolsa string) (bool, error) {
	p.llamadas++
	p.bolsa = bolsa
	return p.permiso, p.err
}

func peticionCapacidadPoliticaPrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaCapacidadPoliticaOfertas, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	return r
}

func TestCapacidadPoliticaOfertasBooleanoSinMaterial(t *testing.T) {
	for _, tc := range []struct {
		nombre, esperado string
		permiso          bool
	}{
		{nombre: "permitida", permiso: true, esperado: `{"puede_publicar":true}` + "\n"},
		{nombre: "denegada", permiso: false, esperado: `{"puede_publicar":false}` + "\n"},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			p := &preparadorCapacidadPoliticaPrueba{permiso: tc.permiso}
			h, err := NuevoHandlerCapacidadPoliticaOfertas(p)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionCapacidadPoliticaPrueba(`{"bolsa_ref":"bolsa:prueba"}`))
			if w.Code != http.StatusOK || w.Body.String() != tc.esperado || p.llamadas != 1 || p.bolsa != "bolsa:prueba" {
				t.Fatalf("estado=%d cuerpo=%q llamadas=%d bolsa=%q", w.Code, w.Body.String(), p.llamadas, p.bolsa)
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Set-Cookie") != "" {
				t.Fatalf("cabeceras de preflight inseguras: %#v", w.Header())
			}
		})
	}
}

func TestCapacidadPoliticaOfertasEntradaEstrictaAntesDelPDP(t *testing.T) {
	casos := []struct {
		nombre, cuerpo string
		cambiar        func(*http.Request)
		estado         int
	}{
		{nombre: "campo_extra", cuerpo: `{"bolsa_ref":"bolsa:prueba","actor_ref":"otro"}`, estado: 400},
		{nombre: "campo_duplicado", cuerpo: `{"bolsa_ref":"bolsa:prueba","bolsa_ref":"otra"}`, estado: 400},
		{nombre: "sin_bolsa", cuerpo: `{}`, estado: 400},
		{nombre: "referencia_vacia", cuerpo: `{"bolsa_ref":" "}`, estado: 400},
		{nombre: "lista", cuerpo: `["bolsa:prueba"]`, estado: 400},
		{nombre: "json_adicional", cuerpo: `{"bolsa_ref":"bolsa:prueba"}{}`, estado: 400},
		{nombre: "query", cuerpo: `{"bolsa_ref":"bolsa:prueba"}`, cambiar: func(r *http.Request) { r.URL.RawQuery = "bolsa_ref=otra" }, estado: 400},
		{nombre: "cookie", cuerpo: `{"bolsa_ref":"bolsa:prueba"}`, cambiar: func(r *http.Request) { r.Header.Set("Cookie", "sesion=libre") }, estado: 400},
		{nombre: "autorizacion_libre", cuerpo: `{"bolsa_ref":"bolsa:prueba"}`, cambiar: func(r *http.Request) { r.Header.Set("Authorization", "Bearer libre") }, estado: 400},
		{nombre: "identidad_heredada", cuerpo: `{"bolsa_ref":"bolsa:prueba"}`, cambiar: func(r *http.Request) { r.Header.Set("X-Vec-Actor", "otro") }, estado: 400},
		{nombre: "sin_accept", cuerpo: `{"bolsa_ref":"bolsa:prueba"}`, cambiar: func(r *http.Request) { r.Header.Del("Accept") }, estado: 400},
		{nombre: "metodo_get", cuerpo: `{"bolsa_ref":"bolsa:prueba"}`, cambiar: func(r *http.Request) { r.Method = http.MethodGet }, estado: 405},
		{nombre: "ruta_escapada", cuerpo: `{"bolsa_ref":"bolsa:prueba"}`, cambiar: func(r *http.Request) { r.URL.RawPath = "/api/vec/bolsa/politica-ofertas/%63apacidad" }, estado: 404},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			p := &preparadorCapacidadPoliticaPrueba{permiso: true}
			h, err := NuevoHandlerCapacidadPoliticaOfertas(p)
			if err != nil {
				t.Fatal(err)
			}
			r := peticionCapacidadPoliticaPrueba(tc.cuerpo)
			if tc.cambiar != nil {
				tc.cambiar(r)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.estado || p.llamadas != 0 || strings.Contains(w.Body.String(), "puede_publicar") {
				t.Fatalf("estado=%d cuerpo=%q llamadas=%d", w.Code, w.Body.String(), p.llamadas)
			}
		})
	}
}

func TestCapacidadPoliticaOfertasErroresNoConceden(t *testing.T) {
	for _, tc := range []struct {
		err    error
		estado int
	}{
		{dominiovec.ErrAutorizacionDenegada, http.StatusForbidden},
		{errors.New("dependencia indisponible"), http.StatusServiceUnavailable},
	} {
		p := &preparadorCapacidadPoliticaPrueba{permiso: true, err: tc.err}
		h, err := NuevoHandlerCapacidadPoliticaOfertas(p)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionCapacidadPoliticaPrueba(`{"bolsa_ref":"bolsa:prueba"}`))
		if w.Code != tc.estado || strings.Contains(w.Body.String(), "puede_publicar") || p.llamadas != 1 {
			t.Fatalf("estado=%d cuerpo=%q llamadas=%d", w.Code, w.Body.String(), p.llamadas)
		}
	}
}

func TestCapacidadPoliticaOfertasSinPreparador(t *testing.T) {
	if h, err := NuevoHandlerCapacidadPoliticaOfertas(nil); h != nil || err == nil {
		t.Fatalf("sin preparador: handler=%v error=%v", h, err)
	}
}
