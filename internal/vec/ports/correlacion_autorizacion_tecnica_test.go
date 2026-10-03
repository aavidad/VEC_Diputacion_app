package ports

import (
	"context"
	"errors"
	"testing"
)

func TestReferenciaCorrelacionAutorizacionV2DePeticionConservaOrigenPrivado(t *testing.T) {
	ctx, err := ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	bruta, ok := CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		t.Fatal("correlacion privada no disponible")
	}
	ctx, cancelar := context.WithCancel(ctx)
	cancelar()
	referencia, err := ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	valor, err := referencia.ValorCanonico()
	if err != nil || valor != "correlacion_"+bruta {
		t.Fatal("referencia V3 distinta de la correlacion tecnica")
	}
	repetida, err := ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	valorRepetido, err := repetida.ValorCanonico()
	if err != nil || valorRepetido != valor {
		t.Fatal("referencia regenerada")
	}
}

func TestReferenciaCorrelacionAutorizacionV2DePeticionFallaSinContextoEmitido(t *testing.T) {
	for _, ctx := range []context.Context{nil, context.Background()} {
		if _, err := ReferenciaCorrelacionAutorizacionV2DePeticion(ctx); !errors.Is(err, ErrCorrelacionIncidenciasNoDisponible) {
			t.Fatalf("referencia inventada sin contexto: %v", err)
		}
	}
}
