package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const registrarFronteraCarteraSQL = `SELECT vec_meritos.registrar_denegacion_frontera_cartera_v1(
 $1::text,$2::text,$3::text,$4::text,NULLIF($5::text,''))`

const preflightFronteraCarteraSQL = `SELECT session_user=current_user
 AND pg_has_role(session_user,'vec_meritos_frontera_externo','MEMBER')
 AND pg_has_role(session_user,'vec_meritos_frontera_externo','USAGE')
 AND has_function_privilege(session_user,
  'vec_meritos.registrar_denegacion_frontera_cartera_v1(text,text,text,text,text)','EXECUTE')
 AND NOT has_schema_privilege(session_user,'vec_meritos','CREATE')
 AND NOT pg_is_in_recovery()`

type consultorFronteraCartera interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// RegistradorFronteraCartera usa únicamente el LOGIN nominal de denegaciones.
// Nunca transporta query, certificado, cabeceras, cuerpo ni detalle del error.
type RegistradorFronteraCartera struct{ consultor consultorFronteraCartera }

var _ vecports.RegistradorAuditoriaFronteraRutaExacta = (*RegistradorFronteraCartera)(nil)

func NuevoRegistradorFronteraCartera(ctx context.Context, pool *pgxpool.Pool) (*RegistradorFronteraCartera, error) {
	if nulo(pool) {
		return nil, meritosports.ErrCarteraNoDisponible
	}
	return nuevoRegistradorFronteraCartera(ctx, pool)
}

func nuevoRegistradorFronteraCartera(ctx context.Context, consultor consultorFronteraCartera) (*RegistradorFronteraCartera, error) {
	if ctx == nil || nulo(consultor) {
		return nil, meritosports.ErrCarteraNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var permitido bool
	if err := consultor.QueryRow(ctx, preflightFronteraCarteraSQL).Scan(&permitido); err != nil || !permitido {
		return nil, meritosports.ErrCarteraNoDisponible
	}
	return &RegistradorFronteraCartera{consultor}, nil
}

func (r *RegistradorFronteraCartera) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context,
	orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if ctx == nil || r == nil || nulo(r.consultor) || orden.Validar() != nil ||
		orden.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaMeritosCartera {
		return meritosports.ErrCarteraNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var registrada bool
	err := r.consultor.QueryRow(ctx, registrarFronteraCarteraSQL,
		orden.CorrelacionRef, string(orden.Motivo), orden.Superficie, orden.Ruta, orden.ActorRef).Scan(&registrada)
	if err != nil {
		return falloCarteraSQL{causa: err}
	}
	if !registrada {
		return meritosports.ErrCarteraNoDisponible
	}
	return nil
}
