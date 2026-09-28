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

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorConsultaReciboPrueba struct {
	resultado ports.ReciboRespuestaConsultado
	err       error
	llamadas  int
}

func (e *ejecutorConsultaReciboPrueba) Consultar(_ context.Context, _ ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error) {
	e.llamadas++
	return e.resultado, e.err
}

func TestConsultaReciboRespuestaSoloExponeVistaMinima(t *testing.T) {
	e := &ejecutorConsultaReciboPrueba{resultado: ports.ReciboRespuestaConsultado{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba", ComunicacionRef: "comunicacion:prueba",
		Respuesta: ports.RespuestaLlamamientoAceptada, JustificanteRef: "justificante:prueba",
		ReciboRef: "recibo:prueba", AuditoriaRef: "auditoria:prueba",
		RegistradaEn: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC), Estado: ports.EstadoRespuestaRecibidaRegistrada,
	}}
	h, err := NuevoManejadorConsultaReciboRespuesta(e)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaConsultaReciboRespuesta+"?organizacion_ref=organizacion:prueba&expediente_ref=expediente:prueba&comunicacion_ref=comunicacion:prueba", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || e.llamadas != 1 {
		t.Fatalf("estado=%d llamadas=%d", w.Code, e.llamadas)
	}
	var got map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	data := got["data"]
	if len(data) != 10 || data["esquema"] != EsquemaConsultaReciboRespuesta || data["expediente_ref"] != "expediente:prueba" || data["recibo_ref"] != "recibo:prueba" {
		t.Fatalf("contrato inesperado: %v", data)
	}
	for _, prohibido := range []string{"recibo_json", "correo_ref", "correo_sha256", "clave_idempotencia", "actor_ref", "perfil_ref"} {
		if strings.Contains(w.Body.String(), prohibido) {
			t.Fatalf("campo privado: %s", prohibido)
		}
	}
	if w.Header().Get("Cache-Control") == "" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("cabeceras de privacidad")
	}
}

func TestConsultaReciboRespuestaNoFiltraAusenciaNiAjeno(t *testing.T) {
	var cuerpos []string
	for _, causa := range []error{ports.ErrReciboRespuestaNoEncontrado, errors.Join(ports.ErrReciboRespuestaNoEncontrado, errors.New("actor ajeno"))} {
		e := &ejecutorConsultaReciboPrueba{err: causa}
		h, err := NuevoManejadorConsultaReciboRespuesta(e)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaConsultaReciboRespuesta+"?organizacion_ref=organizacion:prueba&expediente_ref=expediente:prueba&comunicacion_ref=comunicacion:prueba", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("estado=%d", w.Code)
		}
		cuerpos = append(cuerpos, w.Body.String())
	}
	// La correlación aleatoria cambia, pero el código y la clave pública no.
	for _, cuerpo := range cuerpos {
		if !strings.Contains(cuerpo, `"codigo":"recurso_no_encontrado"`) || strings.Contains(cuerpo, "actor ajeno") {
			t.Fatalf("respuesta pública divergente: %s", cuerpo)
		}
	}
}

func TestConsultaReciboRespuestaRechazaParametrosExtra(t *testing.T) {
	e := &ejecutorConsultaReciboPrueba{}
	h, err := NuevoManejadorConsultaReciboRespuesta(e)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaConsultaReciboRespuesta+"?organizacion_ref=organizacion:prueba&expediente_ref=expediente:prueba&comunicacion_ref=comunicacion:prueba&perfil=otro", nil))
	if w.Code != http.StatusBadRequest || e.llamadas != 0 {
		t.Fatalf("estado=%d llamadas=%d", w.Code, e.llamadas)
	}
}

func TestConsultaReciboRespuestaAdmiteTresReferenciasValidasLargas(t *testing.T) {
	e := &ejecutorConsultaReciboPrueba{err: ports.ErrReciboRespuestaNoEncontrado}
	h, err := NuevoManejadorConsultaReciboRespuesta(e)
	if err != nil {
		t.Fatal(err)
	}
	referencia := strings.Repeat("a", 160)
	query := "?organizacion_ref=" + referencia + "&expediente_ref=" + referencia + "&comunicacion_ref=" + referencia
	if len(query) <= 400 {
		t.Fatal("la prueba no supera el límite anterior")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaConsultaReciboRespuesta+query, nil))
	if w.Code != http.StatusNotFound || e.llamadas != 1 {
		t.Fatalf("estado=%d llamadas=%d", w.Code, e.llamadas)
	}
}

func TestConsultaReciboRespuestaExigeExpedienteYVerificaRecibo(t *testing.T) {
	e := &ejecutorConsultaReciboPrueba{resultado: ports.ReciboRespuestaConsultado{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:otro", ComunicacionRef: "comunicacion:prueba",
		Respuesta: ports.RespuestaLlamamientoAceptada, JustificanteRef: "justificante:prueba",
		ReciboRef: "recibo:prueba", AuditoriaRef: "auditoria:prueba",
		RegistradaEn: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC), Estado: ports.EstadoRespuestaRecibidaRegistrada,
	}}
	h, err := NuevoManejadorConsultaReciboRespuesta(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		query    string
		estado   int
		llamadas int
	}{
		{"?organizacion_ref=organizacion:prueba&comunicacion_ref=comunicacion:prueba", http.StatusBadRequest, 0},
		{"?organizacion_ref=organizacion:prueba&expediente_ref=expediente:prueba&comunicacion_ref=comunicacion:prueba", http.StatusBadGateway, 1},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaConsultaReciboRespuesta+caso.query, nil))
		if w.Code != caso.estado || e.llamadas != caso.llamadas {
			t.Fatalf("estado=%d llamadas=%d", w.Code, e.llamadas)
		}
	}
}

func TestConsultaReciboRespuestaTransitorioNoEsDenegacion(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
	}{
		{ports.ErrConsultaReciboRespuestaFallo, http.StatusServiceUnavailable},
		{ports.ErrConsultaReciboRespuestaDenegada, http.StatusForbidden},
	} {
		e := &ejecutorConsultaReciboPrueba{err: caso.err}
		h, err := NuevoManejadorConsultaReciboRespuesta(e)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
			RutaConsultaReciboRespuesta+"?organizacion_ref=organizacion:prueba&expediente_ref=expediente:prueba&comunicacion_ref=comunicacion:prueba", nil))
		if w.Code != caso.estado {
			t.Fatalf("error=%v: estado=%d", caso.err, w.Code)
		}
	}
}
