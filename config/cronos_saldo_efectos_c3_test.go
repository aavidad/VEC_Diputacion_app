package config

import (
	"errors"
	"testing"
)

func TestSaldoEfectosC3ApagadoOAislado(t *testing.T) {
	for _, v := range []string{"", "false"} {
		activo, err := CronosSaldoEfectosC3DesarrolloActivo(Config{}, v)
		if activo || err != nil {
			t.Fatalf("apagado %q: %v", v, err)
		}
	}
	for _, v := range []string{"true", "TRUE", "1"} {
		activo, err := CronosSaldoEfectosC3DesarrolloActivo(Config{}, v)
		if activo || !errors.Is(err, ErrConfiguracionSaldoEfectosC3) {
			t.Fatalf("selector %q: %v", v, err)
		}
	}
}
