package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"path"

	"vec-diputacion-granada/internal/shared/i18n"
)

//go:embed static/textos/*/cronos-calendario-ensayo.json static/textos/idiomas.json
var textosCronosCalendarioEnsayo embed.FS

// CatalogoCronosCalendarioEnsayo usa el índice común de idiomas y devuelve
// también los mensajes para la explicación JSON del ensayo local.
func CatalogoCronosCalendarioEnsayo() (*i18n.Catalog, map[string]map[string]string, error) {
	contenido, err := textosCronosCalendarioEnsayo.ReadFile("static/textos/idiomas.json")
	if err != nil {
		return nil, nil, err
	}
	var indice struct {
		PorDefecto string `json:"por_defecto"`
		Idiomas    []struct {
			Codigo string `json:"codigo"`
		} `json:"idiomas"`
	}
	if err := json.Unmarshal(contenido, &indice); err != nil {
		return nil, nil, err
	}
	permitidos := map[string]bool{}
	for _, idioma := range indice.Idiomas {
		permitidos[idioma.Codigo] = true
	}
	if indice.PorDefecto == "" || !permitidos[indice.PorDefecto] {
		return nil, nil, i18n.ErrLocaleRequired
	}
	archivos, err := fs.Glob(textosCronosCalendarioEnsayo, "static/textos/*/cronos-calendario-ensayo.json")
	if err != nil {
		return nil, nil, err
	}
	mensajes := make(map[string]map[string]string, len(archivos))
	for _, archivo := range archivos {
		idioma := path.Base(path.Dir(archivo))
		if !permitidos[idioma] {
			return nil, nil, i18n.ErrLocaleRequired
		}
		contenido, err := textosCronosCalendarioEnsayo.ReadFile(archivo)
		if err != nil {
			return nil, nil, err
		}
		var traducciones map[string]string
		if err := json.Unmarshal(contenido, &traducciones); err != nil {
			return nil, nil, err
		}
		mensajes[idioma] = traducciones
	}
	if len(mensajes[indice.PorDefecto]) == 0 {
		return nil, nil, i18n.ErrNoLocalesFound
	}
	catalogo, err := i18n.New(indice.PorDefecto, mensajes)
	if err != nil {
		return nil, nil, err
	}
	return catalogo, mensajes, nil
}
