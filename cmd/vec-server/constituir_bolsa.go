package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

type argumentosConstituirBolsa struct {
	fichero, categoria string
}

func leerArgumentosConstituirBolsa(args []string, salida io.Writer) (argumentosConstituirBolsa, error) {
	f := flag.NewFlagSet("constituir-bolsa", flag.ContinueOnError)
	f.SetOutput(salida)
	var a argumentosConstituirBolsa
	f.StringVar(&a.fichero, "fichero", "", "XLS de Convoca ya importado (identifica el acta por su huella)")
	f.StringVar(&a.categoria, "categoria", "", "clave RPT con la que se importó")
	if err := f.Parse(args); err != nil {
		return a, err
	}
	a.fichero = strings.TrimSpace(a.fichero)
	a.categoria = strings.TrimSpace(a.categoria)
	if a.fichero == "" || a.categoria == "" || f.NArg() != 0 {
		return a, fmt.Errorf("uso: vec-server constituir-bolsa --fichero X.xls --categoria <clave-rpt>")
	}
	return a, nil
}
