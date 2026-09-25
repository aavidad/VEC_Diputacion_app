package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/ports"
)

const (
	firmaRevalidarVinculoCorporativoRRHHV1 = `vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)`
	// ContextoActor 000009 exige el runtime acreditado dentro de la función.
	consultaRevalidarVinculoCorporativoRRHHV1 = `
		SELECT vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(
		       $1, $2, $3, $4, $5::numeric)`
	// Falla con error si la función no existe: sin 000009 no se arranca.
	consultaSondaVinculoCorporativoRRHHV1 = `
		SELECT pg_catalog.has_function_privilege($1, 'EXECUTE')`
)

type consultorVinculoCorporativoPostgreSQL interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// RevalidadorVinculoCorporativoRRHHPostgreSQLV1 pregunta a ContextoActor, en
// cada llamada, por el vínculo corporativo vigente. Usa el mismo LOGIN runtime
// que la resolución F1 y no conserva respuestas.
type RevalidadorVinculoCorporativoRRHHPostgreSQLV1 struct {
	pool consultorVinculoCorporativoPostgreSQL
}

// NuevoRevalidadorVinculoCorporativoRRHHPostgreSQLV1 acredita el runtime y
// exige EXECUTE sobre la función de 000009 antes de atender peticiones.
func NuevoRevalidadorVinculoCorporativoRRHHPostgreSQLV1(
	ctx context.Context, pool *pgxpool.Pool,
) (*RevalidadorVinculoCorporativoRRHHPostgreSQLV1, error) {
	if ctx == nil || pool == nil {
		return nil, ports.ErrVinculoCorporativoRRHHNoDisponible
	}
	var identidad string
	var acreditada, ejecutable bool
	if err := pool.QueryRow(ctx, `
		SELECT identidad_login, acreditada
		  FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()`,
	).Scan(&identidad, &acreditada); err != nil || !acreditada || identidad == "" {
		return nil, errorVinculoCorporativoPostgreSQL(ctx)
	}
	if err := pool.QueryRow(ctx, consultaSondaVinculoCorporativoRRHHV1, firmaRevalidarVinculoCorporativoRRHHV1).
		Scan(&ejecutable); err != nil || !ejecutable {
		return nil, errorVinculoCorporativoPostgreSQL(ctx)
	}
	return &RevalidadorVinculoCorporativoRRHHPostgreSQLV1{pool: pool}, nil
}

func (r *RevalidadorVinculoCorporativoRRHHPostgreSQLV1) RevalidarVinculoCorporativoRRHHV1(
	ctx context.Context, s ports.SolicitudRevalidacionVinculoCorporativoRRHHV1,
) error {
	if ctx == nil || r == nil || valorNuloContextoActorPostgreSQL(r.pool) {
		return ports.ErrVinculoCorporativoRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.Validar(); err != nil {
		return err
	}
	var vigente bool
	if err := r.pool.QueryRow(ctx, consultaRevalidarVinculoCorporativoRRHHV1,
		s.CuentaRef, s.PerfilRef, s.PersonaRef, s.VinculoContextoRef,
		strconv.FormatUint(s.VinculoContextoVersion, 10),
	).Scan(&vigente); err != nil {
		return errorVinculoCorporativoPostgreSQL(ctx)
	}
	if !vigente {
		return ports.ErrVinculoCorporativoRRHHNoVigente
	}
	return nil
}

func errorVinculoCorporativoPostgreSQL(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return errors.Join(ports.ErrVinculoCorporativoRRHHNoDisponible, err)
		}
	}
	return ports.ErrVinculoCorporativoRRHHNoDisponible
}

var _ ports.RevalidadorVinculoCorporativoRRHHV1 = (*RevalidadorVinculoCorporativoRRHHPostgreSQLV1)(nil)
