package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionConfirmarAnalisis        = "vec_contratacion_temporal.confirmar_operacion_analisis_v1"
	maximoIntentosConfirmarAnalisis = 3
)

var _ ports.TransaccionOperacionesAnalisis = (*TransaccionOperacionesAnalisisPostgreSQL)(nil)

type TransaccionOperacionesAnalisisPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevaTransaccionOperacionesAnalisisPostgreSQL(
	pool *pgxpool.Pool,
) (*TransaccionOperacionesAnalisisPostgreSQL, error) {
	return nuevaTransaccionOperacionesAnalisisPostgreSQL(pool)
}

func nuevaTransaccionOperacionesAnalisisPostgreSQL(
	pool iniciadorTransacciones,
) (*TransaccionOperacionesAnalisisPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	return &TransaccionOperacionesAnalisisPostgreSQL{pool: pool}, nil
}

func (t *TransaccionOperacionesAnalisisPostgreSQL) ConfirmarOperacionAnalisis(
	ctx context.Context,
	orden ports.OrdenConfirmarOperacionAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	if ctx == nil || t == nil || dependenciaNula(t.pool) {
		return ports.ReciboOperacionAnalisis{},
			ports.ErrOrdenOperacionAnalisisInvalida
	}
	contenido, err := codificarOperacionConfirmarAnalisis(orden)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	defer borrarBytes(contenido)
	for intento := 1; intento <= maximoIntentosConfirmarAnalisis; intento++ {
		recibo, err := t.confirmarEnTransaccion(ctx, orden, contenido)
		if err == nil {
			return recibo, nil
		}
		if ctx.Err() != nil {
			return ports.ReciboOperacionAnalisis{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(err) ||
			intento == maximoIntentosConfirmarAnalisis {
			return ports.ReciboOperacionAnalisis{},
				normalizarErrorConfirmacionAnalisis(ctx, err)
		}
	}
	return ports.ReciboOperacionAnalisis{},
		ports.ErrPersistenciaOperacionAnalisisNoDisponible
}

func (t *TransaccionOperacionesAnalisisPostgreSQL) confirmarEnTransaccion(
	ctx context.Context,
	orden ports.OrdenConfirmarOperacionAnalisis,
	contenido []byte,
) (ports.ReciboOperacionAnalisis, error) {
	tx, err := t.iniciar(ctx)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	defer revertirTransaccion(tx)
	var ahora time.Time
	err = tx.QueryRow(ctx, `
		SELECT date_trunc('microseconds', clock_timestamp())`,
	).Scan(&ahora)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	if err := orden.ValidarConfirmacionDentroDeTransaccion(ahora); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	var reciboJSON string
	err = tx.QueryRow(ctx, `
		SELECT recibo_json::text
		  FROM `+funcionConfirmarAnalisis+`($1::jsonb)`,
		contenido,
	).Scan(&reciboJSON)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	recibo, err := decodificarReciboConfirmacionAnalisis(reciboJSON)
	if err != nil ||
		recibo.ValidarParaOrdenDentroDeTransaccion(orden) != nil {
		return ports.ReciboOperacionAnalisis{},
			ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	return recibo, nil
}

func (t *TransaccionOperacionesAnalisisPostgreSQL) iniciar(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		revertirTransaccion(tx)
		return nil, err
	}
	return tx, nil
}

func normalizarErrorConfirmacionAnalisis(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, ports.ErrOrdenOperacionAnalisisInvalida) ||
		errors.Is(
			causa,
			ports.ErrResultadoOperacionAnalisisNoConfiable,
		) {
		return causa
	}
	return errorConfirmacionAnalisis(causa)
}
