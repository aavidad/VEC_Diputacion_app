package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func normalizarErrorRecuperacionResultadoCoberturaO405(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if errors.Is(causa, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(causa, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var postgres *pgconn.PgError
	if errors.As(causa, &postgres) {
		switch postgres.Code {
		case "23505":
			return cobertura.ErrHistoriaResultadoOperacionDecisionCoberturaDivergente
		case "42501", "55000":
			return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
		}
	}
	return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
}

func normalizarErrorFilasRecuperacionResultadoCoberturaO405(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if causa == nil {
		return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	var postgres *pgconn.PgError
	if errors.As(causa, &postgres) {
		switch postgres.Code {
		case "23505":
			return cobertura.ErrHistoriaResultadoOperacionDecisionCoberturaDivergente
		case "42501", "55000":
			return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
		}
	}
	if errors.Is(causa, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(causa, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
}

func normalizarErrorCallbackRecuperacionResultadoCoberturaO405(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	switch {
	case errors.Is(causa, context.Canceled):
		return context.Canceled
	case errors.Is(causa, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(
		causa,
		cobertura.ErrHistoriaResultadoOperacionDecisionCoberturaDivergente,
	):
		return cobertura.ErrHistoriaResultadoOperacionDecisionCoberturaDivergente
	case errors.Is(
		causa,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	):
		return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	case errors.Is(
		causa,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	):
		return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	default:
		return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
}
