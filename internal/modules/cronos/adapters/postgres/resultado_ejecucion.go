package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// RegistroResultadoEjecucionMarcajePostgreSQL escribe fallos confirmados y
// resultados indeterminados en vec_cronos_v1.resultado_ejecucion_marcaje con
// un LOGIN que solo hereda vec_cronos_v1_auditor: pool y transacción propios,
// separados de los del marcaje. Solo confirma después del COMMIT.
type RegistroResultadoEjecucionMarcajePostgreSQL struct {
	db *pgxpool.Pool
}

func NuevoRegistroResultadoEjecucionMarcajePostgreSQL(pool *pgxpool.Pool) (*RegistroResultadoEjecucionMarcajePostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrAuditoriaMarcajeNoDisponible
	}
	return &RegistroResultadoEjecucionMarcajePostgreSQL{db: pool}, nil
}

func (r *RegistroResultadoEjecucionMarcajePostgreSQL) RegistrarResultadoEjecucionMarcaje(ctx context.Context, e ports.ResultadoEjecucionMarcaje) error {
	if r == nil || r.db == nil || ctx == nil || e.DecisionRef == "" || e.ContextoRef == "" || e.ActorRef == "" ||
		e.PerfilRef == "" || e.Accion == "" || e.RecursoRef == "" || e.Resultado == "" || e.Causa == "" || e.ObservadaEn.IsZero() {
		return ports.ErrAuditoriaMarcajeNoDisponible
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ErrAuditoriaMarcajeNoDisponible
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	var ref string
	if err := tx.QueryRow(ctx, `SELECT vec_cronos_v1.registrar_resultado_ejecucion_marcaje_v1($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.DecisionRef, e.ContextoRef, e.ActorRef, e.PerfilRef, e.Accion, e.RecursoRef, e.Resultado, e.Causa, e.ObservadaEn.UTC()).Scan(&ref); err != nil || ref == "" {
		return ports.ErrAuditoriaMarcajeNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Join(ports.ErrAuditoriaMarcajeNoDisponible, errAuditoriaSinConfirmar)
	}
	return nil
}

var errAuditoriaSinConfirmar = errors.New("cronos: COMMIT de auditoría sin confirmar")

var _ ports.RegistroResultadoEjecucionMarcaje = (*RegistroResultadoEjecucionMarcajePostgreSQL)(nil)
