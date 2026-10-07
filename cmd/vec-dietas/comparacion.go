package main

import (
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/dietas/adapters/informeliquidacion"
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

func ejecutarInformeComparacion(in io.Reader, out io.Writer, rutaTextos, rutaTema string) int {
	e, err := leerEntradaTipada[preparacionliquidacion.EntradaComparacion](in)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	comparacion, err := preparacionliquidacion.Comparar(e)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	datos, err := leerArchivoInforme(rutaTextos)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	textos, err := informeliquidacion.CargarTextosComparacion(datos)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	tema, err := leerArchivoInforme(rutaTema)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	renderizador, err := informeliquidacion.Nuevo(informeliquidacion.Configuracion{TemaCSS: string(tema)})
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	html, err := renderizador.RenderizarComparacion(comparacion, textos)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	escritos, err := out.Write(html)
	if err != nil {
		return informarFalloInforme(err)
	}
	if escritos != len(html) {
		return informarFalloInforme(io.ErrShortWrite)
	}
	return 0
}
