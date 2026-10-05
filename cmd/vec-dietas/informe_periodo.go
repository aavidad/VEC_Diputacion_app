package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"unicode/utf8"

	csvinforme "vec-diputacion-granada/internal/modules/dietas/adapters/informeperiodo"
	"vec-diputacion-granada/internal/modules/dietas/application/informeperiodo"
)

var errArgumentosInformePeriodo = errors.New("argumentos_no_admitidos")

// ejecutarInformePeriodoCSV lee el paquete sintético por la entrada estándar y
// escribe la selección en CSV. Catálogo y configuración son ficheros locales
// elegidos por el operador; los filtros son los mismos que ofrece la vista.
func ejecutarInformePeriodoCSV(args []string, in io.Reader, out io.Writer) int {
	fs := flag.NewFlagSet("informe-periodo-csv", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	textos := fs.String("textos", "", "")
	configuracion := fs.String("configuracion", "", "")
	var f informeperiodo.Filtros
	fs.StringVar(&f.Persona, "persona", "", "")
	fs.StringVar(&f.Unidad, "unidad", "", "")
	fs.StringVar(&f.Situacion, "situacion", "", "")
	fs.StringVar(&f.Desde, "desde", "", "")
	fs.StringVar(&f.Hasta, "hasta", "", "")
	if fs.Parse(args) != nil || fs.NArg() != 0 || *textos == "" || *configuracion == "" {
		return escribirFalloPreparacion(out, errArgumentosInformePeriodo)
	}
	rawCatalogo, err := leerArchivoInforme(*textos)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	catalogo, err := csvinforme.CargarCatalogo(bytes.NewReader(rawCatalogo))
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	rawConfiguracion, err := leerArchivoInforme(*configuracion)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	var cfg informeperiodo.Configuracion
	if decodificarEstricto(bytes.NewReader(rawConfiguracion), &cfg) != nil {
		return escribirFalloPreparacion(out, informeperiodo.ErrDatosInvalidos)
	}
	var datos informeperiodo.Datos
	if decodificarEstricto(io.LimitReader(in, limiteEntrada+1), &datos) != nil {
		return escribirFalloPreparacion(out, informeperiodo.ErrDatosInvalidos)
	}
	informe, err := informeperiodo.Preparar(cfg, datos, f)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	contenido, err := csvinforme.Escribir(catalogo, informe)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	escritos, err := out.Write(contenido)
	if err != nil {
		return informarFalloInforme(err)
	}
	if escritos != len(contenido) {
		return informarFalloInforme(io.ErrShortWrite)
	}
	return 0
}

// decodificarEstricto exige UTF-8, un único valor JSON y ningún campo desconocido.
func decodificarEstricto(r io.Reader, destino any) error {
	b, err := io.ReadAll(r)
	if err != nil || len(b) > limiteEntrada || !utf8.Valid(b) {
		return errJSON
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(destino) != nil || dec.Decode(new(any)) != io.EOF {
		return errJSON
	}
	return nil
}
