package contactopropio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

type ejecutorReciboContactoPropioPrueba struct {
	llamadas  int
	version   uint64
	resultado ports.ResultadoConsultaReciboContactoUsuario
	err       error
}

func (e *ejecutorReciboContactoPropioPrueba) ConsultarRecibo(_ context.Context, version uint64) (ports.ResultadoConsultaReciboContactoUsuario, error) {
	e.llamadas++
	e.version = version
	return e.resultado, e.err
}

func TestManejadorReciboContactoPropioRechazaTransporteAntesDeEjecutar(t *testing.T) {
	casos := []struct {
		nombre, metodo, destino, tipo, cuerpo string
		esperado                              int
	}{
		{"metodo", http.MethodGet, RutaReciboContactoPropio, "application/json", `{"version":1}`, http.StatusMethodNotAllowed},
		{"query", http.MethodPost, RutaReciboContactoPropio + "?sujeto=forjado", "application/json", `{"version":1}`, http.StatusNotFound},
		{"tipo", http.MethodPost, RutaReciboContactoPropio, "text/plain", `{"version":1}`, http.StatusBadRequest},
		{"desconocido", http.MethodPost, RutaReciboContactoPropio, "application/json", `{"version":1,"sujeto_ref":"forjado"}`, http.StatusBadRequest},
		{"duplicado", http.MethodPost, RutaReciboContactoPropio, "application/json", `{"version":1,"version":2}`, http.StatusBadRequest},
		{"nulo", http.MethodPost, RutaReciboContactoPropio, "application/json", `{"version":null}`, http.StatusBadRequest},
		{"cero", http.MethodPost, RutaReciboContactoPropio, "application/json", `{"version":0}`, http.StatusBadRequest},
		{"no entero", http.MethodPost, RutaReciboContactoPropio, "application/json", `{"version":1.5}`, http.StatusBadRequest},
		{"exceso", http.MethodPost, RutaReciboContactoPropio, "application/json", `{"version":9007199254740992}`, http.StatusBadRequest},
		{"cuerpo grande", http.MethodPost, RutaReciboContactoPropio, "application/json", `{"version":1}` + strings.Repeat(" ", maximoCuerpoContacto), http.StatusBadRequest},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			ejecutor := new(ejecutorReciboContactoPropioPrueba)
			h, err := NuevoManejadorRecibo(ejecutor, catalogoContactoPropioPrueba(t))
			if err != nil {
				t.Fatalf("NuevoManejadorRecibo: %v", err)
			}
			r := httptest.NewRequest(caso.metodo, caso.destino, strings.NewReader(caso.cuerpo))
			r.Header.Set("Content-Type", caso.tipo)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.esperado || ejecutor.llamadas != 0 {
				t.Fatalf("status=%d llamadas=%d", w.Code, ejecutor.llamadas)
			}
		})
	}
}

func TestManejadorReciboContactoPropioDevuelveReciboMinimo(t *testing.T) {
	ejecutor := &ejecutorReciboContactoPropioPrueba{resultado: ports.ResultadoConsultaReciboContactoUsuario{
		Encontrado: true,
		SujetoRef:  "per_sintetica_no_publicable",
		Version:    7,
		ReciboOriginal: ports.ReciboContactoUsuario{
			EvidenciaCentral: ports.EvidenciaAuditoriaCentralContactoUsuario{Referencia: "acc_sintetica_opaca"},
		},
	}}
	h, err := NuevoManejadorRecibo(ejecutor, catalogoContactoPropioPrueba(t))
	if err != nil {
		t.Fatalf("NuevoManejadorRecibo: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaReciboContactoPropio, strings.NewReader(`{"version":7}`))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || ejecutor.llamadas != 1 || ejecutor.version != 7 {
		t.Fatalf("status=%d llamadas=%d version=%d", w.Code, ejecutor.llamadas, ejecutor.version)
	}
	if got, want := strings.TrimSpace(w.Body.String()), `{"recibo_ref":"acc_sintetica_opaca","version":7}`; got != want || strings.Contains(got, "per_sintetica") {
		t.Fatalf("respuesta=%q", got)
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cabeceras inesperadas: %#v", w.Header())
	}
}

func TestManejadorReciboContactoPropioNoInfiereAusenciaComoExito(t *testing.T) {
	ejecutor := new(ejecutorReciboContactoPropioPrueba)
	h, err := NuevoManejadorRecibo(ejecutor, catalogoContactoPropioPrueba(t))
	if err != nil {
		t.Fatalf("NuevoManejadorRecibo: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaReciboContactoPropio, strings.NewReader(`{"version":7}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "recibo_ref") {
		t.Fatalf("ausencia convertida en exito: status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestManejadorReciboContactoPropioClasificaErroresYDependencias(t *testing.T) {
	catalogo := catalogoContactoPropioPrueba(t)
	if _, err := NuevoManejadorRecibo(nil, catalogo); !errors.Is(err, ErrManejadorContactoPropioInvalido) {
		t.Fatalf("ejecutor nulo: %v", err)
	}
	if _, err := NuevoManejadorRecibo(new(ejecutorReciboContactoPropioPrueba), nil); !errors.Is(err, ErrManejadorContactoPropioInvalido) {
		t.Fatalf("catalogo nulo: %v", err)
	}
	ejecutor := &ejecutorReciboContactoPropioPrueba{err: ErrContactoPropioNoDisponible}
	h, err := NuevoManejadorRecibo(ejecutor, catalogo)
	if err != nil {
		t.Fatalf("NuevoManejadorRecibo: %v", err)
	}
	peticion := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, RutaReciboContactoPropio, strings.NewReader(`{"version":7}`))
		r.Header.Set("Content-Type", "application/json")
		return r
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion())
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "Permisos insuficientes") {
		t.Fatalf("denegacion: status=%d body=%q", w.Code, w.Body.String())
	}
	ejecutor.err = ErrContactoPropioInvalido
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion())
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Solicitud no valida") {
		t.Fatalf("invalida: status=%d body=%q", w.Code, w.Body.String())
	}
}
