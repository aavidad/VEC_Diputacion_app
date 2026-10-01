package main

import (
	"encoding/json"
	"io"
	"os"

	adapter "vec-diputacion-granada/internal/modules/carrera/adapters/json"
	"vec-diputacion-granada/internal/modules/carrera/application"
)

func run(in io.Reader, out, errOut io.Writer) int {
	if err := (application.Servicio{}).Ejecutar(adapter.Entrada{Reader: in}, adapter.Salida{Writer: out}); err != nil {
		_ = json.NewEncoder(errOut).Encode(struct {
			Clave string `json:"error_clave"`
		}{err.Error()})
		return 1
	}
	return 0
}
func main() { os.Exit(run(os.Stdin, os.Stdout, os.Stderr)) }
