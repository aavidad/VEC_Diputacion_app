package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"path"

	"vec-diputacion-granada/internal/shared/i18n"
)

// Los mensajes HTTP de B10 comparten los catálogos web; el binario no depende
// del directorio de trabajo ni necesita incorporar otros activos del portal.
//
//go:embed static/textos/*/bolsas-publicas.json
var textosBolsasPublicas embed.FS

func CatalogoBolsasPublicas() (*i18n.Catalog, error) {
	archivos, err := fs.Glob(textosBolsasPublicas, "static/textos/*/bolsas-publicas.json")
	if err != nil {
		return nil, err
	}
	mensajes := make(map[string]map[string]string, len(archivos))
	for _, archivo := range archivos {
		contenido, err := textosBolsasPublicas.ReadFile(archivo)
		if err != nil {
			return nil, err
		}
		var catalogo map[string]string
		if err := json.Unmarshal(contenido, &catalogo); err != nil {
			return nil, err
		}
		mensajes[path.Base(path.Dir(archivo))] = catalogo
	}
	return i18n.New(i18n.DefaultLocale, mensajes)
}
