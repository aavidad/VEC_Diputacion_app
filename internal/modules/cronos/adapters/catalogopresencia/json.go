package catalogopresencia

import (
	"strings"
	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const LimiteJSON = catalogoefectos.LimiteJSON

type CatalogoTextos struct {
	VersionEsquema string            `json:"version_esquema"`
	Idioma         string            `json:"idioma"`
	Textos         map[string]string `json:"textos"`
}

// CargarSnapshot verifies the exact bytes. SHA identifies a fixture, not an
// authorization, signature, audit record or proof that data is synthetic.
func CargarSnapshot(b []byte, sha string) (ports.SnapshotEnsayoPresencia, error) {
	var s ports.SnapshotEnsayoPresencia
	if catalogoefectos.DecodificarEstricto(b, sha, &s) != nil || s.VersionEsquema != 1 || !s.Demostracion {
		return ports.SnapshotEnsayoPresencia{}, domain.ErrPresenciaInvalida
	}
	s.SHA256 = sha
	return s, nil
}
func CargarTextos(b []byte, sha, idioma string) (CatalogoTextos, error) {
	var c CatalogoTextos
	if catalogoefectos.DecodificarEstricto(b, sha, &c) != nil || c.VersionEsquema != "1" || (idioma == "" || strings.TrimSpace(idioma) != idioma || len(idioma) > 64) || c.Idioma != idioma || len(c.Textos) != 13 {
		return CatalogoTextos{}, domain.ErrPresenciaInvalida
	}
	for _, k := range []string{"titulo", "aviso", "entrada_registrada", "pausa_registrada", "salida_registrada", "indeterminado", "agregado", "cobertura", "limites", "entrada_invalida", "cobertura_incompleta", "sin_marcajes", "secuencia_ambigua"} {
		v := c.Textos[k]
		if strings.TrimSpace(v) != v || v == "" || len(v) > 1024 {
			return CatalogoTextos{}, domain.ErrPresenciaInvalida
		}
	}
	return c, nil
}
