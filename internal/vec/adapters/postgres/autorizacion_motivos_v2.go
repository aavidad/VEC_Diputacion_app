package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaResolverMotivoAutorizacionV2Historico = `
	SELECT vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
		$1::text, $2::integer, $3::text, $4::text, $5::timestamptz
	)`

// ValidadorReferenciaMotivoPostgreSQLV2 comprueba una referencia contra la
// proyeccion historica publicada. Es de solo consulta: no proyecta catalogos,
// no resuelve la vigencia actual y no conserva ni administra conexiones.
//
// La composicion debe proporcionarle un pool exclusivo cuya identidad solo
// pueda ejecutar resolver_motivo_autorizacion_v2_historico. No debe reutilizar
// el pool del almacen V1 ni una identidad con privilegios de proyeccion.
type ValidadorReferenciaMotivoPostgreSQLV2 struct {
	consulta   consultorFilaMotivoAutorizacionV2
	catalogoID string
}

type consultorFilaMotivoAutorizacionV2 interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// NuevoValidadorReferenciaMotivoPostgreSQLV2 recibe un pool ya creado por la
// composicion. No acepta ni conserva el DSN y no abre ni cierra conexiones.
func NuevoValidadorReferenciaMotivoPostgreSQLV2(
	pool *pgxpool.Pool,
	catalogoID string,
) (*ValidadorReferenciaMotivoPostgreSQLV2, error) {
	return nuevoValidadorReferenciaMotivoPostgreSQLV2(pool, catalogoID)
}

func nuevoValidadorReferenciaMotivoPostgreSQLV2(
	consulta consultorFilaMotivoAutorizacionV2,
	catalogoID string,
) (*ValidadorReferenciaMotivoPostgreSQLV2, error) {
	centinela := domain.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogoID,
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EntradaClave:         "motivo_00000000000000000000000000000001",
	}
	if valorNuloPostgreSQL(consulta) || !domain.ReferenciaMotivoAutorizacionV2Valida(centinela) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ValidadorReferenciaMotivoPostgreSQLV2{
		consulta: consulta, catalogoID: catalogoID,
	}, nil
}

// ValidarReferenciaMotivoAutorizacionV2 resuelve exclusivamente el estado que
// existia en instante. La barrera de vigencia actual de una concesion pertenece
// a la misma transaccion que registra o consume su efecto y no a este puerto.
func (v *ValidadorReferenciaMotivoPostgreSQLV2) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context,
	referencia domain.ReferenciaEntradaCatalogo,
	instante time.Time,
) error {
	if v == nil || valorNuloPostgreSQL(v.consulta) || valorNuloPostgreSQL(ctx) ||
		referencia.CatalogoID != v.catalogoID ||
		!domain.ReferenciaMotivoAutorizacionV2Valida(referencia) ||
		!instanteHistoricoMotivoAutorizacionV2Valido(instante) {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return errorValidacionMotivoAutorizacionV2PostgreSQL(ctx, err, false)
	}

	var resuelta bool
	err := v.consulta.QueryRow(
		ctx,
		consultaResolverMotivoAutorizacionV2Historico,
		referencia.CatalogoID,
		referencia.CatalogoVersion,
		referencia.CatalogoHuellaSHA256,
		referencia.EntradaClave,
		instante,
	).Scan(&resuelta)
	if err != nil {
		return errorValidacionMotivoAutorizacionV2PostgreSQL(ctx, err, true)
	}
	if err = ctx.Err(); err != nil {
		return errorValidacionMotivoAutorizacionV2PostgreSQL(ctx, err, false)
	}
	if !resuelta {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func instanteHistoricoMotivoAutorizacionV2Valido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func errorValidacionMotivoAutorizacionV2PostgreSQL(
	ctx context.Context,
	causa error,
	falloFuente bool,
) error {
	var errorContextoLlamador error
	if !valorNuloPostgreSQL(ctx) {
		errorContextoLlamador = ctx.Err()
	}
	if errorContextoLlamador != nil {
		// Cancelar una operacion no demuestra una averia de PostgreSQL. Se
		// conserva el centinela estandar, pero no se marca la fuente como caida.
		return errors.Join(domain.ErrSolicitudAutorizacionInvalida, errorContextoLlamador)
	}
	var errorContextoFuente error
	switch {
	case errors.Is(causa, context.Canceled):
		errorContextoFuente = context.Canceled
	case errors.Is(causa, context.DeadlineExceeded):
		errorContextoFuente = context.DeadlineExceeded
	}
	if errorContextoFuente != nil {
		// Si el contexto del llamador sigue valido, un timeout o una cancelacion
		// devueltos por el controlador son un fallo interno de la consulta. No se
		// oculta el centinela, pero tampoco se propaga la causa con sus detalles.
		if falloFuente {
			return errors.Join(
				domain.ErrSolicitudAutorizacionInvalida,
				ports.ErrFuenteAutorizacionNoDisponible,
				errorContextoFuente,
			)
		}
		return errors.Join(domain.ErrSolicitudAutorizacionInvalida, errorContextoFuente)
	}
	if falloFuente {
		return errors.Join(
			domain.ErrSolicitudAutorizacionInvalida,
			ports.ErrFuenteAutorizacionNoDisponible,
		)
	}
	return domain.ErrSolicitudAutorizacionInvalida
}

var _ ports.ValidadorReferenciaMotivoAutorizacionV2 = (*ValidadorReferenciaMotivoPostgreSQLV2)(nil)
