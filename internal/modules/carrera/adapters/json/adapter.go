package json

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"io"
	"strings"

	"vec-diputacion-granada/internal/modules/carrera/domain"
)

const MaxBytes = 1024 * 1024

var ErrEntrada = errors.New("carrera.error.json_invalido")
var ErrSalida = errors.New("carrera.error.salida")

type Entrada struct{ Reader io.Reader }
type Salida struct{ Writer io.Writer }

func (e Entrada) Leer() (domain.Escenario, error) {
	if e.Reader == nil {
		return domain.Escenario{}, ErrEntrada
	}
	b, err := io.ReadAll(io.LimitReader(e.Reader, MaxBytes+1))
	if err != nil || len(b) > MaxBytes {
		return domain.Escenario{}, ErrEntrada
	}
	// Reject duplicate members, including escaped aliases, before translating DTOs.
	tokens := stdjson.NewDecoder(bytes.NewReader(b))
	if err := unicos(tokens, 0); err != nil {
		return domain.Escenario{}, ErrEntrada
	}
	if _, err := tokens.Token(); err != io.EOF {
		return domain.Escenario{}, ErrEntrada
	}
	d := stdjson.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var s escenario
	if d.Decode(&s) != nil {
		return domain.Escenario{}, ErrEntrada
	}
	out := domain.Escenario{Alcance: s.Alcance, Version: s.Version, Fuente: s.Fuente.domain()}
	for _, c := range s.Casos {
		out.Casos = append(out.Casos, c.domain())
	}
	return out, nil
}
func (s Salida) Escribir(p domain.Preparacion) error {
	if s.Writer == nil {
		return ErrSalida
	}
	enc := stdjson.NewEncoder(s.Writer)
	enc.SetIndent("", "  ")
	if enc.Encode(proyectar(p)) != nil {
		return ErrSalida
	}
	return nil
}
func unicos(d *stdjson.Decoder, depth int) error {
	if depth > 32 {
		return ErrEntrada
	}
	t, err := d.Token()
	if err != nil {
		return ErrEntrada
	}
	delim, ok := t.(stdjson.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return ErrEntrada
			}
			k, ok := key.(string)
			if !ok || k != strings.ToLower(k) || strings.Trim(k, "abcdefghijklmnopqrstuvwxyz_") != "" || seen[k] {
				return ErrEntrada
			}
			seen[k] = true
			if unicos(d, depth+1) != nil {
				return ErrEntrada
			}
		}
		end, err := d.Token()
		if err != nil || end != stdjson.Delim('}') {
			return ErrEntrada
		}
	case '[':
		for d.More() {
			if unicos(d, depth+1) != nil {
				return ErrEntrada
			}
		}
		end, err := d.Token()
		if err != nil || end != stdjson.Delim(']') {
			return ErrEntrada
		}
	default:
		return ErrEntrada
	}
	return nil
}
