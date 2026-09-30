package ajustesreglas

import (
	"bytes"
	"encoding/json"
	"io"
)

// Motivo es una clave gobernada por catálogo. TextoClave apunta a i18n;
// ningún nombre visible se compila en Go.
type Motivo struct {
	Clave      string `json:"clave"`
	TextoClave string `json:"texto_clave"`
}

type CatalogoMotivos struct {
	ID      string   `json:"id"`
	Version int      `json:"version"`
	Motivos []Motivo `json:"motivos"`
}

// LeerCatalogoMotivos recibe los datos del adaptador de configuración. Rechaza
// claves repetidas o ajenas a la gramática de CT148.
func LeerCatalogoMotivos(contenido []byte) (CatalogoMotivos, error) {
	var catalogo CatalogoMotivos
	if len(contenido) == 0 || len(contenido) > 16*1024 {
		return catalogo, ErrNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	if d.Decode(&catalogo) != nil || d.Decode(&struct{}{}) != io.EOF ||
		catalogo.ID != "vec.contratacion_temporal.reglas.motivos_ajuste" ||
		catalogo.Version < 1 || len(catalogo.Motivos) == 0 || len(catalogo.Motivos) > 16 {
		return CatalogoMotivos{}, ErrNoDisponible
	}
	vistos := make(map[string]bool, len(catalogo.Motivos))
	for _, m := range catalogo.Motivos {
		if !claveMotivoValida(m.Clave) || m.TextoClave != "ajustesMotivo_"+m.Clave || vistos[m.Clave] {
			return CatalogoMotivos{}, ErrNoDisponible
		}
		vistos[m.Clave] = true
	}
	return catalogo, nil
}

func (c CatalogoMotivos) Admite(clave string) bool {
	for _, m := range c.Motivos {
		if m.Clave == clave {
			return true
		}
	}
	return false
}

func claveMotivoValida(clave string) bool {
	if len(clave) < 3 || len(clave) > 64 || clave[0] < 'a' || clave[0] > 'z' {
		return false
	}
	for _, c := range clave[1:] {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
