package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/administracion/adapters/releasebinario"
)

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout)) }

func ejecutar(argumentos []string, salida io.Writer) int {
	fs := flag.NewFlagSet("vec-copias-inspeccionar-binario", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	raiz := fs.String("raiz", "", "")
	ruta := fs.String("binario", "", "")
	maxBytes := fs.Int64("max-bytes", 0, "")
	if err := fs.Parse(argumentos); err != nil || fs.NArg() != 0 || *raiz == "" || *ruta == "" || *maxBytes <= 0 {
		_ = json.NewEncoder(salida).Encode(struct {
			Estado   string `json:"estado"`
			Campo    string `json:"campo"`
			Esperado string `json:"esperado"`
			Obtenido string `json:"obtenido"`
		}{"no_comprobable", "argumentos", "raiz_binario_y_limite_explicitos", "ausentes_o_invalidos"})
		return 2
	}
	informe := releasebinario.Inspeccionar(*raiz, *ruta, *maxBytes)
	if err := json.NewEncoder(salida).Encode(informe); err != nil {
		return 2
	}
	if informe.Estado == "no_comprobable" {
		return 1
	}
	return 0
}
