package ajustesreglas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecpruebas "vec-diputacion-granada/internal/vec/pruebas"
)

type actorPrueba struct {
	actor    vecdomain.ContextoActor
	llamadas int
}

func (a *actorPrueba) ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error) {
	a.llamadas++
	return a.actor, nil
}

type servicioPrueba struct{ publicaciones int }

func (s *servicioPrueba) Consultar(context.Context, vecdomain.ContextoActor, int, *int64) (app.Lectura, error) {
	return app.Lectura{PuedeAjustar: true}, nil
}
func (s *servicioPrueba) Publicar(context.Context, vecdomain.ContextoActor, app.Solicitud) (app.Resultado, error) {
	s.publicaciones++
	return app.Resultado{}, app.ErrEntradaInvalida
}
func (s *servicioPrueba) Motivos() []app.Motivo { return nil }

func TestRutaAjustesRechazaCuerpoDuplicadoYCabecerasDeIdentidad(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	actor, _, err := vecpruebas.NuevoContextoYVinculo(ahora, "per_0123456789abcdef0123456789abcdef",
		"prf_0123456789abcdef0123456789abcdef", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	a := &actorPrueba{actor: actor}
	s := &servicioPrueba{}
	h, err := NuevoManejador(a, s)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		ruta, cuerpo, cabecera string
		estado                 int
	}{
		{Ruta + "/", "", "", http.StatusNotFound},
		{Ruta, `{"version_esperada":0,"version_esperada":1}`, "", http.StatusBadRequest},
		{Ruta, `{}`, "X-VEC-Rol", http.StatusBadRequest},
		{Ruta, `{"cambios":[{"nuevo":"7","nuevo":"8"}]}`, "", http.StatusBadRequest},
	}
	for _, c := range casos {
		r := httptest.NewRequest(http.MethodPost, c.ruta, strings.NewReader(c.cuerpo))
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Content-Type", "application/json")
		if c.cabecera != "" {
			r.Header.Set(c.cabecera, "rrhh")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.estado || s.publicaciones != 0 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("ruta %s: estado %d, publicaciones %d", c.ruta, w.Code, s.publicaciones)
		}
	}
	r := httptest.NewRequest(http.MethodPost, Ruta, strings.NewReader(`{}`))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://ajeno.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || s.publicaciones != 0 {
		t.Fatalf("origen cruzado llegó a escritura: %d", w.Code)
	}
	if a.llamadas != 2 {
		t.Fatalf("frontera invocada para ruta o cabecera inválida: %d", a.llamadas)
	}
}

func TestJSONSinDuplicadosCompruebaObjetosAnidados(t *testing.T) {
	if !jsonSinDuplicados([]byte(`{"cambios":[{"regla_clave":"c03","campo":"cantidad","nuevo":"7"}]}`)) ||
		jsonSinDuplicados([]byte(`{"cambios":[{"nuevo":"7","nuevo":"8"}]}`)) ||
		jsonSinDuplicados([]byte(`{"cambios":[{}],"cambios":[]}`)) {
		t.Fatal("detector de claves duplicadas")
	}
}

func TestSoloLecturaNoAnunciaNiEjecutaGuardado(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	actor, _, err := vecpruebas.NuevoContextoYVinculo(ahora, "per_0123456789abcdef0123456789abcdef",
		"prf_0123456789abcdef0123456789abcdef", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	a := &actorPrueba{actor: actor}
	s := &servicioPrueba{}
	h, err := NuevoManejadorSoloLectura(a, s)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, Ruta, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"puede_ajustar":false`) {
		t.Fatalf("GET habilitó escritura: %d %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodPost, Ruta, strings.NewReader(`{}`))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || s.publicaciones != 0 || a.llamadas != 1 {
		t.Fatalf("POST alcanzó operación en solo lectura: %d %d %d", w.Code, s.publicaciones, a.llamadas)
	}
}
