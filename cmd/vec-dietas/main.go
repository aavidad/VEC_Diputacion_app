package main

import (
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/dietas/application/simulaciondevengo"
)

type fallo struct {
	Codigo string `json:"codigo"`
}

func main() {
	os.Exit(ejecutarConArgumentos(os.Args[1:], os.Stdin, os.Stdout))
}
func ejecutarConArgumentos(args []string, in io.Reader, out io.Writer) int {
	if len(args) != 0 {
		if json.NewEncoder(out).Encode(fallo{"argumentos_no_admitidos"}) != nil {
			return 1
		}
		return 2
	}
	return ejecutar(in, out)
}
func ejecutar(in io.Reader, out io.Writer) int {
	entrada, err := leerEntrada(in)
	if err != nil {
		if json.NewEncoder(out).Encode(fallo{err.Error()}) != nil {
			return 1
		}
		return 2
	}
	salida, err := simulaciondevengo.Simular(entrada)
	if err != nil {
		if json.NewEncoder(out).Encode(fallo{err.Error()}) != nil {
			return 1
		}
		return 2
	}
	if json.NewEncoder(out).Encode(salida) != nil {
		return 1
	}
	return 0
}
