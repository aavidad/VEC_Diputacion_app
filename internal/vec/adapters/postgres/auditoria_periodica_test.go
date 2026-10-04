package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/ports"
)

type poolPeriodicoPrueba struct {
	txs    []*txPeriodicoPrueba
	usados int
}

func (p *poolPeriodicoPrueba) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if ctx.Err() != nil || o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		return nil, errors.New("transaccion_invalida")
	}
	if p.usados >= len(p.txs) {
		return nil, errors.New("transaccion_extra")
	}
	tx := p.txs[p.usados]
	p.usados++
	return tx, nil
}

type txPeriodicoPrueba struct {
	pgx.Tx
	queryErr, commitErr error
	raw                 []byte
	cerrada             bool
	consulta            string
	contextoCancelado   bool
}

func (tx *txPeriodicoPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (tx *txPeriodicoPrueba) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	tx.consulta = q
	tx.contextoCancelado = ctx.Err() != nil
	return filaPeriodicaPrueba{raw: tx.raw, err: tx.queryErr}
}
func (tx *txPeriodicoPrueba) Rollback(context.Context) error {
	if tx.cerrada {
		return pgx.ErrTxClosed
	}
	tx.cerrada = true
	return nil
}
func (tx *txPeriodicoPrueba) Commit(context.Context) error { tx.cerrada = true; return tx.commitErr }

type filaPeriodicaPrueba struct {
	raw []byte
	err error
}

func (f filaPeriodicaPrueba) Scan(args ...any) error {
	if f.err != nil {
		return f.err
	}
	*args[0].(*[]byte) = append([]byte(nil), f.raw...)
	return nil
}
func TestPeriodicoNoEntregaAcuseSiCommitEsIncierto(t *testing.T) {
	tx := &txPeriodicoPrueba{raw: []byte(`{"estado":"no_vencido","acuse":{}}`), commitErr: errors.New("respuesta_perdida")}
	p := &poolPeriodicoPrueba{txs: []*txPeriodicoPrueba{tx}}
	f := &FuenteCheckpointPeriodicoPostgreSQL{pool: p, maxRegistros: 10}
	ctx, _ := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	c, err := f.CapturarCheckpointPendiente(ctx)
	if !errors.Is(err, ErrCheckpointPeriodicoCommitIndeterminado) || c.Estado != "" || p.usados != 1 {
		t.Fatal("un COMMIT incierto no debe entregar captura ni registrar falso rollback")
	}
}
func TestPeriodicoAuditaErrorDespuesDeRollback(t *testing.T) {
	original := &txPeriodicoPrueba{queryErr: &pgconn.PgError{Code: "42501", Message: "dato_que_no_debe_salir"}}
	registro := &txPeriodicoPrueba{raw: []byte(`{"auditoria_ref":"aud_v3_per_sintetico","secuencia":1}`)}
	p := &poolPeriodicoPrueba{txs: []*txPeriodicoPrueba{original, registro}}
	f := &FuenteCheckpointPeriodicoPostgreSQL{pool: p, maxRegistros: 10}
	ctx, _ := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	_, err := f.CapturarCheckpointPendiente(ctx)
	if err == nil || strings.Contains(err.Error(), "dato_que_no_debe_salir") || p.usados != 2 || !original.cerrada ||
		!registro.cerrada || !strings.Contains(registro.consulta, "registrar_intento_periodico_v1") || registro.contextoCancelado {
		t.Fatal("debe conservar el fallo en otra transacción sin exponer mensajes SQL")
	}
}
