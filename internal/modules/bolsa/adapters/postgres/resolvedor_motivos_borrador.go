package postgres

import (
	"context"
	"errors"
	"time"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var _ gobiernoconvocatorias.ResolvedorMotivoBorrador = (*ResolvedorMotivoBorradorPostgreSQL)(nil)

// ResolvedorMotivoBorradorPostgreSQL adapta exclusivamente la validacion
// historica V2 de motivos al caso de uso de borradores. No busca versiones,
// no sustituye la referencia recibida y no vincula actor, recurso o accion:
// esa vinculacion pertenece al PDP de AutorizadorIntencionBorrador.
//
// La composicion debe construir el validador con un pool PostgreSQL exclusivo
// de solo consulta. El adaptador no recibe DSN, no abre conexiones y nunca
// devuelve detalles procedentes del controlador o de PostgreSQL.
type ResolvedorMotivoBorradorPostgreSQL struct {
	validador puertosvec.ValidadorReferenciaMotivoAutorizacionV2
}

// NuevoResolvedorMotivoBorradorPostgreSQL exige el validador PostgreSQL real.
// El constructor estrecho impide componer por accidente el adaptador de
// memoria o el catalogo de demostracion en esta frontera productiva.
func NuevoResolvedorMotivoBorradorPostgreSQL(
	validador *postgresvec.ValidadorReferenciaMotivoPostgreSQLV2,
) (*ResolvedorMotivoBorradorPostgreSQL, error) {
	return nuevoResolvedorMotivoBorradorPostgreSQL(validador)
}

func nuevoResolvedorMotivoBorradorPostgreSQL(
	validador puertosvec.ValidadorReferenciaMotivoAutorizacionV2,
) (*ResolvedorMotivoBorradorPostgreSQL, error) {
	if valorNulo(validador) {
		return nil, gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	return &ResolvedorMotivoBorradorPostgreSQL{validador: validador}, nil
}

// ResolverMotivoBorrador valida exactamente catalogo, version, huella,
// entrada e instante historico. Un resultado positivo devuelve la misma
// referencia por valor; cualquier incertidumbre falla cerrada.
func (r *ResolvedorMotivoBorradorPostgreSQL) ResolverMotivoBorrador(
	ctx context.Context,
	referencia dominiovec.ReferenciaEntradaCatalogo,
	instante time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	if r == nil || valorNulo(r.validador) || valorNulo(ctx) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(referencia) ||
		!instanteResolucionMotivoBorradorPostgreSQLValido(instante) {
		return dominiovec.ReferenciaEntradaCatalogo{},
			gobiernoconvocatorias.ErrOrdenBorradorInvalida
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorResolucionMotivoBorradorPostgreSQL(ctx, err)
	}
	if err := r.validador.ValidarReferenciaMotivoAutorizacionV2(
		ctx, referencia, instante,
	); err != nil {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorResolucionMotivoBorradorPostgreSQL(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorResolucionMotivoBorradorPostgreSQL(ctx, err)
	}
	return referencia, nil
}

func instanteResolucionMotivoBorradorPostgreSQLValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func errorResolucionMotivoBorradorPostgreSQL(ctx context.Context, causa error) error {
	componentes := []error{gobiernoconvocatorias.ErrOrdenBorradorInvalida}
	if !valorNulo(ctx) {
		if err := ctx.Err(); err != nil {
			return errors.Join(componentes[0], err)
		}
	}
	if errors.Is(causa, dominiovec.ErrSolicitudAutorizacionInvalida) {
		componentes = append(componentes, dominiovec.ErrSolicitudAutorizacionInvalida)
	}
	if errors.Is(causa, puertosvec.ErrFuenteAutorizacionNoDisponible) {
		componentes = append(componentes, puertosvec.ErrFuenteAutorizacionNoDisponible)
	}
	if errors.Is(causa, context.Canceled) {
		componentes = append(componentes, context.Canceled)
	}
	if errors.Is(causa, context.DeadlineExceeded) {
		componentes = append(componentes, context.DeadlineExceeded)
	}
	return errors.Join(componentes...)
}
