package plantillascatalogo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecpruebas "vec-diputacion-granada/internal/vec/pruebas"
)

type resolverPrueba struct {
	actor    vecdomain.ContextoActor
	llamadas int
}

func (r *resolverPrueba) ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error) {
	r.llamadas++
	return r.actor, nil
}

type servicioPrueba struct {
	lecturas      int
	ediciones     int
	publicaciones int
}

func (s *servicioPrueba) Consultar(context.Context, vecdomain.ContextoActor) (app.Lectura, error) {
	s.lecturas++
	return app.Lectura{}, nil
}
func (s *servicioPrueba) Editar(context.Context, vecdomain.ContextoActor, app.SolicitudEditar) (app.ResultadoCambio, error) {
	s.ediciones++
	return app.ResultadoCambio{}, app.ErrEntradaInvalida
}
func (s *servicioPrueba) Publicar(context.Context, vecdomain.ContextoActor, app.SolicitudPublicar) (app.ResultadoCambio, error) {
	s.publicaciones++
	return app.ResultadoCambio{}, app.ErrEntradaInvalida
}
func actorPrueba(t *testing.T) vecdomain.ContextoActor {
	t.Helper()
	h := sha256.Sum256([]byte("plantillas-http"))
	s := hex.EncodeToString(h[:16])
	a, _, err := vecpruebas.NuevoContextoYVinculo(time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), "per_"+s, "prf_"+s, vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestHTTPRechazaJSONAmbiguoAntesDeResolverIdentidad(t *testing.T) {
	a := &resolverPrueba{actor: actorPrueba(t)}
	s := &servicioPrueba{}
	h, _ := NuevoManejador(a, s)
	r := httptest.NewRequest(http.MethodPost, RutaEntradas, strings.NewReader(`{"clave_idempotencia":"a","clave_idempotencia":"b"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 || a.llamadas != 0 || s.ediciones != 0 {
		t.Fatalf("JSON ambiguo llegó a negocio: %d, %d, %d", w.Code, a.llamadas, s.ediciones)
	}
}

func TestHTTPNoAdmiteConsultaNiOrigenCruzado(t *testing.T) {
	a := &resolverPrueba{actor: actorPrueba(t)}
	s := &servicioPrueba{}
	h, _ := NuevoManejador(a, s)
	for _, caso := range []struct {
		ruta, origen string
		codigo       int
	}{{RutaCatalogo + "?actor=otro", "", 404}, {RutaCatalogo, "https://otro.example", 403}} {
		r := httptest.NewRequest(http.MethodGet, caso.ruta, nil)
		if caso.origen != "" {
			r.Header.Set("Origin", caso.origen)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.codigo {
			t.Fatalf("%s: %d", caso.ruta, w.Code)
		}
	}
	if a.llamadas != 0 || s.lecturas != 0 {
		t.Fatal("frontera negativa consultó catálogo")
	}
}

func TestHTTPConsultaAutenticadaSinCacheNiCookie(t *testing.T) {
	a := &resolverPrueba{actor: actorPrueba(t)}
	s := &servicioPrueba{}
	h, _ := NuevoManejador(a, s)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCatalogo, nil))
	if w.Code != 200 || s.lecturas != 1 || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" || !strings.Contains(w.Body.String(), `"borrador":null`) {
		t.Fatalf("consulta: %d, %q", w.Code, w.Body.String())
	}
}
