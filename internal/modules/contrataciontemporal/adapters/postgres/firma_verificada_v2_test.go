package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type poolFirma172Prueba struct{ tx *txFirma172Prueba }

func (p *poolFirma172Prueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		p.tx.t.Fatal("registro V2 fuera de transacción serializable")
	}
	return p.tx, nil
}

type txFirma172Prueba struct {
	pgx.Tx
	t                  *testing.T
	commits, rollbacks int
}

func (tx *txFirma172Prueba) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	if q != ajustesRegistroIncorporacionTXV2 {
		tx.t.Fatal("ajustes V2 incorrectos")
	}
	return pgconn.CommandTag{}, nil
}
func (tx *txFirma172Prueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	if q != registrarFirmaSQL172 || len(args) != 12 {
		tx.t.Fatal("registro V2 no conserva observación separada ni fachada nominal")
	}
	return filaConsultaR5Prueba{contenido: []byte(`{"FirmaRef":"firma-ct:00000000-0000-4000-8000-000000000001"}`)}
}
func (tx *txFirma172Prueba) Commit(context.Context) error   { tx.commits++; return nil }
func (tx *txFirma172Prueba) Rollback(context.Context) error { tx.rollbacks++; return nil }
func TestRegistroV2ValidaAntesDeConfirmar(t *testing.T) {
	tx := &txFirma172Prueba{t: t}
	r := &RegistroFirmasVerificadasV2PostgreSQL{pool: &poolFirma172Prueba{tx}}
	err := r.operarFirma172(context.Background(), registrarFirmaSQL172, make([]any, 12), func([]byte) error { return ports.ErrResultadoFirmaDocumentoInvalido })
	if !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("recibo inválido confirmado: %v %d/%d", err, tx.commits, tx.rollbacks)
	}
}
func TestLecturaV2NoAdmiteCamposAusentes(t *testing.T) {
	w := respuestaFirmasSQL172{}
	if _, e := lecturaFirma172(w, ports.MaterialConsultaFirmasR5{}); !errors.Is(e, ports.ErrResultadoFirmaDocumentoInvalido) {
		t.Fatal("se aceptó una proyección incompleta")
	}
}
