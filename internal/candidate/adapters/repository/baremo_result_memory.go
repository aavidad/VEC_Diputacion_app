package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

type BaremoResultRepository struct {
	mu      sync.RWMutex
	results map[string]domain.BaremoResult
}

func NewBaremoResultRepository() *BaremoResultRepository {
	return &BaremoResultRepository{results: make(map[string]domain.BaremoResult)}
}

func (r *BaremoResultRepository) Save(
	ctx context.Context,
	candidateID string,
	result domain.BaremoResult,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return domain.ErrCandidateIDRequired
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapLocked()
	r.results[candidateID] = copyBaremoResult(result)
	return nil
}

func (r *BaremoResultRepository) GetByCandidate(
	ctx context.Context,
	candidateID string,
) (domain.BaremoResult, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.BaremoResult{}, false, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.results[strings.TrimSpace(candidateID)]
	if !ok {
		return domain.BaremoResult{}, false, nil
	}
	return copyBaremoResult(result), true, nil
}

func (r *BaremoResultRepository) ensureMapLocked() {
	if r.results == nil {
		r.results = make(map[string]domain.BaremoResult)
	}
}

func copyBaremoResult(result domain.BaremoResult) domain.BaremoResult {
	result.SectionPoints = copySectionPoints(result.SectionPoints)
	result.Details = append([]domain.BaremoMeritScore(nil), result.Details...)
	return result
}

func (s *DurableFileStore) snapshot() durableFileSnapshot {
	snap := durableFileSnapshot{SchemaVersion: durableFileSchemaVersion}
	s.memory.mu.RLock()
	for _, id := range sortedKeys(s.memory.candidates) {
		rec := s.memory.candidates[id]
		snap.Candidates = append(snap.Candidates, durableCandidateRecord{CallID: rec.callID, Candidate: rec.candidate})
	}
	for _, candidateID := range sortedKeys(s.memory.meritsByCandidate) {
		for _, id := range sortedKeys(s.memory.meritsByCandidate[candidateID]) {
			snap.Merits = append(snap.Merits, durableMeritRecord{CandidateID: candidateID, Merit: s.memory.meritsByCandidate[candidateID][id]})
		}
	}
	s.memory.mu.RUnlock()
	s.snapshotBaremo(&snap)
	s.snapshotProcedure(&snap)
	s.snapshotAdministrative(&snap)
	return snap
}

func (s *DurableFileStore) snapshotBaremo(snap *durableFileSnapshot) {
	s.baremo.mu.RLock()
	defer s.baremo.mu.RUnlock()
	for _, candidateID := range sortedKeys(s.baremo.results) {
		result := copyBaremoResult(s.baremo.results[candidateID])
		snap.BaremoResults = append(snap.BaremoResults, durableBaremoResultRecord{CandidateID: candidateID, Result: result})
	}
}

func (s *DurableFileStore) snapshotProcedure(snap *durableFileSnapshot) {
	s.procedure.mu.RLock()
	defer s.procedure.mu.RUnlock()
	for _, id := range sortedKeys(s.procedure.convocatorias) {
		record := s.procedure.convocatorias[id]
		snap.Convocatorias = append(snap.Convocatorias, durableConvocatoriaRecord{
			Convocatoria: record.Convocatoria,
			RuleSet:      record.RuleSet.Config(),
		})
	}
	for _, id := range sortedKeys(s.procedure.solicitudes) {
		snap.Solicitudes = append(snap.Solicitudes, copySolicitudRecord(s.procedure.solicitudes[id]))
	}
}

func (s *DurableFileStore) snapshotAdministrative(snap *durableFileSnapshot) {
	s.admin.mu.RLock()
	defer s.admin.mu.RUnlock()
	for _, id := range sortedKeys(s.admin.documents.records) {
		snap.Documents = append(snap.Documents, copyCandidateDocument(s.admin.documents.records[id]))
	}
	for _, id := range sortedKeys(s.admin.claims.records) {
		snap.Claims = append(snap.Claims, copyClaim(s.admin.claims.records[id]))
	}
	for _, id := range sortedKeys(s.admin.notifications.records) {
		snap.Notifications = append(snap.Notifications, copyNotification(s.admin.notifications.records[id]))
	}
	for _, scope := range sortedKeys(s.admin.auditByScope) {
		snap.Audit = append(snap.Audit, durableAuditRecord{Scope: scope, Entries: copyAuditEntries(s.admin.auditByScope[scope])})
	}
}

