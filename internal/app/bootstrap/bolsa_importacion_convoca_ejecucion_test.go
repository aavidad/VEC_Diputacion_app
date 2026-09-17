package bootstrap

import (
	"testing"
	"time"
)

func TestBolsaRefImportacionConvocaDerivaCategoriaYFechaUTC(t *testing.T) {
	instante := time.Date(2026, 9, 17, 23, 30, 0, 0, time.FixedZone("oeste", -2*60*60))
	if recibida := bolsaRefImportacionConvoca("", " administrativo ", instante); recibida != "bolsa:administrativo:2026-09-18" {
		t.Fatalf("bolsa derivada inesperada: %q", recibida)
	}
	if recibida := bolsaRefImportacionConvoca("bolsa:existente:2026-01-01", "administrativo", instante); recibida != "bolsa:existente:2026-01-01" {
		t.Fatalf("la referencia indicada no se conserva: %q", recibida)
	}
}
