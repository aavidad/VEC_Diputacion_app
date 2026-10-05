package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/administracion/adapters/paquetestemas"
	"vec-diputacion-granada/internal/modules/administracion/adapters/temascss"
	"vec-diputacion-granada/internal/modules/administracion/domain/temas"
)

// MotivoFalloDiagnostico se observa por el estado de salida cuando el propio
// canal de diagnóstico no permite escribir su JSON; no expone el error recibido.
const MotivoFalloDiagnostico = 2

func ejecutar(args []string, entrada io.Reader, salida, diagnostico io.Writer) int {
	fallar := func(clave string) int {
		b, err := json.Marshal(struct {
			ErrorClave string `json:"error_clave"`
		}{clave})
		if err != nil {
			return MotivoFalloDiagnostico
		}
		b = append(b, '\n')
		n, err := diagnostico.Write(b)
		if err != nil || n != len(b) {
			return MotivoFalloDiagnostico
		}
		return 1
	}
	f := flag.NewFlagSet("", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	politicaRuta := f.String("politica", "", "")
	huellaEsperada := f.String("politica-sha256", "", "")
	salidaCSS := f.Bool("css", false, "")
	if f.Parse(args) != nil || f.NArg() != 0 || *politicaRuta == "" || *huellaEsperada == "" {
		return fallar("temas.paquetes.error.argumentos")
	}
	// La ruta es una elección local explícita de quien ejecuta la CLI. No
	// procede del paquete y no se imprime en el diagnóstico.
	archivo, err := os.Open(*politicaRuta) // #nosec G703 -- entrada local explícita, sólo lectura
	if err != nil {
		return fallar("temas.paquetes.error.entrada")
	}
	politica, err := paquetestemas.LeerPolitica(archivo, *huellaEsperada)
	errCierre := archivo.Close()
	if err == nil {
		err = errCierre
	}
	if err != nil {
		return fallar(claveError(err))
	}
	resultado, err := paquetestemas.Preparar(entrada, politica)
	if err != nil {
		return fallar(claveError(err))
	}
	// Codificar antes de escribir evita producir un documento parcial por
	// errores de serialización; una escritura fallida nunca devuelve éxito.
	var b []byte
	if *salidaCSS {
		hoja, err := temascss.Generar(resultado.Material, politica.Copia())
		if err != nil {
			return fallar(claveError(err))
		}
		b = hoja.Contenido
	} else {
		b, err = json.Marshal(resultado)
		if err != nil {
			return fallar("temas.paquetes.error.salida")
		}
		b = append(b, '\n')
	}
	n, err := salida.Write(b)
	if err != nil || n != len(b) {
		return fallar("temas.paquetes.error.salida")
	}
	return 0
}

func claveError(err error) string {
	switch {
	case errors.Is(err, paquetestemas.ErrHuella):
		return "temas.paquetes.error.huella_politica"
	case errors.Is(err, temas.ErrPolitica):
		return "temas.paquetes.error.politica"
	case errors.Is(err, temas.ErrContraste):
		return "temas.paquetes.error.contraste"
	case errors.Is(err, temas.ErrPaquete):
		return "temas.paquetes.error.paquete"
	default:
		return "temas.paquetes.error.entrada"
	}
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
