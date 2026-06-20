package handler

import (
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/candidate/ports"
)

func (h *Handler) handleCreateCandidate(
	w http.ResponseWriter,
	r *http.Request,
	principal ports.AuthPrincipal,
) {
	if !h.requireCandidate(w, principal) || !h.requireMethod(w, r, http.MethodPost) {
		return
	}
	var request CreateCandidateCommand
	if err := decodeJSON(r, &request); err != nil {
		h.writeError(w, http.StatusBadRequest, "api.error.bad_request", err)
		return
	}
	if candidateID := strings.TrimSpace(request.ID); candidateID != "" && !h.requireCandidateOwner(w, principal, candidateID) {
		return
	}
	candidate, err := h.service.CreateCandidate(r.Context(), request)
	if err != nil {
		h.writeError(w, statusFromError(err), errorKey(err), err)
		return
	}
	h.writeJSON(w, http.StatusCreated, responseEnvelope{
		Message: h.t("api.candidate.created"),
		Data:    candidate,
	})
}

func (h *Handler) handleCandidateAction(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	principal ports.AuthPrincipal,
) {
	if !h.requireCandidate(w, principal) {
		return
	}
	candidateID, action, ok := parseCandidateAction(path)
	if !ok {
		h.writeError(w, http.StatusNotFound, "api.error.not_found", errInvalidRoute)
		return
	}
	if !h.requireCandidateOwner(w, principal, candidateID) {
		return
	}

	switch action {
	case "merits":
		if h.requireMethod(w, r, http.MethodPost) {
			h.handleAddMerit(w, r, candidateID)
		}
	case "baremo":
		if h.requireMethod(w, r, http.MethodPost) {
			h.handleCalculateBaremo(w, r, candidateID)
		}
	case "expediente":
		if h.requireMethod(w, r, http.MethodGet) {
			h.handleExportExpediente(w, r, candidateID)
		}
	default:
		h.writeError(w, http.StatusNotFound, "api.error.not_found", errInvalidRoute)
	}
}

func (h *Handler) handleAddMerit(
	w http.ResponseWriter,
	r *http.Request,
	candidateID string,
) {
	var request AddMeritCommand
	if err := decodeJSON(r, &request); err != nil {
		h.writeError(w, http.StatusBadRequest, "api.error.bad_request", err)
		return
	}
	merit, err := h.service.AddMerit(r.Context(), candidateID, request)
	if err != nil {
		h.writeError(w, statusFromError(err), errorKey(err), err)
		return
	}
	h.writeJSON(w, http.StatusCreated, responseEnvelope{
		Message: h.t("api.candidate.merit_added"),
		Data:    merit,
	})
}

func (h *Handler) handleCalculateBaremo(
	w http.ResponseWriter,
	r *http.Request,
	candidateID string,
) {
	result, err := h.service.CalculateBaremo(r.Context(), candidateID)
	if err != nil {
		h.writeError(w, statusFromError(err), errorKey(err), err)
		return
	}
	h.writeJSON(w, http.StatusOK, responseEnvelope{
		Message: h.t("api.candidate.baremo_calculated"),
		Data:    result,
	})
}

func (h *Handler) handleExportExpediente(
	w http.ResponseWriter,
	r *http.Request,
	candidateID string,
) {
	result, err := h.service.ExportExpediente(r.Context(), candidateID)
	if err != nil {
		h.writeError(w, statusFromError(err), errorKey(err), err)
		return
	}
	h.writeJSON(w, http.StatusOK, responseEnvelope{
		Message: h.t("api.candidate.expediente_exported"),
		Data:    result,
	})
}
