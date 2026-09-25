package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type gestorSancionesHTTPPrueba struct {
	registro ports.RegistroSancion
	recurso  ports.RegistroRecursoSancion
	vista    ports.VistaSancionesParticipacion
	err      error
	ultima   *ports.SolicitudRegistrarSancion
	ultimoR  *ports.SolicitudRegistrarRecursoSancion
}

func (g *gestorSancionesHTTPPrueba) Registrar(_ context.Context, q ports.SolicitudRegistrarSancion) (ports.RegistroSancion, error) {
	g.ultima = &q
	return g.registro, g.err
}
func (g *gestorSancionesHTTPPrueba) RegistrarRecurso(_ context.Context, q ports.SolicitudRegistrarRecursoSancion) (ports.RegistroRecursoSancion, error) {
	g.ultimoR = &q
	return g.recurso, g.err
}
func (g *gestorSancionesHTTPPrueba) Consultar(context.Context, ports.SolicitudCambiarSituacionParticipacion) (ports.VistaSancionesParticipacion, error) {
	return g.vista, g.err
}

func peticionSancion(metodo, ruta, cuerpo string) *http.Request {
	var r *http.Request
	if cuerpo == "" {
		r = httptest.NewRequest(metodo, ruta, nil)
	} else {
		r = httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "sancion-01")
	}
	r.Header.Set("Accept", "application/json")
	return r
}

const rutaSancionesPrueba = RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/sanciones"

func TestSancionesRutas(t *testing.T) {
	casos := map[string][3]string{
		rutaSancionesPrueba: {"bolsa:01", "participacion:01", ""},
		rutaSancionesPrueba + "/sancion:" + strings.Repeat("a", 64) + "/recursos": {"bolsa:01", "participacion:01", "sancion:" + strings.Repeat("a", 64)},
	}
	for ruta, esperado := range casos {
		b, p, s, ok := ReferenciasRutaSancionesParticipacion(httptest.NewRequest(http.MethodGet, ruta, nil))
		if !ok || b != esperado[0] || p != esperado[1] || s != esperado[2] {
			t.Fatalf("%s: %q %q %q %v", ruta, b, p, s, ok)
		}
	}
	for _, ruta := range []string{rutaSancionesPrueba + "/", rutaSancionesPrueba + "/x", rutaSancionesPrueba + "/x/otra", RutaBolsasGestion + "/b/candidatos/p/operaciones", rutaSancionesPrueba + "?a=1"} {
		if _, _, _, ok := ReferenciasRutaSancionesParticipacion(httptest.NewRequest(http.MethodGet, ruta, nil)); ok {
			t.Fatalf("ruta aceptada: %s", ruta)
		}
	}
}

