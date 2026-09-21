package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type RepositorioSituacionParticipacionPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioSituacionParticipacion = (*RepositorioSituacionParticipacionPostgreSQL)(nil)

func NuevoRepositorioSituacionParticipacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioSituacionParticipacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	return &RepositorioSituacionParticipacionPostgreSQL{pool}, nil
}
func (r *RepositorioSituacionParticipacionPostgreSQL) SituacionVigente(ctx context.Context, ref string) (ports.SituacionParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" {
		return ports.SituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	var resultado ports.SituacionParticipacion
	resultado.ParticipacionRef = ref
	err := r.pool.QueryRow(ctx, `SELECT situacion,desde,fecha_disponible FROM vec_bolsa_llamamientos.leer_situacion_participacion_v1($1)`, ref).Scan(&resultado.Situacion, &resultado.Desde, &resultado.FechaDisponible)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.SituacionParticipacion{}, ports.ErrSituacionParticipacionNoEncontrada
	}
	if err != nil {
		return ports.SituacionParticipacion{}, errorSituacionParticipacion(err)
	}
	return resultado, nil
}
func (r *RepositorioSituacionParticipacionPostgreSQL) BuscarRegistroSituacion(ctx context.Context, ref, clave string) (ports.RegistroSituacionParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" || clave == "" {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	var resultado ports.RegistroSituacionParticipacion
	resultado.ParticipacionRef = ref
	err := r.pool.QueryRow(ctx, `SELECT recibo_ref,situacion,desde,fecha_disponible,motivo FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=$1 AND clave_idempotencia=$2`, ref, clave).Scan(&resultado.ReciboRef, &resultado.Situacion, &resultado.Desde, &resultado.FechaDisponible, &resultado.Motivo)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoEncontrada
	}
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, errorSituacionParticipacion(err)
	}
	resultado.Reutilizada = true
	return resultado, nil
}
func (r *RepositorioSituacionParticipacionPostgreSQL) RegistrarSituacion(ctx context.Context, ref, situacion string, desde time.Time, fecha *time.Time, motivo, actor, clave, recibo string, registrada time.Time) (ports.RegistroSituacionParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" || situacion == "" || motivo == "" || actor == "" || clave == "" || recibo == "" || desde.IsZero() || registrada.IsZero() {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	var resultado ports.RegistroSituacionParticipacion
	resultado.ParticipacionRef = ref
	err = tx.QueryRow(ctx, `SELECT reutilizada,recibo_ref,situacion,desde,fecha_disponible FROM vec_bolsa_llamamientos.registrar_situacion_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9)`, ref, situacion, desde.UTC(), fecha, motivo, actor, clave, recibo, registrada.UTC()).Scan(&resultado.Reutilizada, &resultado.ReciboRef, &resultado.Situacion, &resultado.Desde, &resultado.FechaDisponible)
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, errorSituacionParticipacion(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return resultado, nil
}
func errorSituacionParticipacion(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ports.ErrSituacionParticipacionNoEncontrada
	}
	return ports.ErrSituacionParticipacionNoDisponible
}
