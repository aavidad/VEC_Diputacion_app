package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// RegistroDenegacionFronteraPostgreSQL usa el LOGIN del auditor de Cronos,
// separado del ejecutor, en una transacción propia.
type RegistroDenegacionFronteraPostgreSQL struct {
	db *pgxpool.Pool
}

func NuevoRegistroDenegacionFronteraPostgreSQL(pool *pgxpool.Pool) (*RegistroDenegacionFronteraPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrDenegacionFronteraNoRegistrada
	}
	return &RegistroDenegacionFronteraPostgreSQL{db: pool}, nil
}

func (r *RegistroDenegacionFronteraPostgreSQL) RegistrarDenegacionFronteraCronos(ctx context.Context, o ports.OrdenDenegacionFronteraCronos) error {
	if r == nil || r.db == nil || ctx == nil || o.Validar() != nil {
		return ports.ErrDenegacionFronteraNoRegistrada
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ErrDenegacionFronteraNoRegistrada
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	var ref string
	if err := tx.QueryRow(ctx, `SELECT vec_cronos_v1.registrar_denegacion_frontera_v1($1,$2,$3,$4,$5)`,
		o.CorrelacionRef, o.Motivo, o.Ruta, o.Metodo, o.ActorRef).Scan(&ref); err != nil || ref == "" {
		return ports.ErrDenegacionFronteraNoRegistrada
	}
	if tx.Commit(ctx) != nil {
		return ports.ErrDenegacionFronteraNoRegistrada
	}
	return nil
}

var _ ports.RegistroDenegacionFronteraCronos = (*RegistroDenegacionFronteraPostgreSQL)(nil)
