package catalogoincidencias

import (
	"regexp"
	"strings"
	"unicode/utf8"
	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const LimiteJSON = catalogoefectos.LimiteJSON

var etiquetaIdioma = regexp.MustCompile(`^[A-Za-z]{2,8}(-[A-Za-z0-9]{1,8})*$`)

type CatalogoTextos struct {
	VersionEsquema string            `json:"version_esquema"`
	Idioma         string            `json:"idioma"`
	Textos         map[string]string `json:"textos"`
}

func CargarSnapshot(b []byte, sha string) (ports.SnapshotEnsayoIncidencias, error) {
	var s ports.SnapshotEnsayoIncidencias
	if !utf8.Valid(b) || catalogoefectos.DecodificarEstricto(b, sha, &s) != nil || s.VersionEsquema != 1 || !s.Demostracion {
		return ports.SnapshotEnsayoIncidencias{}, domain.ErrIncidenciasPeriodoInvalido
	}
	s.SHA256 = sha
	return s, nil
}

func CargarTextos(b []byte, sha, idioma string) (CatalogoTextos, error) {
	var c CatalogoTextos
	claves := []string{"titulo", "aviso", "registrado", "registro_incompleto", "sin_registros", "indeterminado", "no_evaluado", "complete", "incomplete", "not_available", "cobertura", "limites", "entrada_invalida"}
	if !utf8.Valid(b) || catalogoefectos.DecodificarEstricto(b, sha, &c) != nil || c.VersionEsquema != "1" || (len(idioma) > 64 || !etiquetaIdioma.MatchString(idioma)) || c.Idioma != idioma || len(c.Textos) != len(claves) {
		return CatalogoTextos{}, domain.ErrIncidenciasPeriodoInvalido
	}
	for _, k := range claves {
		v := c.Textos[k]
		if strings.TrimSpace(v) != v || v == "" || len(v) > 1024 {
			return CatalogoTextos{}, domain.ErrIncidenciasPeriodoInvalido
		}
	}
	return c, nil
}
