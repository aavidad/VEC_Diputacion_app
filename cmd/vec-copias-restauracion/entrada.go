package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"time"
)

// Límites de transporte del formato v1; no son política de retención o versión.
const maxEntradaBytes = 4 << 20
const maxProfundidad = 32

var errEntrada = errors.New("copias_restauracion_error_entrada")

func leerEstricto(r io.Reader, dst any) error {
	b, err := io.ReadAll(io.LimitReader(r, maxEntradaBytes+1))
	if err != nil || len(b) > maxEntradaBytes {
		return errEntrada
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err = validarTokens(d, 0); err != nil {
		return errEntrada
	}
	if _, err = d.Token(); err != io.EOF {
		return errEntrada
	}
	var raw any
	d = json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if d.Decode(&raw) != nil || !validarForma(raw, reflect.TypeOf(dst).Elem()) {
		return errEntrada
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(dst) != nil {
		return errEntrada
	}
	return nil
}

// Rechaza claves repetidas incluso en objetos anidados, antes de que JSON
// sobrescriba una declaración. Ningún error reproduce entrada del operador.
func validarTokens(d *json.Decoder, depth int) error {
	if depth > maxProfundidad {
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
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return errEntrada
			}
			s, ok := key.(string)
			if !ok || seen[s] {
				return errEntrada
			}
			seen[s] = true
			if validarTokens(d, depth+1) != nil {
				return errEntrada
			}
		}
	case '[':
		for d.More() {
			if validarTokens(d, depth+1) != nil {
				return errEntrada
			}
		}
	default:
		return errEntrada
	}
	closing, err := d.Token()
	if err != nil {
		return errEntrada
	}
	if delim == '{' && closing != json.Delim('}') || delim == '[' && closing != json.Delim(']') {
		return errEntrada
	}
	return nil
}

// encoding/json acepta nombres de campo sin distinguir mayúsculas. El contrato
// v1 exige los nombres exactos y todos los campos, también listas vacías.
func validarForma(raw any, t reflect.Type) bool {
	if raw == nil {
		return t.Kind() == reflect.Pointer
	}
	if t == reflect.TypeFor[time.Time]() {
		_, ok := raw.(string)
		return ok
	}
	switch t.Kind() {
	case reflect.Struct:
		obj, ok := raw.(map[string]any)
		if !ok || len(obj) != t.NumField() {
			return false
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			key := strings.Split(f.Tag.Get("json"), ",")[0]
			value, exists := obj[key]
			if !exists || !validarForma(value, f.Type) {
				return false
			}
		}
		return true
	case reflect.Slice:
		xs, ok := raw.([]any)
		if !ok {
			return false
		}
		for _, x := range xs {
			if !validarForma(x, t.Elem()) {
				return false
			}
		}
		return true
	case reflect.Map:
		obj, ok := raw.(map[string]any)
		if !ok || t.Key().Kind() != reflect.String {
			return false
		}
		for _, x := range obj {
			if !validarForma(x, t.Elem()) {
				return false
			}
		}
		return true
	case reflect.Pointer:
		return validarForma(raw, t.Elem())
	case reflect.String:
		_, ok := raw.(string)
		return ok
	case reflect.Bool:
		_, ok := raw.(bool)
		return ok
	default:
		_, ok := raw.(json.Number)
		return ok // Decode comprueba tipo y rango entero.
	}
}
