package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/administracion/adapters/releasebinario"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
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
		return responder(salida, releasebinario.Informe{
			Estado: "no_comprobable", Autenticidad: "no_comprobada",
			Razones: []copias.Razon{{Codigo: "argumentos_no_validos", Clave: "argumentos", Esperado: "raiz_binario_y_limite_explicitos", Obtenido: "ausentes_o_invalidos", Accion: "revisar_argumentos"}},
		}, 2)
	}
	informe := releasebinario.Inspeccionar(*raiz, *ruta, *maxBytes)
	codigo := 0
	if informe.Estado == "no_comprobable" {
		codigo = 1
	}
	return responder(salida, informe, codigo)
}

func responder(salida io.Writer, informe releasebinario.Informe, codigo int) int {
	if json.NewEncoder(salida).Encode(informe) != nil {
		return 4
	}
	return codigo
}
