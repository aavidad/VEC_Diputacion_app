// Package fichero supplies strictly bounded local rehearsal data.
package fichero

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/certificados/domain"
)

const LimiteJSON = 256 << 10

// LeerJSON rejects unknown fields, duplicate keys, trailing JSON, non-regular
// files, malformed UTF-8 and oversized input. Paths are chosen by the local operator.
func LeerJSON(ruta string, destino any) error {
	info, e := os.Lstat(ruta)
	if e != nil || !info.Mode().IsRegular() || info.Size() > LimiteJSON {
		return domain.ErrEntrada
	}
	f, e := os.Open(ruta) // #nosec G304 -- explicit local input path, no remote or HTTP caller.
	if e != nil {
		return domain.ErrEntrada
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, LimiteJSON+1))
	if e != nil || len(b) > LimiteJSON || !utf8.Valid(b) {
		return domain.ErrEntrada
	}
	tokens := json.NewDecoder(bytes.NewReader(b))
	if validarTokens(tokens, 0) != nil {
		return domain.ErrEntrada
	}
	if _, e = tokens.Token(); e != io.EOF {
		return domain.ErrEntrada
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return domain.ErrEntrada
	}
	return nil
}
func validarTokens(d *json.Decoder, profundidad int) error {
	if profundidad > 24 {
		return domain.ErrEntrada
	}
	t, e := d.Token()
	if e != nil {
		return domain.ErrEntrada
	}
	delimitador, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delimitador {
	case '{':
		vistos := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return domain.ErrEntrada
			}
			clave, ok := k.(string)
			if !ok || !claveCanonica(clave) || vistos[clave] {
				return domain.ErrEntrada
			}
			vistos[clave] = true
			if validarTokens(d, profundidad+1) != nil {
				return domain.ErrEntrada
			}
		}
		fin, e := d.Token()
		if e != nil || fin != json.Delim('}') {
			return domain.ErrEntrada
		}
	case '[':
		for d.More() {
			if validarTokens(d, profundidad+1) != nil {
				return domain.ErrEntrada
			}
		}
		fin, e := d.Token()
		if e != nil || fin != json.Delim(']') {
			return domain.ErrEntrada
		}
	default:
		return domain.ErrEntrada
	}
	return nil
}

// Go acepta variantes de mayúsculas para un mismo campo JSON. Exigir claves
// canónicas evita que una segunda grafía sustituya un valor ya leído.
func claveCanonica(clave string) bool {
	if len(clave) == 0 || clave[0] < 'a' || clave[0] > 'z' {
		return false
	}
	for i := 1; i < len(clave); i++ {
		c := clave[i]
		if c < 'a' || c > 'z' {
			if c < '0' || c > '9' {
				if c != '_' && c != '.' {
					return false
				}
			}
		}
	}
	return true
}

type FuenteServicios struct{ Ruta string }

func (f FuenteServicios) Obtener(ctx context.Context) (domain.FuenteServicios, error) {
	var fuente domain.FuenteServicios
	if ctx == nil || ctx.Err() != nil {
		return fuente, domain.ErrNoDisponible
	}
	// La forma V1 solo entra por el traductor de personalv1, nunca escrita a mano.
	if LeerJSON(f.Ruta, &fuente) != nil || fuente.Esquema != domain.EsquemaFuenteEnsayo || fuente.ValidarEnsayo() != nil {
		return domain.FuenteServicios{}, domain.ErrEntrada
	}
	return fuente, nil
}

type CatalogoPlantillas struct{ RutaPlantilla, RutaTextos string }

func (c CatalogoPlantillas) Obtener(ctx context.Context, id string, version int, idioma string) (domain.Plantilla, domain.Textos, error) {
	var p domain.Plantilla
	var t domain.Textos
	if ctx == nil || ctx.Err() != nil {
		return p, t, domain.ErrNoDisponible
	}
	if LeerJSON(c.RutaPlantilla, &p) != nil || LeerJSON(c.RutaTextos, &t) != nil ||
		p.Validar(id, version) != nil || t.Validar() != nil || t.Idioma != idioma {
		return domain.Plantilla{}, domain.Textos{}, domain.ErrCatalogo
	}
	return p, t, nil
}
