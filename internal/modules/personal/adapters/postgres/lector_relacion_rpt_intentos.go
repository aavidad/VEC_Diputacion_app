package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const registrarIntentoLectorRPTSQL = `SELECT vec_personal.registrar_denegacion_relacion_para_rpt_v1($1::text,$2::text,NULLIF($3::text,''),NULLIF($4::text,''))`

type filaIntentoLectorRPT interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type RegistroIntentosLectorRPTPostgreSQL struct{ pool filaIntentoLectorRPT }

func NuevoRegistroIntentosLectorRPTPostgreSQL(pool *pgxpool.Pool) (*RegistroIntentosLectorRPTPostgreSQL, error) {
	if pool == nil {
		return nil, domain.ErrLectorRelacionRPTNoDisponible
	}
	return &RegistroIntentosLectorRPTPostgreSQL{pool}, nil
}

// La selección queda cerrada si falta la fachada o su permiso técnico propio.
func (r *RegistroIntentosLectorRPTPostgreSQL) VerificarDestinoRelacionRPT(ctx context.Context) error {
	if r == nil || ctx == nil || ctx.Err() != nil || nuloRegistroEmpleadoB2(r.pool) {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	var disponible bool
	const consulta = `SELECT has_function_privilege('vec_personal.registrar_denegacion_relacion_para_rpt_v1(text,text,text,text)','EXECUTE')`
	if err := r.pool.QueryRow(ctx, consulta).Scan(&disponible); err != nil || !disponible {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	return nil
}

// Pool separado del lector. La fachada nominal registra y confirma el intento
// fuera del rollback de lectura/consumo; no reutiliza registro B2, CT ni Dietas.
func (r *RegistroIntentosLectorRPTPostgreSQL) RegistrarEventoRelacionRPT(ctx context.Context, e ports.EventoIntentoLectorRelacionRPT) error {
	if r == nil || ctx == nil || ctx.Err() != nil || nuloRegistroEmpleadoB2(r.pool) || !vecdomain.ReferenciaCorrelacionAutorizacionV2Valida(e.CorrelacionRef) ||
		(e.Motivo != "entrada_invalida" && e.Motivo != "denegado" && e.Motivo != "no_disponible") ||
		(e.ActorRef != "" && !domain.ReferenciaPersonaValida(e.ActorRef)) || (e.RelacionRef != "" && !domain.ReferenciaRelacionValida(e.RelacionRef)) {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	var confirmado bool
	if err := r.pool.QueryRow(ctx, registrarIntentoLectorRPTSQL, e.CorrelacionRef, e.Motivo, e.ActorRef, e.RelacionRef).Scan(&confirmado); err != nil || !confirmado {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	return nil
}

var _ ports.DestinoIntentosLectorRelacionRPT = (*RegistroIntentosLectorRPTPostgreSQL)(nil)
