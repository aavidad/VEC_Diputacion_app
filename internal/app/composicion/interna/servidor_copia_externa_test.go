package interna_test

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/app/composicion/interna"
)

func TestCopiaPorValorDeCapsulaFallaCerrada(t *testing.T) {
	original := interna.NuevaCapsulaOpacaParaPrueba(t)
	clon := *original
	if err := clon.EscucharYServir(); !errors.Is(err, interna.ErrServidorInternoInvalido) {
		t.Fatalf("copia abrio escucha: %v", err)
	}
	if err := clon.Apagar(context.Background()); !errors.Is(err, interna.ErrServidorInternoInvalido) {
		t.Fatalf("copia controlo ciclo de vida: %v", err)
	}
	if err := original.Apagar(context.Background()); err != nil {
		t.Fatalf("copia altero original: %v", err)
	}
}
