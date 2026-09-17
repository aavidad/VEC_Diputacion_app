package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestClasificacionConflictoEstadoNoSeDegradaA503(t *testing.T) {
	for nombre, causa := range map[string]error{
		"propuesta": application.ErrPresentacionPropuestaCoberturaEstadoNoAdmite,
		"seleccion": ports.ErrEstadoExpedienteNoSeleccionable,
	} {
		t.Run(nombre, func(t *testing.T) {
			var problema errorPublicoCobertura
			if nombre == "propuesta" {
				problema = clasificarErrorCobertura(causa)
			} else {
				problema = clasificarErrorSeleccionLlamamiento(causa)
			}
			if problema.estado != http.StatusConflict || problema.codigo != "conflicto_estado" {
				t.Fatalf("clasificación=%+v", problema)
			}
		})
	}
}

func TestConflictoEstadoSeRegistraSinExponerLaCausa(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	var registro bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))

	w := httptest.NewRecorder()
	causa := errors.New("CONTENIDO_PRIVADO")
	responderErrorCobertura(
		w,
		httptest.NewRequest(http.MethodPost, RutaPropuestaCobertura, nil),
		errorConflictoEstadoCobertura,
		causa,
	)
	var cuerpo envoltorioErrorCobertura
	var entrada map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(registro.Bytes(), &entrada); err != nil {
		t.Fatalf("registro=%q: %v", registro.String(), err)
	}
	if w.Code != http.StatusConflict || cuerpo.Error.Codigo != "conflicto_estado" ||
		entrada["codigo"] != "conflicto_estado" ||
		entrada["correlacion_ref"] != cuerpo.Error.CorrelacionRef ||
		strings.Contains(registro.String()+w.Body.String(), "CONTENIDO_PRIVADO") {
		t.Fatalf("respuesta o registro inesperados: respuesta=%s registro=%s", w.Body.String(), registro.String())
	}
}
