package main

import (
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/dietas/application/preparacionliquidacion"
)

func ejecutarComparacion(in io.Reader, out io.Writer) int {
	e, err := leerEntradaTipada[preparacionliquidacion.EntradaComparacion](in)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	c, err := preparacionliquidacion.Comparar(e)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	if json.NewEncoder(out).Encode(c) != nil {
		return 1
	}
	return 0
}
