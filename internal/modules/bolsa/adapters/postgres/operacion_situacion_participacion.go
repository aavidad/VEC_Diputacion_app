package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var _ ports.RepositorioOperacionSituacion = (*RepositorioSituacionParticipacionPostgreSQL)(nil)

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
	if cmd.SolicitudRef != "" {
		if cmd.SolicitudVersionEsperada != 1 || len(cmd.ContextoRecursoCanonico) == 0 {
			return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
		}
		madrid, locErr := time.LoadLocation("Europe/Madrid")
		if locErr != nil || cmd.CausaFinalizadaEn == nil {
			return ports.RegistroSituacionParticipacion{}, ports.ErrSituacionParticipacionNoDisponible
		}
		fechaCivil := cmd.CausaFinalizadaEn.In(madrid).Format("2006-01-02")
		err = tx.QueryRow(ctx, `SELECT reutilizada,recibo_ref,situacion,desde,fecha_disponible,recibo_resolucion_ref,resuelta_en FROM vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1($1,$2::integer,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::date,$18,$19,$20,$21,$22::numeric,$23::numeric,$24,$25,$26,$27,$28)`, cmd.SolicitudRef, int32(1), cmd.SolicitudContenidoSHA256, cmd.BolsaRef, c.ParticipacionRef, c.Desde.UTC(), cmd.SituacionEsperadaDesde.UTC(), c.Motivo, cmd.Actor, cmd.ClaveIdempotencia, cmd.ReciboRef, c.RegistradaEn.UTC(), cmd.Validador, cmd.ValidadaEn.UTC(), cmd.Justificante.Referencia, cmd.Justificante.SHA256, fechaCivil, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(), cmd.ContextoRecursoCanonico).Scan(&result.Reutilizada, &result.ReciboRef, &result.Situacion, &result.Desde, &result.FechaDisponible, &result.ReciboResolucionRef, &result.ResueltaEn)
	} else if cmd.Operacion == "revisar" || cmd.Operacion == "regularizar" || (cmd.Operacion == "excluir" && !cmd.SituacionEsperadaDesde.IsZero()) {
		err = tx.QueryRow(ctx, `SELECT reutilizada,recibo_ref,situacion,desde,fecha_disponible FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22::numeric,$23::numeric,$24,$25,$26,$27)`, cmd.BolsaRef, c.ParticipacionRef, cmd.Operacion, c.Desde.UTC(), c.FechaDisponible, c.Motivo, cmd.Actor, cmd.ClaveIdempotencia, cmd.ReciboRef, c.RegistradaEn.UTC(), cmd.Justificante.Tipo, cmd.Justificante.Referencia, cmd.Justificante.SHA256, cmd.Validador, cmd.ValidadaEn.UTC(), cmd.SituacionEsperadaDesde.UTC(), cmd.CausaFinalizadaEn, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&result.Reutilizada, &result.ReciboRef, &result.Situacion, &result.Desde, &result.FechaDisponible)
	} else {
		err = tx.QueryRow(ctx, `SELECT reutilizada,recibo_ref,situacion,desde,fecha_disponible FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20::numeric,$21::numeric,$22,$23,$24,$25)`, cmd.BolsaRef, c.ParticipacionRef, cmd.Operacion, c.Desde.UTC(), c.FechaDisponible, c.Motivo, cmd.Actor, cmd.ClaveIdempotencia, cmd.ReciboRef, c.RegistradaEn.UTC(), cmd.Justificante.Tipo, cmd.Justificante.Referencia, cmd.Justificante.SHA256, cmd.Validador, cmd.ValidadaEn.UTC(), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&result.Reutilizada, &result.ReciboRef, &result.Situacion, &result.Desde, &result.FechaDisponible)
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && cmd.SolicitudRef != "" && pgErr.Code == "22023" {
			return ports.RegistroSituacionParticipacion{}, dominiobolsa.ErrOperacionSituacionParticipacionInvalida
		}
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

func (r *RepositorioSituacionParticipacionPostgreSQL) ListarOperaciones(ctx context.Context, ref, actor string, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ports.RegistroOperacionSituacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" || actor == "" || m.ValidarEstructura() != nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	rows, err := tx.Query(ctx, `SELECT desde,operacion,situacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,motivo FROM vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`, ref, actor, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
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
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, errorSituacionParticipacion(err)
	}
	return items, nil
}
