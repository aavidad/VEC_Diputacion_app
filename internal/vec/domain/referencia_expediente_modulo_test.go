package domain

import (
	"strings"
	"testing"
)

func TestModuloReferenciaExpedienteV1(t *testing.T) {
	id := strings.Repeat("a", 64)
	if got := ModuloReferenciaExpedienteV1("expediente:ct:" + id); got != "contratacion_temporal" {
		t.Fatalf("módulo CT = %q", got)
	}
	if got := ModuloReferenciaExpedienteV1("expediente:ct:" + strings.Repeat("0", 63) + "1"); got != "contratacion_temporal" {
		t.Fatalf("ID CT no cero con prefijo cero = %q", got)
	}
	for _, referencia := range []string{
		"expediente:ct:" + strings.Repeat("0", 64),
		"ref:" + id,
		"expediente:bolsa:" + id,
		"expediente:CT:" + id,
		"Expediente:ct:" + id,
		"expediente:ct:" + strings.ToUpper(id),
		"expediente:ct:" + id[:63],
		"expediente:ct:" + id + "0",
		"expediente:ct:" + id + ":extra",
	} {
		if got := ModuloReferenciaExpedienteV1(referencia); got != "" {
			t.Errorf("referencia %q resolvió módulo %q", referencia, got)
		}
	}
}
