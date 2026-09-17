package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

type argumentosImportarConvoca struct {
	fichero, categoria, bolsaRef string
	admitirRechazos              bool
}

func leerArgumentosImportarConvoca(args []string, salida io.Writer) (argumentosImportarConvoca, error) {
	f := flag.NewFlagSet("importar-convoca", flag.ContinueOnError)
	f.SetOutput(salida)
	var a argumentosImportarConvoca
	f.StringVar(&a.fichero, "fichero", "", "XLS de Convoca")
	f.StringVar(&a.categoria, "categoria", "", "clave RPT")
	f.StringVar(&a.bolsaRef, "bolsa-ref", "", "referencia de bolsa")
	f.BoolVar(&a.admitirRechazos, "admitir-rechazos", false, "conservar lote con filas rechazadas")
	if err := f.Parse(args); err != nil {
		return a, err
	}
	a.fichero = strings.TrimSpace(a.fichero)
	a.categoria = strings.TrimSpace(a.categoria)
	a.bolsaRef = strings.TrimSpace(a.bolsaRef)
	if a.fichero == "" || a.categoria == "" || f.NArg() != 0 {
		return a, fmt.Errorf("uso: vec-server importar-convoca --fichero X.xls --categoria <clave-rpt> [--bolsa-ref referencia] [--admitir-rechazos]")
	}
	return a, nil
}
