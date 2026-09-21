package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const funcionRegistrarIntentoBorradorLlamamientoPostgreSQL = "vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1"

var _ puertosbolsa.RegistradorIntentoBorradorLlamamiento = (*AuditoriaIntentoBorradorLlamamientoPostgreSQL)(nil)

// AuditoriaIntentoBorradorLlamamientoPostgreSQL abre siempre una transacción
// nueva. Debe llamarse después del rollback del caso principal, de modo que
// una denegación o fallo no pierda su bitácora nominal.
type AuditoriaIntentoBorradorLlamamientoPostgreSQL struct{ pool iniciadorTransacciones }

func NuevaAuditoriaIntentoBorradorLlamamientoPostgreSQL(pool *pgxpool.Pool) (*AuditoriaIntentoBorradorLlamamientoPostgreSQL, error) {
	return nuevaAuditoriaIntentoBorradorLlamamientoPostgreSQL(pool)
}

func nuevaAuditoriaIntentoBorradorLlamamientoPostgreSQL(pool iniciadorTransacciones) (*AuditoriaIntentoBorradorLlamamientoPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	return &AuditoriaIntentoBorradorLlamamientoPostgreSQL{pool: pool}, nil
}

func (a *AuditoriaIntentoBorradorLlamamientoPostgreSQL) RegistrarIntentoBorradorLlamamiento(ctx context.Context, intento puertosbolsa.IntentoBorradorLlamamiento) error {
	if ctx == nil || a == nil || valorNulo(a.pool) || intento.Validar() != nil {
		return puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	correlacion, err := intento.Correlacion.ValorCanonico()
	if err != nil {
		return puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return errorBorradorLlamamientoPostgreSQL(ctx, err)
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return errorBorradorLlamamientoPostgreSQL(ctx, err)
	}
	if _, err = tx.Exec(ctx, `SELECT `+funcionRegistrarIntentoBorradorLlamamientoPostgreSQL+`($1,$2,$3,$4,$5)`, correlacion, string(intento.Accion), string(intento.ClaseRuta), actorNuloIntentoBorradorLlamamiento(intento.ActorVerificado), string(intento.Resultado)); err != nil {
		return errorBorradorLlamamientoPostgreSQL(ctx, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return errorBorradorLlamamientoPostgreSQL(ctx, err)
	}
	return nil
}

func actorNuloIntentoBorradorLlamamiento(actor string) any {
	if actor == "" {
		return nil
	}
	return actor
}
