package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

const maximoEntradaJSON = 1024 * 1024

var errEntradaJSON = errors.New("seleccion.admision.entrada_invalida")

func leerJSON(r io.Reader, destino any) error {
	return leerJSONHasta(r, destino, maximoEntradaJSON)
}

// leerJSONHasta admite un límite propio para la lista, que reúne muchas revisiones.
func leerJSONHasta(r io.Reader, destino any, limite int) error {
	if r == nil {
		return errEntradaJSON
	}
	material, err := io.ReadAll(io.LimitReader(r, int64(limite)+1))
	if err != nil || len(material) > limite || !utf8.Valid(material) {
		return errEntradaJSON
	}
	tokens := json.NewDecoder(bytes.NewReader(material))
	if clavesUnicas(tokens, 0) != nil {
		return errEntradaJSON
	}
	if _, err := tokens.Token(); err != io.EOF {
		return errEntradaJSON
	}
	decoder := json.NewDecoder(bytes.NewReader(material))
	decoder.DisallowUnknownFields()
	if decoder.Decode(destino) != nil {
		return errEntradaJSON
	}
	return nil
}

// La entrada usa claves exactas de catálogo, sin duplicados ni alias de mayúsculas.
func clavesUnicas(d *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return errEntradaJSON
	}
	token, err := d.Token()
	if err != nil {
		return errEntradaJSON
	}
	delim, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	if delim != '{' && delim != '[' {
		return errEntradaJSON
	}
	claves := map[string]bool{}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			clave, ok := key.(string)
			if err != nil || !ok || clave == "" || strings.Trim(clave, "abcdefghijklmnopqrstuvwxyz_0123456789.") != "" || claves[clave] {
				return errEntradaJSON
			}
			claves[clave] = true
		}
		if clavesUnicas(d, profundidad+1) != nil {
			return errEntradaJSON
		}
	}
	fin, err := d.Token()
	if err != nil || delim == '{' && fin != json.Delim('}') || delim == '[' && fin != json.Delim(']') {
		return errEntradaJSON
	}
	return nil
}
