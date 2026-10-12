package application

import (
	"regexp"
	"testing"
)

func TestCarteraSeparaReferenciasDeLecturaEIntentoComun(t *testing.T) {
	lectura, err := identificadorRegistroCartera(false)
	if err != nil || !regexp.MustCompile(`^lectura_[0-9a-f]{32}$`).MatchString(lectura) {
		t.Fatalf("lectura positiva: %q / %v", lectura, err)
	}
	intento, err := identificadorRegistroCartera(true)
	if err != nil || !regexp.MustCompile(`^intento_[0-9a-f]{32}$`).MatchString(intento) {
		t.Fatalf("intento nominal común: %q / %v", intento, err)
	}
	if lectura == intento {
		t.Fatal("lectura e intento comparten referencia")
	}
}
