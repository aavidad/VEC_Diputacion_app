package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type poolFirmaVerificadaPrueba struct {
	tx *txFirmaVerificadaPrueba
}

func (p *poolFirmaVerificadaPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		p.tx.t.Fatal("la firma verificada exige SERIALIZABLE y escritura")
	}
	return p.tx, nil
}

type txFirmaVerificadaPrueba struct {
	pgx.Tx
	t                     *testing.T
	commits, rollbacks    int
	consulta, configurada bool
}

func (tx *txFirmaVerificadaPrueba) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	if q != ajustesRegistroIncorporacionTXV2 || len(args) != 0 {
		tx.t.Fatal("los ajustes de la transacción no son los aprobados")
	}
	tx.configurada = true
	return pgconn.CommandTag{}, nil
}

func (tx *txFirmaVerificadaPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	if !tx.configurada || q != registrarFirmaExternaSQL170 || len(args) != 11 {
		tx.t.Fatal("consulta SQL fuera del contrato CT170")
	}
	tx.consulta = true
	return filaFirmaVerificadaPrueba{}
}
func (tx *txFirmaVerificadaPrueba) Commit(context.Context) error {
	tx.commits++
	return nil
}
func (tx *txFirmaVerificadaPrueba) Rollback(context.Context) error {
	tx.rollbacks++
	return nil
}

type filaFirmaVerificadaPrueba struct{}

func (filaFirmaVerificadaPrueba) Scan(dst ...any) error {
	*dst[0].(*[]byte) = []byte(`{"FirmaRef":"firma-ct:00000000-0000-4000-8000-000000000001"}`)
	return nil
}

func TestFirmaVerificadaValidaReciboAntesDeCommit(t *testing.T) {
	for _, rechazar := range []bool{true, false} {
		t.Run(map[bool]string{true: "recibo_invalido", false: "recibo_valido"}[rechazar], func(t *testing.T) {
			tx := &txFirmaVerificadaPrueba{t: t}
			r := &RegistroFirmasExternasPostgreSQL{pool: &poolFirmaVerificadaPrueba{tx: tx}}
			fallo := ports.ErrResultadoFirmaDocumentoInvalido
			err := r.registrarFirmaExternaUnaVez(context.Background(), make([]any, 11), func(reciboFirmaSQL118) error {
				if rechazar {
					return fallo
				}
				return nil
			})
			if !tx.consulta {
				t.Fatal("el registro SQL no se ejecutó")
			}
			if rechazar {
				if !errors.Is(err, fallo) || tx.commits != 0 || tx.rollbacks != 1 {
					t.Fatalf("recibo inválido confirmado: err=%v commits=%d rollbacks=%d", err, tx.commits, tx.rollbacks)
				}
			} else if err != nil || tx.commits != 1 || tx.rollbacks != 0 {
				t.Fatalf("recibo válido sin confirmar: err=%v commits=%d rollbacks=%d", err, tx.commits, tx.rollbacks)
			}
		})
	}
}
