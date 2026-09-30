package ajustesreglas

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecpruebas "vec-diputacion-granada/internal/vec/pruebas"
	"vec-diputacion-granada/internal/vec/reglas"
)

type actorPrueba struct {
	actor    vecdomain.ContextoActor
	llamadas int
	err      error
}

func (a *actorPrueba) ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error) {
	a.llamadas++
	return a.actor, a.err
}

type servicioPrueba struct{ publicaciones int }

func (s *servicioPrueba) Consultar(context.Context, vecdomain.ContextoActor, int, *int64) (app.Lectura, error) {
	return app.Lectura{PuedeAjustar: true, Reglas: []reglas.Regla{{
		Clave: reglas.CTPlazoFiscalizacion, Etiqueta: "Fiscalización", Unidad: reglas.UnidadDiasHabiles,
		Cantidad: 10, Computo: reglas.ComputoAdministrativo,
		Atributos: map[string]string{reglas.CampoCantidadUrgente: "5"},
		Edicion: &reglas.Edicion{Campos: []string{reglas.CampoCantidad, reglas.CampoCantidadUrgente, reglas.CampoUnidad},
			OpcionesUnidad: []reglas.Unidad{reglas.UnidadDiasHabiles, reglas.UnidadDiasNaturales}, CantidadMinima: 1, CantidadMaxima: 60},
	}}}, nil
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
		{Ruta, `{}`, "Idempotency-Key", http.StatusBadRequest},
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
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"puede_ajustar":false`) ||
		!strings.Contains(w.Body.String(), `"cantidad_urgente":"5"`) ||
		!strings.Contains(w.Body.String(), `"opciones_unidad"`) ||
		!strings.Contains(w.Body.String(), `"opciones_computo":[]`) ||
		!strings.Contains(w.Body.String(), `"historial":[]`) ||
		strings.Contains(w.Body.String(), `"OpcionesUnidad"`) {
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

func TestResolverActorDistingueCaidaNominalDeDenegacion(t *testing.T) {
	s := &servicioPrueba{}
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"dependencia_caida", app.ErrNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"denegacion", vecdomain.ErrAutorizacionDenegada, http.StatusForbidden, "acceso_denegado"},
		{"error_no_nominal", errors.New("fallo no clasificado"), http.StatusForbidden, "acceso_denegado"},
		{"actor_invalido", nil, http.StatusForbidden, "acceso_denegado"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			a := &actorPrueba{err: c.err}
			h, err := NuevoManejador(a, s)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodGet, Ruta, nil)
			r.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != c.estado || !strings.Contains(w.Body.String(), `"`+c.codigo+`"`) || a.llamadas != 1 {
				t.Fatalf("clasificación del resolver: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
