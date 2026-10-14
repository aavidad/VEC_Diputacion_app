package config

import (
	"errors"
	"testing"
)

func TestSelectorCalendarioC4NoAfectaArranqueComun(t *testing.T) {
	for _, selector := range []string{"", "false"} {
		activo, err := CronosCalendarioC4DesarrolloActivo(Config{}, selector)
		if activo || err != nil {
			t.Fatalf("apagado %q: %v", selector, err)
		}
	}
	for _, selector := range []string{"true", "TRUE", " false ", "1"} {
		activo, err := CronosCalendarioC4DesarrolloActivo(Config{}, selector)
		if activo || !errors.Is(err, ErrConfiguracionCronosCalendarioC4) {
			t.Fatalf("selector %q: %v", selector, err)
		}
	}
}
