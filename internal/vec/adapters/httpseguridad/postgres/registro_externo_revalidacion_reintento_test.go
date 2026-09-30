package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestRegistroExternoRevalidaTrasAbortoConLaConsultaOriginal(t *testing.T) {
	for _, fase := range []string{"consulta", "commit"} {
		for _, codigo := range []string{"40001", "40P01"} {
			t.Run(fase+codigo, func(t *testing.T) {
				abortada := &transaccionDoble{filas: [][]any{{true}}}
				if fase == "consulta" {
					abortada.errConsulta = &pgconn.PgError{Code: codigo}
				} else {
					abortada.errCommit = &pgconn.PgError{Code: codigo}
				}
				confirmada := &transaccionDoble{filas: [][]any{{true}}}
				pool := &iniciadorDoble{transacciones: []*transaccionDoble{abortada, confirmada}}
				adaptador := nuevoRegistroExternoPrueba(t, &iniciadorDoble{}, pool)
				if err := adaptador.ComprobarSesionYCuentaActivas(context.Background(), consultaValida(altaExternaValida())); err != nil {
					t.Fatal("no recuperó la revalidación abortada", err)
				}
				if pool.llamadas != 2 || abortada.rollbacks != 1 || confirmada.rollbacks != 1 || confirmada.commits != 1 || !reflect.DeepEqual(abortada.argumentos, confirmada.argumentos) || len(confirmada.argumentos[0]) != 20 {
					t.Fatal("cambió la consulta original o dejó un intento abierto")
				}
				comprobarSerializable(t, pool.opciones)
			})
		}
	}
}

func TestRegistroExternoRevalidacionDeniegaSinRepetirResultadoDefinitivo(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		tx     *transaccionDoble
	}{
		{"cuenta_inactiva", &transaccionDoble{filas: [][]any{{false}}}},
		{"permiso", &transaccionDoble{errConsulta: &pgconn.PgError{Code: "42501", Message: "dato privado"}}},
		{"bloqueo", &transaccionDoble{errConsulta: &pgconn.PgError{Code: "55P03"}}},
		{"commit_incierto", &transaccionDoble{filas: [][]any{{true}}, errCommit: errors.New("dato privado")}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			pool := &iniciadorDoble{transacciones: []*transaccionDoble{caso.tx}}
			adaptador := nuevoRegistroExternoPrueba(t, &iniciadorDoble{}, pool)
			err := adaptador.ComprobarSesionYCuentaActivas(context.Background(), consultaValida(altaExternaValida()))
			if !errors.Is(err, httpseguridad.ErrSesionNoValida) || strings.Contains(err.Error(), "dato privado") || pool.llamadas != 1 || caso.tx.rollbacks != 1 {
				t.Fatal("repitió un resultado definitivo, filtró datos o abrió el acceso")
			}
		})
	}
}

func TestRegistroExternoRevalidacionRecompruebaRevocacionTrasAborto(t *testing.T) {
	pool := &iniciadorDoble{transacciones: []*transaccionDoble{
		{filas: [][]any{{true}}, errCommit: &pgconn.PgError{Code: "40001"}},
		{filas: [][]any{{false}}},
	}}
	adaptador := nuevoRegistroExternoPrueba(t, &iniciadorDoble{}, pool)
	err := adaptador.ComprobarSesionYCuentaActivas(context.Background(), consultaValida(altaExternaValida()))
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || pool.llamadas != 2 || pool.transacciones[1].commits != 0 {
		t.Fatal("reutilizó la respuesta activa del intento abortado")
	}
}

func TestRegistroExternoRevalidacionCanceladaNoAbreOtroIntento(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	tx := &transaccionDoble{errConsulta: &pgconn.PgError{Code: "40001"}}
	pool := &poolAltaExternaPrueba{transacciones: []pgx.Tx{&transaccionCancelaAltaExternaPrueba{tx, cancelar}}}
	adaptador := nuevoRegistroExternoPrueba(t, &iniciadorDoble{}, pool)
	if err := adaptador.ComprobarSesionYCuentaActivas(ctx, consultaValida(altaExternaValida())); !errors.Is(err, context.Canceled) || pool.llamadas != 1 || tx.rollbacks != 1 {
		t.Fatal("abrió un intento después de cancelar")
	}
	if err := adaptador.ComprobarSesionYCuentaActivas(ctx, consultaValida(altaExternaValida())); !errors.Is(err, context.Canceled) || pool.llamadas != 1 {
		t.Fatal("abrió una transacción con contexto cancelado")
	}
}

func TestRegistroExternoRevalidacionAgotaPoliticaSinAbrirAcceso(t *testing.T) {
	pool := &iniciadorDoble{}
	for i := 0; i < postgresqlcomun.IntentosMaximosCarreraSerializable; i++ {
		pool.transacciones = append(pool.transacciones, &transaccionDoble{errConsulta: &pgconn.PgError{Code: "40001"}})
	}
	adaptador := nuevoRegistroExternoPrueba(t, &iniciadorDoble{}, pool)
	err := adaptador.ComprobarSesionYCuentaActivas(context.Background(), consultaValida(altaExternaValida()))
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || pool.llamadas != postgresqlcomun.IntentosMaximosCarreraSerializable {
		t.Fatal("agotó los intentos sin denegar el acceso")
	}
}

type poolRevalidacionConcurrente struct {
	inicios atomic.Int32
	commits atomic.Int32
	barrera sync.WaitGroup
}

func (p *poolRevalidacionConcurrente) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	if opciones != opcionesTransaccion() {
		return nil, errors.New("aislamiento inesperado")
	}
	return &txRevalidacionConcurrente{transaccionDoble: &transaccionDoble{filas: [][]any{{true}}}, pool: p, numero: p.inicios.Add(1)}, nil
}

type txRevalidacionConcurrente struct {
	*transaccionDoble
	pool   *poolRevalidacionConcurrente
	numero int32
}

func (t *txRevalidacionConcurrente) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if t.numero <= 2 {
		t.pool.barrera.Done()
		t.pool.barrera.Wait()
	}
	if t.numero == 1 {
		return filaDoble{err: &pgconn.PgError{Code: "40001"}}
	}
	return t.transaccionDoble.QueryRow(ctx, sql, args...)
}

func (t *txRevalidacionConcurrente) Commit(ctx context.Context) error {
	t.pool.commits.Add(1)
	return t.transaccionDoble.Commit(ctx)
}

func TestRegistroExternoRevalidacionesParalelasSoloConfirmanIntentosValidos(t *testing.T) {
	pool := &poolRevalidacionConcurrente{}
	pool.barrera.Add(2)
	adaptador := nuevoRegistroExternoPrueba(t, &iniciadorDoble{}, pool)
	resultados := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			resultados <- adaptador.ComprobarSesionYCuentaActivas(context.Background(), consultaValida(altaExternaValida()))
		}()
	}
	for i := 0; i < 2; i++ {
		if err := <-resultados; err != nil {
			t.Fatal("la carrera abortada impidió una consulta válida", err)
		}
	}
	if pool.inicios.Load() != 3 || pool.commits.Load() != 2 {
		t.Fatal("se confirmó un intento abortado o se repitió uno confirmado")
	}
}
