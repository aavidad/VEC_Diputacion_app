package domain

import (
	"errors"
	"regexp"
)

var ErrVinculoCategoriaRPTInvalido = errors.New("contratacion temporal: vinculo de categoria RPT invalido")

var claveCategoriaRPT = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)

// PublicacionCategoriaRPT identifica una entrada de una publicacion exacta.
// La denominacion y el grupo no constituyen una identidad alternativa.
type PublicacionCategoriaRPT struct {
	CatalogoID      string `json:"catalogo_id"`
	ModuloID        string `json:"modulo_id"`
	CatalogoVersion uint64 `json:"catalogo_version"`
	CatalogoHuella  string `json:"catalogo_huella_sha256"`
	CategoriaID     string `json:"categoria_id"`
	CategoriaClave  string `json:"categoria_clave"`
}

func (p PublicacionCategoriaRPT) Validar() error {
	if !claveCategoriaRPT.MatchString(p.CatalogoID) ||
		!claveCategoriaRPT.MatchString(p.ModuloID) ||
		p.CatalogoVersion == 0 || p.CatalogoVersion > 2147483647 ||
		!HuellaSHA256FirmaValida(p.CatalogoHuella) ||
		!claveCategoriaRPT.MatchString(p.CategoriaID) ||
		!claveCategoriaRPT.MatchString(p.CategoriaClave) {
		return ErrVinculoCategoriaRPTInvalido
	}
	return nil
}

// CategoriaRefDelAnalisis debe coincidir literalmente con los dos identificadores
// publicados. Ningun nombre, grupo o correspondencia de demostracion autoriza
// convertir una categoria historica distinta.
func (p PublicacionCategoriaRPT) CorrespondeA(categoriaRefDelAnalisis string) bool {
	return p.Validar() == nil && categoriaRefDelAnalisis == p.CategoriaID &&
		categoriaRefDelAnalisis == p.CategoriaClave
}
