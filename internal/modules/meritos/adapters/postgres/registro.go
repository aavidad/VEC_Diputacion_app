package postgres

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const consultaOperacion = `SELECT vec_meritos.operar_hecho_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

type iniciador interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type Registro struct{ pool iniciador }

var _ ports.Registro = (*Registro)(nil)

func NuevoRegistro(pool *pgxpool.Pool) (*Registro, error) { return nuevoRegistro(pool) }

func nuevoRegistro(pool iniciador) (*Registro, error) {
	if nulo(pool) {
		return nil, ports.ErrRegistroNoDisponible
	}
	return &Registro{pool: pool}, nil
}

// EjecutarOperacion devuelve resultados nominales solo tras COMMIT. El consumo
// V3, lectura y todas las escrituras se hacen en la misma invocación SQL y
// transacción; el cotejo puro de la respuesta se ejecuta antes de COMMIT.
func (r *Registro) EjecutarOperacion(ctx context.Context, orden ports.OrdenOperacion) (ports.ResultadoOperacion, error) {
	if ctx == nil || r == nil || nulo(r.pool) {
		return ports.ResultadoOperacion{}, ports.ErrRegistroNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacion{}, err
	}
	comando, err := serializarOrden(orden)
	if err != nil {
		return ports.ResultadoOperacion{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nulo(tx) {
		return ports.ResultadoOperacion{}, errorSQL(ctx, err)
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return ports.ResultadoOperacion{}, errorSQL(ctx, err)
	}
	m := orden.Autorizacion.Material
	var raw []byte
	err = tx.QueryRow(ctx, consultaOperacion, comando, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&raw)
	if err != nil {
		return ports.ResultadoOperacion{}, errorSQL(ctx, err)
	}
	resultado, err := leerResultado(raw)
	if err != nil || application.ValidarResultadoOperacion(orden, resultado) != nil {
		return ports.ResultadoOperacion{}, ports.ErrRegistroNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ResultadoOperacion{}, errorSQL(ctx, err)
	}
	return resultado, nil
}

func revertir(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
	defer cancel()
	_ = tx.Rollback(ctx)
}

func errorSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "42501" {
		return vec.ErrAutorizacionDenegada
	}
	return ports.ErrRegistroNoDisponible
}

func nulo(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}
