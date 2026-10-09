package plantillascatalogo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	resultado     *app.ResultadoCambio
	err           error
}

func (s *servicioPrueba) Consultar(context.Context, vecdomain.ContextoActor) (app.Lectura, error) {
	s.lecturas++
	return app.Lectura{}, nil
}
func (s *servicioPrueba) Editar(context.Context, vecdomain.ContextoActor, app.SolicitudEditar) (app.ResultadoCambio, error) {
	s.ediciones++
	if s.resultado != nil {
		return *s.resultado, s.err
	}
	return app.ResultadoCambio{}, app.ErrEntradaInvalida
}
func (s *servicioPrueba) Publicar(context.Context, vecdomain.ContextoActor, app.SolicitudPublicar) (app.ResultadoCambio, error) {
	s.publicaciones++
	if s.resultado != nil {
		return *s.resultado, s.err
	}
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

func TestHTTPEstadoYVinculoDelRecibo(t *testing.T) {
	clave := "11111111-1111-4111-8111-111111111111"
	z := app.ResultadoCambio{Recibo: app.Recibo{
		ReciboRef: "recibo:prueba", ClaveIdempotencia: clave, Operacion: "editar",
		Version: 2, Revision: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64),
		RegistradoEn: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), EstadoReplay: "registrado",
	}}
	s := &servicioPrueba{resultado: &z}
	h, _ := NuevoManejador(&resolverPrueba{actor: actorPrueba(t)}, s)
	peticion := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, RutaEntradas, strings.NewReader(`{"clave_idempotencia":"`+clave+`"}`))
		r.Header.Set("Content-Type", "application/json")
		return r
	}
	for _, caso := range []struct {
		estado string
		codigo int
	}{{"registrado", 201}, {"replay", 200}, {"desconocido", 503}} {
		z.Recibo.EstadoReplay = caso.estado
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion())
		if w.Code != caso.codigo {
			t.Fatalf("%s: HTTP %d", caso.estado, w.Code)
		}
		if caso.codigo == 503 {
			continue
		}
		var cuerpo struct {
			Recibo app.Recibo `json:"recibo"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil || cuerpo.Recibo != z.Recibo {
			t.Fatalf("%s: recibo divergente: %+v, %v", caso.estado, cuerpo.Recibo, err)
		}
	}
	s.err = app.ErrConflicto
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion())
	if w.Code != 409 || strings.Contains(w.Body.String(), "recibo_ref") {
		t.Fatalf("conflicto respondió con recibo: %d %s", w.Code, w.Body.String())
	}
}

// Detrás del proxy de cidonia el Host llega reescrito a "localhost" y el
// navegador envía el Origin público: el propio portal no puede quedar fuera.
// Lo admitido llega al servicio (que en la prueba responde entrada inválida);
// lo ajeno se corta en 403 sin tocarlo.
func TestHTTPAdmiteMismoOrigenTrasProxyYRechazaAjeno(t *testing.T) {
	casos := []struct {
		origenes, sitios []string
		codigo, llegan   int
	}{
		{[]string{"https://vec.example.org"}, []string{"same-origin"}, http.StatusBadRequest, 1},
		{[]string{"https://ajeno.example"}, []string{"cross-site"}, http.StatusForbidden, 0},
		{[]string{"https://ajeno.example"}, []string{"same-site"}, http.StatusForbidden, 0},
		{[]string{"https://ajeno.example"}, nil, http.StatusForbidden, 0},
		{[]string{"https://localhost"}, nil, http.StatusBadRequest, 1},
		{nil, []string{"same-origin", "cross-site"}, http.StatusForbidden, 0},
		{[]string{"https://localhost", "https://ajeno.example"}, []string{"same-origin"}, http.StatusForbidden, 0},
	}
	for _, c := range casos {
		s := &servicioPrueba{}
		h, _ := NuevoManejador(&resolverPrueba{actor: actorPrueba(t)}, s)
		r := httptest.NewRequest(http.MethodPost, RutaEntradas, strings.NewReader(`{"clave_idempotencia":"11111111-1111-4111-8111-111111111111"}`))
		r.Host = "localhost"
		r.Header.Set("Content-Type", "application/json")
		for _, o := range c.origenes {
			r.Header.Add("Origin", o)
		}
		for _, v := range c.sitios {
			r.Header.Add("Sec-Fetch-Site", v)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.codigo || s.ediciones != c.llegan {
			t.Fatalf("%v %v: HTTP %d, ediciones %d", c.origenes, c.sitios, w.Code, s.ediciones)
		}
	}
}
