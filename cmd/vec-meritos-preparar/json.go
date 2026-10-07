package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"vec-diputacion-granada/internal/modules/meritos/domain"
)

const maxBytes = 1024 * 1024

var errEntrada = errors.New("meritos.error.json_invalido")
var errSalida = errors.New("meritos.error.salida")

type entradaJSON struct{ reader io.Reader }
type salidaJSON struct{ writer io.Writer }

func (e entradaJSON) Leer() (domain.Paquete, error) {
	if e.reader == nil {
		return domain.Paquete{}, errEntrada
	}
	b, err := io.ReadAll(io.LimitReader(e.reader, maxBytes+1))
	if err != nil || len(b) > maxBytes {
		return domain.Paquete{}, errEntrada
	}
	tokens := json.NewDecoder(bytes.NewReader(b))
	if unicos(tokens, 0) != nil {
		return domain.Paquete{}, errEntrada
	}
	if _, err := tokens.Token(); err != io.EOF {
		return domain.Paquete{}, errEntrada
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var p domain.Paquete
	if d.Decode(&p) != nil {
		return domain.Paquete{}, errEntrada
	}
	return p, nil
}

func (s salidaJSON) Escribir(p domain.Preparacion) error {
	if s.writer == nil {
		return errSalida
	}
	enc := json.NewEncoder(s.writer)
	enc.SetIndent("", "  ")
	if enc.Encode(p) != nil {
		return errSalida
	}
	return nil
}

// Rechaza claves repetidas o alias de capitalización antes de decodificar.
// Así un estado o persona no puede ser sustituido silenciosamente en el JSON.
func unicos(d *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return errEntrada
	}
	t, err := d.Token()
	if err != nil {
		return errEntrada
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return errEntrada
			}
			k, ok := key.(string)
			if !ok || k == "" || strings.Trim(k, "abcdefghijklmnopqrstuvwxyz_") != "" || seen[k] {
				return errEntrada
			}
			seen[k] = true
			if unicos(d, profundidad+1) != nil {
				return errEntrada
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return errEntrada
		}
	case '[':
		for d.More() {
			if unicos(d, profundidad+1) != nil {
				return errEntrada
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return errEntrada
		}
	default:
		return errEntrada
	}
	return nil
}
