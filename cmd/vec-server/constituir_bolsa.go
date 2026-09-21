package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
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

// describirSustituidas resume las bolsas que la constitución deja extinguidas (B9).
func describirSustituidas(sustituidas []ports.BolsaSustituida) string {
	if len(sustituidas) == 0 {
		return "ninguna"
	}
	partes := make([]string, 0, len(sustituidas))
	for _, s := range sustituidas {
		partes = append(partes, fmt.Sprintf("%s@%d", s.BolsaRef, s.VersionBolsa))
	}
	return strings.Join(partes, ",")
}
