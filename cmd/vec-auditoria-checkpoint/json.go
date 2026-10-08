package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

func decodificar(b []byte, v any) error {
	// Claves ASCII en minúsculas y sin duplicadas: encoding/json también
	// acepta alias por mayúsculas; aquí se exige la representación canónica.
	if err := sinDuplicadas(json.NewDecoder(bytes.NewReader(b))); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return errEntrada
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errEntrada
	}
	return nil
}
func sinDuplicadas(d *json.Decoder) error { return valorUnico(d, 0) }

func valorUnico(d *json.Decoder, profundidad int) error {
	if profundidad > 16 {
		return errEntrada
	}
	t, err := d.Token()
	if err != nil || t == nil {
		return errEntrada
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		vistas := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			s, ok := k.(string)
			if e != nil || !ok || vistas[s] || strings.ContainsFunc(s, func(r rune) bool { return r > 127 || r >= 'A' && r <= 'Z' }) {
				return errEntrada
			}
			vistas[s] = true
			if valorUnico(d, profundidad+1) != nil {
				return errEntrada
			}
		}
	case '[':
		for d.More() {
			if valorUnico(d, profundidad+1) != nil {
				return errEntrada
			}
		}
	default:
		return errEntrada
	}
	cierre, err := d.Token()
	if err != nil || delim == '{' && cierre != json.Delim('}') || delim == '[' && cierre != json.Delim(']') {
		return errEntrada
	}
	return nil
}
