package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type poolConsultaPrueba struct {
	tx          *txConsultaPrueba
	falloInicio error
	opciones    pgx.TxOptions
	comienzos   int
}

func (p *poolConsultaPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.comienzos++
	p.opciones = o
	if p.falloInicio != nil {
		return nil, p.falloInicio
	}
	return p.tx, nil
}

func TestInicioDenegadoNoSimulaLecturaNiFiltraSQL(t *testing.T) {
	p := &poolConsultaPrueba{tx: &txConsultaPrueba{}, falloInicio: &pgconn.PgError{Code: "42501", Message: "dato privado"}}
	salida, err := consultarTransaccion(context.Background(), p, nil, &peticion{}, listarSQL)
	if salida != nil || !errors.Is(err, ports.ErrLecturaUsuariosAdministrablesNoDisponible) || errors.Is(err, domain.ErrAutorizacionDenegada) || p.comienzos != 1 || p.tx.consultas != 0 || p.tx.commits != 0 {
		t.Fatalf("inicio: salida=%v err=%v consultas=%d", salida, err, p.tx.consultas)
	}
}
func (*poolConsultaPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("consulta externa inesperada")
}

type txConsultaPrueba struct {
	pgx.Tx
	respuesta                     []byte
	falloCommit                   error
	consultas, commits, rollbacks int
}

func (t *txConsultaPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	t.consultas++
	return filaConsultaPrueba{t.respuesta}
}
func (t *txConsultaPrueba) Commit(context.Context) error   { t.commits++; return t.falloCommit }
func (t *txConsultaPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

type filaConsultaPrueba struct{ dato []byte }

func (f filaConsultaPrueba) Scan(dest ...any) error {
	*dest[0].(*[]byte) = append([]byte(nil), f.dato...)
	return nil
}

func TestConsultaConfirmaSoloTrasAcuseYCommitConocido(t *testing.T) {
	a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
	p, err := materialListar(a, ports.FiltrosUsuariosAdministrables{})
	if err != nil {
		t.Fatal(err)
	}
	p.decisionRef = "dec_prueba"
	p.contextoSHA = fmt.Sprintf("%064x", 1)
	c := consumoPrueba()
	c["efecto_ref"] = p.recurso.Referencia
	correcta, _ := json.Marshal(map[string]any{"datos": map[string]any{"personas": []any{}, "siguiente_cursor": ""}, "consumo": c})
	for _, caso := range []struct {
		nombre      string
		respuesta   []byte
		falloCommit error
		commits     int
		debeSalir   bool
	}{
		{"exito", correcta, nil, 1, true},
		{"commit_incierto", correcta, errors.New("commit incierto"), 1, false},
		{"acuse_adulterado", []byte(`{"datos":{},"consumo":{}}`), nil, 0, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txConsultaPrueba{respuesta: caso.respuesta, falloCommit: caso.falloCommit}
			pool := &poolConsultaPrueba{tx: tx}
			salida, err := consultarTransaccion(context.Background(), pool, nil, &p, listarSQL)
			if (err == nil) != caso.debeSalir || (salida != nil) != caso.debeSalir {
				t.Fatalf("salida=%t error=%v", salida != nil, err)
			}
			if pool.comienzos != 1 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite || tx.consultas != 1 || tx.commits != caso.commits || tx.rollbacks != 1 {
				t.Fatalf("transacción: comienzos=%d consultas=%d commits=%d rollback=%d", pool.comienzos, tx.consultas, tx.commits, tx.rollbacks)
			}
		})
	}
}
