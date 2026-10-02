package config

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrCopiasADMIN = errors.New("copias_admin_configuracion_invalida")

// CopiasADMIN solo localiza registros privados fuera del rollback. La identidad,
// las concesiones y la auditoría se reciben de las autoridades centrales.
// Ningún destino o ruta de esta configuración procede de la petición web.
type CopiasADMIN struct {
	DiarioDirectorio   string   `json:"diario_directorio"`
	PoliticaDirectorio string   `json:"politica_directorio"`
	RaicesRestauradas  []string `json:"raices_restauradas"`
	DestinoRef         string   `json:"destino_ref"`
	LimiteListado      int      `json:"limite_listado"`
}

func (c CopiasADMIN) Validar() error {
	if len(c.RaicesRestauradas) == 0 || len(c.RaicesRestauradas) > 64 || c.LimiteListado < 1 || c.LimiteListado > 100 || c.DestinoRef == "" || len(c.DestinoRef) > 96 {
		return ErrCopiasADMIN
	}
	for _, p := range append([]string{c.DiarioDirectorio, c.PoliticaDirectorio}, c.RaicesRestauradas...) {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || p == "/" {
			return ErrCopiasADMIN
		}
	}
	if c.DiarioDirectorio == c.PoliticaDirectorio {
		return ErrCopiasADMIN
	}
	for _, r := range c.RaicesRestauradas {
		for _, p := range []string{c.DiarioDirectorio, c.PoliticaDirectorio} {
			rel, e := filepath.Rel(r, p)
			if e != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return ErrCopiasADMIN
			}
		}
	}
	return nil
}
