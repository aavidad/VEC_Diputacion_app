package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"vec-diputacion-granada/internal/app/separacionportales"
)

const subcomandoComprobarSeparacion = "comprobar-separacion-portales"

const tamanoMaximoGuionEntorno = 256 << 10

var errArgumentosSeparacion = errors.New("argumentos de comprobar-separacion-portales invalidos")

// subcomandosSoloInterno operan sobre datos de RRHH (bolsas, CONVOCA,
// Dietas). Un proceso externo no los ejecuta aunque alguien lo intente dentro
// de su contenedor.
var subcomandosSoloInterno = map[string]struct{}{
	"rellenar-vinculos-bolsa":     {},
	"comprobar-dietas":            {},
	"publicar-proyeccion-publica": {},
	"constituir-bolsa":            {},
	"importar-convoca":            {},
}

// comprobarSubcomandoEnPortal impide ejecutar tareas internas desde el
// proceso externo. Un valor de portal no válido también lo impide.
func comprobarSubcomandoEnPortal(args []string, valorPortal string) error {
	if len(args) < 2 {
		return nil
	}
	if _, interno := subcomandosSoloInterno[args[1]]; !interno {
		return nil
	}
	portal, err := separacionportales.Parsear(valorPortal)
	if err != nil {
		return err
	}
	if portal == separacionportales.PortalExterno {
		return fmt.Errorf("%s: tarea del portal interno en un proceso externo", args[1])
	}
	return nil
}

// ejecutarComprobacionSeparacion la lanza quien despliega, con acceso a los
// dos directorios de material y a los dos guiones de arranque. No imprime
// ningún valor: solo recuentos y, si falla, el nombre del elemento repetido.
func ejecutarComprobacionSeparacion(args []string, salida io.Writer) error {
	opciones := flag.NewFlagSet(subcomandoComprobarSeparacion, flag.ContinueOnError)
	opciones.SetOutput(io.Discard)
	materialInterno := opciones.String("material-interno", "", "directorio de material del proceso interno")
	materialExterno := opciones.String("material-externo", "", "directorio de material del proceso externo")
	entornoInterno := opciones.String("entorno-interno", "", "guion que exporta las variables del proceso interno")
	entornoExterno := opciones.String("entorno-externo", "", "guion que exporta las variables del proceso externo")
	if err := opciones.Parse(args); err != nil || opciones.NArg() != 0 || *materialInterno == "" || *materialExterno == "" {
		return errArgumentosSeparacion
	}
	interno := separacionportales.Proceso{Material: *materialInterno}
	externo := separacionportales.Proceso{Material: *materialExterno}
	var err error
	if interno.Entorno, err = leerGuionEntorno(*entornoInterno); err != nil {
		return err
	}
	if externo.Entorno, err = leerGuionEntorno(*entornoExterno); err != nil {
		return err
	}
	informe, err := separacionportales.ComprobarSeparacion(interno, externo)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(salida, "separacion_portales=correcta secretos_interno=%d secretos_externo=%d usuarios_interno=%d usuarios_externo=%d\n",
		informe.SecretosInterno, informe.SecretosExterno, informe.UsuariosInterno, informe.UsuariosExterno)
	return err
}

// leerGuionEntorno lee un guion de arranque acotado. Sin ruta no hay nada que
// comparar y se devuelve vacío.
func leerGuionEntorno(ruta string) ([]byte, error) {
	if ruta == "" {
		return nil, nil
	}
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Size() > tamanoMaximoGuionEntorno {
		return nil, fmt.Errorf("%s: guion de entorno no valido", subcomandoComprobarSeparacion)
	}
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, fmt.Errorf("%s: guion de entorno no legible", subcomandoComprobarSeparacion)
	}
	defer fichero.Close()
	contenido, err := io.ReadAll(io.LimitReader(fichero, tamanoMaximoGuionEntorno+1))
	if err != nil || len(contenido) > tamanoMaximoGuionEntorno {
		return nil, fmt.Errorf("%s: guion de entorno no valido", subcomandoComprobarSeparacion)
	}
	return contenido, nil
}
