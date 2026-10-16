package bootstrap

import (
	"slices"
	"testing"
)

func TestMaterialSubidaLocalConservaCarteraSinPDF(t *testing.T) {
	base := audienciasInventarioPortalExternoV3(consumidorMeritosCarteraPortalExternoV3, 2)
	completo := audienciasInventarioPortalExternoV3(consumidorMeritosCarteraPortalExternoV3, 4)
	if len(base) != 2 || len(completo) != 4 || !slices.Equal(base, completo[:2]) ||
		completo[2] != audienciaSubidaLocalActoModulo || completo[3] != audienciaSubidaLocalDescargaPropia {
		t.Fatal("se perdió el inventario anterior de cartera o la separación documental")
	}
	for _, cantidad := range []int{0, 1, 3, 5} {
		if audienciasInventarioPortalExternoV3(consumidorMeritosCarteraPortalExternoV3, cantidad) != nil {
			t.Fatal("se admitió un inventario parcial o con audiencias adicionales")
		}
	}
	for _, nuevo := range descriptoresMaterialSubidaLocalExternaV3() {
		presentes := 0
		for _, descriptor := range descriptoresMaterialPortalExternoV3() {
			if descriptor.Audiencia == nuevo.Audiencia {
				presentes++
			} else if descriptor.Dominio == nuevo.Dominio || descriptor.Prefijo == nuevo.Prefijo || descriptor.ProveedorNominal == nuevo.ProveedorNominal {
				t.Fatal("la subida comparte una raíz nominal con otra operación")
			}
		}
		if presentes != 1 {
			t.Fatal("la audiencia documental no tiene un único descriptor")
		}
	}
}
