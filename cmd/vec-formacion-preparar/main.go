package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/formacion/adapters/jsonio"
	"vec-diputacion-granada/internal/modules/formacion/application"
)

func run(in io.Reader, out, diagnostico io.Writer) int {
	d, err := jsonio.Leer(in)
	if err == nil {
		var p application.Preparacion
		p, err = application.Preparar(d)
		if err == nil {
			if err = jsonio.Escribir(out, p); err == nil {
				return 0
			}
		}
	}
	clave := "formacion.error.salida"
	if !errors.Is(err, jsonio.ErrSalida) {
		clave = err.Error()
	}
	// Solo se emiten claves controladas; nunca se devuelve el contenido de entrada.
	_ = json.NewEncoder(diagnostico).Encode(struct {
		ErrorClave string `json:"error_clave"`
	}{clave})
	return 1
}
func main() { os.Exit(run(os.Stdin, os.Stdout, os.Stderr)) }
