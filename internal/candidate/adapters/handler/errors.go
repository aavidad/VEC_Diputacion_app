package handler

import (
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/candidate/application"
	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
	"vec-diputacion-granada/internal/candidate/usecases"
)

func statusFromError(err error) int {
	switch {
	case errors.Is(err, ports.ErrCandidateNotFound),
		errors.Is(err, ports.ErrConvocatoriaNotFound),
		errors.Is(err, ports.ErrSolicitudNotFound):
		return http.StatusNotFound
	case isValidationError(err):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func errorKey(err error) string {
	if errors.Is(err, ports.ErrCandidateNotFound) ||
		errors.Is(err, ports.ErrConvocatoriaNotFound) ||
		errors.Is(err, ports.ErrSolicitudNotFound) {
		return "api.error.not_found"
	}
	if isValidationError(err) {
		return "api.error.bad_request"
	}
	return "api.error.internal"
}

func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrCandidateIDRequired) ||
		errors.Is(err, domain.ErrCandidateDNIRequired) ||
		errors.Is(err, domain.ErrCandidateNombreRequired) ||
		errors.Is(err, domain.ErrCandidateEmailRequired) ||
		errors.Is(err, domain.ErrMeritIDRequired) ||
		errors.Is(err, domain.ErrMeritTypeInvalid) ||
		errors.Is(err, domain.ErrMeritStateInvalid) ||
		errors.Is(err, domain.ErrMeritDataInvalid) ||
		errors.Is(err, domain.ErrProcedureInvalid) ||
		errors.Is(err, domain.ErrProcedureTransition) ||
		errors.Is(err, domain.ErrProcedureRanking) ||
		errors.Is(err, usecases.ErrProcedureConvocatoriaRequired) ||
		errors.Is(err, usecases.ErrProcedureSolicitudRequired) ||
		errors.Is(err, application.ErrCallIDRequired) ||
		errors.Is(err, application.ErrCallNotConfigured) ||
		errors.Is(err, domain.ErrBaremoRuleSetInvalid) ||
		errors.Is(err, domain.ErrBaremoMeritNoRule)
}
