// Package catalogoalta carga las opciones versionadas de necesidad para la
// petición de personal temporal. El fichero incluido es un ejemplo sustituible.
package catalogoalta

import (
	_ "embed"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

//go:embed necesidades_v1.ejemplo.json
var necesidadesEjemplo []byte

// CargarNecesidades carga la distribución incluida o una publicación local.
// Una ruta declarada que falta o es inválida produce error, sin fallback.
func CargarNecesidades(ruta string) (domain.CatalogoNecesidadesAlta, error) {
	contenido := necesidadesEjemplo
	if ruta != "" {
		// #nosec G304 -- La ruta la aporta la configuración local, nunca HTTP.
		f, err := os.Open(ruta)
		if err != nil {
			return domain.CatalogoNecesidadesAlta{}, err
		}
		defer func() { _ = f.Close() }()
		contenido, err = io.ReadAll(io.LimitReader(f, domain.MaximoInstantaneaCatalogoNecesidadesAltaBytes+1))
		if err != nil || len(contenido) > domain.MaximoInstantaneaCatalogoNecesidadesAltaBytes {
			return domain.CatalogoNecesidadesAlta{}, domain.ErrNecesidadAltaInvalida
		}
	}
	return domain.RestaurarCatalogoNecesidadesAlta(contenido)
}
