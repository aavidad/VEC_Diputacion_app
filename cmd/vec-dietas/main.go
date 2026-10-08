package main

import (
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/dietas/application/preparacionliquidacion"
	"vec-diputacion-granada/internal/modules/dietas/application/simulaciondevengo"
	"vec-diputacion-granada/internal/modules/dietas/domain"
)

type fallo struct {
	Codigo string `json:"codigo"`
}

func main() {
	os.Exit(ejecutarConArgumentos(os.Args[1:], os.Stdin, os.Stdout))
}
func ejecutarConArgumentos(args []string, in io.Reader, out io.Writer) int {
	if len(args) > 0 && args[0] == "--informe-periodo-csv" {
		return ejecutarInformePeriodoCSV(args[1:], in, out)
	}
	if len(args) > 0 && args[0] == "--informe-periodo-pdf" {
		return ejecutarInformePeriodoPDF(args[1:], in, out)
	}
	if len(args) == 6 && args[0] == "--comparar-liquidaciones" && args[1] == "--informe" && args[2] == "--textos" && args[4] == "--tema" {
		return ejecutarInformeComparacion(in, out, args[3], args[5])
	}
	if len(args) == 1 && args[0] == "--comparar-liquidaciones" {
		return ejecutarComparacion(in, out)
	}
	if len(args) == 7 && args[0] == "--preparar-liquidacion" && args[1] == "--desde-instantanea" && args[2] == "--informe" && args[3] == "--textos" && args[5] == "--tema" {
		return ejecutarInformeRecuperacion(in, out, args[4], args[6])
	}
	if len(args) == 2 && args[0] == "--preparar-liquidacion" && args[1] == "--desde-instantanea" {
		return ejecutarRecuperacion(in, out)
	}
	if len(args) == 6 && args[0] == "--preparar-liquidacion" && args[1] == "--informe" && args[2] == "--textos" && args[4] == "--tema" {
		return ejecutarInformePreparacion(in, out, args[3], args[5])
	}
	if len(args) == 1 && args[0] == "--preparar-liquidacion" {
		return ejecutarPreparacion(in, out)
	}
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

func ejecutarPreparacion(in io.Reader, out io.Writer) int {
	e, err := leerEntradaTipada[preparacionliquidacion.Entrada](in)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	p, err := preparacionliquidacion.Preparar(e)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	if json.NewEncoder(out).Encode(p.Instantanea()) != nil {
		return 1
	}
	return 0
}
func escribirFalloPreparacion(out io.Writer, err error) int {
	if json.NewEncoder(out).Encode(fallo{err.Error()}) != nil {
		return 1
	}
	return 2
}

func leerRecuperacion(in io.Reader) (*domain.PreparacionLiquidacion, error) {
	s, err := leerEntradaTipada[domain.InstantaneaLiquidacionPropuesta](in)
	if err != nil {
		return nil, err
	}
	return preparacionliquidacion.Recuperar(s)
}

func ejecutarRecuperacion(in io.Reader, out io.Writer) int {
	p, err := leerRecuperacion(in)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	if json.NewEncoder(out).Encode(p.Instantanea()) != nil {
		return 1
	}
	return 0
}
