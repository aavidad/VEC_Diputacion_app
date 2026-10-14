package web

import "embed"

//go:embed static/textos/*/cronos-equipo-informes-c10.json static/textos/idiomas.json
var textosCronosInformes embed.FS

// CatalogosCronosInformes conserva los catálogos del idioma activo para la
// representación documental; no incorpora permisos ni política de campos.
func CatalogosCronosInformes() (map[string][]byte, error) {
	return catalogosDeTextosIndexados(textosCronosInformes, "cronos-equipo-informes-c10.json")
}
