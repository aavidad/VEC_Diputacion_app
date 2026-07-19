package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type motorRutasPrueba struct {
	resultado dietasports.ResultadoCalculoRuta
	error     error
}

func (m motorRutasPrueba) Calcular(_ context.Context, _ dietasports.SolicitudCalculoRuta) (dietasports.ResultadoCalculoRuta, error) {
	return m.resultado, m.error
}

func TestServicioCalculoRutasConservaElPuertoIntercambiable(t *testing.T) {
	esperado := dietasports.ResultadoCalculoRuta{VersionGrafo: "grafo-v1", Motor: "motor-prueba"}
	servicio, err := NuevoServicioCalculoRutas(motorRutasPrueba{resultado: esperado})
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := servicio.Calcular(context.Background(), dietasports.SolicitudCalculoRuta{})
	if err != nil || !reflect.DeepEqual(obtenido, esperado) {
		t.Fatalf("resultado=%+v error=%v", obtenido, err)
	}
}

func TestServicioCalculoRutasRechazaMotorAusente(t *testing.T) {
	if servicio, err := NuevoServicioCalculoRutas(nil); servicio != nil || !errors.Is(err, ErrComposicionCalculoRutasInvalida) {
		t.Fatalf("servicio=%v error=%v", servicio, err)
	}
	var servicio *ServicioCalculoRutas
	if _, err := servicio.Calcular(context.Background(), dietasports.SolicitudCalculoRuta{}); !errors.Is(err, ErrComposicionCalculoRutasInvalida) {
		t.Fatalf("servicio nil: error=%v", err)
	}
}
