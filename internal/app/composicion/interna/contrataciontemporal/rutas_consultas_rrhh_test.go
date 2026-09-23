package contrataciontemporal

import (
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestRutasConsultasRRHHSoloCuadroYDetalle(t *testing.T) {
	rutas, err := NuevasRutasConsultasRRHH(DependenciasConsultasRRHH{
		Cuadro:  &consultorCuadroRRHHComposicionPrueba{},
		Detalle: &consultorDetalleRRHHComposicionPrueba{},
	})
	if err != nil {
		t.Fatal(err)
	}
	obtenidas := make([]string, len(rutas))
	for i, ruta := range rutas {
		if ruta.Manejador == nil {
			t.Fatalf("ruta %d sin handler", i)
		}
		obtenidas[i] = ruta.Ruta
	}
	esperadas := []string{httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH}
	if !reflect.DeepEqual(obtenidas, esperadas) {
		t.Fatalf("rutas = %v", obtenidas)
	}
}

func TestRutasConsultasRRHHRechazanDependenciasIncompletas(t *testing.T) {
	var cuadroNulo *consultorCuadroRRHHComposicionPrueba
	casos := []DependenciasConsultasRRHH{
		{},
		{Cuadro: &consultorCuadroRRHHComposicionPrueba{}},
		{Cuadro: cuadroNulo, Detalle: &consultorDetalleRRHHComposicionPrueba{}},
	}
	for _, caso := range casos {
		rutas, err := NuevasRutasConsultasRRHH(caso)
		if rutas != nil || !errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
			t.Fatalf("montaje incompleto = (%v, %v)", rutas, err)
		}
	}
}
