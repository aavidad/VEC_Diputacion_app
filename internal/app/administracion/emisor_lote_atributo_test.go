package administracion

import (
	"testing"

	lote "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
)

// Una decisión liga un único atributo con la huella del material: el del lote
// o el de su preparación. Dos atributos, otro nombre u otra huella se rechazan.
func TestAtributoMaterialLoteLoteOPreparacion(t *testing.T) {
	material := []byte(`{"Esquema":"x"}`)
	h := huellaMaterialLote(material)
	for nombre, caso := range map[string]struct {
		atributos map[string]string
		valido    bool
	}{
		"lote":          {map[string]string{lote.AtributoSolicitudLote: h}, true},
		"preparacion":   {map[string]string{lote.AtributoPreparacionLote: h}, true},
		"dos_atributos": {map[string]string{lote.AtributoSolicitudLote: h, lote.AtributoPreparacionLote: h}, false},
		"otro_nombre":   {map[string]string{"huella_sha256": h}, false},
		"otra_huella":   {map[string]string{lote.AtributoPreparacionLote: huellaMaterialLote([]byte("otro"))}, false},
		"sin_atributos": {map[string]string{}, false},
	} {
		if atributoMaterialLote(caso.atributos, material) != caso.valido {
			t.Fatal(nombre)
		}
	}
}
