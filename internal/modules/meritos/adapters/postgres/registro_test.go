package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/meritos/ports"
)

type poolPrueba struct {
	tx       *txPrueba
	opciones pgx.TxOptions
	llamadas int
}

func (p *poolPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.llamadas++
	p.opciones = o
	return p.tx, nil
}

type txPrueba struct {
	pgx.Tx
	raw                    []byte
	errConsulta, errCommit error
	commits, rollbacks     int
	consulta               string
	argumentos             []any
}

func (tx *txPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (tx *txPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	tx.consulta = q
	tx.argumentos = args
	return filaPrueba{tx.raw, tx.errConsulta}
}
func (tx *txPrueba) Commit(context.Context) error   { tx.commits++; return tx.errCommit }
func (tx *txPrueba) Rollback(context.Context) error { tx.rollbacks++; return nil }

type filaPrueba struct {
	raw []byte
	err error
}

func (f filaPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*dest[0].(*[]byte) = append([]byte{}, f.raw...)
	return nil
}

func TestAdaptadorValidaResultadoAntesDeCommit(t *testing.T) {
	orden, resultado := ordenPrueba(t)
	raw, err := json.Marshal(resultado)
	if err != nil {
		t.Fatal(err)
	}
	tx := &txPrueba{raw: raw}
	pool := &poolPrueba{tx: tx}
	repo, _ := nuevoRegistro(pool)
	out, err := repo.EjecutarOperacion(context.Background(), orden)
	if err != nil || out.Recibo.Referencia != resultado.Recibo.Referencia || tx.commits != 1 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite || tx.consulta != consultaOperacion || len(tx.argumentos) != 11 {
		t.Fatal("transacción no confirma resultado exacto", err)
	}
}

func TestAdaptadorDenegacionNominalConfirmaAuditoriaAntesDeRetornar(t *testing.T) {
	orden, _ := ordenPrueba(t)
	tx := &txPrueba{raw: []byte(`{"codigo":"conflicto_version","auditoria_ref":"auditoria:rechazo","anterior":null,"recibo":null}`)}
	repo, _ := nuevoRegistro(&poolPrueba{tx: tx})
	out, err := repo.EjecutarOperacion(context.Background(), orden)
	if err != nil || out.Codigo != "conflicto_version" || out.Recibo != nil || tx.commits != 1 {
		t.Fatal("rechazo auditado no confirmado", err)
	}
}

func TestAdaptadorNoDevuelveResultadoDeCommitIncierto(t *testing.T) {
	orden, resultado := ordenPrueba(t)
	raw, _ := json.Marshal(resultado)
	for _, codigo := range []string{"confirmada", "denegada"} {
		t.Run(codigo, func(t *testing.T) {
			if codigo == "denegada" {
				raw = []byte(`{"codigo":"denegada","auditoria_ref":"auditoria:rechazo","anterior":null,"recibo":null}`)
			}
			tx := &txPrueba{raw: raw, errCommit: errors.New("test")}
			repo, _ := nuevoRegistro(&poolPrueba{tx: tx})
			out, err := repo.EjecutarOperacion(context.Background(), orden)
			if !errors.Is(err, ports.ErrRegistroNoDisponible) || out.Codigo != "" || out.Recibo != nil || tx.commits != 1 || tx.rollbacks != 1 {
				t.Fatal("resultado no durable expuesto", err)
			}
		})
	}
}

func TestAdaptadorRespuestaInvalidaHaceRollback(t *testing.T) {
	orden, resultado := ordenPrueba(t)
	resultado.Recibo.Registro.Hecho.PersonaRef = "persona:ajena"
	ajena, _ := json.Marshal(resultado)
	for _, raw := range [][]byte{
		ajena, []byte(`{"codigo":"denegada","auditoria_ref":"auditoria:rechazo","anterior":{},"recibo":null}`),
		[]byte(`{"codigo":"denegada","auditoria_ref":"auditoria:rechazo","extra":1}`),
		[]byte(`{"codigo":"denegada","auditoria_ref":"auditoria:rechazo"} {}`),
	} {
		tx := &txPrueba{raw: raw}
		repo, _ := nuevoRegistro(&poolPrueba{tx: tx})
		out, err := repo.EjecutarOperacion(context.Background(), orden)
		if err == nil || out.Codigo != "" || tx.commits != 0 || tx.rollbacks != 1 {
			t.Fatal("respuesta inválida llegó a COMMIT", err)
		}
	}
}

func TestAdaptadorLigaduraExactaAntesDeAbrirTransaccion(t *testing.T) {
	for _, caso := range []string{"contenido", "huella", "ambito", "actor", "contexto"} {
		t.Run(caso, func(t *testing.T) {
			orden, _ := ordenPrueba(t)
			switch caso {
			case "contenido":
				orden.Hecho.Denominacion = "Contenido cambiado"
			case "huella":
				orden.HuellaComando = "00"
			case "ambito":
				orden.VersionEsperada = 1
			case "actor":
				orden.ActorRef = "persona:ajena"
			case "contexto":
				orden.Autorizacion.Contexto.RegistroContextoRef = "otro"
			}
			pool := &poolPrueba{tx: &txPrueba{}}
			repo, _ := nuevoRegistro(pool)
			if _, err := repo.EjecutarOperacion(context.Background(), orden); err == nil || pool.llamadas != 0 {
				t.Fatal("orden cambiada alcanza SQL", err)
			}
		})
	}
}

func TestAdaptadorDenegacionSQLNoEsReciboConfirmado(t *testing.T) {
	orden, _ := ordenPrueba(t)
	tx := &txPrueba{errConsulta: &pgconn.PgError{Code: "42501", Message: "dato privado de prueba"}}
	repo, _ := nuevoRegistro(&poolPrueba{tx: tx})
	if out, err := repo.EjecutarOperacion(context.Background(), orden); err == nil || out.Codigo != "" || tx.commits != 0 || tx.rollbacks != 1 || err.Error() == "dato privado de prueba" {
		t.Fatal("error SQL afirma confirmación o filtra mensaje")
	}
}
