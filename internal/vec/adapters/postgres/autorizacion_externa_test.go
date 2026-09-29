package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/ports"
)

type iniciadorFuenteExternaPrueba struct{ opciones pgx.TxOptions }

func (i *iniciadorFuenteExternaPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.opciones = opciones
	return nil, errors.New("fallo sintético antes de consultar")
}

func TestFuenteExternaUsaBarreraSerializableDeContextoActor(t *testing.T) {
	i := &iniciadorFuenteExternaPrueba{}
	a, err := nuevoAlmacenAutorizacion(i)
	if err != nil {
		t.Fatal(err)
	}
	a.externo = true
	_, err = a.ObtenerInstantaneaAutorizacion(context.Background(), "per_0123456789abcdef0123456789abcdef", "prf_0123456789abcdef0123456789abcdef")
	if !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("fallo cerrado: %v", err)
	}
	if i.opciones.IsoLevel != pgx.Serializable || i.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("ContextoActor exige SERIALIZABLE de escritura: %#v", i.opciones)
	}
}
