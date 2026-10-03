package adminperfiles

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/ports"
)

type filaContextoFalsa struct{}

func (filaContextoFalsa) Scan(...any) error { return errors.New("fila sintética ausente") }

type txContextoFalsa struct{ pgx.Tx }

func (txContextoFalsa) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (txContextoFalsa) QueryRow(context.Context, string, ...any) pgx.Row { return filaContextoFalsa{} }
func (txContextoFalsa) Rollback(context.Context) error                   { return nil }

type poolContextoFalso struct{ opciones []pgx.TxOptions }

func (p *poolContextoFalso) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = append(p.opciones, opciones)
	return txContextoFalsa{}, nil
}
func (*poolContextoFalso) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaContextoFalsa{}
}

func TestContextoADMINRecuperaConInstantaneaReciente(t *testing.T) {
	pool := &poolContextoFalso{}
	resolver := &resolutorContexto{base: &PostgreSQL{pool: pool}}
	for _, consulta := range []string{registrarContexto, recuperarContexto} {
		_, _, err := resolver.ejecutar(context.Background(), consulta, nil,
			ports.SolicitudResolucionRegistroContextoActorV2{})
		if err == nil {
			t.Fatal("la fila sintética ausente se aceptó")
		}
	}
	if len(pool.opciones) != 2 || pool.opciones[0].IsoLevel != pgx.Serializable ||
		pool.opciones[1].IsoLevel != pgx.ReadCommitted ||
		pool.opciones[0].AccessMode != pgx.ReadWrite || pool.opciones[1].AccessMode != pgx.ReadWrite {
		t.Fatalf("aislamiento de registrar/reconciliar incorrecto: %+v", pool.opciones)
	}
}
