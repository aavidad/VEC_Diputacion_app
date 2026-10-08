package config

import (
	"errors"
	"testing"
)

func TestInscripcionesRequiereSelectorYDSNLector(t *testing.T) {
	var c Config
	if activo, err := c.BolsaInscripcionesActivo(); activo || err != nil {
		t.Fatalf("sin selector: %v %v", activo, err)
	}
	c.BolsaInscripcionesEnabled = "otro"
	if activo, err := c.BolsaInscripcionesActivo(); activo || !errors.Is(err, ErrBolsaInscripcionesSelector) {
		t.Fatalf("selector invalido: %v %v", activo, err)
	}
	c.BolsaInscripcionesEnabled = "true"
	if activo, err := c.BolsaInscripcionesActivo(); activo || err == nil {
		t.Fatalf("sin lector: %v %v", activo, err)
	}
}
