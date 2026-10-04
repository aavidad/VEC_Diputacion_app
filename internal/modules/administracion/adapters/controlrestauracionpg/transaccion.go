package controlrestauracionpg

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	ordenpg "vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias/postgres"
	dominio "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrCierre = errors.New("control_restauracion_postgres_cierre_no_confirmado")

// CanalSQL is private infrastructure within this adapter. It deliberately omits
// Begin, Commit, Rollback and Conn; its concrete wrapper does not expose pgx.Tx.
// Trusted providers must not issue transaction-control SQL or retain this channel.
type CanalSQL interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// MaterializadorEnTX must use this same channel for current central V3 state.
// No network, filesystem, other module table or other connection call is allowed.
// It cannot wrap the former outside-TX materializer as a nominal substitute.
type MaterializadorEnTX interface {
	MaterializarOrdenV3EnTX(context.Context, CanalSQL, dominio.Orden, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (ordenpg.MaterialV3, error)
}

type iniciador interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Registro owns the transaction. Its pool, provider and close deadline belong
// to trusted server composition, never to HTTP requests.
type Registro struct {
	pool        iniciador
	material    MaterializadorEnTX
	plazoCierre time.Duration
}

func Nuevo(pool *pgxpool.Pool, material MaterializadorEnTX, plazoCierre time.Duration) (*Registro, error) {
	if pool == nil || esNulo(material) || plazoCierre <= 0 || plazoCierre > 30*time.Second {
		return nil, ErrRegistro
	}
	return &Registro{pool: pool, material: material, plazoCierre: plazoCierre}, nil
}

type canalSQL struct{ tx pgx.Tx }

func (c canalSQL) QueryRow(ctx context.Context, consulta string, args ...any) pgx.Row {
	return c.tx.QueryRow(ctx, consulta, args...)
}

func (c canalSQL) Exec(ctx context.Context, consulta string, args ...any) (pgconn.CommandTag, error) {
	return c.tx.Exec(ctx, consulta, args...)
}

func (r *Registro) enTransaccion(ctx context.Context, trabajo func(CanalSQL) (Resultado, error)) (resultado Resultado, err error) {
	if ctx == nil || ctx.Err() != nil || r == nil || esNulo(r.pool) || trabajo == nil || r.plazoCierre <= 0 || r.plazoCierre > 30*time.Second {
		return Resultado{}, ErrRegistro
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || esNulo(tx) {
		return Resultado{}, ErrRegistro
	}
	confirmada := false
	defer func() {
		if confirmada {
			return
		}
		limpieza, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.plazoCierre)
		defer cancel()
		if cierre := tx.Rollback(limpieza); cierre != nil && !errors.Is(cierre, pgx.ErrTxClosed) {
			resultado, err = Resultado{}, ErrCierre
		}
	}()
	if _, err := tx.Exec(ctx, "SET LOCAL timezone = 'UTC'"); err != nil {
		return Resultado{}, ErrRegistro
	}
	resultado, err = trabajo(canalSQL{tx: tx})
	if err != nil || ctx.Err() != nil {
		return Resultado{}, ErrRegistro
	}
	cierre, cancel := context.WithTimeout(ctx, r.plazoCierre)
	defer cancel()
	if err := tx.Commit(cierre); err != nil {
		// Cleanup is attempted, but a failed COMMIT never proves rollback or the
		// absence of an effect. No receipt and no automatic retry are returned.
		return Resultado{}, ErrRegistro
	}
	confirmada = true
	// A confirmed COMMIT is not turned into uncertainty by late cancellation.
	return resultado, nil
}

func esNulo(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Func, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}
