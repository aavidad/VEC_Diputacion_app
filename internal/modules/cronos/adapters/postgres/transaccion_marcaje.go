package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type iniciadorMarcaje interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// ejecutarTransaccionMarcaje conserva el resultado de COMMIT separado de la
// entrega de su respuesta. Sólo un rollback confirmado permite afirmar fallo.
func (r *RepositorioMarcajes) ejecutarTransaccionMarcaje(ctx context.Context, evento ports.ResultadoEjecucionMarcaje, aplicar func(pgx.Tx) (ports.ReciboMarcajePropio, error)) (ports.ReciboMarcajePropio, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ReciboMarcajePropio{}, r.auditarFallo(ctx, evento, "fallo_confirmado", "persistencia", errorSeguro(ctx, err))
	}
	terminada := false
	defer func() {
		if !terminada {
			rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer cancel()
			_ = tx.Rollback(rollbackCtx)
		}
	}()
	recibo, err := aplicar(tx)
	if err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		rollbackErr := tx.Rollback(rollbackCtx)
		terminada = true
		cancel()
		if rollbackErr != nil {
			return ports.ReciboMarcajePropio{}, r.auditarFallo(ctx, evento, "resultado_indeterminado", "rollback", ports.ErrResultadoMarcajeIndeterminado)
		}
		causa := "persistencia"
		if errors.Is(err, ports.ErrClaveOperacionEnConflicto) || errors.Is(err, ports.ErrMovimientoRemotoNoPermitido) ||
			errors.Is(err, ports.ErrContinuidadMarcajeNoConfirmada) || errors.Is(err, ports.ErrTeletrabajoNoAutorizado) {
			causa = "conflicto"
		} else if errors.Is(err, errReciboMarcajeInvalido) {
			causa = "recibo_invalido"
			err = ports.ErrDependenciaNoDisponible
		}
		return ports.ReciboMarcajePropio{}, r.auditarFallo(ctx, evento, "fallo_confirmado", causa, err)
	}
	err = tx.Commit(ctx)
	terminada = true
	if err != nil {
		// pgx cierra la transacción incluso si se pierde la respuesta del
		// COMMIT. Un Rollback posterior con ErrTxClosed no prueba reversión.
		resultado, causa := "resultado_indeterminado", ports.ErrResultadoMarcajeIndeterminado
		var pg *pgconn.PgError
		if errors.Is(err, pgx.ErrTxCommitRollback) || (errors.As(err, &pg) && pg.Severity == "ERROR") {
			resultado, causa = "fallo_confirmado", errorSeguro(ctx, err)
		}
		return ports.ReciboMarcajePropio{}, r.auditarFallo(ctx, evento, resultado, "commit", causa)
	}
	return recibo, nil
}

var errReciboMarcajeInvalido = errors.New("cronos recibo transaccional invalido")

func (r *RepositorioMarcajes) auditarFallo(ctx context.Context, evento ports.ResultadoEjecucionMarcaje, resultado, causa string, original error) error {
	evento.Resultado, evento.Causa = resultado, causa
	evento.ObservadaEn = time.Now().UTC().Truncate(time.Microsecond)
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if r.auditoria == nil || r.auditoria.RegistrarResultadoEjecucionMarcaje(auditCtx, evento) != nil {
		if resultado == "resultado_indeterminado" {
			return errors.Join(ports.ErrResultadoMarcajeIndeterminado, ports.ErrAuditoriaMarcajeNoDisponible)
		}
		return ports.ErrAuditoriaMarcajeNoDisponible
	}
	return original
}
