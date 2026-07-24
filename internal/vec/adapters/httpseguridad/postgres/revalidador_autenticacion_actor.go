package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrRevalidadorAutenticacionActorNoDisponible = errors.New(
	"revalidador de autenticacion de actor no disponible",
)

const consultaRevalidarAutenticacionActorV1 = `
	SELECT autenticacion_ref, autenticacion_huella_sha256, asercion_ref,
	       sesion_ref, control_sesion_ref, control_sesion_revision,
	       control_sesion_huella_sha256, cuenta_ref, cuenta_ordinaria_ref,
	       cuenta_privilegiada, superficie, metodo_observado,
	       garantia_observada, politica_garantia_ref,
	       politica_garantia_huella_sha256, autenticacion_verificada_en,
	       sesion_emitida_en, sesion_valida_hasta, sesion_revalidada_en
	  FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1($1,$2)`

// RevalidadorAutenticacionActorPostgreSQL proyecta una sesion durable a partir
// de sus dos referencias opacas. No acepta del llamador cuenta, perfil,
// superficie, metodo ni garantia.
type RevalidadorAutenticacionActorPostgreSQL struct {
	pool iniciadorTransacciones
}

// NuevoRevalidadorAutenticacionActorPostgreSQL acredita que el pool usa un
// LOGIN dedicado que hereda exclusivamente la capacidad de revalidacion.
func NuevoRevalidadorAutenticacionActorPostgreSQL(
	ctx context.Context,
	pool *pgxpool.Pool,
) (*RevalidadorAutenticacionActorPostgreSQL, error) {
	if valorNulo(ctx) || pool == nil || ctx.Err() != nil {
		return nil, ErrRevalidadorAutenticacionActorNoDisponible
	}
	if _, err := acreditarCapacidadPool(ctx, pool, capacidadRevalidar); err != nil {
		return nil, ErrRevalidadorAutenticacionActorNoDisponible
	}
	return nuevoRevalidadorAutenticacionActorPostgreSQL(pool)
}

func nuevoRevalidadorAutenticacionActorPostgreSQL(
	pool iniciadorTransacciones,
) (*RevalidadorAutenticacionActorPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, ErrRevalidadorAutenticacionActorNoDisponible
	}
	return &RevalidadorAutenticacionActorPostgreSQL{pool: pool}, nil
}

func (r *RevalidadorAutenticacionActorPostgreSQL) RevalidarAutenticacionActorV1(
	ctx context.Context,
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	if r == nil || valorNulo(ctx) || valorNulo(r.pool) || solicitud.Validar() != nil {
		return domain.AutenticacionRevalidadaV1{},
			domain.ErrAutenticacionRevalidadaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.AutenticacionRevalidadaV1{}, err
	}

	tx, err := r.pool.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return domain.AutenticacionRevalidadaV1{}, errorRevalidacionActorSaneado(ctx)
	}
	defer revertir(tx)
	if err = prepararTransaccion(ctx, tx); err != nil {
		return domain.AutenticacionRevalidadaV1{}, errorRevalidacionActorSaneado(ctx)
	}

	resultado, err := consultarAutenticacionActorV1(ctx, tx, solicitud)
	if err != nil || resultado.Validar() != nil ||
		resultado.AutenticacionRef != solicitud.AutenticacionRef ||
		resultado.SesionRef != solicitud.SesionRef {
		return domain.AutenticacionRevalidadaV1{}, errorRevalidacionActorSaneado(ctx)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.AutenticacionRevalidadaV1{}, errorRevalidacionActorSaneado(ctx)
	}
	return resultado, nil
}

func consultarAutenticacionActorV1(
	ctx context.Context,
	tx pgx.Tx,
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	var resultado domain.AutenticacionRevalidadaV1
	var revisionTexto, superficie, metodo, garantia string
	err := tx.QueryRow(
		ctx,
		consultaRevalidarAutenticacionActorV1,
		solicitud.AutenticacionRef,
		solicitud.SesionRef,
	).Scan(
		&resultado.AutenticacionRef,
		&resultado.AutenticacionHuellaSHA256,
		&resultado.AsercionRef,
		&resultado.SesionRef,
		&resultado.ControlSesionRef,
		&revisionTexto,
		&resultado.ControlSesionHuellaSHA256,
		&resultado.CuentaRef,
		&resultado.CuentaOrdinariaRef,
		&resultado.CuentaPrivilegiada,
		&superficie,
		&metodo,
		&garantia,
		&resultado.PoliticaGarantiaRef,
		&resultado.PoliticaGarantiaHuellaSHA256,
		&resultado.AutenticacionVerificadaEn,
		&resultado.SesionEmitidaEn,
		&resultado.SesionValidaHasta,
		&resultado.SesionRevalidadaEn,
	)
	if err != nil {
		return domain.AutenticacionRevalidadaV1{}, err
	}
	revision, err := strconv.ParseUint(revisionTexto, 10, 64)
	if err != nil || revision == 0 {
		return domain.AutenticacionRevalidadaV1{},
			domain.ErrAutenticacionRevalidadaInvalida
	}
	resultado.ControlSesionRevision = revision
	resultado.Superficie = domain.SuperficieAutenticacionActorV1(superficie)
	resultado.MetodoObservado = domain.AuthMethod(metodo)
	resultado.GarantiaObservada = domain.AuthAssurance(garantia)
	resultado.AutenticacionVerificadaEn = instanteUTCPostgreSQL(
		resultado.AutenticacionVerificadaEn,
	)
	resultado.SesionEmitidaEn = instanteUTCPostgreSQL(resultado.SesionEmitidaEn)
	resultado.SesionValidaHasta = instanteUTCPostgreSQL(resultado.SesionValidaHasta)
	resultado.SesionRevalidadaEn = instanteUTCPostgreSQL(resultado.SesionRevalidadaEn)
	return resultado, nil
}

func instanteUTCPostgreSQL(instante time.Time) time.Time {
	return instante.UTC().Truncate(time.Microsecond)
}

func errorRevalidacionActorSaneado(ctx context.Context) error {
	if !valorNulo(ctx) {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return domain.ErrAutenticacionRevalidadaInvalida
}

var _ ports.RevalidadorAutenticacionActorV1 = (*RevalidadorAutenticacionActorPostgreSQL)(nil)
