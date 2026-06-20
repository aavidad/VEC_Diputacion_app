package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

var (
	ErrAdministrativeFlowPortsRequired = errors.New("administrative flow usecase: ports are required")
	ErrAdministrativeDocumentRequired  = errors.New("administrative flow usecase: document is required")
	ErrAdministrativeClaimRequired     = errors.New("administrative flow usecase: claim is required")
	ErrAdministrativeNoticeRequired    = errors.New("administrative flow usecase: notification is required")
	ErrAdministrativeRecipientMismatch = errors.New("administrative flow usecase: notification recipient mismatch")
	ErrAdministrativeAuditRequired     = errors.New("administrative flow usecase: audit scope is required")
)

type AdministrativeFlowUseCase struct {
	documents     ports.CandidateDocumentRepository
	claims        ports.ClaimRepository
	notifications ports.NotificationRepository
	audit         ports.AdministrativeAuditTrail
}

type RegisterCandidateDocumentCommand struct {
	ID           string
	CandidateID  string
	SolicitudID  string
	ProcedureID  string
	Purpose      domain.DocumentPurpose
	Evidence     domain.DocumentEvidence
	RegisteredBy string
	RegisteredAt time.Time
}

type PresentClaimCommand struct {
	ID          string
	CandidateID string
	SolicitudID string
	Text        string
	Documents   []domain.CandidateDocument
	PresentedBy string
	PresentedAt time.Time
	ReceiptCSV  string
}

type CreateNotificationCommand struct {
	ID          string
	CandidateID string
	SolicitudID string
	Type        string
	Subject     string
	Body        string
	CreatedBy   string
	CreatedAt   time.Time
}

type ReceiptCommand struct {
	NotificationID string
	CSV            string
	RecipientID    string
	Channel        string
	IssuedAt       time.Time
}

func NewAdministrativeFlowUseCase(
	documents ports.CandidateDocumentRepository,
	claims ports.ClaimRepository,
	notifications ports.NotificationRepository,
	audit ports.AdministrativeAuditTrail,
) (*AdministrativeFlowUseCase, error) {
	if documents == nil || claims == nil || notifications == nil || audit == nil {
		return nil, ErrAdministrativeFlowPortsRequired
	}
	return &AdministrativeFlowUseCase{
		documents: documents, claims: claims, notifications: notifications, audit: audit,
	}, nil
}

func (u *AdministrativeFlowUseCase) RegisterCandidateDocument(
	ctx context.Context,
	command RegisterCandidateDocumentCommand,
) (domain.CandidateDocument, domain.AuditEntry, error) {
	ctx = normalizeContext(ctx)
	document, err := domain.NewCandidateDocument(domain.CandidateDocumentInput(command))
	if err != nil {
		return domain.CandidateDocument{}, domain.AuditEntry{}, err
	}
	if err := u.documents.Save(ctx, document); err != nil {
		return domain.CandidateDocument{}, domain.AuditEntry{}, fmt.Errorf("save candidate document: %w", err)
	}
	entry, err := u.appendAudit(ctx, document.CandidateID, document.RegisteredBy, "candidate.document.registered", document.AuditPayload(), document.RegisteredAt)
	if err != nil {
		return domain.CandidateDocument{}, domain.AuditEntry{}, err
	}
	return document, entry, nil
}

func (u *AdministrativeFlowUseCase) PresentClaim(
	ctx context.Context,
	command PresentClaimCommand,
) (domain.Claim, domain.AuditEntry, error) {
	ctx = normalizeContext(ctx)
	claim, err := domain.NewClaim(domain.ClaimInput(command))
	if err != nil {
		return domain.Claim{}, domain.AuditEntry{}, err
	}
	if err := u.claims.Save(ctx, claim); err != nil {
		return domain.Claim{}, domain.AuditEntry{}, fmt.Errorf("save claim: %w", err)
	}
	entry, err := u.appendAudit(ctx, claim.CandidateID, claim.PresentedBy, "candidate.claim.presented", claim.AuditPayload(), claim.PresentedAt)
	if err != nil {
		return domain.Claim{}, domain.AuditEntry{}, err
	}
	return claim, entry, nil
}

func (u *AdministrativeFlowUseCase) CreateNotification(
	ctx context.Context,
	command CreateNotificationCommand,
) (domain.Notification, domain.AuditEntry, error) {
	ctx = normalizeContext(ctx)
	notification, err := domain.NewNotification(domain.NotificationInput(command))
	if err != nil {
		return domain.Notification{}, domain.AuditEntry{}, err
	}
	if err := u.notifications.Save(ctx, notification); err != nil {
		return domain.Notification{}, domain.AuditEntry{}, fmt.Errorf("save notification: %w", err)
	}
	entry, err := u.appendAudit(ctx, notification.CandidateID, notification.CreatedBy, "candidate.notification.created", notification.AuditPayload(), notification.CreatedAt)
	if err != nil {
		return domain.Notification{}, domain.AuditEntry{}, err
	}
	return notification, entry, nil
}

