package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
	"vec-diputacion-granada/internal/candidate/usecases"
)

type AdministrativeFlowService interface {
	RegisterCandidateDocument(context.Context, string, ports.AuthPrincipal, administrativeDocumentRequest) (administrativeDocumentView, error)
	ListCandidateDocuments(context.Context, string) ([]administrativeDocumentView, error)
	PresentCandidateClaim(context.Context, string, ports.AuthPrincipal, administrativeClaimRequest) (administrativeClaimView, error)
	ListCandidateClaims(context.Context, string, string) ([]administrativeClaimView, error)
	CreateCandidateNotification(context.Context, string, ports.AuthPrincipal, administrativeNotificationRequest) (administrativeNotificationView, error)
	ListCandidateNotifications(context.Context, string) ([]administrativeNotificationView, error)
	SendNotification(context.Context, ports.AuthPrincipal, administrativeNotificationReceiptRequest) (administrativeNotificationView, error)
	MarkNotificationRead(context.Context, ports.AuthPrincipal, administrativeNotificationReceiptRequest) (administrativeNotificationView, error)
	ListCandidateAudit(context.Context, string) ([]administrativeAuditView, error)
	ListAuditByScope(context.Context, string) ([]administrativeAuditView, error)
}

type administrativeFlowService struct {
	documents ports.CandidateDocumentRepository
	usecase   *usecases.AdministrativeFlowUseCase
	now       func() time.Time
}

var errCargaDocumentalSeguraNoDisponible = errors.New("carga documental segura no disponible")
var errFlujoProbatorioSeguroNoDisponible = errors.New("flujo probatorio seguro no disponible")

type administrativeDocumentRequest struct {
	ID               string                 `json:"id"`
	SolicitudID      string                 `json:"solicitud_id"`
	ProcedureID      string                 `json:"procedure_id"`
	Purpose          domain.DocumentPurpose `json:"purpose"`
	CSV              string                 `json:"csv"`
	DigestSHA256     string                 `json:"digest_sha256"`
	StorageObjectRef string                 `json:"storage_object_ref"`
	TSAStampRef      string                 `json:"tsa_stamp_ref"`
	DocumentID       string                 `json:"document_id"`
	DocumentType     string                 `json:"document_type"`
	Title            string                 `json:"title"`
	Format           string                 `json:"format"`
	Language         string                 `json:"language"`
	SignatureRef     string                 `json:"signature_ref"`
	RegisteredAt     time.Time              `json:"registered_at"`
}

type administrativeDocumentView struct {
	ID             string                 `json:"id"`
	CandidateID    string                 `json:"candidate_id"`
	SolicitudID    string                 `json:"solicitud_id"`
	ProcedureID    string                 `json:"procedure_id"`
	Purpose        domain.DocumentPurpose `json:"purpose"`
	CSV            string                 `json:"csv"`
	AVStatus       domain.AVStatus        `json:"av_status"`
	RegisteredBy   string                 `json:"registered_by"`
	RegisteredAt   time.Time              `json:"registered_at"`
	AuditSequence  int                    `json:"audit_sequence,omitempty"`
	ReceiptI18nKey string                 `json:"receipt_i18n_key"`
}

func NewAdministrativeFlowService(
	documents ports.CandidateDocumentRepository,
	usecase *usecases.AdministrativeFlowUseCase,
) AdministrativeFlowService {
	if documents == nil || usecase == nil {
		return nil
	}
	return administrativeFlowService{documents: documents, usecase: usecase, now: time.Now}
}

func (s administrativeFlowService) RegisterCandidateDocument(
	ctx context.Context,
	candidateID string,
	principal ports.AuthPrincipal,
	request administrativeDocumentRequest,
) (administrativeDocumentView, error) {
	at := request.RegisteredAt
	if at.IsZero() {
		at = s.now().UTC()
	}
	evidence, err := s.evidence(candidateID, principal.Subject, request, at)
	if err != nil {
		return administrativeDocumentView{}, err
	}
	document, audit, err := s.usecase.RegisterCandidateDocument(ctx, usecases.RegisterCandidateDocumentCommand{
		ID: strings.TrimSpace(request.ID), CandidateID: strings.TrimSpace(candidateID),
		SolicitudID: strings.TrimSpace(request.SolicitudID), ProcedureID: strings.TrimSpace(request.ProcedureID),
		Purpose: request.Purpose, Evidence: evidence, RegisteredBy: principal.Subject, RegisteredAt: at,
	})
	if err != nil {
		return administrativeDocumentView{}, err
	}
	view := documentView(document)
	view.AuditSequence = audit.Sequence
	return view, nil
}

func (s administrativeFlowService) ListCandidateDocuments(
	ctx context.Context,
	candidateID string,
) ([]administrativeDocumentView, error) {
	documents, err := s.documents.ListByCandidate(ctx, strings.TrimSpace(candidateID))
	if err != nil {
		return nil, err
	}
	views := make([]administrativeDocumentView, 0, len(documents))
	for _, document := range documents {
		views = append(views, documentView(document))
	}
	return views, nil
}

