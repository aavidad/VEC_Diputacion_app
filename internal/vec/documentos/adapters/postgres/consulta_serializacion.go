package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/shared/postgresql"
)

func (r *Repositorio) transaccionConsulta(ctx context.Context, funcion string, args ...any) ([]byte, error) {
	if r == nil || r.db == nil || ctx == nil {
		return nil, ErrRepositorioNoDisponible
	}
	return consultarConReintento(ctx, func() (transaccionRepositorio, error) {
		tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		return transaccionPGX{tx}, err
	}, funcion, args...)
}

// Cada carrera aborta el intento completo, incluido el consumo y la auditoría.
// Se conserva la misma preimagen y autoridad; solo cambia la instantánea SQL.
// Un COMMIT ambiguo o una denegación no se repiten.
func consultarConReintento(ctx context.Context, iniciar func() (transaccionRepositorio, error), funcion string, args ...any) ([]byte, error) {
	var resultado []byte
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		tx, err := iniciar()
		if err != nil {
			return err
		}
		resultado, err = ejecutarIntentoTransaccion(ctx, tx, funcion, args...)
		return err
	})
	if err != nil {
		return nil, clasificarErrorSQL(err)
	}
	return resultado, nil
}
