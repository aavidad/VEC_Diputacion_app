package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var _ ports.RepositorioOperacionSituacion = (*RepositorioSituacionParticipacionPostgreSQL)(nil)

func (r *RepositorioSituacionParticipacionPostgreSQL) BuscarOperacion(ctx context.Context, ref, clave string) (ports.RegistroOperacionSituacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" || clave == "" {
		return ports.RegistroOperacionSituacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	var o ports.RegistroOperacionSituacion
	o.ParticipacionRef = ref
	err := r.pool.QueryRow(ctx, `SELECT recibo_ref,situacion,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,motivo FROM vec_bolsa_llamamientos.recuperar_operacion_situacion_participacion_v1($1,$2)`, ref, clave).Scan(&o.ReciboRef, &o.Situacion, &o.Desde, &o.Operacion, &o.Justificante.Tipo, &o.Justificante.Referencia, &o.Justificante.SHA256, &o.Actor, &o.Validador, &o.ValidadaEn, &o.Motivo)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.RegistroOperacionSituacion{}, ports.ErrSituacionParticipacionNoEncontrada
	}
	if err != nil {
		return ports.RegistroOperacionSituacion{}, errorSituacionParticipacion(err)
	}
	o.Reutilizada = true
	return o, nil
}

func (r *RepositorioSituacionParticipacionPostgreSQL) RegistrarOperacion(ctx context.Context, cmd ports.ComandoOperacionSituacion) (ports.RegistroSituacionParticipacion, error) {
	c := cmd.Cambio
	if r == nil || r.pool == nil || ctx == nil || c.ParticipacionRef == "" || cmd.BolsaRef == "" || c.Destino == "" || c.Motivo == "" || cmd.Actor == "" || cmd.ClaveIdempotencia == "" || cmd.ReciboRef == "" || cmd.Justificante.Validar() != nil || cmd.Material.ValidarEstructura() != nil {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := cmd.Material
	var result ports.RegistroSituacionParticipacion
	result.ParticipacionRef = c.ParticipacionRef
	err = tx.QueryRow(ctx, `SELECT reutilizada,recibo_ref,situacion,desde,fecha_disponible FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20::numeric,$21::numeric,$22,$23,$24,$25)`, cmd.BolsaRef, c.ParticipacionRef, cmd.Operacion, c.Desde.UTC(), c.FechaDisponible, c.Motivo, cmd.Actor, cmd.ClaveIdempotencia, cmd.ReciboRef, c.RegistradaEn.UTC(), cmd.Justificante.Tipo, cmd.Justificante.Referencia, cmd.Justificante.SHA256, cmd.Validador, cmd.ValidadaEn.UTC(), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&result.Reutilizada, &result.ReciboRef, &result.Situacion, &result.Desde, &result.FechaDisponible)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "VBS01" {
			return ports.RegistroSituacionParticipacion{}, ports.ErrClaveOperacionReutilizada
		}
		return ports.RegistroSituacionParticipacion{}, errorSituacionParticipacion(err)
	}
	if result.ReciboRef != cmd.ReciboRef || result.Situacion != c.Destino || !mismaFechaDisponiblePostgreSQL(result.FechaDisponible, c.FechaDisponible) {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return result, nil
}

func (r *RepositorioSituacionParticipacionPostgreSQL) ListarOperaciones(ctx context.Context, ref string) ([]ports.RegistroOperacionSituacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	rows, err := r.pool.Query(ctx, `SELECT desde,operacion,situacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,motivo FROM vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1($1)`, ref)
	if err != nil {
		return nil, errorSituacionParticipacion(err)
	}
	defer rows.Close()
	items := make([]ports.RegistroOperacionSituacion, 0)
	for rows.Next() {
		var o ports.RegistroOperacionSituacion
		o.ParticipacionRef = ref
		if err := rows.Scan(&o.Desde, &o.Operacion, &o.Situacion, &o.Justificante.Tipo, &o.Justificante.Referencia, &o.Justificante.SHA256, &o.Actor, &o.Validador, &o.ValidadaEn, &o.Motivo); err != nil {
			return nil, errorSituacionParticipacion(err)
		}
		items = append(items, o)
	}
	if rows.Err() != nil {
		return nil, errorSituacionParticipacion(rows.Err())
	}
	return items, nil
}
