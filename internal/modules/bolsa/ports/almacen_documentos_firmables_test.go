package ports

import (
	"context"
	"reflect"
	"testing"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// La implementacion deliberadamente minima convierte el principio de menor
// privilegio en una comprobacion de compilacion: si el puerto incorpora otra
// operacion, esta asignacion deja de compilar.
var _ AlmacenDocumentosFirmables = (*almacenDocumentosFirmablesMinimo)(nil)

type almacenDocumentosFirmablesMinimo struct{}

func (*almacenDocumentosFirmablesMinimo) Capacidades(context.Context) (puertosvec.CapacidadesAlmacenObjetos, error) {
	return puertosvec.CapacidadesAlmacenObjetos{}, nil
}

func (*almacenDocumentosFirmablesMinimo) Escribir(
	context.Context,
	puertosvec.SolicitudEscribirObjeto,
) (puertosvec.ResultadoOperacionObjeto, error) {
	return puertosvec.ResultadoOperacionObjeto{}, nil
}

func (*almacenDocumentosFirmablesMinimo) AplicarRetencion(
	context.Context,
	puertosvec.SolicitudRetenerObjeto,
) (puertosvec.ResultadoOperacionObjeto, error) {
	return puertosvec.ResultadoOperacionObjeto{}, nil
}

func TestAlmacenDocumentosFirmablesExponeSoloListaPositivaMinima(t *testing.T) {
	tipo := reflect.TypeOf((*AlmacenDocumentosFirmables)(nil)).Elem()
	esperados := map[string]struct{}{
		"AplicarRetencion": {},
		"Capacidades":      {},
		"Escribir":         {},
	}
	if tipo.NumMethod() != len(esperados) {
		t.Fatalf("el puerto expone %d operaciones; se esperaban exactamente %d", tipo.NumMethod(), len(esperados))
	}
	for nombre := range esperados {
		if _, existe := tipo.MethodByName(nombre); !existe {
			t.Errorf("falta la operacion minima %q", nombre)
		}
	}
	for _, prohibido := range []string{
		"Abrir",
		"Eliminar",
		"Promover",
		"Inmovilizar",
		"LevantarInmovilizacion",
	} {
		if _, existe := tipo.MethodByName(prohibido); existe {
			t.Errorf("el puerto concede la operacion no requerida %q", prohibido)
		}
	}
}
