package postgresimportacionconvoca

import (
	"context"
	"crypto/subtle"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
)

func bytesIgualesConstantes(a, b []byte) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare(a, b) == 1
}

func valorNulo(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func formatearInstante(instante time.Time) string {
	return instante.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func iniciarTransaccion(
	ctx context.Context,
	pool iniciadorTransacciones,
	modo pgx.TxAccessMode,
) (pgx.Tx, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: modo,
	})
	if err != nil {
		return nil, errorPostgreSQL(ctx, err)
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '3s', true),
		       set_config('statement_timeout', '45s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		_ = tx.Rollback(context.Background())
		return nil, errorPostgreSQL(ctx, err)
	}
	return tx, nil
}

func revertir(tx pgx.Tx) {
	if tx != nil {
		_ = tx.Rollback(context.Background())
	}
}
