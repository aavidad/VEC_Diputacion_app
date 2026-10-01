package main

import (
	"errors"
	"testing"
)

type salidaFallida struct{}

func (salidaFallida) Write([]byte) (int, error) { return 0, errors.New("salida cerrada") }

func TestErrorDeSalidaNoSeDescarta(t *testing.T) {
	if codigo := ejecutar(nil, salidaFallida{}); codigo != 3 {
		t.Fatalf("codigo con stdout fallido: %d", codigo)
	}
}
