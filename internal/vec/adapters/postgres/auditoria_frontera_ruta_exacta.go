// Package postgres contiene adaptadores duraderos del nucleo para PostgreSQL.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/ports"
)

// ErrAuditoriaFronteraRutaExactaNoDisponible no expone el motivo del fallo de
// infraestructura. La frontera HTTP ya ha decidido la denegacion y no cambia
// su respuesta si la auditoria no puede registrarse.
var ErrAuditoriaFronteraRutaExactaNoDisponible = errors.New(
	"vec postgres: auditoria de frontera de ruta exacta no disponible",
)

const consultaRegistrarAuditoriaFronteraRutaExacta = `
	SELECT vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
		$1::text, $2::text, $3::text, $4::text, NULLIF($5::text, '')
	)`

const consultaPreflightAuditoriaFronteraRutaExacta = `
	SELECT pg_catalog.has_function_privilege(
		current_user,
		'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)',
		'EXECUTE'
	)`

type consultorFilaAuditoriaFronteraRutaExacta interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// RegistradorAuditoriaFronteraRutaExactaPostgreSQL persiste el hecho
// minimizado de una denegacion temprana. No recibe ni transporta cabeceras,
// cuerpo, certificados, IP ni texto de errores.
type RegistradorAuditoriaFronteraRutaExactaPostgreSQL struct {
	consultor consultorFilaAuditoriaFronteraRutaExacta
}

var _ ports.RegistradorAuditoriaFronteraRutaExacta = (*RegistradorAuditoriaFronteraRutaExactaPostgreSQL)(nil)

func NuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(
	pool *pgxpool.Pool,
) (*RegistradorAuditoriaFronteraRutaExactaPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, ErrAuditoriaFronteraRutaExactaNoDisponible
	}
	return nuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(pool)
}

func nuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(
	consultor consultorFilaAuditoriaFronteraRutaExacta,
) (*RegistradorAuditoriaFronteraRutaExactaPostgreSQL, error) {
	if valorNuloPostgreSQL(consultor) {
		return nil, ErrAuditoriaFronteraRutaExactaNoDisponible
	}
	return &RegistradorAuditoriaFronteraRutaExactaPostgreSQL{consultor: consultor}, nil
}

// PreflightAuditoriaFronteraRutaExacta comprueba que el pool ya dispone del
// privilegio EXECUTE de la firma SQL exacta. La composición debe invocarlo al
// acreditar su pool antes de montar la frontera; no se abre una conexión ni se
// ejecuta desde el constructor para conservar el ciclo de vida del llamador.
func (r *RegistradorAuditoriaFronteraRutaExactaPostgreSQL) PreflightAuditoriaFronteraRutaExacta(
	ctx context.Context,
) error {
	if ctx == nil || r == nil || valorNuloPostgreSQL(r.consultor) {
		return ErrAuditoriaFronteraRutaExactaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var permitido bool
	if err := r.consultor.QueryRow(ctx, consultaPreflightAuditoriaFronteraRutaExacta).Scan(&permitido); err != nil || !permitido {
		return ErrAuditoriaFronteraRutaExactaNoDisponible
	}
	return nil
}

func (r *RegistradorAuditoriaFronteraRutaExactaPostgreSQL) RegistrarAuditoriaFronteraRutaExacta(
	ctx context.Context,
	orden ports.OrdenAuditoriaFronteraRutaExacta,
) error {
	if ctx == nil || r == nil || valorNuloPostgreSQL(r.consultor) {
		return ErrAuditoriaFronteraRutaExactaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := orden.Validar(); err != nil {
		return err
	}
	var registrada bool
	err := r.consultor.QueryRow(
		ctx,
		consultaRegistrarAuditoriaFronteraRutaExacta,
		orden.CorrelacionRef,
		string(orden.Motivo),
		orden.Superficie,
		orden.Ruta,
		orden.ActorRef,
	).Scan(&registrada)
	if err != nil || !registrada {
		return ErrAuditoriaFronteraRutaExactaNoDisponible
	}
	return nil
}
