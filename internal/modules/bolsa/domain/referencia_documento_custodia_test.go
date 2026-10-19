package domain

import (
	"strings"
	"testing"
)

func TestReferenciaDocumentoCustodia(t *testing.T) {
	hex := "12345678a" + strings.Repeat("b", 55)
	for _, v := range []string{"registro/2026/0042", "expediente:EXP-2026.17#acta", "ref:" + strings.Repeat("c", 64), "abc", strings.Repeat("a", 256)} {
		if !ReferenciaDocumentoCustodiaValida(v) {
			t.Errorf("rechazó %q", v)
		}
	}
	for _, v := range []string{"dni:prueba", "NIE-archivo", "documento:12345678Z", "X1234567L", "ab", "con espacio", "/registro", "otro:contrato-bolsa:" + hex, strings.Repeat("a", 257), "ref*1"} {
		if ReferenciaDocumentoCustodiaValida(v) {
			t.Errorf("admitió %q", v)
		}
	}
	if !ReferenciaEventoContratoParticipacion(prefijoEventoContratoParticipacion+hex) || ReferenciaDocumentoCustodiaValida(prefijoEventoContratoParticipacion+hex) {
		t.Fatal("el evento de contrato debe admitirse solo por su forma exacta")
	}
	for _, v := range []string{prefijoEventoContratoParticipacion + strings.ToUpper(hex), prefijoEventoContratoParticipacion + hex[:63], "evento:ct:contrato:" + hex} {
		if ReferenciaEventoContratoParticipacion(v) {
			t.Errorf("admitió evento %q", v)
		}
	}
}
