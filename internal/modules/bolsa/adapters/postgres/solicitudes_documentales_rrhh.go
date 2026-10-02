package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var _ ports.ConsultaSolicitudesDocumentalesRRHH = (*RepositorioSituacionParticipacionPostgreSQL)(nil)

func (r *RepositorioSituacionParticipacionPostgreSQL) ListarSolicitudesDocumentalesPendientes(ctx context.Context, bolsa, participacion, actor string, corte time.Time, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ports.SolicitudDocumentalPendienteRRHH, error) {
	if r == nil || r.pool == nil || ctx == nil || bolsa == "" || participacion == "" || actor == "" || corte.IsZero() || m.ValidarEstructura() != nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorSituacionParticipacion(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, configuracionTransaccionPortal); err != nil {
		return nil, errorSituacionParticipacion(err)
	}
	rows, err := tx.Query(ctx, `SELECT solicitud_ref,version,contenido_sha256,documento_ref,documento_sha256,fecha_fin_causa,estado,recibo_ref,registrada_en FROM vec_bolsa_llamamientos.consultar_solicitudes_documentales_rrhh_v1($1::text,$2::text,$3::text,$4::timestamptz,$5::bytea,$6::bytea,$7::bytea,$8::bytea,$9::numeric,$10::numeric,$11::bytea,$12::bytea,$13::bytea,$14::bytea)`,
		bolsa, participacion, actor, corte.UTC(), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return nil, errorSituacionParticipacion(err)
	}
	defer rows.Close()
	items := make([]ports.SolicitudDocumentalPendienteRRHH, 0)
	for rows.Next() {
		var item ports.SolicitudDocumentalPendienteRRHH
		var fin *time.Time
		if err := rows.Scan(&item.SolicitudRef, &item.Version, &item.ContenidoSHA256, &item.DocumentoRef, &item.DocumentoSHA256, &fin, &item.Estado, &item.ReciboRef, &item.RegistradaEn); err != nil {
			return nil, errorSituacionParticipacion(err)
		}
		if fin != nil {
			item.FechaFinCausa = fin.Format(time.DateOnly)
		}
		item.RegistradaEn = item.RegistradaEn.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, errorSituacionParticipacion(err)
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, errorSituacionParticipacion(err)
	}
	return items, nil
}
