package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"regexp"
	"time"
	"unicode/utf8"

	informedoc "vec-diputacion-granada/internal/modules/dietas/adapters/informeperiodo"
	"vec-diputacion-granada/internal/modules/dietas/application/informeperiodo"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
)

var errArgumentosInformePeriodo = errors.New("argumentos_no_admitidos")

// ejecutarInformePeriodoCSV lee el paquete sintético por la entrada estándar y
// escribe la selección en CSV. Catálogo y configuración son ficheros locales
// elegidos por el operador; los filtros son los mismos que ofrece la vista.
func ejecutarInformePeriodoCSV(args []string, in io.Reader, out io.Writer) int {
	return ejecutarInformePeriodo(args, in, out, func(textos []byte, informe informeperiodo.Informe) ([]byte, error) {
		catalogo, err := informedoc.CargarCatalogo(bytes.NewReader(textos))
		if err != nil {
			return nil, err
		}
		return informedoc.Escribir(catalogo, informe)
	})
}

// ejecutarInformePeriodoPDF compone la misma selección con el renderer PDF común.
func ejecutarInformePeriodoPDF(args []string, in io.Reader, out io.Writer) int {
	return ejecutarInformePeriodo(args, in, out, func(textos []byte, informe informeperiodo.Informe) ([]byte, error) {
		catalogo, err := informedoc.CargarCatalogoPDF(bytes.NewReader(textos))
		if err != nil {
			return nil, err
		}
		ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		return informedoc.PrepararPDF(ctx, pdf.Renderizador{Idioma: catalogo.Idioma}, catalogo, informe)
	})
}

func ejecutarInformePeriodo(args []string, in io.Reader, out io.Writer,
	componer func([]byte, informeperiodo.Informe) ([]byte, error)) int {
	fs := flag.NewFlagSet("informe-periodo", flag.ContinueOnError)
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
	contenido, err := componer(rawCatalogo, informe)
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

// decodificarEstricto exige UTF-8, claves en minúscula sin repetir, un único valor JSON y ningún campo desconocido.
func decodificarEstricto(r io.Reader, destino any) error {
	b, err := io.ReadAll(r)
	if err != nil || len(b) > limiteEntrada || !utf8.Valid(b) {
		return errJSON
	}
	if clavesExactas(b) != nil {
		return errJSON
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(destino) != nil || dec.Decode(new(any)) != io.EOF {
		return errJSON
	}
	return nil
}

var patronClaveJSON = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// clavesExactas impide que encoding/json case una clave con otra en distinta
// capitalización o se quede con la última de dos repetidas: todos los
// esquemas de este informe usan claves en minúscula.
func clavesExactas(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	type nivel struct {
		objeto bool
		clave  bool
		vistas map[string]struct{}
	}
	var pila []*nivel
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil || len(pila) > 32 {
			return errJSON
		}
		var actual *nivel
		if len(pila) > 0 {
			actual = pila[len(pila)-1]
		}
		if d, ok := tok.(json.Delim); ok && (d == '}' || d == ']') {
			pila = pila[:len(pila)-1]
			if len(pila) > 0 && pila[len(pila)-1].objeto {
				pila[len(pila)-1].clave = true
			}
			continue
		}
		if actual != nil && actual.objeto && actual.clave {
			clave, _ := tok.(string)
			if _, repetida := actual.vistas[clave]; repetida || !patronClaveJSON.MatchString(clave) {
				return errJSON
			}
			actual.vistas[clave] = struct{}{}
			actual.clave = false
			continue
		}
		if d, ok := tok.(json.Delim); ok {
			pila = append(pila, &nivel{objeto: d == '{', clave: d == '{', vistas: map[string]struct{}{}})
			continue
		}
		if actual != nil && actual.objeto {
			actual.clave = true
		}
	}
}
