package bootstrap

import (
	"net/http"
	"testing"

	dietasdocs "vec-diputacion-granada/internal/modules/dietas/adapters/documentos"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
)

func TestDocumentoLiquidacionCatalogosYRutaAutorizada(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		f, err := abrirCatalogoDocumentoLiquidacion(idioma)
		if err != nil {
			t.Fatal(err)
		}
		textos, err := dietasdocs.CargarTextosLiquidacionDocumento(f)
		_ = f.Close()
		if err != nil || textos.Idioma != idioma {
			t.Fatalf("catalogo %s: %v", idioma, err)
		}
	}
	ruta := dietashttp.RutaLiquidaciones + "dco_aaaaaaaaaaaaaaaaaaaaaa/liquidaciones/3/documento"
	if !esRutaLiquidacionEconomica(ruta) || !metodoLiquidacionEconomicaValido(ruta, http.MethodGet) || metodoLiquidacionEconomicaValido(ruta, http.MethodPost) {
		t.Fatal("ruta documental sin política GET")
	}
}
