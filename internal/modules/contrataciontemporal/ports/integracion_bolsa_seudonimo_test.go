package ports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestSeudonimoSeleccionExigeHMACConDominioCerrado(t *testing.T) {
	t.Parallel()
	noValidos := []string{
		"",
		"12345678Z",
		"X1234567L",
		"persona@example.org",
		"María Pérez",
		"seleccion:persona:42",
		strings.Repeat("a", 64),
		"hmac-sha256:otro.dominio/v1:" + strings.Repeat("a", 64),
		"hmac-sha256:" + dominioSeudonimoSeleccionBolsa +
			"/v01:" + strings.Repeat("a", 64),
	}
	for _, valor := range noValidos {
		if _, err := NuevoSeudonimoSeleccionBolsa(valor); err == nil {
			t.Fatalf("identificador directo o gramática ajena aceptada: %q", valor)
		}
	}
	if _, err := NuevoSeudonimoSeleccionBolsa(
		"hmac-sha256:" + dominioSeudonimoSeleccionBolsa +
			"/v1:" + strings.Repeat("a", 64),
	); err != nil {
		t.Fatalf("seudónimo HMAC válido rechazado: %v", err)
	}
}

func TestSeudonimoSeleccionRedactaDiagnosticoYConservaCodecs(t *testing.T) {
	t.Parallel()
	original := seudonimoSeleccionBolsaPrueba(t, "b")
	crudo := original.valorCanonico()
	formatos := []string{"%s", "%q", "%v", "%+v", "%#v"}
	for _, formato := range formatos {
		salida := fmt.Sprintf(formato, original)
		if strings.Contains(salida, crudo) ||
			!strings.Contains(salida, seudonimoSeleccionRedactado) {
			t.Fatalf("formato %s filtró o no redactó: %s", formato, salida)
		}
	}
	var bitacora bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&bitacora, nil))
	logger.Info("prueba", "seleccion", original)
	if strings.Contains(bitacora.String(), crudo) ||
		!strings.Contains(bitacora.String(), seudonimoSeleccionRedactado) {
		t.Fatalf("slog filtró o no redactó: %s", bitacora.String())
	}

	contenidoJSON, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("codificar JSON: %v", err)
	}
	var desdeJSON SeudonimoSeleccionBolsa
	if err := json.Unmarshal(contenidoJSON, &desdeJSON); err != nil ||
		desdeJSON != original {
		t.Fatalf("codec JSON no conservó el seudónimo: %v", err)
	}
	contenidoTexto, err := original.MarshalText()
	if err != nil {
		t.Fatalf("codificar texto: %v", err)
	}
	var desdeTexto SeudonimoSeleccionBolsa
	if err := desdeTexto.UnmarshalText(contenidoTexto); err != nil ||
		desdeTexto != original {
		t.Fatalf("codec texto no conservó el seudónimo: %v", err)
	}
}
