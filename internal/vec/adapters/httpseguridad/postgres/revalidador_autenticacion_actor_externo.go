package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaRevalidarAutenticacionActorExternoV1 = `
	SELECT autenticacion_ref, autenticacion_huella_sha256, asercion_ref,
	       sesion_ref, control_sesion_ref, control_sesion_revision,
	       control_sesion_huella_sha256, cuenta_ref, cuenta_ordinaria_ref,
	       cuenta_privilegiada, superficie, metodo_observado,
	       garantia_observada, politica_garantia_ref,
	       politica_garantia_huella_sha256, autenticacion_verificada_en,
	       sesion_emitida_en, sesion_valida_hasta, sesion_revalidada_en
	  FROM vec_identidad_externa_v1.revalidar_autenticacion_actor_v1($1,$2)`

type RevalidadorAutenticacionActorExternoPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoRevalidadorAutenticacionActorExternoPostgreSQL(
	ctx context.Context,
	pool *pgxpool.Pool,
) (*RevalidadorAutenticacionActorExternoPostgreSQL, error) {
	if valorNulo(ctx) || pool == nil || ctx.Err() != nil {
		return nil, ErrRevalidadorAutenticacionActorNoDisponible
	}
	if _, err := acreditarCapacidadExterna(ctx, pool, capacidadRevalidarExterna); err != nil {
		return nil, ErrRevalidadorAutenticacionActorNoDisponible
	}
	return nuevoRevalidadorAutenticacionActorExternoPostgreSQL(pool)
}

func nuevoRevalidadorAutenticacionActorExternoPostgreSQL(
	pool iniciadorTransacciones,
) (*RevalidadorAutenticacionActorExternoPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, ErrRevalidadorAutenticacionActorNoDisponible
	}
	return &RevalidadorAutenticacionActorExternoPostgreSQL{pool: pool}, nil
}

func (r *RevalidadorAutenticacionActorExternoPostgreSQL) RevalidarAutenticacionActorV1(
	ctx context.Context,
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	if r == nil || valorNulo(ctx) || valorNulo(r.pool) || solicitud.Validar() != nil {
		return domain.AutenticacionRevalidadaV1{}, domain.ErrAutenticacionRevalidadaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.AutenticacionRevalidadaV1{}, err
	}
	tx, err := r.pool.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return domain.AutenticacionRevalidadaV1{}, errorRevalidacionActorSaneado(ctx)
	}
	defer revertir(tx)
	if prepararTransaccion(ctx, tx) != nil {
		return domain.AutenticacionRevalidadaV1{}, errorRevalidacionActorSaneado(ctx)
	}
	resultado, err := consultarAutenticacionActorExternoV1(ctx, tx, solicitud)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
			return domain.AutenticacionRevalidadaV1{}, domain.ErrAutenticacionRevalidadaInvalida
		}
		return domain.AutenticacionRevalidadaV1{}, errorRevalidacionActorSaneado(ctx)
	}
	if resultado.Validar() != nil ||
		resultado.AutenticacionRef != solicitud.AutenticacionRef ||
		resultado.SesionRef != solicitud.SesionRef ||
		resultado.Superficie != domain.SuperficieAutenticacionExternaPersonalV1 ||
		resultado.CuentaPrivilegiada ||
		resultado.CuentaRef != resultado.CuentaOrdinariaRef ||
		tx.Commit(ctx) != nil {
		return domain.AutenticacionRevalidadaV1{}, errorRevalidacionActorSaneado(ctx)
	}
	return resultado, nil
}

func consultarAutenticacionActorExternoV1(
	ctx context.Context,
	tx pgx.Tx,
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	var resultado domain.AutenticacionRevalidadaV1
	var revisionTexto, superficie, metodo, garantia string
	err := tx.QueryRow(ctx, consultaRevalidarAutenticacionActorExternoV1,
		solicitud.AutenticacionRef, solicitud.SesionRef).Scan(
		&resultado.AutenticacionRef, &resultado.AutenticacionHuellaSHA256,
		&resultado.AsercionRef, &resultado.SesionRef,
		&resultado.ControlSesionRef, &revisionTexto,
		&resultado.ControlSesionHuellaSHA256,
		&resultado.CuentaRef, &resultado.CuentaOrdinariaRef,
		&resultado.CuentaPrivilegiada, &superficie,
		&metodo, &garantia, &resultado.PoliticaGarantiaRef,
		&resultado.PoliticaGarantiaHuellaSHA256,
		&resultado.AutenticacionVerificadaEn, &resultado.SesionEmitidaEn,
		&resultado.SesionValidaHasta, &resultado.SesionRevalidadaEn,
	)
	if err != nil {
		return domain.AutenticacionRevalidadaV1{}, err
	}
	revision, err := strconv.ParseUint(revisionTexto, 10, 64)
	if err != nil || revision == 0 {
		return domain.AutenticacionRevalidadaV1{}, ports.ErrRevalidacionAutenticacionActorNoDisponible
	}
	resultado.ControlSesionRevision = revision
	resultado.Superficie = domain.SuperficieAutenticacionActorV1(superficie)
	resultado.MetodoObservado = domain.AuthMethod(metodo)
	resultado.GarantiaObservada = domain.AuthAssurance(garantia)
	resultado.AutenticacionVerificadaEn = instanteUTCPostgreSQL(resultado.AutenticacionVerificadaEn)
	resultado.SesionEmitidaEn = instanteUTCPostgreSQL(resultado.SesionEmitidaEn)
	resultado.SesionValidaHasta = instanteUTCPostgreSQL(resultado.SesionValidaHasta)
	resultado.SesionRevalidadaEn = instanteUTCPostgreSQL(resultado.SesionRevalidadaEn)
	return resultado, nil
}

var _ ports.RevalidadorAutenticacionActorV1 = (*RevalidadorAutenticacionActorExternoPostgreSQL)(nil)
