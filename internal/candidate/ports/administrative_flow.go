package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/candidate/domain"
)

var (
	ErrCandidateDocumentNotFound = errors.New("administrative flow repository: candidate document not found")
	ErrClaimNotFound             = errors.New("administrative flow repository: claim not found")
	ErrNotificationNotFound      = errors.New("administrative flow repository: notification not found")
)

type CandidateDocumentRepository interface {
	Save(ctx context.Context, document domain.CandidateDocument) error
	GetByID(ctx context.Context, id string) (domain.CandidateDocument, error)
	ListByCandidate(ctx context.Context, candidateID string) ([]domain.CandidateDocument, error)
}

type ClaimRepository interface {
	Save(ctx context.Context, claim domain.Claim) error
	GetByID(ctx context.Context, id string) (domain.Claim, error)
	ListBySolicitud(ctx context.Context, solicitudID string) ([]domain.Claim, error)
}

type NotificationRepository interface {
	Save(ctx context.Context, notification domain.Notification) error
	GetByID(ctx context.Context, id string) (domain.Notification, error)
	ListByCandidate(ctx context.Context, candidateID string) ([]domain.Notification, error)
}

type AdministrativeAuditTrail interface {
	Append(ctx context.Context, scope string, envelope domain.AuditEnvelope) (domain.AuditEntry, error)
	ListByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error)
}
