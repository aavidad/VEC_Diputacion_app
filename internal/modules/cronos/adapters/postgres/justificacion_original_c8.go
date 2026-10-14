package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaAnexoOriginalC8    = `SELECT vec_cronos_v1.anexar_justificacion_original_c8_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaRevisionOriginalC8 = `SELECT vec_cronos_v1.revisar_justificacion_original_c8_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
)

type RepositorioOriginalC8 struct{ db *pgxpool.Pool }

func NuevoRepositorioOriginalC8(db *pgxpool.Pool) (*RepositorioOriginalC8, error) {
	if db == nil {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &RepositorioOriginalC8{db: db}, nil
}

func errorOriginalC8(ctx context.Context, err error) error {
	var pg *pgconn.PgError
	if ctx != nil && ctx.Err() == nil && errors.As(err, &pg) {
		switch pg.Code {
		case "PC001":
			return domain.ErrJustificacionInvalida
		case "PC002", "PC011", "23505":
			return domain.ErrJustificacionConflicto
		case "PC012":
			return ports.ErrResolucionNoCompetente
		case "PC013":
			return ports.ErrBandejaOriginalC8DemasiadoGrande
		}
	}
	return errorResolucion(ctx, err)
}

func (r *RepositorioOriginalC8) ConfirmarOriginal(ctx context.Context, orden ports.OrdenOriginalC8,
	m domain.MaterialJustificacionOriginalC8, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboOriginalC8, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil || actor.PersonaRef != m.ActorRef || actor.PerfilActivoRef != m.PerfilRef {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	recurso, err := application.RecursoJustificacionOriginalC8(m)
	if err != nil || !resumenV3Ligado(v3, application.AudienciaJustificacionOriginalC8, m.Accion, recurso) {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	consulta := consultaAnexoOriginalC8
	if m.Accion == domain.AccionRevisarOriginalC8 {
		consulta = consultaRevisionOriginalC8
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consulta, canonico, v3, errorOriginalC8)
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	defer clear(bruto)
	var sql struct {
		SolicitudRef     string    `json:"solicitud_ref"`
		DocumentoID      string    `json:"documento_id"`
		DocumentoVersion uint64    `json:"documento_version"`
		SHA256           string    `json:"sha256"`
		Estado           string    `json:"estado"`
		Version          int64     `json:"version"`
		ReciboRef        string    `json:"recibo_ref"`
		InstanteUTC      time.Time `json:"instante_utc"`
		Replay           bool      `json:"replay"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.SolicitudRef != m.SolicitudRef || sql.DocumentoID != m.DocumentoID ||
		sql.SHA256 != m.DocumentoSHA256 || sql.ReciboRef == "" || sql.InstanteUTC.IsZero() {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	return ports.ReciboOriginalC8{SolicitudRef: sql.SolicitudRef, DocumentoID: sql.DocumentoID,
		DocumentoVersion: sql.DocumentoVersion, SHA256: sql.SHA256,
		Estado: domain.EstadoJustificacion(sql.Estado), Version: sql.Version,
		ReciboRef: sql.ReciboRef, InstanteUTC: sql.InstanteUTC.UTC(), Replay: sql.Replay}, nil
}
