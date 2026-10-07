package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/incompatibilidades/domain"
)

const maximoEntrada = 16 * 1024

type resultado struct {
	Status string   `json:"status"`
	Fields []string `json:"fields,omitempty"`
}

func main() {
	os.Exit(ejecutar(os.Stdin, os.Stdout))
}

// ejecutar es una precomprobación local de hechos sintéticos. No almacena datos,
// resuelve identidades, genera recibos ni emite actos administrativos.
func ejecutar(entrada io.Reader, salida io.Writer) int {
	limitada := io.LimitReader(entrada, maximoEntrada+1)
	contenido, err := io.ReadAll(limitada)
	if err != nil || len(contenido) > maximoEntrada || !utf8.Valid(contenido) {
		return escribir(salida, resultado{Status: "invalid_input"}, 2)
	}
	var d domain.DeclaracionActividad
	if !objetoSinClavesDuplicadas(contenido) {
		return escribir(salida, resultado{Status: "invalid_input"}, 2)
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&d) != nil || dec.Decode(new(any)) != io.EOF {
		return escribir(salida, resultado{Status: "invalid_input"}, 2)
	}
	if !referenciasDeEnsayo(d) {
		return escribir(salida, resultado{Status: "invalid_input"}, 2)
	}
	if campos := d.CamposInvalidos(); len(campos) > 0 {
		return escribir(salida, resultado{Status: "incomplete", Fields: campos}, 1)
	}
	return escribir(salida, resultado{Status: "structurally_complete"}, 0)
}

func referenciasDeEnsayo(d domain.DeclaracionActividad) bool {
	for _, ref := range []string{d.ActividadRef, d.FuncionesRef, d.TitularRef, d.JornadaRef, d.HorarioRef} {
		if ref != "" && !strings.HasPrefix(ref, "ensayo:") {
			return false
		}
	}
	return true
}

func objetoSinClavesDuplicadas(contenido []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(contenido))
	inicio, err := dec.Token()
	if err != nil || inicio != json.Delim('{') {
		return false
	}
	vistas := make(map[string]bool)
	for dec.More() {
		clave, err := dec.Token()
		if err != nil {
			return false
		}
		nombre, ok := clave.(string)
		if !ok || vistas[nombre] {
			return false
		}
		vistas[nombre] = true
		var valor json.RawMessage
		if dec.Decode(&valor) != nil {
			return false
		}
	}
	fin, err := dec.Token()
	return err == nil && fin == json.Delim('}')
}

func escribir(salida io.Writer, r resultado, codigo int) int {
	if json.NewEncoder(salida).Encode(r) != nil {
		return 2
	}
	return codigo
}
