package config

import (
	"errors"
	"testing"
)

func TestCTFirmaRegistroSelector(t *testing.T) {
	for _, valor := range []string{"", "false", " false "} {
		if activo, err := (Config{CTFirmaRegistroEnabled: valor}).CTFirmaRegistroDesarrolloActivo(); activo || err != nil {
			t.Fatalf("%q: %t %v", valor, activo, err)
		}
	}
	if _, err := (Config{CTFirmaRegistroEnabled: "si"}).CTFirmaRegistroDesarrolloActivo(); !errors.Is(err, ErrConfiguracionCTFirmaRegistroSelector) {
		t.Fatalf("selector inválido: %v", err)
	}
	if _, err := (Config{CTFirmaRegistroEnabled: "true"}).CTFirmaRegistroDesarrolloActivo(); !errors.Is(err, ErrConfiguracionCTFirmaRegistroActivacion) {
		t.Fatalf("fuera de desarrollo: %v", err)
	}
}
