package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var _ ports.RepositorioHistorialParticipacion = (*RepositorioSituacionParticipacionPostgreSQL)(nil)

// ListarHistorial lee con una sola autorización consumida las operaciones B8 y
// la traza de valores de la migración 000034.
func (r *RepositorioSituacionParticipacionPostgreSQL) ListarHistorial(ctx context.Context, ref, actor string, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.HistorialParticipacion, error) {
	vacio := ports.HistorialParticipacion{}
	if r == nil || r.pool == nil || ctx == nil || ref == "" || actor == "" || m.ValidarEstructura() != nil {
		return vacio, ports.ErrSituacionParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, ports.ErrSituacionParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	rows, err := tx.Query(ctx, `SELECT clase,instante,operacion,situacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,motivo,recibo_ref,campo,valor_anterior,valor_nuevo FROM vec_bolsa_llamamientos.listar_historial_participacion_v1($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`, ref, actor, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return vacio, errorSituacionParticipacion(err)
	}
	defer rows.Close()
	h := ports.HistorialParticipacion{Operaciones: []ports.RegistroOperacionSituacion{}, Cambios: []ports.CambioValorParticipacion{}}
	for rows.Next() {
		var f struct {
			clase, actor                                               string
			operacion, situacion, jTipo, jRef, jSHA, validador, motivo *string
			recibo, campo, anterior, nuevo                             *string
			instante                                                   time.Time
			validadaEn                                                 *time.Time
		}
		if err := rows.Scan(&f.clase, &f.instante, &f.operacion, &f.situacion, &f.jTipo, &f.jRef, &f.jSHA, &f.actor, &f.validador, &f.validadaEn, &f.motivo, &f.recibo, &f.campo, &f.anterior, &f.nuevo); err != nil {
			return vacio, errorSituacionParticipacion(err)
		}
		switch f.clase {
		case "operacion":
			if f.operacion == nil || f.situacion == nil || f.jTipo == nil || f.jRef == nil || f.jSHA == nil || f.validador == nil || f.validadaEn == nil || f.motivo == nil || f.recibo == nil {
				return vacio, ports.ErrSituacionParticipacionNoDisponible
			}
			var o ports.RegistroOperacionSituacion
			o.ParticipacionRef, o.Desde, o.Operacion, o.Situacion, o.Motivo, o.ReciboRef = ref, f.instante.UTC(), *f.operacion, *f.situacion, *f.motivo, *f.recibo
			o.Justificante.Tipo, o.Justificante.Referencia, o.Justificante.SHA256 = *f.jTipo, *f.jRef, *f.jSHA
			o.Actor, o.Validador, o.ValidadaEn = f.actor, *f.validador, f.validadaEn.UTC()
			h.Operaciones = append(h.Operaciones, o)
		case "cambio":
			if f.campo == nil || f.recibo == nil || (f.nuevo == nil && f.anterior == nil) {
				return vacio, ports.ErrSituacionParticipacionNoDisponible
			}
			h.Cambios = append(h.Cambios, ports.CambioValorParticipacion{Instante: f.instante.UTC(), ReciboRef: *f.recibo, Campo: *f.campo, ValorAnterior: f.anterior, ValorNuevo: f.nuevo, Actor: f.actor})
		default:
			return vacio, ports.ErrSituacionParticipacionNoDisponible
		}
	}
	if rows.Err() != nil {
		return vacio, errorSituacionParticipacion(rows.Err())
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorSituacionParticipacion(err)
	}
	return h, nil
}
