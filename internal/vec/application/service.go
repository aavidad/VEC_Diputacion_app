package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrServiceDependencyRequired = errors.New("vec service dependency required")

type Service struct {
	modules ports.ModuleRegistryStore
	audit   ports.AuditStore
	events  ports.EventStore
	now     func() time.Time
}

func NewService(
	modules ports.ModuleRegistryStore,
	audit ports.AuditStore,
	events ports.EventStore,
) (*Service, error) {
	if modules == nil || audit == nil || events == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &Service{
		modules: modules,
		audit:   audit,
		events:  events,
		now:     time.Now,
	}, nil
}

func (s *Service) RegisterModule(ctx context.Context, manifest domain.ModuleManifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	return s.modules.SaveModule(ctx, manifest)
}

func (s *Service) Modules(ctx context.Context) ([]domain.ModuleManifest, error) {
	return s.modules.ListModules(ctx)
}

func (s *Service) BuildMenu(ctx context.Context, principal domain.Principal) ([]domain.MenuEntry, error) {
	if err := principal.Validate(); err != nil {
		return nil, err
	}
	modules, err := s.modules.ListModules(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]domain.MenuEntry, 0)
	for _, module := range modules {
		for _, entry := range module.Menu {
			if principal.HasAllPermissions(entry.RequiredPermissions) {
				entries = append(entries, entry)
			}
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Order != entries[j].Order {
			return entries[i].Order < entries[j].Order
		}
		if entries[i].ModuleID != entries[j].ModuleID {
			return entries[i].ModuleID < entries[j].ModuleID
		}
		return entries[i].ID < entries[j].ID
	})
	return entries, nil
}

type AuditCommand struct {
	Principal  domain.Principal
	Action     string
	ModuleID   string
	SubjectRef string
	Result     string
	Metadata   map[string]string
}

func (s *Service) RecordAudit(ctx context.Context, command AuditCommand) (domain.AuditEntry, error) {
	if err := command.Principal.Validate(); err != nil {
		return domain.AuditEntry{}, err
	}
	if strings.TrimSpace(command.Action) == "" {
		return domain.AuditEntry{}, domain.ErrModuleInvalid
	}
	entry := domain.AuditEntry{
		ActorID:    command.Principal.ID,
		Action:     command.Action,
		ModuleID:   command.ModuleID,
		SubjectRef: command.SubjectRef,
		Result:     command.Result,
		Metadata:   command.Metadata,
		OccurredAt: s.now().UTC(),
	}
	return s.audit.AppendAudit(ctx, entry)
}

func (s *Service) PublishEvent(ctx context.Context, event domain.Event) error {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = s.now().UTC()
	}
	return s.events.PublishEvent(ctx, event)
}

func (s *Service) Audit(ctx context.Context, subjectRef string) ([]domain.AuditEntry, error) {
	return s.audit.ListAudit(ctx, subjectRef)
}
