package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	adaptador "vec-diputacion-granada/internal/modules/administracion/adapters/capturacopias"
)

func leerConfiguracion(r io.Reader) (configuracion, error) {
	var c configuracion
	b, err := io.ReadAll(io.LimitReader(r, (4<<20)+1))
	if err != nil || len(b) > 4<<20 {
		return c, adaptador.ErrLocal
	}
	d := json.NewDecoder(bytes.NewReader(b))
	if estructura(d, 0) != nil {
		return c, adaptador.ErrLocal
	}
	if _, err := d.Token(); err != io.EOF {
		return c, adaptador.ErrLocal
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return c, adaptador.ErrLocal
	}
	return c, nil
}

// A configuration has one interpretation: duplicates (including case variants),
// nulls, trailing values and excessive depth are refused before any process.
func estructura(d *json.Decoder, nivel int) error {
	if nivel > 32 {
		return adaptador.ErrLocal
	}
	t, err := d.Token()
	if err != nil || t == nil {
		return adaptador.ErrLocal
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return adaptador.ErrLocal
	}
	vistas := map[string]bool{}
	for d.More() {
		if delim == '{' {
			t, err = d.Token()
			k, ok := t.(string)
			if err != nil || !ok || vistas[strings.ToLower(k)] {
				return adaptador.ErrLocal
			}
			vistas[strings.ToLower(k)] = true
		}
		if estructura(d, nivel+1) != nil {
			return adaptador.ErrLocal
		}
	}
	_, err = d.Token()
	return err
}
