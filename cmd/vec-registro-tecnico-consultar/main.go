package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/domain"
)

//go:embed textos/*.json
var catalogos embed.FS

type textosConsulta struct {
	Esquema    string `json:"esquema"`
	Ayuda      string `json:"ayuda"`
	Terminada  string `json:"terminada"`
	Parcial    string `json:"parcial"`
	Parametros string `json:"parametros"`
	Archivo    string `json:"archivo"`
	Salida     string `json:"salida"`
}

func cargarTextos(idioma string) (textosConsulta, bool) {
	if idioma != "es" && idioma != "en" {
		return textosConsulta{}, false
	}
	datos, err := catalogos.ReadFile("textos/" + idioma + ".json")
	if err != nil {
		return textosConsulta{}, false
	}
	var textos textosConsulta
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(&textos) != nil || d.Decode(new(any)) != io.EOF || textos.Esquema != "1" ||
		textos.Ayuda == "" || textos.Terminada == "" || textos.Parcial == "" ||
		textos.Parametros == "" || textos.Archivo == "" || textos.Salida == "" {
		return textosConsulta{}, false
	}
	return textos, true
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

func validarOpciones(desde, hasta, codigo, ruta string, archivos archivosFlag) (opcionesConsulta, bool) {
	inicio, errInicio := time.Parse(time.RFC3339, desde)
	fin, errFin := time.Parse(time.RFC3339, hasta)
	if errInicio != nil || errFin != nil || !fin.After(inicio) || fin.Sub(inicio) > 31*24*time.Hour ||
		len(archivos) == 0 || len(archivos) > maxArchivosConsulta || (codigo != "" && ruta != "") {
		return opcionesConsulta{}, false
	}
	if codigo != "" {
		if _, ok := domain.DefinicionIncidenciaTecnicaDe(domain.CodigoIncidenciaTecnica(codigo)); !ok {
			return opcionesConsulta{}, false
		}
	}
	if ruta != "" && !rutaFija(ruta) {
		return opcionesConsulta{}, false
	}
	vistos := make(map[string]bool, len(archivos))
	for _, rutaArchivo := range archivos {
		if !filepath.IsAbs(rutaArchivo) || filepath.Clean(rutaArchivo) != rutaArchivo || vistos[rutaArchivo] {
			return opcionesConsulta{}, false
		}
		vistos[rutaArchivo] = true
	}
	return opcionesConsulta{Desde: inicio.UTC(), Hasta: fin.UTC(), Codigo: domain.CodigoIncidenciaTecnica(codigo), Ruta: ruta, Archivos: archivos}, true
}

func respuestaError(destino io.Writer, mensaje string) int {
	_ = json.NewEncoder(destino).Encode(struct {
		Estado  string `json:"estado"`
		Mensaje string `json:"mensaje"`
	}{Estado: "error", Mensaje: mensaje})
	return 2
}

func ejecutar(args []string, salida, diagnostico io.Writer) int {
	if salida == nil || diagnostico == nil {
		return 2
	}
	f := flag.NewFlagSet("vec-registro-tecnico-consultar", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var archivos archivosFlag
	f.Var(&archivos, "archivo", "")
	desde := f.String("desde", "", "")
	hasta := f.String("hasta", "", "")
	codigo := f.String("codigo", "", "")
	ruta := f.String("ruta", "", "")
	idioma := f.String("idioma", "es", "")
	ayuda := f.Bool("ayuda", false, "")
	errorParametros := f.Parse(args)
	textos, ok := cargarTextos(*idioma)
	if !ok {
		textos, _ = cargarTextos("es")
		errorParametros = os.ErrInvalid
	}
	if errorParametros != nil || f.NArg() != 0 {
		return respuestaError(diagnostico, textos.Parametros)
	}
	if *ayuda {
		if json.NewEncoder(salida).Encode(struct {
			Ayuda string `json:"ayuda"`
		}{textos.Ayuda}) != nil {
			return respuestaError(diagnostico, textos.Salida)
		}
		return 0
	}
	opciones, ok := validarOpciones(*desde, *hasta, *codigo, *ruta, archivos)
	if !ok {
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
	if json.NewEncoder(salida).Encode(resumen) != nil {
		return respuestaError(diagnostico, textos.Salida)
	}
	if resumen.Rechazadas > 0 {
		return 3
	}
	return 0
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr)) }
