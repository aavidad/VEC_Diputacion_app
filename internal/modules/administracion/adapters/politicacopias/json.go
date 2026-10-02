package politicacopias

import (
	"bytes"
	"encoding/json"
	"io"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
)

const MaxBytes = 1 << 20

// Decodificar rejects duplicates, case aliases, unknown fields and trailing JSON.
func Decodificar(r io.Reader, target any) error {
	b, e := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if e != nil || len(b) > MaxBytes {
		return d.ErrEntrada
	}
	if claves(json.NewDecoder(bytes.NewReader(b)), 0) != nil {
		return d.ErrEntrada
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(target) != nil || dec.Decode(new(any)) != io.EOF {
		return d.ErrEntrada
	}
	return nil
}
func claves(dec *json.Decoder, depth int) error {
	if depth > 32 {
		return d.ErrEntrada
	}
	token, e := dec.Token()
	if e != nil {
		return d.ErrEntrada
	}
	switch token {
	case json.Delim('{'):
		seen := map[string]bool{}
		for dec.More() {
			key, e := dec.Token()
			s, ok := key.(string)
			if e != nil || !ok || s == "" || seen[s] {
				return d.ErrEntrada
			}
			for _, c := range s {
				if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '_' {
					return d.ErrEntrada
				}
			}
			seen[s] = true
			if claves(dec, depth+1) != nil {
				return d.ErrEntrada
			}
		}
		t, e := dec.Token()
		if e != nil || t != json.Delim('}') {
			return d.ErrEntrada
		}
	case json.Delim('['):
		for dec.More() {
			if claves(dec, depth+1) != nil {
				return d.ErrEntrada
			}
		}
		t, e := dec.Token()
		if e != nil || t != json.Delim(']') {
			return d.ErrEntrada
		}
	}
	return nil
}
