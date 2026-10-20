package httpinterno

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorEmisionCaptura struct{ entrada *EntradaEmitirLlamamiento }

func (p preparadorEmisionCaptura) PrepararSolicitudEmitirLlamamiento(_ context.Context, e EntradaEmitirLlamamiento) (ports.SolicitudEmitirLlamamiento, error) {
	*p.entrada = e
	return ports.SolicitudEmitirLlamamiento{}, nil
}
func (p preparadorEmisionCaptura) PrepararSolicitudRecuperarLlamamiento(context.Context, string, string) (ports.SolicitudRecuperarLlamamiento, error) {
	return ports.SolicitudRecuperarLlamamiento{}, nil
}

type operadorEmisionError struct{ err error }

func (o operadorEmisionError) EmitirLlamamiento(context.Context, ports.SolicitudEmitirLlamamiento) (ports.EmisionLlamamiento, error) {
	return ports.EmisionLlamamiento{}, o.err
}
func (o operadorEmisionError) RecuperarLlamamiento(context.Context, ports.SolicitudRecuperarLlamamiento) (ports.EmisionLlamamiento, error) {
	return ports.EmisionLlamamiento{}, o.err
}

func emitirConError(t *testing.T, err error, cuerpo string) (*httptest.ResponseRecorder, EntradaEmitirLlamamiento) {
	t.Helper()
	var entrada EntradaEmitirLlamamiento
	h, errH := NuevoHandlerEmisionLlamamiento(preparadorEmisionCaptura{&entrada}, operadorEmisionError{err})
	if errH != nil {
		t.Fatal(errH)
	}
	r := httptest.NewRequest(http.MethodPost, RutaEmisionesLlamamiento, strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "b7-emision-0001")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w, entrada
}

func TestHandlerEmisionFueraDeOrdenResponde409ConQuienVaAntes(t *testing.T) {
	cuerpo := `{"bolsa_ref":"bolsa:01","participaciones":["part:03"],"configuracion":{"referencia":"NEC-01"}}`
	w, _ := emitirConError(t, &ports.LlamamientoFueraDeOrden{ParticipacionRef: "part:02", OrdenVigente: 2}, cuerpo)
	var salida struct {
		Error map[string]any `json:"error"`
	}
	if w.Code != http.StatusConflict || json.Unmarshal(w.Body.Bytes(), &salida) != nil {
		t.Fatalf("respuesta inesperada: %d %s", w.Code, w.Body.String())
	}
	if salida.Error["codigo"] != "llamamiento_fuera_de_orden" || salida.Error["participacion_ref"] != "part:02" ||
		salida.Error["orden_vigente"] != float64(2) || fmt.Sprint(salida.Error["motivos_salto"]) != "[]" {
		t.Fatalf("cuerpo del rechazo: %s", w.Body.String())
	}
	// Envuelto y sin identificar a nadie: sigue siendo un 409 sin referencia.
	w, _ = emitirConError(t, fmt.Errorf("x: %w", &ports.LlamamientoFueraDeOrden{}), cuerpo)
	if w.Code != http.StatusConflict || strings.Contains(w.Body.String(), "participacion_ref") {
		t.Fatalf("rechazo sin detalle: %d %s", w.Code, w.Body.String())
	}
}

func TestHandlerEmisionMotivoSalto(t *testing.T) {
	cuerpo := `{"bolsa_ref":"bolsa:01","participaciones":["part:03"],"configuracion":{"referencia":"NEC-01"},"motivo_salto":"b01.salto.x"}`
	w, entrada := emitirConError(t, fmt.Errorf("%w: %w", ports.ErrEmisionLlamamientoInvalida, ports.ErrMotivoSaltoNoAdmitido), cuerpo)
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "motivo_salto_no_admitido") || entrada.MotivoSalto != "b01.salto.x" {
		t.Fatalf("motivo no admitido: %d %s %+v", w.Code, w.Body.String(), entrada)
	}
}
