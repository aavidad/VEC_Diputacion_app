package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type inicioSeleccionOfertaPrueba struct {
	tx       pgx.Tx
	inicios  int
	opciones pgx.TxOptions
}

func (i *inicioSeleccionOfertaPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	i.inicios++
	i.opciones = o
	return i.tx, nil
}

type filaSeleccionOfertaPrueba struct{ raw []byte }

func (f filaSeleccionOfertaPrueba) Scan(dest ...any) error {
	if len(dest) != 1 {
		return errors.New("columnas inesperadas")
	}
	p, ok := dest[0].(*[]byte)
	if !ok {
		return errors.New("destino inesperado")
	}
	*p = append([]byte(nil), f.raw...)
	return nil
}

type txSeleccionOfertaPrueba struct {
	pgx.Tx
	consultas, confirmaciones int
	fallaCommit               error
}

func (t *txSeleccionOfertaPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *txSeleccionOfertaPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	t.consultas++
	return filaSeleccionOfertaPrueba{[]byte(`{"situacion":"ausente"}`)}
}

func (t *txSeleccionOfertaPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return t.fallaCommit
}

func (t *txSeleccionOfertaPrueba) Rollback(context.Context) error { return nil }

func TestRegistroSeleccionOfertaCommitInciertoNoRepiteEfecto(t *testing.T) {
	t.Parallel()
	fallo := errors.New("respuesta de commit perdida")
	tx := &txSeleccionOfertaPrueba{fallaCommit: fallo}
	inicio := &inicioSeleccionOfertaPrueba{tx: tx}
	registro, err := nuevoRegistroSeleccionOfertaPostgreSQL(inicio)
	if err != nil {
		t.Fatal(err)
	}
	_, err = registro.ejecutar(t.Context(), "SELECT operacion_ficticia()")
	if !errors.Is(err, errRegistroSeleccionOfertaPostgreSQL) {
		t.Fatalf("commit incierto debe cerrar el resultado: %v", err)
	}
	if inicio.inicios != 1 || tx.consultas != 1 || tx.confirmaciones != 1 ||
		inicio.opciones.IsoLevel != pgx.Serializable || inicio.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("mutación repetida o transacción débil: inicio=%d consulta=%d commit=%d opciones=%+v",
			inicio.inicios, tx.consultas, tx.confirmaciones, inicio.opciones)
	}
}

func TestRegistroSeleccionOfertaAusenteNoAceptaComandoFabricado(t *testing.T) {
	t.Parallel()
	_, err := decodificarEstadoSeleccionOferta([]byte(`{"situacion":"ausente","comando":{"bolsa_ref":"bolsa:falsa"}}`),
		ports.IntencionSeleccionOfertaCT{})
	if !errors.Is(err, errRegistroSeleccionOfertaPostgreSQL) {
		t.Fatalf("un estado ausente con comando no puede publicarse: %v", err)
	}
}
