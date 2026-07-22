package canonico

import (
	"testing"
	"time"
)

func catalogoCategoriasPrueba(t *testing.T) CatalogoCategoriasV1 {
	t.Helper()
	catalogo, err := NuevoCatalogoCategoriasV1("categorias-profesionales", 1, []CategoriaCatalogoV1{{
		Clave: "auxiliar-administrativo", Etiqueta: "Auxiliar administrativo",
		Descripcion: "Categoría profesional de auxiliar administrativo.",
		Semantica:   "informacion", Orden: 1, Area: "administracion",
		AreaEtiqueta: "Administración", Suscribible: true,
		VigenteDesde: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return catalogo
}

func TestHuellaCatalogoCategoriasVinculaContenidoPublico(t *testing.T) {
	catalogo := catalogoCategoriasPrueba(t)
	huella, err := catalogo.HuellaSHA256()
	if err != nil || len(huella) != 64 {
		t.Fatalf("huella = %q, %v", huella, err)
	}
	mutado := catalogo
	mutado.Categorias = clonarCategorias(catalogo.Categorias)
	mutado.Categorias[0].Etiqueta = "Etiqueta alterada"
	huellaMutada, err := mutado.HuellaSHA256()
	if err != nil || huellaMutada == huella {
		t.Fatalf("la huella no vincula la etiqueta: %q, %v", huellaMutada, err)
	}
	if huella != "4125f5b5f12f3da31fff30aa699239592d02b01b1676e98d8fa1ab7beb30ad7d" {
		t.Fatalf("la preimagen canónica cambió: %s", huella)
	}
}
