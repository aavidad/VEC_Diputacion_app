package domain

import "testing"

func TestConfirmacionAdjudicacionNoAdmiteModosSinImplementar(t *testing.T) {
	if !ConfirmacionAdjudicacionValida("") || !ConfirmacionAdjudicacionValida(ConfirmacionOfertaAceptacionPrevia) {
		t.Fatal("políticas históricas y aceptación previa deben admitirse")
	}
	for _, valor := range []string{"automatica", "aceptacion_previa ", "otra"} {
		if ConfirmacionAdjudicacionValida(valor) {
			t.Fatalf("modo no implementado admitido: %q", valor)
		}
	}
}
