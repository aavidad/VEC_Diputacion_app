package httppersonal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ejecutorPortalPrueba struct {
	llamadas int
	hasta    time.Time
	comando  mibolsa.ComandoRespuestaPortal
	err      error
	repetida bool
}

func (e *ejecutorPortalPrueba) SolicitarPausa(_ context.Context, _ mibolsa.Orden, bolsa string, hasta time.Time, clave string) (puertosbolsa.ReciboSolicitudPortal, error) {
	e.llamadas++
	e.hasta = hasta
	return puertosbolsa.ReciboSolicitudPortal{Reutilizada: e.repetida, SolicitudRef: "solicitud-portal:x", ReciboRef: "recibo:solicitud-portal:x", RegistradaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}, e.err
}

func (e *ejecutorPortalPrueba) SolicitarReactivacion(_ context.Context, _ mibolsa.Orden, _, _ string) (puertosbolsa.ReciboSolicitudPortal, error) {
	e.llamadas++
	return puertosbolsa.ReciboSolicitudPortal{SolicitudRef: "solicitud-portal:y", ReciboRef: "recibo:solicitud-portal:y", RegistradaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}, e.err
}

func (e *ejecutorPortalPrueba) Responder(_ context.Context, _ mibolsa.Orden, c mibolsa.ComandoRespuestaPortal) (puertosbolsa.ReciboRespuestaPortal, error) {
	e.llamadas++
	e.comando = c
	return puertosbolsa.ReciboRespuestaPortal{RespuestaRef: "respuesta-portal:x", ReciboRef: "recibo:respuesta-portal:x", Modo: "firme", RespondidaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC), VenceAntesDe: time.Date(2026, 9, 26, 22, 0, 0, 0, time.UTC)}, e.err
}

func peticionPortal(ruta, cuerpo string, cabeceras map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	for k, v := range cabeceras {
		r.Header.Set(k, v)
	}
	return r
}

func TestPortalRegistraSolicitudYRespuesta(t *testing.T) {
	e := new(ejecutorPortalPrueba)
	h, err := NuevoPortal(RutaMiBolsaSolicitudes, new(preparadorPrueba), e)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"pausa","bolsa":"bolsa:01","pausa_hasta":"2026-12-31T22:59:59Z","clave":"clave-pausa-0001"}`, nil))
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"estado":"pendiente_rrhh"`) || !e.hasta.Equal(time.Date(2026, 12, 31, 22, 59, 59, 0, time.UTC)) {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	e.repetida = true
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"pausa","bolsa":"bolsa:01","pausa_hasta":"2026-12-31T22:59:59Z","clave":"clave-pausa-0001"}`, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"repetida":true`) {
		t.Fatalf("repetición: %d %s", w.Code, w.Body.String())
	}
	r, _ := NuevoPortal(RutaMiBolsaRespuestas, new(preparadorPrueba), e)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, peticionPortal(RutaMiBolsaRespuestas, `{"bolsa":"bolsa:01","respuesta":"renuncia_justificada","causa":"enfermedad","justificante_ref":"justificante:1","justificante_sha256":"`+strings.Repeat("a", 64)+`","clave":"clave-respuesta-1"}`, nil))
	if w.Code != 201 || e.comando.Causa != "enfermedad" || !strings.Contains(w.Body.String(), `"estado":"firme"`) || !strings.Contains(w.Body.String(), `"vence_antes_de"`) {
		t.Fatalf("respuesta: %d %s", w.Code, w.Body.String())
	}
}

func TestPortalRechazaPeticionesNoSegurasSinEjecutar(t *testing.T) {
	e := new(ejecutorPortalPrueba)
	h, _ := NuevoPortal(RutaMiBolsaSolicitudes, new(preparadorPrueba), e)
	valido := `{"tipo":"reactivacion","bolsa":"bolsa:01","clave":"clave-reactiva-01"}`
	for nombre, r := range map[string]*http.Request{
		"otro origen":      peticionPortal(RutaMiBolsaSolicitudes, valido, map[string]string{"Sec-Fetch-Site": "cross-site"}),
		"formulario":       peticionPortal(RutaMiBolsaSolicitudes, valido, map[string]string{"Content-Type": "text/plain"}),
		"cookie":           peticionPortal(RutaMiBolsaSolicitudes, valido, map[string]string{"Cookie": "a=b"}),
		"campo ajeno":      peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"reactivacion","bolsa":"bolsa:01","clave":"clave-reactiva-01","candidato":"can_x"}`, nil),
		"pausa sin fin":    peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"pausa","bolsa":"bolsa:01","clave":"clave-pausa-0001"}`, nil),
		"reactiva con fin": peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"reactivacion","bolsa":"bolsa:01","pausa_hasta":"2026-12-31T22:59:59Z","clave":"clave-reactiva-01"}`, nil),
		"consulta":         peticionPortal(RutaMiBolsaSolicitudes+"?x=1", valido, nil),
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 || w.Code >= 500 {
			t.Fatalf("%s: status %d", nombre, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsaSolicitudes, nil))
	if w.Code != 405 || e.llamadas != 0 {
		t.Fatalf("método o ejecución indebida: %d %d", w.Code, e.llamadas)
	}
}

func TestPortalTraduceConflictosDeNegocio(t *testing.T) {
	for err, codigo := range map[error]string{
		puertosbolsa.ErrPortalSolicitudPendiente:    "solicitud_pendiente",
		puertosbolsa.ErrPortalSituacionNoAdmite:     "situacion_no_admite",
		puertosbolsa.ErrPortalSinLlamamientoAbierto: "sin_llamamiento_abierto",
		puertosbolsa.ErrPortalRespuestaFueraDePlazo: "fuera_de_plazo",
		puertosbolsa.ErrPortalClaveReutilizada:      "clave_reutilizada",
	} {
		h, _ := NuevoPortal(RutaMiBolsaSolicitudes, new(preparadorPrueba), &ejecutorPortalPrueba{err: err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"reactivacion","bolsa":"bolsa:01","clave":"clave-reactiva-01"}`, nil))
		if w.Code != 409 || !strings.Contains(w.Body.String(), codigo) {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
	h, _ := NuevoPortal(RutaMiBolsaSolicitudes, new(preparadorPrueba), &ejecutorPortalPrueba{err: puertosbolsa.ErrPortalCandidatoNoDisponible})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"reactivacion","bolsa":"bolsa:01","clave":"clave-reactiva-01"}`, nil))
	if w.Code != 503 {
		t.Fatalf("indisponibilidad: %d", w.Code)
	}
	if AccionPortalEn(http.MethodPost, RutaMiBolsaRespuestas)[0] != puertosbolsa.AccionResponderLlamamientoPropio || AccionPortalEn(http.MethodGet, RutaMiBolsaRespuestas) != nil || !EsRutaPortal(RutaMiBolsa) {
		t.Fatal("rutas y acciones del portal")
	}
}
