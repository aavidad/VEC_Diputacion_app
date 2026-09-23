package interna

import (
	"context"
	"errors"
	"reflect"
	"testing"

	httpinterno "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultorCuadroRRHHMontajePrueba struct{}

func (*consultorCuadroRRHHMontajePrueba) Consultar(context.Context, ctports.SolicitudCuadroRRHH) (ctports.PaginaCuadroRRHH, error) {
	return ctports.PaginaCuadroRRHH{}, errors.New("no debe consultar")
}

type consultorDetalleRRHHMontajePrueba struct{}

func (*consultorDetalleRRHHMontajePrueba) Consultar(context.Context, ctports.SolicitudDetalleRRHH) (ctports.DetalleExpedienteRRHH, error) {
	return ctports.DetalleExpedienteRRHH{}, errors.New("no debe consultar")
}

func TestMontajeConsultasRRHHSoloRutasCuadroYDetalle(t *testing.T) {
	rutas, err := rutasConsultoresRRHH(&consultorCuadroRRHHMontajePrueba{}, &consultorDetalleRRHHMontajePrueba{})
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
	var cuadroNulo *consultorCuadroRRHHMontajePrueba
	if rutas, err := rutasConsultoresRRHH(cuadroNulo, &consultorDetalleRRHHMontajePrueba{}); rutas != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) {
		t.Fatalf("montaje con consultor nulo = (%v, %v)", rutas, err)
	}
}

func TestLecturasRRHHSinPoolNominalNoPublicanRutas(t *testing.T) {
	rutas, err := nuevasRutasConsultasRRHH(dependenciasLecturasRRHH{}, autoridadContextoConsultaRRHH{})
	if rutas != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) {
		t.Fatalf("lecturas CT sin pool = (%v, %v)", rutas, err)
	}
}
