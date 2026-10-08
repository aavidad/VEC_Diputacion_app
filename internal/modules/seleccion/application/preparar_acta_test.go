package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func TestPrepararActaRespetaCancelacion(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := PrepararMaterialActa(ctx, domain.MaterialActaPropuesto{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion: %v", err)
	}
}
