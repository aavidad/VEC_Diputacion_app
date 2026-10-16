package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type operadorConfirmacionDirectaPrueba struct {
	llamadas int
	repetida bool
	err      error
}

func (o *operadorConfirmacionDirectaPrueba) ConfirmarAceptacionDirecta(_ context.Context,
	q ports.SolicitudConfirmarAceptacionDirecta,
) (ports.RegistroConfirmacionAceptacionDirecta, error) {
	o.llamadas++
	if o.err != nil {
		return ports.RegistroConfirmacionAceptacionDirecta{}, o.err
	}
	ahora := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	return ports.RegistroConfirmacionAceptacionDirecta{
		BolsaRef: q.BolsaRef, LlamamientoRef: q.LlamamientoRef,
		RespuestaRef: "respuesta-portal:" + strings.Repeat("a", 64), ResolucionRef: "resolucion:01",
		RegistroSituacionParticipacion: ports.RegistroSituacionParticipacion{
			Reutilizada: o.repetida, ReciboRef: "recibo:respuesta-revision:01", ResueltaEn: &ahora,
			Motivo: q.Motivo,
			SituacionParticipacion: ports.SituacionParticipacion{ParticipacionRef: q.ParticipacionRef,
				Situacion: "pendiente_incorporacion", Desde: ahora}},
	}, nil
}

func peticionConfirmacionDirecta(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaBolsasGestion+
		"/bolsa:01/candidatos/participacion:01/respuestas/llamamiento:01/confirmacion", strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "confirmacion-0001")
	return r
}

func TestConfirmacionDirectaHTTPReciboReplayYConflicto(t *testing.T) {
	o := &operadorConfirmacionDirectaPrueba{}
	h, err := NuevoHandlerConfirmacionDirecta(preparadorSituacionHTTPPrueba{}, o)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"motivo":"Aceptación comprobada","situacion_desde_esperada":"2026-10-16T08:00:00.000000Z"}`
	for _, caso := range []struct {
		repetida bool
		estado   int
	}{{false, 201}, {true, 200}} {
		o.repetida = caso.repetida
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionConfirmacionDirecta(cuerpo))
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"respuesta_ref":"respuesta-portal:`) {
			t.Fatalf("recibo: %d %s", w.Code, w.Body.String())
		}
	}
	o.err = ports.ErrConfirmacionAceptacionDirectaConflicto
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionConfirmacionDirecta(cuerpo))
	if w.Code != 409 || o.llamadas != 3 {
		t.Fatalf("conflicto: %d llamadas=%d", w.Code, o.llamadas)
	}
	o.err = vecports.ErrInstantaneaAutorizacionObsoleta
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionConfirmacionDirecta(cuerpo))
	if w.Code != 403 || !strings.Contains(w.Body.String(), `"acceso_denegado"`) {
		t.Fatalf("rol retirado: %d %s", w.Code, w.Body.String())
	}
}

func TestConfirmacionDirectaHTTPRechazaCuerpoAmbiguoAntesDeActuar(t *testing.T) {
	o := &operadorConfirmacionDirectaPrueba{}
	h, _ := NuevoHandlerConfirmacionDirecta(preparadorSituacionHTTPPrueba{}, o)
	longitudFalsa := peticionConfirmacionDirecta(`{"motivo":"A","situacion_desde_esperada":"2026-10-16T08:00:00Z"}`)
	longitudFalsa.ContentLength++
	for _, r := range []*http.Request{
		peticionConfirmacionDirecta(`{"motivo":"A","motivo":"B","situacion_desde_esperada":"2026-10-16T08:00:00Z"}`),
		longitudFalsa,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 || o.llamadas != 0 {
			t.Fatalf("entrada ambigua: %d llamadas=%d", w.Code, o.llamadas)
		}
	}
}

func TestConfirmacionDirectaHTTPAdmiteMotivoEscapadoDentroDelLimite(t *testing.T) {
	o := &operadorConfirmacionDirectaPrueba{}
	h, _ := NuevoHandlerConfirmacionDirecta(preparadorSituacionHTTPPrueba{}, o)
	cuerpo := `{"motivo":"` + strings.Repeat(`\u0041`, 1000) +
		`","situacion_desde_esperada":"2026-10-16T08:00:00.000000Z"}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionConfirmacionDirecta(cuerpo))
	if w.Code != 201 || o.llamadas != 1 {
		t.Fatalf("motivo escapado: %d llamadas=%d", w.Code, o.llamadas)
	}
}
