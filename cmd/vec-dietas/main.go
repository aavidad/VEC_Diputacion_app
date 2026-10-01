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
	if len(os.Args) != 1 {
		_ = json.NewEncoder(os.Stdout).Encode(fallo{"argumentos_no_admitidos"})
		os.Exit(2)
	}
	os.Exit(ejecutar(os.Stdin, os.Stdout))
}
func ejecutar(in io.Reader, out io.Writer) int {
	entrada, err := leerEntrada(in)
	if err != nil {
		if json.NewEncoder(out).Encode(fallo{"entrada_json_invalida"}) != nil {
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
