package main

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/personal/application"
)

const limiteEntrada = 8 << 20

// La revisión sintáctica exige claves exactas, sin duplicados, valores nulos,
// campos desconocidos ni sustituciones silenciosas del decodificador estándar.
func leerPaquete(input io.Reader) (application.PaquetePreparacionOrganizacion, error) {
	var p application.PaquetePreparacionOrganizacion
	datos, err := io.ReadAll(io.LimitReader(input, limiteEntrada+1))
	if err != nil {
		return p, err
	}
	if len(datos) > limiteEntrada || !utf8.Valid(datos) || !json.Valid(datos) {
		return p, io.ErrUnexpectedEOF
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	if err := revisarForma(d, reflect.TypeOf(p), 0); err != nil {
		return p, err
	}
	if _, err := d.Token(); err != io.EOF {
		return p, io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(bytes.NewReader(datos))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return p, err
	}
	return p, nil
}

func revisarForma(d *json.Decoder, tipo reflect.Type, profundidad int) error {
	if profundidad > 32 {
		return io.ErrUnexpectedEOF
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return io.ErrUnexpectedEOF
	}
	switch tipo.Kind() {
	case reflect.Struct:
		if token != json.Delim('{') {
			return io.ErrUnexpectedEOF
		}
		campos := map[string]reflect.Type{}
		for i := 0; i < tipo.NumField(); i++ {
			f := tipo.Field(i)
			campos[strings.Split(f.Tag.Get("json"), ",")[0]] = f.Type
		}
		vistos := map[string]bool{}
		for d.More() {
			clave, err := d.Token()
			if err != nil {
				return err
			}
			nombre, ok := clave.(string)
			if !ok || vistos[nombre] {
				return io.ErrUnexpectedEOF
			}
			campo, ok := campos[nombre]
			if !ok {
				return io.ErrUnexpectedEOF
			}
			vistos[nombre] = true
			if err := revisarForma(d, campo, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := d.Token()
		if err != nil {
			return err
		}
		if cierre != json.Delim('}') {
			return io.ErrUnexpectedEOF
		}
	case reflect.Slice:
		if token != json.Delim('[') {
			return io.ErrUnexpectedEOF
		}
		for d.More() {
			if err := revisarForma(d, tipo.Elem(), profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := d.Token()
		if err != nil {
			return err
		}
		if cierre != json.Delim(']') {
			return io.ErrUnexpectedEOF
		}
	default:
		if _, ok := token.(json.Delim); ok {
			return io.ErrUnexpectedEOF
		}
	}
	return nil
}