func (s administrativeFlowService) evidence(
	candidateID string,
	subject string,
	request administrativeDocumentRequest,
	at time.Time,
) (domain.DocumentEvidence, error) {
	return domain.NewDocumentEvidence(domain.DocumentEvidenceInput{
		CSV: strings.TrimSpace(request.CSV), DigestSHA256: strings.TrimSpace(request.DigestSHA256),
		Refs: domain.DocumentExternalRefs{StorageObjectRef: strings.TrimSpace(request.StorageObjectRef), TSAStampRef: strings.TrimSpace(request.TSAStampRef)},
		ENI: domain.ENIMetadata{
			DocumentID: strings.TrimSpace(defaultString(request.DocumentID, request.ID)), Organ: "Diputacion de Granada",
			Procedure: strings.TrimSpace(request.ProcedureID), DocumentType: strings.TrimSpace(request.DocumentType),
			Title: strings.TrimSpace(request.Title), Format: strings.TrimSpace(request.Format),
			Language: strings.TrimSpace(defaultString(request.Language, "es")), CreatedAt: at,
		},
		AVStatus: domain.AVStatusPending, SubmittedBy: strings.TrimSpace(defaultString(subject, candidateID)), SubmittedAt: at,
		Signatures: []domain.SignatureEvidence{{SignerID: strings.TrimSpace(defaultString(subject, candidateID)), SignatureRef: strings.TrimSpace(request.SignatureRef), SignedAt: at}},
	})
}

func (h *Handler) handleCandidateDocumentsRoute(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	principal ports.AuthPrincipal,
) {
	if h.administrative == nil {
		h.writeError(w, http.StatusNotFound, "api.error.not_found", errInvalidRoute)
		return
	}
	candidateID, _, ok := parseCandidateAction(path)
	if !ok {
		h.writeError(w, http.StatusNotFound, "api.error.not_found", errInvalidRoute)
		return
	}
	if !h.requireCandidateOwner(w, principal, candidateID) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleListCandidateDocuments(w, r, candidateID)
	case http.MethodPost:
		// El contrato heredado acepta CSV, huella, objeto, sello y firma
		// declarados por el navegador. Se mantiene cerrado hasta sustituirlo por
		// reserva de carga, cuarentena y confirmacion desde fuentes internas.
		h.writeError(w, http.StatusServiceUnavailable, "api.error.secure_upload_unavailable", errCargaDocumentalSeguraNoDisponible)
	default:
		w.Header().Set("Allow", "GET, POST")
		h.writeError(w, http.StatusMethodNotAllowed, "api.error.method_not_allowed", nil)
	}
}

func (h *Handler) handleRegisterCandidateDocument(
	w http.ResponseWriter,
	r *http.Request,
	candidateID string,
	principal ports.AuthPrincipal,
) {
	var request administrativeDocumentRequest
	if err := decodeJSON(r, &request); err != nil {
		h.writeError(w, http.StatusBadRequest, "api.error.bad_request", err)
		return
	}
	document, err := h.administrative.RegisterCandidateDocument(r.Context(), candidateID, principal, request)
	if err != nil {
		h.writeError(w, administrativeStatusFromError(err), administrativeErrorKey(err), err)
		return
	}
	h.writeJSON(w, http.StatusCreated, responseEnvelope{Message: h.t("api.candidate.document_registered"), Data: document})
}

func (h *Handler) handleListCandidateDocuments(w http.ResponseWriter, r *http.Request, candidateID string) {
	documents, err := h.administrative.ListCandidateDocuments(r.Context(), candidateID)
	if err != nil {
		h.writeError(w, administrativeStatusFromError(err), administrativeErrorKey(err), err)
		return
	}
	h.writeJSON(w, http.StatusOK, responseEnvelope{Message: h.t("api.candidate.documents_listed"), Data: documents})
}

func documentView(document domain.CandidateDocument) administrativeDocumentView {
	return administrativeDocumentView{
		ID: document.ID, CandidateID: document.CandidateID, SolicitudID: document.SolicitudID,
		ProcedureID: document.ProcedureID, Purpose: document.Purpose, CSV: string(document.Evidence.CSV),
		AVStatus: document.Evidence.AVStatus, RegisteredBy: document.RegisteredBy,
		RegisteredAt: document.RegisteredAt, ReceiptI18nKey: "module.bolsa.document.registered",
	}
}

func administrativeStatusFromError(err error) int {
	if errors.Is(err, ports.ErrCandidateDocumentNotFound) ||
		errors.Is(err, ports.ErrClaimNotFound) ||
		errors.Is(err, ports.ErrNotificationNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, domain.ErrDocumentInvalid) ||
		errors.Is(err, domain.ErrCandidateDocumentInvalid) ||
		errors.Is(err, domain.ErrClaimInvalid) ||
		errors.Is(err, domain.ErrNotificationInvalid) ||
		errors.Is(err, domain.ErrNotificationTransition) ||
		errors.Is(err, domain.ErrAuditInvalid) ||
		errors.Is(err, usecases.ErrAdministrativeClaimRequired) ||
		errors.Is(err, usecases.ErrAdministrativeNoticeRequired) ||
		errors.Is(err, usecases.ErrAdministrativeRecipientMismatch) ||
		errors.Is(err, usecases.ErrAdministrativeAuditRequired) {
		return http.StatusBadRequest
	}
	return statusFromError(err)
}

func administrativeErrorKey(err error) string {
	if administrativeStatusFromError(err) == http.StatusBadRequest {
		return "api.error.bad_request"
	}
	if administrativeStatusFromError(err) == http.StatusNotFound {
		return "api.error.not_found"
	}
	return errorKey(err)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
