package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverPrueba struct {
	llamadas int
	err      error
}

func (r *resolverPrueba) ResolverMarcajePropio(*http.Request) (ports.ContextoMarcajePropio, error) {
	r.llamadas++
	return ports.ContextoMarcajePropio{}, r.err
}

type casoPrueba struct {
	llamadas  int
	solicitud ports.SolicitudMarcajePropio
	err       error
}

func (c *casoPrueba) RegistrarMarcajePropio(_ context.Context, _ ports.ContextoMarcajePropio, s ports.SolicitudMarcajePropio) (ports.ReciboMarcajePropio, error) {
	c.llamadas++
	c.solicitud = s
	return ports.ReciboMarcajePropio{Referencia: "recibo:cronos:00000000-0000-4000-8000-000000000001", MarcajeOriginalRef: "marcaje:cronos:" + s.ClaveOperacion, InstanteUTC: time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)}, c.err
}
func TestHTTPMarcajeEntregaDTOPropioSinIdentidad(t *testing.T) {
	resolver := &resolverPrueba{}
	caso := &casoPrueba{}
	h, _ := NuevoManejadorMarcajes(caso, resolver)
	r := httptest.NewRequest(http.MethodPost, RutaRegistrarMarcajePropio, strings.NewReader(`{"movimiento":"entrada","clave_operacion":"op-cronos-0001"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Empleado", "ignorado")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var cuerpo map[string]map[string]any
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil || cuerpo["recibo"]["marcaje_original_ref"] != "marcaje:cronos:op-cronos-0001" || cuerpo["recibo"]["instante_utc"] == nil || resolver.llamadas != 1 || caso.llamadas != 1 {
		t.Fatalf("contrato HTTP incorrecto: %d", w.Code)
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("privacidad HTTP")
	}
}
func TestHTTPMarcajeRechazaIdentidadEnCuerpoYRutasAjena(t *testing.T) {
	casos := []struct {
		ruta, metodo, cuerpo, media string
		estado                      int
	}{
		{RutaRegistrarMarcajePropio, "POST", `{"movimiento":"entrada","clave_operacion":"op-cronos-0001","empleado_ref":"emp_ajeno"}`, "application/json", 400},
		{RutaRegistrarMarcajePropio, "POST", strings.Repeat(" ", 4097), "application/json", 400},
		{RutaRegistrarMarcajePropio, "POST", `{"movimiento":"entrada","movimiento":"salida","clave_operacion":"op-cronos-0001"}`, "application/json", 400},
		{RutaRegistrarMarcajePropio, "POST", `null`, "application/json", 400},
		{RutaRegistrarMarcajePropio, "POST", `{}`, "application/json", 400},
		{RutaRegistrarMarcajePropio + "?empleado=otro", "POST", "{}", "application/json", 404},
		{"/otra", "POST", "{}", "application/json", 404},
		{RutaRegistrarMarcajePropio, "GET", "", "application/json", 405},
		{RutaRegistrarMarcajePropio, "POST", "{}", "application/jsonx", 400},
	}
	for _, c := range casos {
		resolver := &resolverPrueba{}
		caso := &casoPrueba{}
		h, _ := NuevoManejadorMarcajes(caso, resolver)
		r := httptest.NewRequest(c.metodo, c.ruta, strings.NewReader(c.cuerpo))
		r.Header.Set("Content-Type", c.media)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.estado || resolver.llamadas != 0 || caso.llamadas != 0 {
			t.Fatalf("entrada indebida: %d", w.Code)
		}
	}
}
func TestHTTPMarcajeDenegacionYConflictoCerrados(t *testing.T) {
	for _, c := range []struct {
		resolver, errorCaso error
		status              int
	}{
		{ErrAutenticacionCronosRequerida, nil, 401},
		{ErrAccesoCronosDenegado, nil, 403},
		{ports.ErrDependenciaNoDisponible, nil, 503},
		{errors.New("detalle_privado"), nil, 503},
		{nil, ports.ErrClaveOperacionEnConflicto, 409},
		{nil, ErrAccesoCronosDenegado, 403},
		{nil, errors.New("detalle_sql"), 503},
	} {
		resolver := &resolverPrueba{err: c.resolver}
		caso := &casoPrueba{err: c.errorCaso}
		h, _ := NuevoManejadorMarcajes(caso, resolver)
		r := httptest.NewRequest("POST", RutaRegistrarMarcajePropio, strings.NewReader(`{"movimiento":"entrada","clave_operacion":"op-cronos-0001"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.status || strings.Contains(w.Body.String(), "detalle") {
			t.Fatal("fallo no cerrado")
		}
	}
}

func TestNuevoManejadorMarcajesRechazaDependenciasNulasTipadas(t *testing.T) {
	var caso *casoPrueba
	var resolver *resolverPrueba
	if _, err := NuevoManejadorMarcajes(caso, &resolverPrueba{}); err == nil {
		t.Fatal("caso de uso nulo tipado aceptado")
	}
	if _, err := NuevoManejadorMarcajes(&casoPrueba{}, resolver); err == nil {
		t.Fatal("resolver nulo tipado aceptado")
	}
}
