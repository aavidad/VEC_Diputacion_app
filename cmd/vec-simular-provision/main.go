// vec-simular-provision ejecuta un ejercicio local con JSON sintético. No
// conecta Personal, RPT, autorización, persistencia, registro ni firma.
package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func ejecutar(args []string, entrada io.Reader, salida, diagnostico io.Writer) int {
	flags := flag.NewFlagSet("vec-simular-provision", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	archivo := flags.String("entrada", "-", "")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return emitirError(diagnostico, &domain.Error{Codigo: "argumentos_invalidos", Campo: "entrada"})
	}
	if *archivo != "-" {
		info, err := os.Stat(*archivo)
		if err != nil || !info.Mode().IsRegular() {
			return emitirError(diagnostico, &domain.Error{Codigo: "archivo_entrada_invalido", Campo: "entrada"})
		}
		if info.Size() > simulacion.MaximoBytes {
			return emitirError(diagnostico, &domain.Error{Codigo: "json_excesivo", Campo: "documento"})
		}
		// #nosec G304 -- CLI local: el operador elige explícitamente el archivo
		// sintético; no es una ruta recibida por HTTP ni un almacén de producto.
		f, err := os.Open(*archivo)
		if err != nil {
			return emitirError(diagnostico, &domain.Error{Codigo: "archivo_entrada_invalido", Campo: "entrada"})
		}
		defer func() { _ = f.Close() }()
		entrada = f
	}
	p, err := simulacion.DecodificarProceso(entrada)
	if err != nil {
		return emitirError(diagnostico, err)
	}
	r, err := application.SimularProceso(p)
	if err != nil {
		return emitirError(diagnostico, err)
	}
	if err := json.NewEncoder(salida).Encode(r); err != nil {
		return emitirError(diagnostico, &domain.Error{Codigo: "salida_json_fallida", Campo: "documento"})
	}
	return 0
}

func emitirError(salida io.Writer, err error) int {
	e, ok := err.(*domain.Error)
	if !ok {
		e = &domain.Error{Codigo: "simulacion_fallida", Campo: "documento"}
	}
	// Algunos errores del motor usan como campo un ID editable de regla.
	// El diagnóstico CLI sólo publica nombres técnicos de esta lista cerrada.
	campo := "documento"
	switch e.Campo {
	case "entrada", "documento", "proceso", "solicitud", "instantanea", "preferencias", "puestos", "requisitos", "valoraciones":
		campo = e.Campo
	}
	nominal := &domain.Error{Codigo: e.Codigo, Campo: campo}
	_ = json.NewEncoder(salida).Encode(struct {
		Error *domain.Error `json:"error"`
	}{nominal})
	return 1
}
