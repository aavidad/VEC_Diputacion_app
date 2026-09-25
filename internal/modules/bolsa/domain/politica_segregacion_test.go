package domain

import (
	"errors"
	"slices"
	"testing"
)

func TestPoliticaSegregacionMinimaEsLaExclusion(t *testing.T) {
	var cero PoliticaSegregacion
	for _, politica := range []PoliticaSegregacion{cero, PoliticaSegregacionMinima()} {
		if !slices.Equal(politica.Operaciones(), []string{OperacionExcluir}) ||
			!politica.ExigeSegundaPersona(OperacionExcluir) ||
			politica.ExigeSegundaPersona(OperacionPausar) || politica.ExigeSegundaPersona(OperacionReactivar) {
			t.Fatalf("la política mínima debe ser la de hoy: %v", politica.Operaciones())
		}
	}
}

func TestPoliticaSegregacionConfiguradaSoloAnade(t *testing.T) {
	politica, err := NuevaPoliticaSegregacion([]string{OperacionExcluir, OperacionPausar})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(politica.Operaciones(), []string{OperacionPausar, OperacionExcluir}) ||
		!politica.ExigeSegundaPersona(OperacionPausar) || politica.ExigeSegundaPersona(OperacionReactivar) {
		t.Fatalf("política inesperada: %v", politica.Operaciones())
	}
	copia := politica.Operaciones()
	copia[0] = "otra"
	if !politica.ExigeSegundaPersona(OperacionPausar) {
		t.Fatal("la lista devuelta no es una copia")
	}
	for _, invalida := range [][]string{
		nil, {}, {OperacionPausar}, {OperacionExcluir, OperacionExcluir},
		{OperacionExcluir, "cancelar"}, {OperacionExcluir, ""},
	} {
		if _, err := NuevaPoliticaSegregacion(invalida); !errors.Is(err, ErrPoliticaSegregacionInvalida) {
			t.Errorf("%v debe rechazarse: %v", invalida, err)
		}
	}
}
