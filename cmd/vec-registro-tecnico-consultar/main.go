package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/web"
)

type textosConsulta struct {
	Esquema    string `json:"esquema"`
	Ayuda      string `json:"ayuda"`
	Terminada  string `json:"terminada"`
	Parcial    string `json:"parcial"`
	Parametros string `json:"parametros"`
	Archivo    string `json:"archivo"`
	Salida     string `json:"salida"`
}

func cargarTextos(idioma string) (textosConsulta, error) {
	datos, err := web.TextosRegistroTecnicoConsulta(idioma)
	if err != nil {
		return textosConsulta{}, err
	}
	var textos textosConsulta
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(&textos) != nil || d.Decode(new(any)) != io.EOF || textos.Esquema != "1" ||
		textos.Ayuda == "" || textos.Terminada == "" || textos.Parcial == "" ||
		textos.Parametros == "" || textos.Archivo == "" || textos.Salida == "" {
		return textosConsulta{}, os.ErrInvalid
	}
	return textos, nil
}

type archivosFlag []string

func (a *archivosFlag) String() string { return "" }
func (a *archivosFlag) Set(valor string) error {
	if len(*a) >= maxArchivosConsulta {
		return os.ErrInvalid
	}
	*a = append(*a, valor)
	return nil
}

type opcionesConsulta struct {
	Desde, Hasta time.Time
	Codigo       domain.CodigoIncidenciaTecnica
	Ruta         string
	Archivos     []string
}

func validarOpciones(desde, hasta, codigo, ruta string, archivos archivosFlag) (opcionesConsulta, error) {
	inicio, errInicio := time.Parse(time.RFC3339, desde)
	fin, errFin := time.Parse(time.RFC3339, hasta)
	if errInicio != nil || errFin != nil || !fin.After(inicio) || fin.Sub(inicio) > 31*24*time.Hour ||
		len(archivos) == 0 || len(archivos) > maxArchivosConsulta || (codigo != "" && ruta != "") {
		return opcionesConsulta{}, os.ErrInvalid
	}
	if codigo != "" {
		if _, ok := domain.DefinicionIncidenciaTecnicaDe(domain.CodigoIncidenciaTecnica(codigo)); !ok {
			return opcionesConsulta{}, os.ErrInvalid
		}
	}
	if ruta != "" && !rutaFija(ruta) {
		return opcionesConsulta{}, os.ErrInvalid
	}
	vistos := make(map[string]bool, len(archivos))
	for _, rutaArchivo := range archivos {
		if !filepath.IsAbs(rutaArchivo) || filepath.Clean(rutaArchivo) != rutaArchivo || vistos[rutaArchivo] {
			return opcionesConsulta{}, os.ErrInvalid
		}
		vistos[rutaArchivo] = true
	}
	return opcionesConsulta{Desde: inicio.UTC(), Hasta: fin.UTC(), Codigo: domain.CodigoIncidenciaTecnica(codigo), Ruta: ruta, Archivos: archivos}, nil
}

var errSalidaConsultaNoDisponible = errors.New("consulta_tecnica_salida_no_disponible")

func escribirJSON(destino io.Writer, valor any) error {
	var linea bytes.Buffer
	if err := json.NewEncoder(&linea).Encode(valor); err != nil {
		return errors.Join(errSalidaConsultaNoDisponible, err)
	}
	n, err := destino.Write(linea.Bytes())
	if err != nil {
		return errors.Join(errSalidaConsultaNoDisponible, err)
	}
	if n != linea.Len() {
		return errors.Join(errSalidaConsultaNoDisponible, io.ErrShortWrite)
	}
	return nil
}

func respuestaError(destino io.Writer, mensaje string) (int, error) {
	err := escribirJSON(destino, struct {
		Estado  string `json:"estado"`
		Mensaje string `json:"mensaje"`
	}{Estado: "error", Mensaje: mensaje})
	return 2, err
}

func ejecutar(args []string, salida, diagnostico io.Writer) (int, error) {
	if salida == nil || diagnostico == nil {
		return 2, errSalidaConsultaNoDisponible
	}
	f := flag.NewFlagSet("vec-registro-tecnico-consultar", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var archivos archivosFlag
	f.Var(&archivos, "archivo", "")
	desde := f.String("desde", "", "")
	hasta := f.String("hasta", "", "")
	codigo := f.String("codigo", "", "")
	ruta := f.String("ruta", "", "")
	idioma := f.String("idioma", "", "")
	ayuda := f.Bool("ayuda", false, "")
	errorParametros := f.Parse(args)
	textos, errTextos := cargarTextos(*idioma)
	if errTextos != nil {
		textos, errTextos = cargarTextos("")
		if errTextos != nil {
			return 2, errTextos
		}
		errorParametros = os.ErrInvalid
	}
	if errorParametros != nil || f.NArg() != 0 {
		return respuestaError(diagnostico, textos.Parametros)
	}
	if *ayuda {
		if err := escribirJSON(salida, struct {
			Ayuda string `json:"ayuda"`
		}{textos.Ayuda}); err != nil {
			codigo, avisoErr := respuestaError(diagnostico, textos.Salida)
			return codigo, errors.Join(err, avisoErr)
		}
		return 0, nil
	}
	opciones, errOpciones := validarOpciones(*desde, *hasta, *codigo, *ruta, archivos)
	if errOpciones != nil {
		return respuestaError(diagnostico, textos.Parametros)
	}
	catalogo, err := catalogoincidencias.Predeterminado()
	if err != nil {
		return respuestaError(diagnostico, textos.Archivo)
	}
	resumen, err := consultarArchivos(opciones, catalogo)
	if err != nil {
		return respuestaError(diagnostico, textos.Archivo)
	}
	if resumen.Rechazadas > 0 {
		resumen.Estado, resumen.Mensaje = "parcial", textos.Parcial
	} else {
		resumen.Estado, resumen.Mensaje = "terminada", textos.Terminada
	}
	if err := escribirJSON(salida, resumen); err != nil {
		codigo, avisoErr := respuestaError(diagnostico, textos.Salida)
		return codigo, errors.Join(err, avisoErr)
	}
	if resumen.Rechazadas > 0 {
		return 3, nil
	}
	return 0, nil
}

func main() {
	codigo, err := ejecutar(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		os.Exit(2)
	}
	os.Exit(codigo)
}
