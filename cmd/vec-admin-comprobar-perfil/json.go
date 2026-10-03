package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"syscall"
	"unicode/utf8"
)

func leerJSON(ruta string, destino any, maximo int64) error {
	// #nosec G304 G703 -- ruta explícita de la CLI offline; lectura acotada, sin servidor ni escritura.
	fichero, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return errEntrada
	}
	defer fichero.Close()
	info, err := fichero.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximo {
		return errEntrada
	}
	datos, err := io.ReadAll(io.LimitReader(fichero, maximo+1))
	if err != nil || int64(len(datos)) > maximo || !utf8.Valid(datos) {
		return errEntrada
	}
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.UseNumber()
	if consumirJSON(dec, 0) != nil {
		return errEntrada
	}
	if _, err := dec.Token(); err != io.EOF {
		return errEntrada
	}
	dec = json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if dec.Decode(destino) != nil || dec.Decode(new(any)) != io.EOF {
		return errEntrada
	}
	return nil
}

// Rechaza claves repetidas antes del DTO: encoding/json conservaría solo la
// última, haciendo ambigua una propuesta de permisos. También acota estructura.
func consumirJSON(dec *json.Decoder, profundidad int) error {
	if profundidad > 16 {
		return errEntrada
	}
	token, err := dec.Token()
	if err != nil {
		return errEntrada
	}
	d, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch d {
	case '{':
		vistos := make(map[string]bool)
		for dec.More() {
			token, err := dec.Token()
			clave, ok := token.(string)
			if err != nil || !ok || clave != strings.ToLower(clave) || vistos[clave] || len(vistos) >= 64 {
				return errEntrada
			}
			vistos[clave] = true
			if consumirJSON(dec, profundidad+1) != nil {
				return errEntrada
			}
		}
		if cierre, err := dec.Token(); err != nil || cierre != json.Delim('}') {
			return errEntrada
		}
	case '[':
		cuenta := 0
		for dec.More() {
			cuenta++
			if cuenta > 512 || consumirJSON(dec, profundidad+1) != nil {
				return errEntrada
			}
		}
		if cierre, err := dec.Token(); err != nil || cierre != json.Delim(']') {
			return errEntrada
		}
	default:
		return errEntrada
	}
	return nil
}
