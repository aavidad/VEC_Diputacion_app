package lecturaincorporacion

import (
	"strings"
	"testing"
	"time"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestLectorV2UnidadOriginalDelExpediente(t *testing.T) {
	const unidad = "unidad:desarrollo:rrhh"
	for _, ref := range []string{unidad, "ref:" + strings.Repeat("a", 64), "ref:" + strings.Repeat("0", 64)} {
		if !unidadLecturaV2Valida(ref) {
			t.Fatal("unidad original o legacy rechazada")
		}
	}
	for _, ref := range []string{"unidad:", "unidad:con espacio", "Unidad:rrhh", "centro:rrhh", "unidad:" + strings.Repeat("a", 154)} {
		if unidadLecturaV2Valida(ref) {
			t.Fatal("unidad ajena admitida")
		}
	}
	// Aceptar la gramática no sustituye selector ni contexto nominal.
	if _, err := NuevoMaterialV2(Selector{}, unidad, ct.ContextoAutorizacionAltaV3{}, time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("unidad por sí sola permitió preparar una lectura")
	}
}
