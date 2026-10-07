package main

import (
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

func ejecutar(entrada ports.EntradaPreparacion, salida ports.SalidaPreparacion) error {
	p, err := entrada.Leer()
	if err != nil {
		return err
	}
	out, err := domain.Preparar(p)
	if err != nil {
		return err
	}
	return salida.Escribir(out)
}

func run(in io.Reader, out, errOut io.Writer) int {
	if err := ejecutar(entradaJSON{in}, salidaJSON{out}); err != nil {
		_ = json.NewEncoder(errOut).Encode(struct {
			Clave string `json:"error_clave"`
		}{err.Error()})
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Stdin, os.Stdout, os.Stderr)) }
