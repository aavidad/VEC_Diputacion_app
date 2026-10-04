package main

import (
	"bytes"
	"testing"
)

func TestPreservacionCLIRechazaPrestamosDeFirmaYDatos(t *testing.T) {
	for _, args := range [][]string{
		{"--operacion", "configurar-preservacion", "--kms-master", "material"},
		{"--operacion", "consultar-preservacion", "--entrada", "datos"},
		{"--operacion", "configurar-preservacion", "--version", "1"},
	} {
		var out bytes.Buffer
		if run(args, &out, &out) != 1 {
			t.Fatal("modo incompatible aceptado")
		}
	}
}
