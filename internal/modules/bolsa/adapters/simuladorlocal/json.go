package simuladorlocal

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"
)

const MaximoBytes = 256 * 1024

// Decodificar exige objeto único, claves no repetidas y volumen acotado antes
// de traducir al DTO. No convierte cantidades JSON a float64.
func Decodificar(b []byte) (Solicitud, error) {
	if len(b) == 0 || len(b) > MaximoBytes || !utf8.Valid(b) {
		return Solicitud{}, ErrSolicitud
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	nodos := 0
	if err := recorrer(d, 0, &nodos, true); err != nil {
		return Solicitud{}, err
	}
	if _, err := d.Token(); err != io.EOF {
		return Solicitud{}, ErrSolicitud
	}
	var s Solicitud
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil || len(s.Modo) > 32 || len(s.EjemploRef) > 64 {
		return Solicitud{}, ErrSolicitud
	}
	if len(s.Reglas) == 0 || s.Reglas[0] != '{' {
		return Solicitud{}, ErrSolicitud
	}
	return s, nil
}

func recorrer(d *json.Decoder, profundidad int, nodos *int, objeto bool) error {
	*nodos = *nodos + 1
	if profundidad > 32 || *nodos > 8192 {
		return ErrSolicitud
	}
	t, err := d.Token()
	if err != nil {
		return ErrSolicitud
	}
	delim, coleccion := t.(json.Delim)
	if objeto && (!coleccion || delim != '{') {
		return ErrSolicitud
	}
	if !coleccion {
		if s, ok := t.(string); ok && len(s) > 4096 {
			return ErrSolicitud
		}
		return nil
	}
	if delim != '{' && delim != '[' {
		return ErrSolicitud
	}
	claves := make(map[string]bool)
	cantidad := 0
	for d.More() {
		cantidad++
		if cantidad > 128 {
			return ErrSolicitud
		}
		if delim == '{' {
			clave, err := d.Token()
			s, ok := clave.(string)
			if err != nil || !ok || claves[s] || !claveJSON(s) {
				return ErrSolicitud
			}
			claves[s] = true
		}
		if err := recorrer(d, profundidad+1, nodos, false); err != nil {
			return err
		}
	}
	cierre, err := d.Token()
	if err != nil || (delim == '{' && cierre != json.Delim('}')) || (delim == '[' && cierre != json.Delim(']')) {
		return ErrSolicitud
	}
	return nil
}

// Todos los esquemas admitidos usan claves ASCII minúsculas. Esto excluye
// alias que encoding/json iguala sin distinguir mayúsculas o por Unicode.
func claveJSON(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if c != '_' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}
