package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorEmisionPrueba struct{ err error }

func (p preparadorEmisionPrueba) PrepararSolicitudEmitirLlamamiento(context.Context, EntradaEmitirLlamamiento) (ports.SolicitudEmitirLlamamiento, error) {
	return ports.SolicitudEmitirLlamamiento{}, p.err
}

type operadorEmisionPrueba struct{ reutilizada bool }

func (o operadorEmisionPrueba) EmitirLlamamiento(context.Context, ports.SolicitudEmitirLlamamiento) (ports.EmisionLlamamiento, error) {
	return ports.EmisionLlamamiento{LlamamientoRef: "llamamiento:abc", ReciboRef: "recibo:llamamiento:abc", Estado: "emitido_pendiente_respuesta", EmitidoEn: time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC), Reutilizada: o.reutilizada}, nil
}

type recuperadorEmisionPrueba struct{}

func (recuperadorEmisionPrueba) Recuperar(context.Context, string, string) (ports.EmisionLlamamiento, error) {
	return ports.EmisionLlamamiento{ReciboRef: "recibo:llamamiento:abc", Reutilizada: true}, nil
}

func TestHandlerEmisionLlamamientoCreaYRecuperaSinCambiarRuta(t *testing.T) {
	for _, caso := range []struct {
		nombre      string
		reutilizada bool
		estado      int
	}{{"alta", false, http.StatusCreated}, {"replay", true, http.StatusOK}} {
		t.Run(caso.nombre, func(t *testing.T) {
			h, err := NuevoHandlerEmisionLlamamiento(preparadorEmisionPrueba{}, operadorEmisionPrueba{caso.reutilizada}, recuperadorEmisionPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, RutaEmisionesLlamamiento, strings.NewReader("{\"bolsa_ref\":\"bolsa:01\",\"participaciones\":[\"part:01\"],\"configuracion\":{\"referencia\":\"NEC-01\"}}"))
			r.Header.Set("Accept", "application/json")
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", "b7-emision-0001")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || !strings.Contains(w.Body.String(), "recibo:llamamiento:abc") {
				t.Fatalf("respuesta inesperada: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandlerEmisionLlamamientoDeniegaSinAmbito(t *testing.T) {
	h, err := NuevoHandlerEmisionLlamamiento(preparadorEmisionPrueba{err: dominiovec.ErrAutorizacionDenegada}, operadorEmisionPrueba{}, recuperadorEmisionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaEmisionesLlamamiento, strings.NewReader("{\"bolsa_ref\":\"bolsa:01\",\"participaciones\":[\"part:01\"],\"configuracion\":{\"referencia\":\"NEC-01\"}}"))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "b7-emision-0001")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "acceso_denegado") {
		t.Fatalf("respuesta inesperada: %d %s", w.Code, w.Body.String())
	}
}
