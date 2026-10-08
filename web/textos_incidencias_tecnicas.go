package web

import (
	"embed"
	"encoding/json"
	"os"
	"path"
)

//go:embed static/textos/*/incidencias_tecnicas.json static/textos/idiomas.json
var textosIncidenciasTecnicas embed.FS

// TextosIncidenciasTecnicas devuelve los datos canónicos sin depender del cwd.
// El idioma vacío se resuelve con el índice común de idiomas, nunca en Go.
func TextosIncidenciasTecnicas(idioma string) ([]byte, error) {
	b, err := textosIncidenciasTecnicas.ReadFile("static/textos/idiomas.json")
	if err != nil {
		return nil, os.ErrInvalid
	}
	var indice struct {
		PorDefecto string `json:"por_defecto"`
		Idiomas    []struct {
			Codigo string `json:"codigo"`
		} `json:"idiomas"`
	}
	if json.Unmarshal(b, &indice) != nil {
		return nil, os.ErrInvalid
	}
	if idioma == "" {
		idioma = indice.PorDefecto
	}
	for _, i := range indice.Idiomas {
		if i.Codigo == idioma && idioma != "" {
			b, err := textosIncidenciasTecnicas.ReadFile(path.Join("static/textos", idioma, "incidencias_tecnicas.json"))
			if err != nil {
				return nil, os.ErrInvalid
			}
			return b, nil
		}
	}
	return nil, os.ErrInvalid
}
