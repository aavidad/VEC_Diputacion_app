package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func TestPrepararTribunalRespetaContexto(t *testing.T) {
	if _, err := PrepararMaterialTribunal(nil, domain.MaterialTribunalPropuesto{}); err != ErrPreparacionTribunalNoDisponible {
		t.Fatalf("contexto ausente: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := PrepararMaterialTribunal(ctx, domain.MaterialTribunalPropuesto{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion: %v", err)
	}
}
