package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type RepositorioSituacionParticipacionPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioSituacionParticipacion = (*RepositorioSituacionParticipacionPostgreSQL)(nil)

func NuevoRepositorioSituacionParticipacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioSituacionParticipacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	return &RepositorioSituacionParticipacionPostgreSQL{pool}, nil
}
func (r *RepositorioSituacionParticipacionPostgreSQL) ParticipacionPerteneceABolsa(ctx context.Context, bolsaRef, participacionRef string) (bool, error) {
	if r == nil || r.pool == nil || ctx == nil || bolsaRef == "" || participacionRef == "" {
		return false, ports.ErrSituacionParticipacionNoDisponible
	}
	var pertenece bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1
		  FROM vec_bolsa_llamamientos.constitucion_entrada AS entrada
		  JOIN vec_bolsa_llamamientos.constitucion AS constitucion
		    USING (instantanea_ref,version_instantanea)
		 WHERE entrada.participacion_ref=$1 AND constitucion.bolsa_ref=$2
	)`, participacionRef, bolsaRef).Scan(&pertenece)
	if err != nil {
		return false, errorSituacionParticipacion(err)
	}
	return pertenece, nil
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
	err := r.pool.QueryRow(ctx, `SELECT recibo_ref,situacion,desde,fecha_disponible,motivo FROM vec_bolsa_llamamientos.recuperar_situacion_participacion_v1($1,$2)`, ref, clave).Scan(&resultado.ReciboRef, &resultado.Situacion, &resultado.Desde, &resultado.FechaDisponible, &resultado.Motivo)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoEncontrada
	}
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, errorSituacionParticipacion(err)
	}
	resultado.Reutilizada = true
	return resultado, nil
}
func (r *RepositorioSituacionParticipacionPostgreSQL) RegistrarSituacion(ctx context.Context, comando ports.ComandoCambiarSituacionParticipacion) (ports.RegistroSituacionParticipacion, error) {
	cambio := comando.Cambio
	if r == nil || r.pool == nil || ctx == nil || cambio.ParticipacionRef == "" || comando.BolsaRef == "" || cambio.Destino == "" || cambio.Motivo == "" || comando.Actor == "" || comando.ClaveIdempotencia == "" || comando.ReciboRef == "" || cambio.Desde.IsZero() || cambio.RegistradaEn.IsZero() || comando.Material.ValidarEstructura() != nil {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	var resultado ports.RegistroSituacionParticipacion
	resultado.ParticipacionRef = cambio.ParticipacionRef
	m := comando.Material
	err = tx.QueryRow(ctx, `SELECT reutilizada,recibo_ref,situacion,desde,fecha_disponible FROM vec_bolsa_llamamientos.registrar_situacion_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::numeric,$16::numeric,$17,$18,$19,$20)`, comando.BolsaRef, cambio.ParticipacionRef, cambio.Destino, cambio.Desde.UTC(), cambio.FechaDisponible, cambio.Motivo, comando.Actor, comando.ClaveIdempotencia, comando.ReciboRef, cambio.RegistradaEn.UTC(), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&resultado.Reutilizada, &resultado.ReciboRef, &resultado.Situacion, &resultado.Desde, &resultado.FechaDisponible)
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, errorSituacionParticipacion(err)
	}
	if resultado.ReciboRef != comando.ReciboRef || resultado.Situacion != cambio.Destino ||
		!mismaFechaDisponiblePostgreSQL(resultado.FechaDisponible, cambio.FechaDisponible) {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return resultado, nil
}
func errorSituacionParticipacion(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return ports.ErrSituacionParticipacionNoEncontrada
		case "VBS01", "22023":
			return dominiobolsa.ErrCambioSituacionParticipacionInvalido
		case "23505":
			return ports.ErrSituacionParticipacionNoDisponible
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		}
	}
	return ports.ErrSituacionParticipacionNoDisponible
}

func mismaFechaDisponiblePostgreSQL(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.UTC().Truncate(time.Microsecond).Equal(b.UTC().Truncate(time.Microsecond))
}
