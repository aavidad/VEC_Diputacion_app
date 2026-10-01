package catalogoagregadosincidencias

import (
	"regexp"
	"strings"
	"unicode/utf8"
	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const LimiteJSON = catalogoefectos.LimiteJSON

var etiquetaIdioma = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

type CatalogoTextos struct {
	VersionEsquema string            `json:"version_esquema"`
	Idioma         string            `json:"idioma"`
	Textos         map[string]string `json:"textos"`
}

func CargarSnapshot(b []byte, sha string) (ports.SnapshotEnsayoAgregadosIncidencias, error) {
	var s ports.SnapshotEnsayoAgregadosIncidencias
	if !utf8.Valid(b) || catalogoefectos.DecodificarEstricto(b, sha, &s) != nil || s.VersionEsquema != 1 || !s.Demo {
		return ports.SnapshotEnsayoAgregadosIncidencias{}, domain.ErrAgregadosIncidenciasInvalidos
	}
	s.SHA256 = sha
	return s, nil
}

func CargarTextos(b []byte, sha, idioma string) (CatalogoTextos, error) {
	var c CatalogoTextos
	if !utf8.Valid(b) || catalogoefectos.DecodificarEstricto(b, sha, &c) != nil || c.VersionEsquema != "1" || len(idioma) > 64 || !etiquetaIdioma.MatchString(idioma) || c.Idioma != idioma || len(c.Textos) != 12 {
		return CatalogoTextos{}, domain.ErrAgregadosIncidenciasInvalidos
	}
	for _, k := range []string{"titulo", "aviso", "completo", "incompleto", "desconocido", "registro_incompleto", "pendiente", "resuelta", "observados", "cobertura", "limites", "entrada_invalida"} {
		v := c.Textos[k]
		if v == "" || strings.TrimSpace(v) != v || len(v) > 1024 || strings.ContainsAny(v, "\x00\r\n") || strings.ContainsRune(v, utf8.RuneError) {
			return CatalogoTextos{}, domain.ErrAgregadosIncidenciasInvalidos
		}
	}
	return c, nil
}
