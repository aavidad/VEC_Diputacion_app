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
	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	"vec-diputacion-granada/internal/shared/i18n"
)

type Handler struct {
	service        Service
	demoRunner     ProcedureDemoRunner
	administrative AdministrativeFlowService
	authenticator  ports.Authenticator
	messages       *i18n.Catalog
	status         bolsamodule.OperationalStatus
}

type administrativeClaimRequest struct {
	ID          string    `json:"id"`
	SolicitudID string    `json:"solicitud_id"`
	Text        string    `json:"text"`
	ReceiptCSV  string    `json:"receipt_csv"`
	PresentedAt time.Time `json:"presented_at"`
}

type administrativeClaimView struct {
	ID             string            `json:"id"`
	CandidateID    string            `json:"candidate_id"`
	SolicitudID    string            `json:"solicitud_id"`
	Text           string            `json:"text"`
	State          domain.ClaimState `json:"state"`
	PresentedBy    string            `json:"presented_by"`
	PresentedAt    time.Time         `json:"presented_at"`
	ReceiptCSV     string            `json:"receipt_csv"`
	AuditSequence  int               `json:"audit_sequence,omitempty"`
	ReceiptI18nKey string            `json:"receipt_i18n_key"`
}

type administrativeNotificationRequest struct {
	ID          string    `json:"id"`
	SolicitudID string    `json:"solicitud_id"`
	Type        string    `json:"type"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
}

type administrativeNotificationView struct {
	ID             string                   `json:"id"`
	CandidateID    string                   `json:"candidate_id"`
	SolicitudID    string                   `json:"solicitud_id"`
	Type           string                   `json:"type"`
	Subject        string                   `json:"subject"`
	State          domain.NotificationState `json:"state"`
	CreatedBy      string                   `json:"created_by"`
	CreatedAt      time.Time                `json:"created_at"`
	AuditSequence  int                      `json:"audit_sequence,omitempty"`
	ReceiptI18nKey string                   `json:"receipt_i18n_key"`
}

type administrativeAuditView struct {
	Sequence      int       `json:"sequence"`
	OccurredAt    time.Time `json:"occurred_at"`
	Actor         string    `json:"actor"`
	Action        string    `json:"action"`
	PayloadHash   string    `json:"payload_hash"`
	PrevSignature string    `json:"prev_signature,omitempty"`
	Signature     string    `json:"signature"`
}

func (s administrativeFlowService) PresentCandidateClaim(ctx context.Context, candidateID string, principal ports.AuthPrincipal, request administrativeClaimRequest) (administrativeClaimView, error) {
	at := request.PresentedAt
	if at.IsZero() {
		at = s.now().UTC()
	}
	claim, audit, err := s.usecase.PresentClaim(ctx, usecases.PresentClaimCommand{
		ID: strings.TrimSpace(request.ID), CandidateID: strings.TrimSpace(candidateID),
		SolicitudID: defaultString(request.SolicitudID, candidateID), Text: strings.TrimSpace(request.Text),
		PresentedBy: strings.TrimSpace(principal.Subject), PresentedAt: at, ReceiptCSV: strings.TrimSpace(request.ReceiptCSV),
	})
	if err != nil {
		return administrativeClaimView{}, err
	}
	view := claimView(claim)
	view.AuditSequence = audit.Sequence
	return view, nil
}

func (s administrativeFlowService) ListCandidateClaims(ctx context.Context, candidateID, solicitudID string) ([]administrativeClaimView, error) {
	claims, err := s.usecase.ListClaimsBySolicitud(ctx, defaultString(solicitudID, candidateID))
	if err != nil {
		return nil, err
	}
	views := make([]administrativeClaimView, 0, len(claims))
	for _, claim := range claims {
		if claim.CandidateID == strings.TrimSpace(candidateID) {
			views = append(views, claimView(claim))
		}
	}
	return views, nil
}

func (s administrativeFlowService) CreateCandidateNotification(ctx context.Context, candidateID string, principal ports.AuthPrincipal, request administrativeNotificationRequest) (administrativeNotificationView, error) {
	at := request.CreatedAt
	if at.IsZero() {
		at = s.now().UTC()
	}
	notification, audit, err := s.usecase.CreateNotification(ctx, usecases.CreateNotificationCommand{
		ID: strings.TrimSpace(request.ID), CandidateID: strings.TrimSpace(candidateID),
		SolicitudID: defaultString(request.SolicitudID, candidateID), Type: strings.TrimSpace(request.Type),
		Subject: strings.TrimSpace(request.Subject), Body: strings.TrimSpace(request.Body),
		CreatedBy: strings.TrimSpace(principal.Subject), CreatedAt: at,
	})
	if err != nil {
		return administrativeNotificationView{}, err
	}
	view := notificationView(notification)
	view.AuditSequence = audit.Sequence
	return view, nil
}

func (s administrativeFlowService) ListCandidateNotifications(ctx context.Context, candidateID string) ([]administrativeNotificationView, error) {
	notifications, err := s.usecase.ListNotificationsByCandidate(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	views := make([]administrativeNotificationView, 0, len(notifications))
	for _, notification := range notifications {
		views = append(views, notificationView(notification))
	}
	return views, nil
}

func (s administrativeFlowService) ListCandidateAudit(ctx context.Context, candidateID string) ([]administrativeAuditView, error) {
	entries, err := s.usecase.ListAuditByScope(ctx, "candidate:"+strings.TrimSpace(candidateID))
	if err != nil {
		return nil, err
	}
	views := make([]administrativeAuditView, 0, len(entries))
	for _, entry := range entries {
		views = append(views, auditView(entry))
	}
	return views, nil
}

func claimView(claim domain.Claim) administrativeClaimView {
	return administrativeClaimView{
		ID: claim.ID, CandidateID: claim.CandidateID, SolicitudID: claim.SolicitudID, Text: claim.Text,
		State: claim.State, PresentedBy: claim.PresentedBy, PresentedAt: claim.PresentedAt,
		ReceiptCSV: string(claim.Receipt.CSV), ReceiptI18nKey: "module.bolsa.claim.presented",
	}
}

func notificationView(notification domain.Notification) administrativeNotificationView {
	return administrativeNotificationView{
		ID: notification.ID, CandidateID: notification.CandidateID, SolicitudID: notification.SolicitudID,
		Type: notification.Type, Subject: notification.Subject, State: notification.State,
		CreatedBy: notification.CreatedBy, CreatedAt: notification.CreatedAt,
		ReceiptI18nKey: "module.bolsa.notification.created",
	}
}

func auditView(entry domain.AuditEntry) administrativeAuditView {
	return administrativeAuditView{
		Sequence: entry.Sequence, OccurredAt: entry.OccurredAt, Actor: entry.Actor,
		Action: entry.Action, PayloadHash: entry.PayloadHash, PrevSignature: entry.PrevSignature,
		Signature: entry.Signature,
	}
}

func NewHTTPHandler(
	service Service,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
) (*Handler, error) {
	return NewHTTPHandlerWithProcedure(service, nil, authenticator, messages)
}

func NewHTTPHandlerWithProcedure(
	service Service,
	procedure ProcedureUseCase,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
) (*Handler, error) {
	return NewHTTPHandlerWithDemoRunner(service, NewProcedureDemoRunner(procedure), authenticator, messages)
}

func NewHTTPHandlerWithDemoRunner(
	service Service,
	demoRunner ProcedureDemoRunner,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
) (*Handler, error) {
	return NewHTTPHandlerWithModules(service, demoRunner, nil, authenticator, messages)
}

func NewHTTPHandlerWithModules(
	service Service,
	demoRunner ProcedureDemoRunner,
	administrative AdministrativeFlowService,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
) (*Handler, error) {
	return NewHTTPHandlerWithModulesAndStatus(
		service,
		demoRunner,
		administrative,
		authenticator,
		messages,
		bolsamodule.OperationalStatusDefault(demoRunner != nil),
	)
}

func NewHTTPHandlerWithModulesAndStatus(
	service Service,
	demoRunner ProcedureDemoRunner,
	administrative AdministrativeFlowService,
	authenticator ports.Authenticator,
	messages *i18n.Catalog,
	status bolsamodule.OperationalStatus,
) (*Handler, error) {
	if service == nil || authenticator == nil {
		return nil, errors.New("handler: service and authenticator are required")
	}
	if messages == nil {
		messages = fallbackCatalog()
	}
	if strings.TrimSpace(status.ModuleRef) == "" {
		status = bolsamodule.OperationalStatusDefault(demoRunner != nil)
	}
	return &Handler{
		service:        service,
		demoRunner:     demoRunner,
		administrative: administrative,
		authenticator:  authenticator,
		messages:       messages,
		status:         status,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	principal = normalizePrincipalRole(principal)
	if !hasAllowedRole(principal.Role) {
		h.writeError(w, http.StatusForbidden, "api.error.forbidden", nil)
		return
	}
	h.dispatch(w, r, principal)
}

func normalizePrincipalRole(principal ports.AuthPrincipal) ports.AuthPrincipal {
	roles := make(map[ports.AuthRole]struct{}, len(principal.Roles)+1)
	invalid := false
	add := func(role ports.AuthRole) {
		if role == "" {
			return
		}
		if !role.IsValid() {
			invalid = true
			return
		}
		roles[role] = struct{}{}
	}
	add(principal.Role)
	for _, role := range principal.Roles {
		add(role)
	}
	if invalid || len(roles) != 1 {
		// El handler heredado autoriza por perfil grueso y no puede componer
		// asignaciones con alcance. Un conjunto ambiguo o parcialmente invalido
		// queda sin autoridad; ningun valor se descarta para salvar los restantes.
		principal.Role = ""
		principal.Roles = nil
		return principal
	}
	for role := range roles {
		principal.Role = role
		principal.Roles = []ports.AuthRole{role}
	}
	return principal
}
