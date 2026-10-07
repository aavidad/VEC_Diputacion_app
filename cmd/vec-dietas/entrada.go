package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/dietas/application/simulaciondevengo"
)

const limiteEntrada = 1 << 20

var errJSON = errors.New("entrada_json_invalida")

func leerEntrada(in io.Reader) (simulaciondevengo.Entrada, error) {
	return leerEntradaTipada[simulaciondevengo.Entrada](in)
}

func leerEntradaTipada[T any](in io.Reader) (T, error) {
	var e T
	b, err := io.ReadAll(io.LimitReader(in, limiteEntrada+1))
	if err != nil || len(b) > limiteEntrada || !utf8.Valid(b) {
		return e, errJSON
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := validarForma(d, reflect.TypeFor[T](), 0); err != nil {
		return e, errJSON
	}
	if _, err := d.Token(); err != io.EOF {
		return e, errJSON
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil {
		var vacio T
		return vacio, errJSON
	}
	return e, nil
}

// validarForma exige claves exactas, sin duplicados, campos obligatorios y sin null.
func validarForma(d *json.Decoder, t reflect.Type, profundidad int) error {
	if profundidad > 32 {
		return errJSON
	}
	tok, err := d.Token()
	if err != nil || tok == nil {
		return errJSON
	}
	switch t.Kind() {
	case reflect.Struct:
		if tok != json.Delim('{') {
			return errJSON
		}
		campos := map[string]reflect.StructField{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			campos[strings.Split(f.Tag.Get("json"), ",")[0]] = f
		}
		vistos := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return errJSON
			}
			nombre, ok := key.(string)
			if !ok || vistos[nombre] {
				return errJSON
			}
			f, ok := campos[nombre]
			if !ok {
				return errJSON
			}
			vistos[nombre] = true
			if err := validarForma(d, f.Type, profundidad+1); err != nil {
				return err
			}
		}
		if fin, err := d.Token(); err != nil || fin != json.Delim('}') {
			return errJSON
		}
		for nombre, f := range campos {
			if !vistos[nombre] && !strings.Contains(f.Tag.Get("json"), ",omitempty") {
				return errJSON
			}
		}
	case reflect.Slice:
		if tok != json.Delim('[') {
			return errJSON
		}
		for d.More() {
			if err := validarForma(d, t.Elem(), profundidad+1); err != nil {
				return err
			}
		}
		if fin, err := d.Token(); err != nil || fin != json.Delim(']') {
			return errJSON
		}
	default:
		if _, ok := tok.(json.Delim); ok {
			return errJSON
		}
	}
	return nil
}
