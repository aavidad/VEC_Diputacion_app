package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"time"
)

func decodificarEstricto(b []byte, destino any) error {
	if err := clavesUnicas(b); err != nil {
		return err
	}
	if err := clavesExactas(b, reflect.TypeOf(destino).Elem()); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	var sobra any
	if !errors.Is(d.Decode(&sobra), io.EOF) {
		return errors.New("json_extra")
	}
	return nil
}

func clavesExactas(b []byte, tipo reflect.Type) error {
	if tipo.Kind() == reflect.Pointer {
		if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
			return nil
		}
		return clavesExactas(b, tipo.Elem())
	}
	if tipo == reflect.TypeOf(time.Time{}) {
		var valor string
		if json.Unmarshal(b, &valor) != nil || len(valor) != 20 {
			return errors.New("json_instante")
		}
		fecha, err := time.Parse(time.RFC3339, valor)
		if err != nil || fecha.Format(time.RFC3339) != valor || valor[19] != 'Z' {
			return errors.New("json_instante")
		}
		return nil
	}
	switch tipo.Kind() {
	case reflect.Array, reflect.Slice:
		var elementos []json.RawMessage
		if err := json.Unmarshal(b, &elementos); err != nil || elementos == nil || tipo.Kind() == reflect.Array && len(elementos) != tipo.Len() {
			return errors.New("json_longitud")
		}
		for _, x := range elementos {
			if err := clavesExactas(x, tipo.Elem()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		var campos map[string]json.RawMessage
		if err := json.Unmarshal(b, &campos); err != nil || campos == nil {
			return errors.New("json_objeto")
		}
		if len(campos) != tipo.NumField() {
			return errors.New("json_campos")
		}
		for i := range tipo.NumField() {
			campo := tipo.Field(i)
			nombre := campo.Tag.Get("json")
			valor, ok := campos[nombre]
			if !ok || campo.Type.Kind() != reflect.Pointer && bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
				return errors.New("json_campo")
			}
			if err := clavesExactas(valor, campo.Type); err != nil {
				return err
			}
		}
	}
	return nil
}

// El decodificador de Go acepta claves repetidas; este recorrido las deniega.
func clavesUnicas(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var recorrer func() error
	recorrer = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch t {
		case json.Delim('{'):
			vistas := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				clave, ok := k.(string)
				if !ok || vistas[clave] {
					return errors.New("json_clave_repetida")
				}
				vistas[clave] = true
				if err := recorrer(); err != nil {
					return err
				}
			}
			_, err := d.Token()
			return err
		case json.Delim('['):
			for d.More() {
				if err := recorrer(); err != nil {
					return err
				}
			}
			_, err := d.Token()
			return err
		default:
			return nil
		}
	}
	return recorrer()
}
