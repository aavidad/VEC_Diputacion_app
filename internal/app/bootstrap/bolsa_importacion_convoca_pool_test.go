package bootstrap

import (
	"context"
	"errors"
	"testing"
	"vec-diputacion-granada/config"
)

func TestPoolImportacionConvocaFallaFueraDelPerfilDesarrollo(t *testing.T) {
	_, e := abrirPoolImportacionConvoca(context.Background(), config.Config{})
	if !errors.Is(e, ErrPoolImportacionConvocaNoDisponible) {
		t.Fatalf("error=%v", e)
	}
}
