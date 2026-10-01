package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/administracion/adapters/inventariocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout))
}

// La interfaz devuelve JSON de máquina. Los mensajes para personas pertenecen
// a los catálogos de ADMIN; no se imprimen rutas, errores de sistema ni contenido.
func ejecutar(argumentos []string, salida io.Writer) int {
	opciones := flag.NewFlagSet("vec-copias-inventariar", flag.ContinueOnError)
	opciones.SetOutput(io.Discard)
	opciones.Usage = func() {}
	descriptorRuta := opciones.String("descriptor", "", "")
	observadoRuta := opciones.String("observado", "", "")
	raiz := opciones.String("raiz", "", "")
	if err := opciones.Parse(argumentos); err != nil || opciones.NArg() != 0 || *descriptorRuta == "" || *observadoRuta == "" || *raiz == "" {
		return emitirError(salida, "argumentos_no_validos")
	}
	descriptorArchivo, err := abrirDocumento(*descriptorRuta)
	if err != nil {
		return emitirError(salida, "descriptor_no_disponible")
	}
	descriptor, err := inventariocopias.LeerDescriptor(descriptorArchivo)
	_ = descriptorArchivo.Close()
	if err != nil {
		return emitirError(salida, "descriptor_no_valido")
	}
	observadoArchivo, err := abrirDocumento(*observadoRuta)
	if err != nil {
		return emitirError(salida, "observado_no_disponible")
	}
	observado, err := inventariocopias.LeerInventario(observadoArchivo)
	_ = observadoArchivo.Close()
	if err != nil {
		return emitirError(salida, "observado_no_valido")
	}
	informe := inventariocopias.Inventariar(*raiz, descriptor, observado)
	if err := json.NewEncoder(salida).Encode(informe); err != nil {
		return 2
	}
	if informe.Resultado.Estado != copias.Compatible {
		return 1
	}
	return 0
}

func emitirError(salida io.Writer, codigo string) int {
	campo := "argumentos"
	if codigo == "descriptor_no_disponible" || codigo == "descriptor_no_valido" {
		campo = "descriptor"
	} else if codigo == "observado_no_disponible" || codigo == "observado_no_valido" {
		campo = "observado"
	}
	_ = json.NewEncoder(salida).Encode(struct {
		Estado string       `json:"estado"`
		Razon  copias.Razon `json:"razon"`
	}{Estado: "no_comprobable", Razon: copias.Razon{Codigo: codigo, Clave: campo, Esperado: "entrada_local_valida", Obtenido: "ausente_o_invalida", Accion: "revisar_entrada_autorizada"}})
	return 2
}

func abrirDocumento(ruta string) (*os.File, error) {
	// #nosec G304 -- Rutas elegidas por el operador de esta CLI offline, sin HTTP
	// ni entradas remotas. Solo lectura de JSON regular y acotado; no se muestran.
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, inventariocopias.ErrDocumento
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, inventariocopias.ErrDocumento
	}
	return f, nil
}
