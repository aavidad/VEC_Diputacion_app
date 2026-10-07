package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const maxBytes = 1 << 20

func decodificar(r io.Reader, target any) error {
	b, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil || len(b) > maxBytes {
		return errors.New("entrada_invalida")
	}
	if err := comprobarClaves(json.NewDecoder(bytes.NewReader(b))); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("entrada_invalida")
	}
	return nil
}

// Every key in these closed DTOs and their catalogue uses lowercase ASCII,
// digits or underscore. Check raw keys before encoding/json can fold aliases
// or silently replace repeated values. Each object has its own duplicate set.
func comprobarClaves(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return errors.New("entrada_invalida")
	}
	switch token {
	case json.Delim('{'):
		vistos := make(map[string]bool)
		for d.More() {
			token, err := d.Token()
			clave, ok := token.(string)
			if err != nil || !ok || clave == "" || vistos[clave] {
				return errors.New("entrada_invalida")
			}
			for _, c := range clave {
				if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '_' {
					return errors.New("entrada_invalida")
				}
			}
			vistos[clave] = true
			if err := comprobarClaves(d); err != nil {
				return err
			}
		}
		if token, err := d.Token(); err != nil || token != json.Delim('}') {
			return errors.New("entrada_invalida")
		}
	case json.Delim('['):
		for d.More() {
			if err := comprobarClaves(d); err != nil {
				return err
			}
		}
		if token, err := d.Token(); err != nil || token != json.Delim(']') {
			return errors.New("entrada_invalida")
		}
	}
	return nil
}
