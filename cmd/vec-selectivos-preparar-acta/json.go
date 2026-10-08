package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maximoEntradaJSON = 1024 * 1024

var errEntradaJSON = errors.New("seleccion.acta_preparacion.json_invalido")

func leerJSON(r io.Reader, destino any) error {
	if r == nil {
		return errEntradaJSON
	}
	material, err := io.ReadAll(io.LimitReader(r, maximoEntradaJSON+1))
	if err != nil || len(material) > maximoEntradaJSON || !utf8.Valid(material) {
		return errEntradaJSON
	}
	if err := validarEscapesUnicode(material); err != nil {
		return err
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

// encoding/json sustituye los pares UTF-16 inválidos por U+FFFD. Esta guarda
// evita alterar textos aportados y distingue escapes de barras literales.
func validarEscapesUnicode(material []byte) error {
	for i := 0; i < len(material); i++ {
		if material[i] != '"' {
			continue
		}
		for i++; i < len(material) && material[i] != '"'; i++ {
			if material[i] != '\\' {
				continue
			}
			i++
			if i >= len(material) {
				return errEntradaJSON
			}
			if material[i] != 'u' {
				continue
			}
			if i+4 >= len(material) {
				return errEntradaJSON
			}
			valor, err := strconv.ParseUint(string(material[i+1:i+5]), 16, 16)
			if err != nil {
				return errEntradaJSON
			}
			i += 4
			if valor >= 0xdc00 && valor <= 0xdfff {
				return errEntradaJSON
			}
			if valor >= 0xd800 && valor <= 0xdbff {
				if i+6 >= len(material) || material[i+1] != '\\' || material[i+2] != 'u' {
					return errEntradaJSON
				}
				bajo, err := strconv.ParseUint(string(material[i+3:i+7]), 16, 16)
				if err != nil || bajo < 0xdc00 || bajo > 0xdfff {
					return errEntradaJSON
				}
				i += 6
			}
		}
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
			if err != nil || !ok || clave == "" || strings.Trim(clave, "abcdefghijklmnopqrstuvwxyz_0123456789") != "" || claves[clave] {
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
