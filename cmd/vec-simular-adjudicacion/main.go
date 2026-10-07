// Command vec-simular-adjudicacion consume un único JSON sintético por stdin.
package main

import (
	"encoding/json"
	"io"
	"os"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

func ejecutar(entrada io.Reader, salida, diagnostico io.Writer) int {
	p, err := simulacion.DecodificarAdjudicacion(entrada)
	if err != nil {
		return informarError(diagnostico, err)
	}
	r, err := application.SimularAdjudicacion(p.Configuracion, p.Entrada)
	if err != nil {
		return informarError(diagnostico, err)
	}
	if err := json.NewEncoder(salida).Encode(r); err != nil {
		return informarError(diagnostico, &domain.Error{Codigo: "adjudicacion_salida_no_disponible", Campo: "documento"})
	}
	return 0
}
func informarError(w io.Writer, err error) int {
	e, ok := err.(*domain.Error)
	if !ok {
		e = &domain.Error{Codigo: "adjudicacion_error", Campo: "documento"}
	}
	_ = json.NewEncoder(w).Encode(e)
	return 1
}
func main() { os.Exit(ejecutar(os.Stdin, os.Stdout, os.Stderr)) }