func (s *DurableFileStore) applySnapshot(snapshot durableFileSnapshot) error {
	ctx := context.Background()
	s.memory.mu.Lock()
	s.memory.candidates = map[string]candidateRecord{}
	s.memory.candidatesByCall = map[string]map[string]struct{}{}
	s.memory.meritsByCandidate = map[string]map[string]domain.Merit{}
	s.memory.mu.Unlock()
	candidates, merits := NewCandidateRepository(s.memory), NewMeritRepository(s.memory)
	for _, rec := range snapshot.Candidates {
		if err := candidates.Save(ctx, rec.CallID, rec.Candidate); err != nil {
			return fmt.Errorf("load candidate %q: %w", rec.Candidate.ID, err)
		}
	}
	for _, rec := range snapshot.Merits {
		if err := merits.Save(ctx, rec.CandidateID, rec.Merit); err != nil {
			return fmt.Errorf("load merit %q: %w", rec.Merit.ID, err)
		}
	}
	if err := s.applyBaremoSnapshot(ctx, snapshot); err != nil {
		return err
	}
	if err := s.applyProcedureSnapshot(ctx, snapshot); err != nil {
		return err
	}
	return s.applyAdministrativeSnapshot(ctx, snapshot)
}

func (s *DurableFileStore) applyBaremoSnapshot(ctx context.Context, snapshot durableFileSnapshot) error {
	s.baremo.mu.Lock()
	s.baremo.results = map[string]domain.BaremoResult{}
	s.baremo.mu.Unlock()
	for _, rec := range snapshot.BaremoResults {
		if err := s.baremo.Save(ctx, rec.CandidateID, rec.Result); err != nil {
			return fmt.Errorf("load baremo result %q: %w", rec.CandidateID, err)
		}
	}
	return nil
}

func (s *DurableFileStore) applyProcedureSnapshot(ctx context.Context, snapshot durableFileSnapshot) error {
	s.procedure.mu.Lock()
	s.procedure.convocatorias = map[string]ports.ConvocatoriaRecord{}
	s.procedure.solicitudes = map[string]ports.SolicitudRecord{}
	s.procedure.solicitudesByConvocatoria = map[string]map[string]struct{}{}
	s.procedure.mu.Unlock()
	convocatorias := NewProcedureConvocatoriaRepository(s.procedure)
	solicitudes := NewProcedureSolicitudRepository(s.procedure)
	for _, rec := range snapshot.Convocatorias {
		ruleSet, err := domain.NewBaremoRuleSet(rec.RuleSet)
		if err != nil {
			return fmt.Errorf("load convocatoria rule set %q: %w", rec.Convocatoria.ID, err)
		}
		if err := convocatorias.Save(ctx, ports.ConvocatoriaRecord{Convocatoria: rec.Convocatoria, RuleSet: ruleSet}); err != nil {
			return fmt.Errorf("load convocatoria %q: %w", rec.Convocatoria.ID, err)
		}
	}
	for _, rec := range snapshot.Solicitudes {
		if err := solicitudes.Save(ctx, rec); err != nil {
			return fmt.Errorf("load solicitud %q: %w", rec.ID, err)
		}
	}
	return nil
}

func (s *DurableFileStore) applyAdministrativeSnapshot(ctx context.Context, snapshot durableFileSnapshot) error {
	s.admin.mu.Lock()
	s.admin.documents = newIndexedMemory[domain.CandidateDocument]()
	s.admin.claims = newIndexedMemory[domain.Claim]()
	s.admin.notifications = newIndexedMemory[domain.Notification]()
	s.admin.auditByScope = map[string][]domain.AuditEntry{}
	s.admin.mu.Unlock()
	documents := NewAdministrativeCandidateDocumentRepository(s.admin)
	claims := NewAdministrativeClaimRepository(s.admin)
	notifications := NewAdministrativeNotificationRepository(s.admin)
	for _, document := range snapshot.Documents {
		if err := documents.Save(ctx, document); err != nil {
			return fmt.Errorf("load document %q: %w", document.ID, err)
		}
	}
	for _, claim := range snapshot.Claims {
		if err := claims.Save(ctx, claim); err != nil {
			return fmt.Errorf("load claim %q: %w", claim.ID, err)
		}
	}
	for _, notification := range snapshot.Notifications {
		if err := notifications.Save(ctx, notification); err != nil {
			return fmt.Errorf("load notification %q: %w", notification.ID, err)
		}
	}
	return s.applyAuditSnapshot(snapshot.Audit)
}

func (s *DurableFileStore) applyAuditSnapshot(records []durableAuditRecord) error {
	s.admin.mu.Lock()
	defer s.admin.mu.Unlock()
	for _, rec := range records {
		if len(rec.Entries) == 0 {
			continue
		}
		if err := domain.VerifyAuditChain(rec.Entries, administrativeAuditSigningRef); err != nil {
			return fmt.Errorf("load audit %q: %w", rec.Scope, err)
		}
		s.admin.auditByScope[strings.TrimSpace(rec.Scope)] = copyAuditEntries(rec.Entries)
	}
	return nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open durable dir: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync durable dir: %w", err)
	}
	return nil
}
