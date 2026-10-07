package domain

import (
	"strings"
	"testing"
)

func TestPublicacionCategoriaRPTExigeClaveLiteral(t *testing.T) {
	p := PublicacionCategoriaRPT{CatalogoID: "rpt-categorias", ModuloID: "personal", CatalogoVersion: 2, CatalogoHuella: strings.Repeat("a", 64), CategoriaID: "categoria:tecnica", CategoriaClave: "categoria:tecnica"}
	if !p.CorrespondeA("categoria:tecnica") {
		t.Fatal("la misma clave publicada debe corresponder")
	}
	if p.CorrespondeA("categoria:auxiliar") {
		t.Fatal("una categoria distinta no puede vincularse")
	}
	p.CategoriaClave = "categoria:equivalente"
	if p.CorrespondeA("categoria:tecnica") {
		t.Fatal("la entrada publicada debe tener la clave literal")
	}
	p.CategoriaClave = p.CategoriaID
	p.CatalogoVersion = 2147483648
	if p.Validar() == nil {
		t.Fatal("RPT SQL no admite version fuera de integer")
	}
}
