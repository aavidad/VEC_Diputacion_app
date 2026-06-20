package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type Store struct {
	mu      sync.RWMutex
	modules map[string]domain.ModuleManifest
	audit   []domain.AuditEntry
	events  []domain.Event
}

func NewStore() *Store {
	return &Store{modules: map[string]domain.ModuleManifest{}}
}

func (s *Store) SaveModule(_ context.Context, manifest domain.ModuleManifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modules[manifest.ID] = manifest
	return nil
}

func (s *Store) ListModules(_ context.Context) ([]domain.ModuleManifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	modules := make([]domain.ModuleManifest, 0, len(s.modules))
	for _, module := range s.modules {
		modules = append(modules, module)
	}
	sort.SliceStable(modules, func(i, j int) bool { return modules[i].ID < modules[j].ID })
	return modules, nil
}

func (s *Store) AppendAudit(_ context.Context, entry domain.AuditEntry) (domain.AuditEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.Seq = int64(len(s.audit) + 1)
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("audit-%06d", entry.Seq)
	}
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = time.Now().UTC()
	}
	if len(s.audit) > 0 {
		entry.PrevSignature = s.audit[len(s.audit)-1].Signature
	}
	entry.Signature = auditSignature(entry)
	s.audit = append(s.audit, entry)
	return entry, nil
}

func (s *Store) ListAudit(_ context.Context, subjectRef string) ([]domain.AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.AuditEntry, 0, len(s.audit))
	for _, entry := range s.audit {
		if strings.TrimSpace(subjectRef) == "" || entry.SubjectRef == subjectRef {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (s *Store) PublishEvent(_ context.Context, event domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.ID == "" {
		event.ID = fmt.Sprintf("event-%06d", len(s.events)+1)
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	s.events = append(s.events, event)
	return nil
}

func (s *Store) ListEvents(_ context.Context, types []string) ([]domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	allowed := map[string]bool{}
	for _, eventType := range types {
		allowed[eventType] = true
	}
	result := make([]domain.Event, 0, len(s.events))
	for _, event := range s.events {
		if len(allowed) == 0 || allowed[event.Type] {
			result = append(result, event)
		}
	}
	return result, nil
}

func auditSignature(entry domain.AuditEntry) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%s",
		entry.Seq,
		entry.ActorID,
		entry.Action,
		entry.ModuleID,
		entry.SubjectRef,
		entry.Result,
		entry.OccurredAt.UTC().Format(time.RFC3339Nano),
		entry.PrevSignature,
	)))
	return hex.EncodeToString(sum[:])
}
