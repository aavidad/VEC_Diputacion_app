package ajustesreglas

import (
	"bytes"
	"encoding/json"
	"io"
)

type Motivo struct {
	Clave      string `json:"clave"`
	TextoClave string `json:"texto_clave"`
}
type CatalogoMotivos struct {
	ID      string   `json:"id"`
	Version int      `json:"version"`
	Motivos []Motivo `json:"motivos"`
}

func LeerCatalogoMotivos(b []byte) (CatalogoMotivos, error) {
	var c CatalogoMotivos
	if len(b) == 0 || len(b) > 16*1024 {
		return c, ErrNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || c.ID != CatalogoMotivosID || c.Version < 1 || len(c.Motivos) == 0 || len(c.Motivos) > 16 {
		return CatalogoMotivos{}, ErrNoDisponible
	}
	vistos := map[string]bool{}
	for _, m := range c.Motivos {
		if !motivoClaveValida(m.Clave) || m.TextoClave != "ajustesMotivo_"+m.Clave || vistos[m.Clave] {
			return CatalogoMotivos{}, ErrNoDisponible
		}
		vistos[m.Clave] = true
	}
	return c, nil
}
func (c CatalogoMotivos) Admite(clave string) bool {
	for _, m := range c.Motivos {
		if m.Clave == clave {
			return true
		}
	}
	return false
}
func (c CatalogoMotivos) Lista() []Motivo { return append([]Motivo(nil), c.Motivos...) }
func motivoClaveValida(s string) bool {
	if len(s) < 3 || len(s) > 64 || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, r := range s[1:] {
		if r < 'a' || r > 'z' {
			if r < '0' || r > '9' {
				if r != '_' {
					return false
				}
			}
		}
	}
	return true
}
