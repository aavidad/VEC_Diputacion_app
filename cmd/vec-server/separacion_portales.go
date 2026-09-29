package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"

	"vec-diputacion-granada/internal/app/separacionportales"
)

const subcomandoComprobarSeparacion = "comprobar-separacion-portales"

const tamanoMaximoGuionEntorno = 256 << 10

var errArgumentosSeparacion = errors.New("argumentos de comprobar-separacion-portales invalidos")

// subcomandosSoloInterno operan sobre datos de RRHH (bolsas, CONVOCA,
// Dietas). Un proceso externo no los ejecuta aunque alguien lo intente dentro
// de su contenedor.
var subcomandosSoloInterno = map[string]struct{}{
	"rellenar-vinculos-bolsa":       {},
	"comprobar-dietas":              {},
	"publicar-proyeccion-publica":   {},
	"constituir-bolsa":              {},
	"importar-convoca":              {},
	subcomandoPrepararPortalExterno: {},
}

// subcomandosPortalExterno es la lista positiva de tareas que el proceso
// externo puede ejecutar: solo calcular sus propios alias, sin conexiones.
var subcomandosPortalExterno = map[string]struct{}{
	subcomandoExportarSeudonimosPortalExterno: {},
}

// comprobarSubcomandoEnPortal: el proceso externo solo ejecuta el servidor y
// las tareas de su lista positiva, así que una tarea nueva tampoco corre allí
// por descuido. Un valor de portal no válido impide las tareas internas.
func comprobarSubcomandoEnPortal(args []string, valorPortal string) error {
	if len(args) < 2 {
		return nil
	}
	portal, err := separacionportales.Parsear(valorPortal)
	if err != nil {
		if _, interno := subcomandosSoloInterno[args[1]]; interno {
			return err
		}
		return nil
	}
	if portal == separacionportales.PortalExterno {
		if _, admitido := subcomandosPortalExterno[args[1]]; !admitido {
			return errors.New("el proceso externo no ejecuta esa tarea")
		}
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

const subcomandoPrepararPortalExterno = "preparar-portal-externo"

var errArgumentosPreparacion = errors.New("argumentos de preparar-portal-externo invalidos")

// ejecutarPreparacionPortalExterno se lanza en el lado interno (con su
// entorno y su material): publica el gobierno de las audiencias externas
// pedidas y deja en el material del proceso externo solo sus claves
// derivadas. Imprime recuentos, nunca claves.
func ejecutarPreparacionPortalExterno(ctx context.Context, args []string, salida io.Writer, cfg config.Config,
	preparar func(context.Context, config.Config, string, []string, []byte) (bootstrap.ResumenPreparacionPortalExterno, error),
) error {
	opciones := flag.NewFlagSet(subcomandoPrepararPortalExterno, flag.ContinueOnError)
	opciones.SetOutput(io.Discard)
	destino := opciones.String("material-externo", "", "directorio de material del proceso externo")
	consumidores := opciones.String("consumidores", "", "consumidores separados por comas")
	rutaSeudonimos := opciones.String("seudonimos", "", "fichero con los alias calculados por el proceso externo")
	if err := opciones.Parse(args); err != nil || opciones.NArg() != 0 || *destino == "" || *consumidores == "" {
		return errArgumentosPreparacion
	}
	seudonimos, err := leerGuionEntorno(*rutaSeudonimos)
	if err != nil {
		return err
	}
	resumen, err := preparar(ctx, cfg, *destino, strings.Split(*consumidores, ","), seudonimos)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(salida, "preparacion_portal_externo=correcta consumidores=%s claves=%d alias=%d configuracion=%s\n",
		strings.Join(resumen.Consumidores, ","), resumen.Claves, resumen.Alias, resumen.Configuracion)
	return err
}

const subcomandoExportarSeudonimosPortalExterno = "exportar-seudonimos-portal-externo"

// ejecutarExportacionSeudonimos se lanza en el proceso externo: escribe en
// la salida los alias HMAC de sus cuentas para que el lado interno los
// registre. No abre conexiones ni imprime claves.
func ejecutarExportacionSeudonimos(salida io.Writer, cfg config.Config, exportar func(config.Config) ([]byte, error)) error {
	contenido, err := exportar(cfg)
	if err != nil {
		return err
	}
	_, err = salida.Write(append(contenido, '\n'))
	return err
}
