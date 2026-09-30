package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

type estadoAltaExternaPrueba struct{ altas, auditorias int }

type transaccionAltaExternaPrueba struct {
	*transaccionDoble
	estado *estadoAltaExternaPrueba
}

func (t *transaccionAltaExternaPrueba) Commit(ctx context.Context) error {
	err := t.transaccionDoble.Commit(ctx)
	if err == nil {
		t.estado.altas++
		t.estado.auditorias++
	}
	return err
}

type poolAltaExternaPrueba struct {
	transacciones []pgx.Tx
	llamadas      int
}

func (p *poolAltaExternaPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	if opciones != opcionesTransaccion() || p.llamadas >= len(p.transacciones) {
		return nil, errors.New("transaccion de prueba inesperada")
	}
	tx := p.transacciones[p.llamadas]
	p.llamadas++
	return tx, nil
}

func TestRegistroExternoRepiteSoloAbortosSerializablesConOperacionOriginal(t *testing.T) {
	for _, fase := range []string{"consulta", "commit"} {
		for _, codigo := range []string{"40001", "40P01"} {
			t.Run(fase+codigo, func(t *testing.T) {
				alta := altaExternaValida()
				estado := &estadoAltaExternaPrueba{}
				abortada := &transaccionDoble{filas: [][]any{filaAltaValida(alta)}}
				if fase == "consulta" {
					abortada.errConsulta = &pgconn.PgError{Code: codigo, Message: "dato privado"}
				} else {
					abortada.errCommit = &pgconn.PgError{Code: codigo, Message: "dato privado"}
				}
				confirmada := &transaccionDoble{filas: [][]any{filaAltaValida(alta)}}
				pool := &poolAltaExternaPrueba{transacciones: []pgx.Tx{
					&transaccionAltaExternaPrueba{abortada, estado}, &transaccionAltaExternaPrueba{confirmada, estado}}}
				adaptador := nuevoRegistroExternoPrueba(t, pool, &iniciadorDoble{})
				recibo, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
				if err != nil || recibo.ValidarPara(alta) != nil || pool.llamadas != 2 || estado.altas != 1 || estado.auditorias != 1 {
					t.Fatal("no recuperó una sola alta auditada tras el aborto", err)
				}
				if abortada.rollbacks != 1 || confirmada.rollbacks != 1 || !reflect.DeepEqual(abortada.argumentos, confirmada.argumentos) || len(confirmada.argumentos[0]) != 20 {
					t.Fatal("no cerró cada transacción o cambió operación, tiempos o sellos")
				}
			})
		}
	}
}

func TestRegistroExternoNoRepiteErroresDefinitivosNiRespuestaAbortada(t *testing.T) {
	for _, codigo := range []string{"42501", "55P03", "23505", "42702"} {
		t.Run(codigo, func(t *testing.T) {
			alta := altaExternaValida()
			tx := &transaccionDoble{errConsulta: &pgconn.PgError{Code: codigo, Message: "dato privado"}}
			pool := &iniciadorDoble{transacciones: []*transaccionDoble{tx}}
			adaptador := nuevoRegistroExternoPrueba(t, pool, &iniciadorDoble{})
			recibo, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
			if !errors.Is(err, httpseguridad.ErrSesionNoValida) || strings.Contains(err.Error(), "dato privado") || pool.llamadas != 1 || tx.rollbacks != 1 || !reflect.DeepEqual(recibo, httpseguridad.ConfirmacionAltaSesion{}) {
				t.Fatal("repitió un error definitivo, filtró su causa o entregó respuesta")
			}
		})
	}
}

func TestRegistroExternoCommitInciertoNuncaReenviaPorErrorDeReconciliacion(t *testing.T) {
	for _, fase := range []string{"consulta", "commit"} {
		for _, codigo := range []string{"40001", "40P01"} {
			t.Run(fase+codigo, func(t *testing.T) {
				alta := altaExternaValida()
				altaTx := &transaccionDoble{filas: [][]any{filaAltaValida(alta)}, errCommit: errors.New("commit incierto")}
				recTx := &transaccionDoble{filas: [][]any{filaAltaValida(alta)}}
				if fase == "consulta" {
					recTx.errConsulta = &pgconn.PgError{Code: codigo}
				} else {
					recTx.errCommit = &pgconn.PgError{Code: codigo}
				}
				pool := &iniciadorDoble{transacciones: []*transaccionDoble{altaTx, recTx}}
				adaptador := nuevoRegistroExternoPrueba(t, pool, &iniciadorDoble{})
				recibo, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
				if !errors.Is(err, httpseguridad.ErrSesionNoValida) || pool.llamadas != 2 || len(altaTx.consultas) != 1 || len(recTx.consultas) != 1 || !strings.Contains(recTx.consultas[0], "reconciliar_registro_sesion_v1") || !reflect.DeepEqual(recibo, httpseguridad.ConfirmacionAltaSesion{}) {
					t.Fatal("un error de recuperación incierta volvió a ejecutar el alta")
				}
				if !reflect.DeepEqual(altaTx.argumentos, recTx.argumentos) {
					t.Fatal("reconcilió otra operación")
				}
			})
		}
	}
}

type transaccionCancelaAltaExternaPrueba struct {
	*transaccionDoble
	cancelar context.CancelFunc
}

func (t *transaccionCancelaAltaExternaPrueba) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	t.cancelar()
	return t.transaccionDoble.QueryRow(ctx, sql, args...)
}

func TestRegistroExternoCanceladoNoAbreOtroIntento(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	tx := &transaccionDoble{errConsulta: &pgconn.PgError{Code: "40001"}}
	pool := &poolAltaExternaPrueba{transacciones: []pgx.Tx{&transaccionCancelaAltaExternaPrueba{tx, cancelar}}}
	adaptador := nuevoRegistroExternoPrueba(t, pool, &iniciadorDoble{})
	recibo, err := adaptador.ConsumirAsercionYRegistrar(ctx, altaExternaValida())
	if !errors.Is(err, context.Canceled) || pool.llamadas != 1 || tx.rollbacks != 1 || !reflect.DeepEqual(recibo, httpseguridad.ConfirmacionAltaSesion{}) {
		t.Fatal("la cancelación permitió otro intento o una respuesta parcial")
	}
	if _, err := adaptador.ConsumirAsercionYRegistrar(ctx, altaExternaValida()); !errors.Is(err, context.Canceled) || pool.llamadas != 1 {
		t.Fatal("abrió una transacción con contexto cancelado")
	}
}

func TestRegistroExternoAgotaPoliticaComunSinConfirmacion(t *testing.T) {
	pool := &iniciadorDoble{}
	for i := 0; i < postgresqlcomun.IntentosMaximosCarreraSerializable; i++ {
		pool.transacciones = append(pool.transacciones, &transaccionDoble{errConsulta: &pgconn.PgError{Code: "40001"}})
	}
	adaptador := nuevoRegistroExternoPrueba(t, pool, &iniciadorDoble{})
	recibo, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), altaExternaValida())
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || pool.llamadas != postgresqlcomun.IntentosMaximosCarreraSerializable || !reflect.DeepEqual(recibo, httpseguridad.ConfirmacionAltaSesion{}) {
		t.Fatal("no respetó el límite común o entregó un resultado abortado")
	}
	for _, tx := range pool.transacciones {
		if tx.rollbacks != 1 || tx.commits != 0 {
			t.Fatal("dejó un intento abortado sin cerrar")
		}
	}
}
