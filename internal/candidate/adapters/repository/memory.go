package repository

import (
	"context"
	"sort"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

var (
	_ ports.CandidateRepository = (*CandidateRepository)(nil)
	_ ports.MeritRepository     = (*MeritRepository)(nil)
)

type MemoryStore struct {
	mu sync.RWMutex

	// Shared by memory repositories and the durable file adapter snapshot.
	candidates        map[string]candidateRecord
	candidatesByCall  map[string]map[string]struct{}
	meritsByCandidate map[string]map[string]domain.Merit
}

type CandidateRepository struct {
	store *MemoryStore
}

type MeritRepository struct {
	store *MemoryStore
}

type candidateRecord struct {
	callID    string
	candidate domain.Candidate
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		candidates:        make(map[string]candidateRecord),
		candidatesByCall:  make(map[string]map[string]struct{}),
		meritsByCandidate: make(map[string]map[string]domain.Merit),
	}
}

func NewCandidateRepository(store *MemoryStore) *CandidateRepository {
	if store == nil {
		store = NewMemoryStore()
	}
	return &CandidateRepository{store: store}
}

func NewMeritRepository(store *MemoryStore) *MeritRepository {
	if store == nil {
		store = NewMemoryStore()
	}
	return &MeritRepository{store: store}
}

func NewRepositories() (*CandidateRepository, *MeritRepository) {
	store := NewMemoryStore()
	return NewCandidateRepository(store), NewMeritRepository(store)
}

func (r *CandidateRepository) Save(
	ctx context.Context,
	callID string,
	candidate domain.Candidate,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := candidate.Validate(); err != nil {
		return err
	}
	callID = strings.TrimSpace(callID)
	candidate.ID = strings.TrimSpace(candidate.ID)

	store := r.memoryStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureMapsLocked()

	if previous, ok := store.candidates[candidate.ID]; ok && previous.callID != callID {
		delete(store.candidatesByCall[previous.callID], candidate.ID)
		if len(store.candidatesByCall[previous.callID]) == 0 {
			delete(store.candidatesByCall, previous.callID)
		}
	}

	store.candidates[candidate.ID] = candidateRecord{
		callID:    callID,
		candidate: candidate,
	}
	if store.candidatesByCall[callID] == nil {
		store.candidatesByCall[callID] = make(map[string]struct{})
	}
	store.candidatesByCall[callID][candidate.ID] = struct{}{}
	return nil
}

func (r *CandidateRepository) GetByID(
	ctx context.Context,
	id string,
) (domain.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return domain.Candidate{}, err
	}

	store := r.memoryStore()
	store.mu.RLock()
	defer store.mu.RUnlock()

	record, ok := store.candidates[strings.TrimSpace(id)]
	if !ok {
		return domain.Candidate{}, ports.ErrCandidateNotFound
	}
	return record.candidate, nil
}

func (r *CandidateRepository) ListByCall(
	ctx context.Context,
	callID string,
) ([]domain.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	store := r.memoryStore()
	store.mu.RLock()
	defer store.mu.RUnlock()

	ids := sortedKeys(store.candidatesByCall[strings.TrimSpace(callID)])
	candidates := make([]domain.Candidate, 0, len(ids))
	for _, id := range ids {
		if record, ok := store.candidates[id]; ok {
			candidates = append(candidates, record.candidate)
		}
	}
	return candidates, nil
}

func (r *MeritRepository) Save(
	ctx context.Context,
	candidateID string,
	merit domain.Merit,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := merit.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(candidateID) == "" {
		return domain.ErrCandidateIDRequired
	}
	candidateID = strings.TrimSpace(candidateID)
	merit.ID = strings.TrimSpace(merit.ID)

	store := r.memoryStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureMapsLocked()

	if store.meritsByCandidate[candidateID] == nil {
		store.meritsByCandidate[candidateID] = make(map[string]domain.Merit)
	}
	store.meritsByCandidate[candidateID][merit.ID] = merit
	return nil
}

func (r *MeritRepository) ListByCandidate(
	ctx context.Context,
	candidateID string,
) ([]domain.Merit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	store := r.memoryStore()
	store.mu.RLock()
	defer store.mu.RUnlock()

	ids := sortedKeys(store.meritsByCandidate[strings.TrimSpace(candidateID)])
	merits := make([]domain.Merit, 0, len(ids))
	for _, id := range ids {
		merits = append(merits, store.meritsByCandidate[candidateID][id])
	}
	return merits, nil
}

func (r *CandidateRepository) memoryStore() *MemoryStore {
	if r == nil || r.store == nil {
		return NewMemoryStore()
	}
	return r.store
}

func (r *MeritRepository) memoryStore() *MemoryStore {
	if r == nil || r.store == nil {
		return NewMemoryStore()
	}
	return r.store
}

func (r *MemoryStore) ensureMapsLocked() {
	if r.candidates == nil {
		r.candidates = make(map[string]candidateRecord)
	}
	if r.candidatesByCall == nil {
		r.candidatesByCall = make(map[string]map[string]struct{})
	}
	if r.meritsByCandidate == nil {
		r.meritsByCandidate = make(map[string]map[string]domain.Merit)
	}
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
