package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

type ModuleRegistryStore interface {
	SaveModule(context.Context, domain.ModuleManifest) error
	ListModules(context.Context) ([]domain.ModuleManifest, error)
}

type AuditStore interface {
	AppendAudit(context.Context, domain.AuditEntry) (domain.AuditEntry, error)
	ListAudit(context.Context, string) ([]domain.AuditEntry, error)
}

type EventStore interface {
	PublishEvent(context.Context, domain.Event) error
	ListEvents(context.Context, []string) ([]domain.Event, error)
}

type CertAuthPort interface {
	Challenge(context.Context) (domain.AuthChallenge, error)
	Verify(context.Context, domain.AuthChallengeResponse) (domain.Principal, error)
}

type SignaturePort interface {
	Sign(context.Context, domain.Principal, domain.SignRequest) (domain.SignReceipt, error)
	VerifySignature(context.Context, string) (domain.SignVerification, error)
}
