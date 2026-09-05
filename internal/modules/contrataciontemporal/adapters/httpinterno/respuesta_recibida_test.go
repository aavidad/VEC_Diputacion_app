package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorRespuestaPrueba struct {
	llamadas int
	estado   string
	err      error
}

func (e *ejecutorRespuestaPrueba) Registrar(_ context.Context, s ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
	e.llamadas++
	if e.err != nil {
		return ports.RespuestaRecibidaRegistrada{}, e.err
	}
	return ports.RespuestaRecibidaRegistrada{Solicitud: s, Estado: e.estado, JustificanteRef: "respuesta:sintetica",
		ReciboRef: "recibo:sintetico", AuditoriaRef: "auditoria:sintetica", RegistradaEn: s.RecibidaEn.Add(time.Minute)}, nil
}

func entradaRespuestaPrueba() respuestaRecibidaJSON {
	return respuestaRecibidaJSON{ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", OrganizacionRef: "org:sintetica",
		ExpedienteRef: "exp:sintetico", LlamamientoRef: "llamamiento:sintetico", ComunicacionRef: "comunicacion:sintetica",
		VersionComunicacionEsperada: 2, Respuesta: "aceptacion", CorreoRef: "correo:sintetico", CorreoSHA256: strings.Repeat("a", 64), RecibidaEn: "2026-09-05T10:00:00.000Z"}
}

func TestRespuestaRecibidaHTTPRegistroReplayYErrores(t *testing.T) {
	for _, caso := range []struct {
		nombre, estado string
		err            error
		http           int
	}{
		{"registro", "registrada_por_rrhh", nil, 201}, {"replay", "replay_registrada_por_rrhh", nil, 200},
		{"denegado", "", application.ErrRespuestaRecibidaDenegada, 403},
		{"clave", "", application.ErrClaveRespuestaRecibidaEnColision, 409},
		{"incierto", "", application.ErrRespuestaRecibidaNoDisponible, 503},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := &ejecutorRespuestaPrueba{estado: caso.estado, err: caso.err}
			h, err := NuevoManejadorRespuestaRecibida(e)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(entradaRespuestaPrueba())
			r := httptest.NewRequest("POST", RutaRegistroRespuestaRecibida, strings.NewReader(string(b)))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.http || e.llamadas != 1 {
				t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
			}
			if caso.err == nil {
				var salida struct {
					Data respuestaRecibidaSalidaJSON `json:"data"`
				}
				if json.Unmarshal(w.Body.Bytes(), &salida) != nil || salida.Data.Esquema != EsquemaRegistroRespuestaRecibida || salida.Data.Estado != caso.estado || salida.Data.VersionComunicacionEsperada != 2 || salida.Data.RecibidaEn != "2026-09-05T10:00:00Z" {
					t.Fatal("recibo desligado", w.Body.String())
				}
			}
			if w.Header().Get("Set-Cookie") != "" {
				t.Fatal("cookie inesperada")
			}
		})
	}
}

func TestRespuestaRecibidaHTTPNoAceptaAutoridadNiEntradaAmbigua(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*http.Request, *respuestaRecibidaJSON)
		raw    string
	}{
		{"cabecera de autoridad libre", func(r *http.Request, s *respuestaRecibidaJSON) { r.Header.Set("X-Vec-Actor-Ref", "actor:ajeno") }, ""},
		{"cookie", func(r *http.Request, s *respuestaRecibidaJSON) { r.Header.Set("Cookie", "x=y") }, ""},
		{"query", func(r *http.Request, s *respuestaRecibidaJSON) { r.URL.RawQuery = "actor=x" }, ""},
		{"version", func(r *http.Request, s *respuestaRecibidaJSON) { s.VersionComunicacionEsperada = 6 }, ""},
		{"expiracion", func(r *http.Request, s *respuestaRecibidaJSON) { s.Respuesta = "expiracion_gobernada" }, ""},
		{"precisión temporal no canónica", func(r *http.Request, s *respuestaRecibidaJSON) { s.RecibidaEn = "2026-09-05T10:00:00.0000001Z" }, ""},
		{"duplicada", nil, `{"respuesta":"aceptacion","respuesta":"renuncia"}`},
		{"campos ajenos", nil, `{"actor_ref":"actor:ajeno"}`},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := &ejecutorRespuestaPrueba{estado: "registrada_por_rrhh"}
			h, _ := NuevoManejadorRespuestaRecibida(e)
			s := entradaRespuestaPrueba()
			r := httptest.NewRequest("POST", RutaRegistroRespuestaRecibida, nil)
			r.Header.Set("Content-Type", "application/json")
			if caso.mutar != nil {
				caso.mutar(r, &s)
			}
			b, _ := json.Marshal(s)
			if caso.raw != "" {
				b = []byte(caso.raw)
			}
			body := httptest.NewRequest("POST", RutaRegistroRespuestaRecibida, strings.NewReader(string(b)))
			r.Body = body.Body
			r.ContentLength = body.ContentLength
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code < 400 || e.llamadas != 0 {
				t.Fatalf("entrada alcanzó registro: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
