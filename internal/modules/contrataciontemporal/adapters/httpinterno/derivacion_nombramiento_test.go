package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorDerivacionPrueba struct{ llamadas int }

func (e *ejecutorDerivacionPrueba) Derivar(_ context.Context, _ ports.EntradaDerivacionNombramiento) (ports.ResultadoDerivacionNombramiento, error) {
	e.llamadas++
	return ports.ResultadoDerivacionNombramiento{}, ports.ErrDerivacionNombramientoNoDisponible
}

type resultadoDerivacionPrueba struct{ estado string }

func (e resultadoDerivacionPrueba) Derivar(_ context.Context, _ ports.EntradaDerivacionNombramiento) (ports.ResultadoDerivacionNombramiento, error) {
	return ports.ResultadoDerivacionNombramiento{Esquema: ports.EsquemaResultadoDerivacionNombramiento,
		Estado: e.estado, PadreRef: "expediente:padre:uno", HijoRef: "expediente:hijo:uno",
		NumeroVisible: "2026/CT-P1", NumeroPlaza: 1, TotalPersonas: 2, Version: 6,
		AceptacionRef: "aceptacion:uno", ReciboRef: "recibo:uno", AuditoriaRef: "auditoria:privada",
		ConfirmadaEn: time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)}, nil
}

func TestDerivacionNombramientoCierraEntradaYConfiguracion(t *testing.T) {
	e := &ejecutorDerivacionPrueba{}
	h, err := NuevoManejadorDerivacionNombramiento(e)
	if err != nil {
		t.Fatal(err)
	}
	const base = `{"padre_ref":"expediente:padre:uno","version_padre":6,"oferta_ref":"oferta:` +
		`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","numero_plaza":1,` +
		`"secuencia_plaza":1,"participacion_ref":"participacion:uno","recibo_adjudicacion_ref":"recibo:uno",` +
		`"clave_idempotencia":"938f47a6-5d2b-4c10-aa11-1234567890ab"}`
	for _, caso := range []struct {
		nombre, cuerpo string
		estado         int
		llamadas       int
	}{
		{"configuracion ausente", base, http.StatusServiceUnavailable, 1},
		{"actor de cliente", strings.Replace(base, `"padre_ref"`, `"actor_ref":"persona:otra","padre_ref"`, 1), http.StatusUnprocessableEntity, 0},
		{"organizacion de cliente", strings.Replace(base, `"padre_ref"`, `"organizacion_ref":"org:otra","padre_ref"`, 1), http.StatusUnprocessableEntity, 0},
		{"duplicado", strings.Replace(base, `"version_padre":6`, `"version_padre":6,"version_padre":6`, 1), http.StatusBadRequest, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e.llamadas = 0
			r := httptest.NewRequest(http.MethodPost, RutaDerivacionNombramiento, strings.NewReader(caso.cuerpo))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || e.llamadas != caso.llamadas {
				t.Fatalf("estado=%d llamadas=%d cuerpo=%s", w.Code, e.llamadas, w.Body.String())
			}
		})
	}
}

func TestDerivacionNombramientoProyectaReciboSinAuditoria(t *testing.T) {
	const cuerpo = `{"padre_ref":"expediente:padre:uno","version_padre":6,"oferta_ref":"oferta:` +
		`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","numero_plaza":1,` +
		`"secuencia_plaza":1,"participacion_ref":"participacion:uno","recibo_adjudicacion_ref":"recibo:uno",` +
		`"clave_idempotencia":"938f47a6-5d2b-4c10-aa11-1234567890ab"}`
	for _, caso := range []struct {
		estado string
		codigo int
	}{{"confirmado", http.StatusCreated}, {"replay_confirmado", http.StatusOK}} {
		h, err := NuevoManejadorDerivacionNombramiento(resultadoDerivacionPrueba{estado: caso.estado})
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodPost, RutaDerivacionNombramiento, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.codigo || !strings.Contains(w.Body.String(), `"estado":"`+caso.estado+`"`) ||
			strings.Contains(w.Body.String(), "auditoria:privada") || strings.Contains(w.Body.String(), "auditoria_ref") {
			t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
		}
	}
}
