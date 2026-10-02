package cotejopromocionjson

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

const MaxBytes = 1 << 20

var ErrEntrada = errors.New("carrera.cotejo.error.json_invalido")

type Entrada struct {
	Consulta ports.ConsultaPromocionSintetica  `json:"consulta"`
	Dictamen *ports.DictamenPromocionSintetico `json:"dictamen_sintetico"`
}

func Leer(r io.Reader) (Entrada, error) {
	if r == nil {
		return Entrada{}, ErrEntrada
	}
	b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil || len(b) > MaxBytes {
		return Entrada{}, ErrEntrada
	}
	t := json.NewDecoder(bytes.NewReader(b))
	if unicos(t, 0, "abcdefghijklmnopqrstuvwxyz_0123456789") != nil {
		return Entrada{}, ErrEntrada
	}
	if _, err := t.Token(); err != io.EOF {
		return Entrada{}, ErrEntrada
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var e Entrada
	if d.Decode(&e) != nil {
		return Entrada{}, ErrEntrada
	}
	return e, nil
}

func unicos(d *json.Decoder, n int, alfabeto string) error {
	if n > 32 {
		return ErrEntrada
	}
	t, err := d.Token()
	if err != nil {
		return ErrEntrada
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return ErrEntrada
	}
	vistos := map[string]bool{}
	for d.More() {
		if delim == '{' {
			t, err := d.Token()
			if err != nil {
				return ErrEntrada
			}
			k, ok := t.(string)
			if !ok || k == "" || k != strings.ToLower(k) || strings.Trim(k, alfabeto) != "" || vistos[k] {
				return ErrEntrada
			}
			vistos[k] = true
		}
		if unicos(d, n+1, alfabeto) != nil {
			return ErrEntrada
		}
	}
	fin, err := d.Token()
	if err != nil || (delim == '{' && fin != json.Delim('}')) || (delim == '[' && fin != json.Delim(']')) {
		return ErrEntrada
	}
	return nil
}

// El adaptador conserva una respuesta aportada de ensayo; no evalúa requisitos
// ni conecta una autoridad nominal o fuente personal.
type Lector struct {
	Dictamen *ports.DictamenPromocionSintetico
}

func (l Lector) ConsultarCotejoPromocionSintetico(ctx context.Context, _ ports.ConsultaPromocionSintetica) (ports.DictamenPromocionSintetico, error) {
	if ctx == nil || ctx.Err() != nil || l.Dictamen == nil {
		return ports.DictamenPromocionSintetico{}, ErrEntrada
	}
	d := *l.Dictamen
	d.Comprobaciones = append([]ports.ComprobacionPromocionSintetica(nil), d.Comprobaciones...)
	for i := range d.Comprobaciones {
		d.Comprobaciones[i].HechosReferencias = append([]string(nil), d.Comprobaciones[i].HechosReferencias...)
	}
	return d, nil
}

// LeerCatalogo aplica al catálogo configurable el mismo rechazo de duplicados
// y documentos concatenados, con un límite propio menor.
func LeerCatalogo(r io.Reader) (map[string]string, error) {
	if r == nil {
		return nil, ErrEntrada
	}
	b, err := io.ReadAll(io.LimitReader(r, 64*1024+1))
	if err != nil || len(b) > 64*1024 {
		return nil, ErrEntrada
	}
	t := json.NewDecoder(bytes.NewReader(b))
	if unicos(t, 0, "abcdefghijklmnopqrstuvwxyz_0123456789.") != nil {
		return nil, ErrEntrada
	}
	if _, err := t.Token(); err != io.EOF {
		return nil, ErrEntrada
	}
	var textos map[string]string
	if json.Unmarshal(b, &textos) != nil || len(textos) == 0 {
		return nil, ErrEntrada
	}
	return textos, nil
}
