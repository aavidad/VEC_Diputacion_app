package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/aspirantes/application"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
)

type servicioPrueba struct {
	err       error
	replay    bool
	llamadas  int
	peticion  application.PeticionFicha
	operacion string
}

func (s *servicioPrueba) Consultar(context.Context, ports.OrdenFicha) (application.VistaFichaPropia, error) {
	s.llamadas++
	return application.VistaFichaPropia{Estado: application.EstadoActiva, Version: 3, Contacto: map[string]string{"telefono": "958123456"}}, s.err
}

func (s *servicioPrueba) mutar(op string, p application.PeticionFicha) (ports.ReciboFicha, error) {
	s.llamadas++
	s.operacion = op
	s.peticion = p
	s.peticion.Campos = map[string]string{}
	for k, v := range p.Campos {
		s.peticion.Campos[k] = v
	}
	return ports.ReciboFicha{ReciboRef: "asprec_" + strings.Repeat("a", 32), Version: p.VersionEsperada + 1, FechaUTC: time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC), Replay: s.replay}, s.err
}
func (s *servicioPrueba) Alta(_ context.Context, _ ports.OrdenFicha, p application.PeticionFicha) (ports.ReciboFicha, error) {
	return s.mutar("alta", p)
}
func (s *servicioPrueba) Rectificar(_ context.Context, _ ports.OrdenFicha, p application.PeticionFicha) (ports.ReciboFicha, error) {
	return s.mutar("rectificar", p)
}

type ordenPrueba struct{ err error }

func (o ordenPrueba) ResolverOrdenFicha(context.Context) (ports.OrdenFicha, error) {
	return ports.OrdenFicha{}, o.err
}

type auditorPrueba struct {
	anotadas []int
	err      error
}

func (a *auditorPrueba) AuditarDenegacion(_ context.Context, estado int) error {
	a.anotadas = append(a.anotadas, estado)
	return a.err
}

func peticion(t *testing.T, m *Manejador, metodo, ruta, cuerpo string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if cuerpo == "" {
		r = httptest.NewRequest(metodo, ruta, nil)
	} else {
		r = httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	return w
}

func codigo(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var c struct {
		Error struct{ Codigo, ClaveI18n string } `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &c)
	return c.Error.Codigo
}

func nuevo(t *testing.T, s *servicioPrueba, o ordenPrueba, a *auditorPrueba) *Manejador {
	t.Helper()
	m, err := NuevoManejador(s, o, a)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestConsultarYCabeceras(t *testing.T) {
	s := &servicioPrueba{}
	w := peticion(t, nuevo(t, s, ordenPrueba{}, &auditorPrueba{}), http.MethodGet, RutaMiFicha, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"telefono":"958123456"`) ||
		w.Header().Get("Cache-Control") != "private, no-store, max-age=0" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("%d %s %v", w.Code, w.Body, w.Header())
	}
	for _, ruta := range []string{RutaMiFicha + "?persona=per_x", RutaMiFicha + "/", "/api/vec/aspirantes/area-personal/mi%2Dficha"} {
		if w := peticion(t, nuevo(t, s, ordenPrueba{}, &auditorPrueba{}), http.MethodGet, ruta, ""); w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d", ruta, w.Code)
		}
	}
	if w := peticion(t, nuevo(t, s, ordenPrueba{}, &auditorPrueba{}), http.MethodPut, RutaMiFicha, ""); w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("PUT %d", w.Code)
	}
}

func TestAltaYRectificarCuerposExactos(t *testing.T) {
	s := &servicioPrueba{}
	m := nuevo(t, s, ordenPrueba{}, &auditorPrueba{})
	w := peticion(t, m, http.MethodPost, RutaMiFicha, `{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{"telefono":"958 12 34 56"}}`)
	if w.Code != http.StatusCreated || s.operacion != "alta" || s.peticion.Campos["telefono"] != "958 12 34 56" || !strings.Contains(w.Body.String(), `"fecha_utc":"2026-09-29T10:00:00.000000Z"`) {
		t.Fatalf("alta %d %s", w.Code, w.Body)
	}
	s.replay = true
	w = peticion(t, m, http.MethodPost, RutaMiFicha, `{"operacion":"rectificar","clave_operacion":"rect-0000000000000001","version_esperada":2,"motivo":"cambio_de_dato","campos":{"telefono":""}}`)
	if w.Code != http.StatusOK || s.operacion != "rectificar" || s.peticion.Motivo != "cambio_de_dato" {
		t.Fatalf("rectificar repetido %d %s", w.Code, w.Body)
	}
	antes := s.llamadas
	malos := []string{
		`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{},"persona_ref":"per_otra"}`,
		`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"motivo":"dato_nuevo","campos":{}}`,
		`{"operacion":"rectificar","clave_operacion":"rect-0000000000000001","version_esperada":2,"campos":{}}`,
		`{"operacion":"alta","operacion":"rectificar","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{}}`,
		`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{"telefono":"1","telefono":"2"}}`,
		`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":null}`,
		`{"operacion":"borrar","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{}}`,
		`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{"telefono":{"a":1}}}`,
		`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":-1,"campos":{}}`,
		`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{}} {}`,
		`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{"domicilio":"` + strings.Repeat("a", 5000) + `"}}`,
	}
	for i, cuerpo := range malos {
		if w := peticion(t, m, http.MethodPost, RutaMiFicha, cuerpo); w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("caso %d: %d", i, w.Code)
		}
	}
	r := httptest.NewRequest(http.MethodPost, RutaMiFicha, strings.NewReader(`{"operacion":"alta","clave_operacion":"alta-0000000000000001","version_esperada":0,"campos":{}}`))
	r.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity || s.llamadas != antes {
		t.Fatal("cuerpos inválidos no llegan al servicio")
	}
}

func TestErroresYAuditoria(t *testing.T) {
	casos := []struct {
		err    error
		estado int
		codigo string
	}{
		{ports.ErrNoAutenticado, 401, "no_autenticado"}, {ports.ErrProhibido, 403, "prohibido"}, {ports.ErrSinFicha, 404, "sin_ficha"},
		{ports.ErrFichaExistente, 409, "ficha_existente"}, {ports.ErrConflicto, 409, "conflicto"}, {ports.ErrInvalida, 422, "peticion_invalida"},
		{errors.New("pq: detalle"), 503, "no_disponible"},
	}
	for _, c := range casos {
		a := &auditorPrueba{}
		w := peticion(t, nuevo(t, &servicioPrueba{err: c.err}, ordenPrueba{}, a), http.MethodGet, RutaMiFicha, "")
		if w.Code != c.estado || codigo(t, w) != c.codigo || strings.Contains(w.Body.String(), "pq") {
			t.Fatalf("%v: %d %s", c.err, w.Code, w.Body)
		}
		if (c.estado == 401 || c.estado == 403) != (len(a.anotadas) == 1) {
			t.Fatalf("%v: auditoría %v", c.err, a.anotadas)
		}
	}
	a := &auditorPrueba{err: errors.New("registro caído")}
	if w := peticion(t, nuevo(t, &servicioPrueba{}, ordenPrueba{err: ports.ErrNoAutenticado}, a), http.MethodGet, RutaMiFicha, ""); w.Code != 503 {
		t.Fatalf("denegación sin rastro: %d", w.Code)
	}
	if _, err := NuevoManejador(nil, ordenPrueba{}, &auditorPrueba{}); err == nil {
		t.Fatal("sin servicio")
	}
}