func (u *AdministrativeFlowUseCase) SendNotification(
	ctx context.Context,
	command ReceiptCommand,
) (domain.Notification, domain.AuditEntry, error) {
	return u.applyNotificationReceipt(ctx, command, "candidate.notification.sent", (*domain.Notification).Send)
}

func (u *AdministrativeFlowUseCase) MarkNotificationRead(
	ctx context.Context,
	command ReceiptCommand,
) (domain.Notification, domain.AuditEntry, error) {
	return u.applyNotificationReceipt(ctx, command, "candidate.notification.read", (*domain.Notification).MarkRead)
}

func (u *AdministrativeFlowUseCase) ListClaimsBySolicitud(ctx context.Context, solicitudID string) ([]domain.Claim, error) {
	ctx = normalizeContext(ctx)
	solicitudID = strings.TrimSpace(solicitudID)
	if solicitudID == "" {
		return nil, ErrAdministrativeClaimRequired
	}
	claims, err := u.claims.ListBySolicitud(ctx, solicitudID)
	if err != nil {
		return nil, fmt.Errorf("list claims by solicitud: %w", err)
	}
	return claims, nil
}

func (u *AdministrativeFlowUseCase) ListNotificationsByCandidate(ctx context.Context, candidateID string) ([]domain.Notification, error) {
	ctx = normalizeContext(ctx)
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return nil, ErrAdministrativeNoticeRequired
	}
	notifications, err := u.notifications.ListByCandidate(ctx, candidateID)
	if err != nil {
		return nil, fmt.Errorf("list notifications by candidate: %w", err)
	}
	return notifications, nil
}

func (u *AdministrativeFlowUseCase) ListAuditByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error) {
	ctx = normalizeContext(ctx)
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil, ErrAdministrativeAuditRequired
	}
	entries, err := u.audit.ListByScope(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("list audit by scope: %w", err)
	}
	return entries, nil
}

func (u *AdministrativeFlowUseCase) applyNotificationReceipt(
	ctx context.Context,
	command ReceiptCommand,
	action string,
	apply func(*domain.Notification, domain.NotificationReceipt) error,
) (domain.Notification, domain.AuditEntry, error) {
	ctx = normalizeContext(ctx)
	id := strings.TrimSpace(command.NotificationID)
	if id == "" {
		return domain.Notification{}, domain.AuditEntry{}, ErrAdministrativeNoticeRequired
	}
	notification, err := u.notifications.GetByID(ctx, id)
	if err != nil {
		return domain.Notification{}, domain.AuditEntry{}, fmt.Errorf("get notification: %w", err)
	}
	recipientID := strings.TrimSpace(command.RecipientID)
	if recipientID == "" {
		return domain.Notification{}, domain.AuditEntry{}, ErrAdministrativeNoticeRequired
	}
	if recipientID != strings.TrimSpace(notification.CandidateID) {
		return domain.Notification{}, domain.AuditEntry{}, ErrAdministrativeRecipientMismatch
	}
	receipt, err := domain.NewNotificationReceipt(command.CSV, recipientID, command.Channel, command.IssuedAt, notification.AuditPayload())
	if err != nil {
		return domain.Notification{}, domain.AuditEntry{}, err
	}
	if err := apply(&notification, receipt); err != nil {
		return domain.Notification{}, domain.AuditEntry{}, err
	}
	if err := u.notifications.Save(ctx, notification); err != nil {
		return domain.Notification{}, domain.AuditEntry{}, fmt.Errorf("save notification receipt: %w", err)
	}
	entry, err := u.appendAudit(ctx, notification.CandidateID, recipientID, action, notification.AuditPayload(), command.IssuedAt)
	if err != nil {
		return domain.Notification{}, domain.AuditEntry{}, err
	}
	return notification, entry, nil
}

func (u *AdministrativeFlowUseCase) appendAudit(
	ctx context.Context,
	candidateID string,
	actor string,
	action string,
	payload []byte,
	occurredAt time.Time,
) (domain.AuditEntry, error) {
	scope := "candidate:" + strings.TrimSpace(candidateID)
	entry, err := u.audit.Append(ctx, scope, domain.AuditEnvelope{
		Actor: strings.TrimSpace(actor), Action: action,
		OccurredAt: occurredAt.UTC(), Payload: payload,
	})
	if err != nil {
		return domain.AuditEntry{}, fmt.Errorf("append audit: %w", err)
	}
	return entry, nil
}
