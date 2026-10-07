package web

import (
	"embed"
	"encoding/json"
	"os"
	"path"
)

//go:embed static/textos/*/registro-tecnico-consulta.json static/textos/idiomas.json
var textosRegistroTecnicoConsulta embed.FS

// TextosRegistroTecnicoConsulta carga el catálogo común sin depender del cwd.
// El idioma vacío usa el valor por defecto de idiomas.json.
func TextosRegistroTecnicoConsulta(idioma string) ([]byte, error) {
	indiceBytes, err := textosRegistroTecnicoConsulta.ReadFile("static/textos/idiomas.json")
	if err != nil {
		return nil, os.ErrInvalid
	}
	var indice struct {
		PorDefecto string `json:"por_defecto"`
		Idiomas    []struct {
			Codigo string `json:"codigo"`
		} `json:"idiomas"`
	}
	if json.Unmarshal(indiceBytes, &indice) != nil {
		return nil, os.ErrInvalid
	}
	if idioma == "" {
		idioma = indice.PorDefecto
	}
	for _, disponible := range indice.Idiomas {
		if disponible.Codigo == idioma && idioma != "" {
			texto, err := textosRegistroTecnicoConsulta.ReadFile(path.Join("static/textos", idioma, "registro-tecnico-consulta.json"))
			if err != nil {
				return nil, os.ErrInvalid
			}
			return texto, nil
		}
	}
	return nil, os.ErrInvalid
}
