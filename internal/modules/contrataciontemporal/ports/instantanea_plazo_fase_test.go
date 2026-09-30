package ports

import (
	"bytes"
	"encoding/json"
	"testing"

	"vec-diputacion-granada/internal/vec/reglas"
)

func TestPaginaCuadroRRHHNoPublicaCatalogosCapturados(t *testing.T) {
	t.Parallel()
	pagina := PaginaCuadroRRHH{InstantaneasPlazo: []*reglas.InstantaneaPersistidaRegla{{
		CatalogoBaseCanonico: []byte("catalogo_privado_ct157"),
		CanonicoAjustes:      []byte("ajustes_privados_ct157"),
	}}}
	contenido, err := json.Marshal(pagina)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(contenido, []byte("catalogo_privado_ct157")) ||
		bytes.Contains(contenido, []byte("ajustes_privados_ct157")) ||
		bytes.Contains(contenido, []byte("instantaneas_plazo")) {
		t.Fatal("la página HTTP expone una definición histórica completa")
	}
}
