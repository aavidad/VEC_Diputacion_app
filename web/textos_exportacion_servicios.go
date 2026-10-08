package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"path"

	"vec-diputacion-granada/internal/shared/i18n"
)

//go:embed static/textos/*/personal-exportacion-servicios.json static/textos/idiomas.json
var textosExportacionServicios embed.FS

// CatalogosExportacionServicios conserva los bytes de los catálogos comunes,
// cuya huella forma parte del material de autorización de la exportación.
func CatalogosExportacionServicios() (map[string][]byte, error) {
	contenido, err := textosExportacionServicios.ReadFile("static/textos/idiomas.json")
	if err != nil {
		return nil, err
	}
	var indice struct {
		Idiomas []struct {
			Codigo string `json:"codigo"`
		} `json:"idiomas"`
	}
	if err := json.Unmarshal(contenido, &indice); err != nil {
		return nil, err
	}
	permitidos := map[string]bool{}
	for _, idioma := range indice.Idiomas {
		permitidos[idioma.Codigo] = true
	}
	archivos, err := fs.Glob(textosExportacionServicios, "static/textos/*/personal-exportacion-servicios.json")
	if err != nil {
		return nil, err
	}
	catalogos := make(map[string][]byte, len(archivos))
	for _, archivo := range archivos {
		idioma := path.Base(path.Dir(archivo))
		if !permitidos[idioma] {
			return nil, i18n.ErrLocaleRequired
		}
		contenido, err := textosExportacionServicios.ReadFile(archivo)
		if err != nil {
			return nil, err
		}
		catalogos[idioma] = contenido
	}
	if len(catalogos) != len(permitidos) || len(catalogos) == 0 {
		return nil, i18n.ErrNoLocalesFound
	}
	return catalogos, nil
}
