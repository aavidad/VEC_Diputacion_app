package cargosct

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// Ejecutar confirms one bounded serializable transaction. There are no
// implicit retries: an uncertain COMMIT is recovered with the original plan.
func Ejecutar(ctx context.Context, pool *pgxpool.Pool, operacion string, s Solicitud) (Resultado, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil || !s.Operador.validar() {
		return Resultado{}, ErrRechazada
	}
	b, err := s.Plan.Canonica()
	if err != nil {
		return Resultado{}, ErrRechazada
	}
	consultas := map[string]string{
		"preparar":  "SELECT vec_autorizacion.preparar_plan_cargo_ct_v1($1,$2,$3,$4,$5)",
		"aprobar":   "SELECT vec_autorizacion.aprobar_plan_cargo_ct_v1($1,$2,$3,$4,$5)",
		"aplicar":   "SELECT vec_autorizacion.aplicar_plan_cargo_ct_v1($1,$2,$3,$4,$5)",
		"recuperar": "SELECT vec_autorizacion.recuperar_plan_cargo_ct_v1($1,$2,$3,$4,$5)",
	}
	sql, ok := consultas[operacion]
	if !ok {
		return Resultado{}, ErrRechazada
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Resultado{}, ErrRechazada
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	if _, err = tx.Exec(ctx, "SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL lock_timeout='2s'; SET LOCAL idle_in_transaction_session_timeout='20s'"); err != nil {
		return Resultado{}, ErrRechazada
	}
	var raw []byte
	m := s.Operador
	if err = tx.QueryRow(ctx, sql, b, m.Decision, m.Motivo, m.PersonaVersion, m.PerfilVersion).Scan(&raw); err != nil {
		return Resultado{}, ErrRechazada
	}
	var r Resultado
	if json.Unmarshal(raw, &r) != nil {
		return Resultado{}, ErrRechazada
	}
	if r.Estado == "denegado" {
		if tx.Commit(ctx) != nil {
			return Resultado{}, ErrRechazada
		}
		return Resultado{}, ErrRechazada
	}
	if r.Clave != s.Plan.Clave || r.Estado == "" || r.AuditoriaRef == "" || r.Fecha.IsZero() {
		return Resultado{}, ErrRechazada
	}
	h, _ := s.Plan.Huella()
	if r.PlanSHA256 != h {
		return Resultado{}, ErrRechazada
	}
	if (operacion == "aplicar" || operacion == "recuperar") && (r.ReciboRef == "" || r.Version < 1) {
		return Resultado{}, ErrRechazada
	}
	if tx.Commit(ctx) != nil {
		return Resultado{}, ErrRechazada
	}
	return r, nil
}