func TestSancionesRegistroContratoHTTP(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	cuerpo := `{"consecuencia":"b24.sancion.suspension","causa":"No se presentó","fecha_notificacion":"2026-09-20","resuelta_por":"persona:jefatura","resolucion":{"referencia":"registro:2026/1","sha256":"` + strings.Repeat("a", 64) + `"}}`
	for _, caso := range []struct {
		reutilizada bool
		estado      int
	}{{false, 201}, {true, 200}} {
		g := &gestorSancionesHTTPPrueba{registro: ports.RegistroSancion{Reutilizada: caso.reutilizada, SancionRef: "sancion:x", ReciboRef: "recibo:situacion:x", Situacion: "no_disponible", Desde: &ahora}}
		h, _ := NuevoHandlerSancionesParticipacion(preparadorSituacionHTTPPrueba{}, g)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionSancion(http.MethodPost, rutaSancionesPrueba, cuerpo))
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"sancion_ref":"sancion:x"`) || g.ultima == nil || g.ultima.Datos.Consecuencia != "b24.sancion.suspension" {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	for nombre, malo := range map[string]string{
		"campo extra":      strings.Replace(cuerpo, `"causa"`, `"extra":1,"causa"`, 1),
		"huella corta":     strings.Replace(cuerpo, strings.Repeat("a", 64), "abc", 1),
		"fecha mal":        strings.Replace(cuerpo, "2026-09-20", "20/09/2026", 1),
		"sin consecuencia": strings.Replace(cuerpo, "b24.sancion.suspension", "", 1),
	} {
		g := &gestorSancionesHTTPPrueba{}
		h, _ := NuevoHandlerSancionesParticipacion(preparadorSituacionHTTPPrueba{}, g)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionSancion(http.MethodPost, rutaSancionesPrueba, malo))
		if w.Code != 400 || g.ultima != nil {
			t.Fatalf("%s: status=%d", nombre, w.Code)
		}
	}
	sinClave := peticionSancion(http.MethodPost, rutaSancionesPrueba, cuerpo)
	sinClave.Header.Del("Idempotency-Key")
	h, _ := NuevoHandlerSancionesParticipacion(preparadorSituacionHTTPPrueba{}, &gestorSancionesHTTPPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, sinClave)
	if w.Code != 400 {
		t.Fatalf("sin clave idempotente: %d", w.Code)
	}
}

func TestSancionesErrores(t *testing.T) {
	cuerpo := `{"consecuencia":"b24.sancion.baja","causa":"Renuncia","fecha_notificacion":"2026-09-20","resuelta_por":"persona:jefatura","resolucion":{"referencia":"registro:2026/1","sha256":"` + strings.Repeat("a", 64) + `"}}`
	for err, esperado := range map[error]int{
		dominiovec.ErrAutorizacionDenegada:             403,
		ports.ErrSancionesNoConfiguradas:               503,
		ports.ErrClaveOperacionReutilizada:             409,
		domain.ErrCambioSituacionParticipacionInvalido: 409,
		domain.ErrSancionParticipacionInvalida:         400,
		ports.ErrSituacionParticipacionNoEncontrada:    404,
	} {
		h, _ := NuevoHandlerSancionesParticipacion(preparadorSituacionHTTPPrueba{}, &gestorSancionesHTTPPrueba{err: err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionSancion(http.MethodPost, rutaSancionesPrueba, cuerpo))
		if w.Code != esperado || strings.Contains(w.Body.String(), "bolsa:") {
			t.Fatalf("%v: status=%d body=%s", err, w.Code, w.Body.String())
		}
	}
	h, _ := NuevoHandlerSancionesParticipacion(preparadorSituacionHTTPPrueba{}, &gestorSancionesHTTPPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionSancion(http.MethodDelete, rutaSancionesPrueba, ""))
	if w.Code != 405 || w.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("DELETE: %d %q", w.Code, w.Header().Get("Allow"))
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionSancion(http.MethodGet, rutaSancionesPrueba+"/sancion:x/recursos", ""))
	if w.Code != 405 || w.Header().Get("Allow") != "POST" {
		t.Fatalf("GET recursos: %d", w.Code)
	}
}

func TestSancionesConsultaYRecurso(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	g := &gestorSancionesHTTPPrueba{vista: ports.VistaSancionesParticipacion{
		CatalogoDisponible: true, EstadosRecurso: []string{"interpuesto"},
		Consecuencias: []ports.ConsecuenciaSancion{{Clave: "b24.sancion.baja", Etiqueta: "Baja", Efecto: "excluir", Articulo: "art. 11.2"}},
		Sanciones: []domain.SancionParticipacion{{SancionRef: "sancion:x", Efecto: "ninguna", RecursoVence: "2026-10-20", RegistradaEn: ahora,
			Recursos: []domain.EventoRecursoSancion{{Estado: "interpuesto", Fecha: "2026-09-24", RegistradaEn: ahora}}}},
	}, recurso: ports.RegistroRecursoSancion{SancionRef: "sancion:x", Estado: "interpuesto", RegistradaEn: ahora}}
	h, _ := NuevoHandlerSancionesParticipacion(preparadorSituacionHTTPPrueba{}, g)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionSancion(http.MethodGet, rutaSancionesPrueba, ""))
	var cuerpo struct {
		Data struct {
			Esquema            string           `json:"esquema"`
			CatalogoDisponible bool             `json:"catalogo_disponible"`
			Items              []map[string]any `json:"items"`
		} `json:"data"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil || cuerpo.Data.Esquema != "vec.bolsa.rrhh.sanciones.v1" || !cuerpo.Data.CatalogoDisponible || len(cuerpo.Data.Items) != 1 {
		t.Fatalf("GET status=%d body=%s", w.Code, w.Body.String())
	}
	recurso := cuerpo.Data.Items[0]["recurso"].(map[string]any)
	if recurso["estado"] != "interpuesto" || cuerpo.Data.Items[0]["situacion_desde"] != nil {
		t.Fatalf("recurso inesperado: %v", cuerpo.Data.Items[0])
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionSancion(http.MethodPost, rutaSancionesPrueba+"/sancion:x/recursos", `{"estado":"interpuesto","fecha":"2026-09-24","documento":{"referencia":"registro:2026/9","sha256":"`+strings.Repeat("d", 64)+`"}}`))
	if w.Code != 201 || g.ultimoR == nil || g.ultimoR.SancionRef != "sancion:x" || g.ultimoR.Evento.Documento == nil {
		t.Fatalf("recurso status=%d body=%s", w.Code, w.Body.String())
	}
}
