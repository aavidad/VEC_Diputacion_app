package domain

import (
	"strings"
	"testing"
)

func TestPropuestaDecisionCoberturaLigaPreparacionGlobalDeEvidencias(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := propuesta.VinculoParaDecision(
		propuesta.ViaPropuesta(),
		datos.GeneradaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	if vinculo.PreparacionEvidenciasRef !=
		datos.PreparacionEvidenciasRef ||
		vinculo.PreparacionEvidenciasHuellaSHA256 !=
			datos.PreparacionEvidenciasHuellaSHA256 {
		t.Fatalf("vinculo probatorio incompleto: %#v", vinculo)
	}

	otraEntrada := datosPropuestaDecisionCoberturaPrueba(t)
	otraEntrada.PreparacionEvidenciasHuellaSHA256 =
		strings.Repeat("d", 64)
	otra, err := CrearPropuestaDecisionCobertura(otraEntrada)
	if err != nil {
		t.Fatal(err)
	}
	if otra.HuellaSHA256() == propuesta.HuellaSHA256() {
		t.Fatal("cambiar la evidencia global no altero la propuesta")
	}
}

func TestPropuestaDecisionCoberturaRechazaPreparacionGlobalAdulterada(
	t *testing.T,
) {
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	publicacion := propuesta.Publicacion()
	publicacion.PreparacionEvidenciasRef =
		"preparacion_evidencias_adulterada_01"
	if _, err := RestaurarPropuestaDecisionCobertura(
		publicacion,
		datos.Catalogo,
		datos.Politica,
	); err == nil {
		t.Fatal("se restauro una preparacion distinta con la huella anterior")
	}

	publicacion = propuesta.Publicacion()
	publicacion.PreparacionEvidenciasHuellaSHA256 = strings.Repeat("e", 64)
	if _, err := RestaurarPropuestaDecisionCobertura(
		publicacion,
		datos.Catalogo,
		datos.Politica,
	); err == nil {
		t.Fatal("se restauro una huella probatoria adulterada")
	}
}

func TestPropuestaDecisionCoberturaExigePreparacionGlobalValida(
	t *testing.T,
) {
	casos := []func(*DatosCrearPropuestaDecisionCobertura){
		func(datos *DatosCrearPropuestaDecisionCobertura) {
			datos.PreparacionEvidenciasRef = ""
		},
		func(datos *DatosCrearPropuestaDecisionCobertura) {
			datos.PreparacionEvidenciasHuellaSHA256 = ""
		},
		func(datos *DatosCrearPropuestaDecisionCobertura) {
			datos.PreparacionEvidenciasHuellaSHA256 =
				strings.Repeat("0", 64)
		},
	}
	for indice, alterar := range casos {
		datos := datosPropuestaDecisionCoberturaPrueba(t)
		alterar(&datos)
		if _, err := CrearPropuestaDecisionCobertura(datos); err == nil {
			t.Fatalf("preparacion invalida aceptada: caso %d", indice)
		}
	}
}
