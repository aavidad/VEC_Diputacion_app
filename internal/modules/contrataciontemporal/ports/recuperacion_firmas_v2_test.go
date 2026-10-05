package ports

import (
	"slices"
	"testing"
)

func TestCamposRecuperacionFirmasV2Son48ExactosYCopia(t *testing.T) {
	campos := CamposRecuperacionFirmasV2()
	base := CamposConsultaFirmasR5V2()
	if len(base) != 44 || len(campos) != 48 || !slices.IsSorted(campos) {
		t.Fatalf("cardinalidad u orden de campos incorrecto: base=%d recuperación=%d", len(base), len(campos))
	}
	for i := 1; i < len(campos); i++ {
		if campos[i] == campos[i-1] {
			t.Fatal("campo duplicado")
		}
	}
	for _, nuevo := range []string{"CanonNominal", "CanonNominalRef", "CanonNominalSHA256", "MaterialRootSHA256"} {
		if !slices.Contains(campos, nuevo) {
			t.Fatalf("falta %s", nuevo)
		}
	}
	campos[0] = "alterado"
	if CamposRecuperacionFirmasV2()[0] == "alterado" || CamposConsultaFirmasR5V2()[0] == "alterado" {
		t.Fatal("lista mutable compartida")
	}
}
