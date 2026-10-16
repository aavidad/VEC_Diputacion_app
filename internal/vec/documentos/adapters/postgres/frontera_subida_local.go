package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const firmaDenegacionSubidaLocal = "vec_documentos.registrar_denegacion_frontera_subida_local_v1(text,text,text,text,timestamptz)"

// RegistradorFronteraSubidaLocal usa un LOGIN dedicado que solo puede
// registrar denegaciones de la carga local. La consulta conserva la ruta
// canónica; una referencia documental nunca entra en el asiento.
type RegistradorFronteraSubidaLocal struct{ db *pgxpool.Pool }

func NuevoRegistradorFronteraSubidaLocal(db *pgxpool.Pool) (*RegistradorFronteraSubidaLocal, error) {
	if db == nil {
		return nil, ErrFronteraNoDisponible
	}
	return &RegistradorFronteraSubidaLocal{db: db}, nil
}

func (r *RegistradorFronteraSubidaLocal) Preflight(ctx context.Context) error {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil {
		return ErrFronteraNoDisponible
	}
	var autorizado bool
	err := r.db.QueryRow(ctx, `SELECT session_user=current_user
 AND session_user='vec_externo_documentos_frontera'
 AND pg_catalog.has_schema_privilege(session_user,'vec_documentos','USAGE')
 AND pg_catalog.has_function_privilege(session_user,$1,'EXECUTE')`, firmaDenegacionSubidaLocal).Scan(&autorizado)
	if err != nil || !autorizado {
		return ErrFronteraNoDisponible
	}
	return nil
}

func (r *RegistradorFronteraSubidaLocal) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context,
	o vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil ||
		o.Superficie != vecdomain.SuperficieAuditoriaFronteraRutaExactaDocumentosSubidaLocal ||
		o.Validar() != nil {
		return ErrOrdenFronteraInvalida
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ErrFronteraNoDisponible
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_catalog.set_config('search_path','pg_catalog',true),
 pg_catalog.set_config('timezone','UTC',true),
 pg_catalog.set_config('row_security','on',true),
 pg_catalog.set_config('statement_timeout','5s',true)`); err != nil {
		return ErrFronteraNoDisponible
	}
	var referencia string
	err = tx.QueryRow(ctx, `SELECT vec_documentos.registrar_denegacion_frontera_subida_local_v1($1,$2,$3,$4,$5)`,
		o.CorrelacionRef, string(o.Motivo), o.Ruta, o.ActorRef, time.Now().UTC()).Scan(&referencia)
	if err != nil || len(referencia) != len("fdc_")+64 || !strings.HasPrefix(referencia, "fdc_") {
		return ErrFronteraNoDisponible
	}
	for _, c := range referencia[len("fdc_"):] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ErrFronteraNoDisponible
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrFronteraNoDisponible
	}
	return nil
}

var _ vecports.RegistradorAuditoriaFronteraRutaExacta = (*RegistradorFronteraSubidaLocal)(nil)
