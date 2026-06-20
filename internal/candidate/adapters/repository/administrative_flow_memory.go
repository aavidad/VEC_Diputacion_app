package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

// Stable signing ref preserves audit signatures across durable reloads.
const administrativeAuditSigningRef = "administrative-flow-memory"

var (
	_ ports.CandidateDocumentRepository = (*AdministrativeCandidateDocumentRepository)(nil)
	_ ports.ClaimRepository             = (*AdministrativeClaimRepository)(nil)
	_ ports.NotificationRepository      = (*AdministrativeNotificationRepository)(nil)
	_ ports.AdministrativeAuditTrail    = (*AdministrativeAuditTrail)(nil)
)

type AdministrativeFlowMemoryStore struct {
	mu sync.RWMutex

	documents     indexedMemory[domain.CandidateDocument]
	claims        indexedMemory[domain.Claim]
	notifications indexedMemory[domain.Notification]
	auditByScope  map[string][]domain.AuditEntry
}

type indexedMemory[T any] struct {
	records map[string]T
	byOwner map[string]map[string]struct{}
}

type AdministrativeCandidateDocumentRepository struct {
	store *AdministrativeFlowMemoryStore
}
type AdministrativeClaimRepository struct {
	store *AdministrativeFlowMemoryStore
}
type AdministrativeNotificationRepository struct {
	store *AdministrativeFlowMemoryStore
}

type AdministrativeAuditTrail struct {
	store *AdministrativeFlowMemoryStore
}

func NewAdministrativeFlowMemoryStore() *AdministrativeFlowMemoryStore {
	return &AdministrativeFlowMemoryStore{
		documents:     newIndexedMemory[domain.CandidateDocument](),
		claims:        newIndexedMemory[domain.Claim](),
		notifications: newIndexedMemory[domain.Notification](),
		auditByScope:  make(map[string][]domain.AuditEntry),
	}
}

func NewAdministrativeCandidateDocumentRepository(store *AdministrativeFlowMemoryStore) *AdministrativeCandidateDocumentRepository {
	return &AdministrativeCandidateDocumentRepository{store: adminStore(store)}
}
func NewAdministrativeClaimRepository(store *AdministrativeFlowMemoryStore) *AdministrativeClaimRepository {
	return &AdministrativeClaimRepository{store: adminStore(store)}
}
func NewAdministrativeNotificationRepository(store *AdministrativeFlowMemoryStore) *AdministrativeNotificationRepository {
	return &AdministrativeNotificationRepository{store: adminStore(store)}
}
func NewAdministrativeAuditTrail(store *AdministrativeFlowMemoryStore) *AdministrativeAuditTrail {
	return &AdministrativeAuditTrail{store: adminStore(store)}
}

func (r *AdministrativeCandidateDocumentRepository) Save(ctx context.Context, document domain.CandidateDocument) error {
	document.ID = strings.TrimSpace(document.ID)
	document.CandidateID = strings.TrimSpace(document.CandidateID)
	store := adminStore(r.store)
	return saveIndexed(ctx, store, &store.documents, document, document.ID, document.CandidateID, candidateDocumentOwner, validateCandidateDocument, copyCandidateDocument)
}
func (r *AdministrativeCandidateDocumentRepository) GetByID(ctx context.Context, id string) (domain.CandidateDocument, error) {
	store := adminStore(r.store)
	return getIndexed(ctx, store, &store.documents, id, ports.ErrCandidateDocumentNotFound, copyCandidateDocument)
}
func (r *AdministrativeCandidateDocumentRepository) ListByCandidate(ctx context.Context, candidateID string) ([]domain.CandidateDocument, error) {
	store := adminStore(r.store)
	return listIndexed(ctx, store, &store.documents, candidateID, copyCandidateDocument)
}
func (r *AdministrativeClaimRepository) Save(ctx context.Context, claim domain.Claim) error {
	claim.ID = strings.TrimSpace(claim.ID)
	claim.SolicitudID = strings.TrimSpace(claim.SolicitudID)
	store := adminStore(r.store)
	return saveIndexed(ctx, store, &store.claims, claim, claim.ID, claim.SolicitudID, claimOwner, validateClaim, copyClaim)
}
func (r *AdministrativeClaimRepository) GetByID(ctx context.Context, id string) (domain.Claim, error) {
	store := adminStore(r.store)
	return getIndexed(ctx, store, &store.claims, id, ports.ErrClaimNotFound, copyClaim)
}
func (r *AdministrativeClaimRepository) ListBySolicitud(ctx context.Context, solicitudID string) ([]domain.Claim, error) {
	store := adminStore(r.store)
	return listIndexed(ctx, store, &store.claims, solicitudID, copyClaim)
}
func (r *AdministrativeNotificationRepository) Save(ctx context.Context, notification domain.Notification) error {
	notification.ID = strings.TrimSpace(notification.ID)
	notification.CandidateID = strings.TrimSpace(notification.CandidateID)
	store := adminStore(r.store)
	return saveIndexed(ctx, store, &store.notifications, notification, notification.ID, notification.CandidateID, notificationOwner, validateNotification, copyNotification)
}
func (r *AdministrativeNotificationRepository) GetByID(ctx context.Context, id string) (domain.Notification, error) {
	store := adminStore(r.store)
	return getIndexed(ctx, store, &store.notifications, id, ports.ErrNotificationNotFound, copyNotification)
}
func (r *AdministrativeNotificationRepository) ListByCandidate(ctx context.Context, candidateID string) ([]domain.Notification, error) {
	store := adminStore(r.store)
	return listIndexed(ctx, store, &store.notifications, candidateID, copyNotification)
}
func (a *AdministrativeAuditTrail) Append(ctx context.Context, scope string, envelope domain.AuditEnvelope) (domain.AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return domain.AuditEntry{}, err
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return domain.AuditEntry{}, fmt.Errorf("%w: scope is required", domain.ErrAuditInvalid)
	}

	store := adminStore(a.store)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureMapsLocked()
	var previous *domain.AuditEntry
	if entries := store.auditByScope[scope]; len(entries) > 0 {
		previous = &entries[len(entries)-1]
	}
	entry, err := domain.NewAuditEntry(previous, envelope, administrativeAuditSigningRef)
	if err != nil {
		return domain.AuditEntry{}, err
	}
	store.auditByScope[scope] = append(store.auditByScope[scope], entry)
	return entry, nil
}

func (a *AdministrativeAuditTrail) ListByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	store := adminStore(a.store)
	store.mu.RLock()
	defer store.mu.RUnlock()
	return copyAuditEntries(store.auditByScope[strings.TrimSpace(scope)]), nil
}

func saveIndexed[T any](
	ctx context.Context,
	store *AdministrativeFlowMemoryStore,
	bucket *indexedMemory[T],
	record T,
	id string,
	owner string,
	ownerOf func(T) string,
	validate func(T) error,
	clone func(T) T,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validate(record); err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureMapsLocked()
	previousOwner := ""
	if previous, ok := bucket.records[id]; ok {
		previousOwner = ownerOf(previous)
	}
	moveIndex(bucket.byOwner, previousOwner, owner, id)
	bucket.records[id] = clone(record)
	return nil
}

func getIndexed[T any](
	ctx context.Context,
	store *AdministrativeFlowMemoryStore,
	bucket *indexedMemory[T],
	id string,
	notFound error,
	clone func(T) T,
) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	record, ok := bucket.records[strings.TrimSpace(id)]
	if !ok {
		return zero, notFound
	}
	return clone(record), nil
}

func listIndexed[T any](
	ctx context.Context,
	store *AdministrativeFlowMemoryStore,
	bucket *indexedMemory[T],
	owner string,
	clone func(T) T,
) ([]T, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	ids := sortedKeys(bucket.byOwner[strings.TrimSpace(owner)])
	records := make([]T, 0, len(ids))
	for _, id := range ids {
		records = append(records, clone(bucket.records[id]))
	}
	return records, nil
}

func adminStore(store *AdministrativeFlowMemoryStore) *AdministrativeFlowMemoryStore {
	if store == nil {
		return NewAdministrativeFlowMemoryStore()
	}
	return store
}

func (s *AdministrativeFlowMemoryStore) ensureMapsLocked() {
	ensureIndexed(&s.documents)
	ensureIndexed(&s.claims)
	ensureIndexed(&s.notifications)
	if s.auditByScope == nil {
		s.auditByScope = make(map[string][]domain.AuditEntry)
	}
}

func newIndexedMemory[T any]() indexedMemory[T] {
	return indexedMemory[T]{records: make(map[string]T), byOwner: make(map[string]map[string]struct{})}
}

func ensureIndexed[T any](bucket *indexedMemory[T]) {
	if bucket.records == nil {
		bucket.records = make(map[string]T)
	}
	if bucket.byOwner == nil {
		bucket.byOwner = make(map[string]map[string]struct{})
	}
}

func moveIndex(index map[string]map[string]struct{}, previousOwner, nextOwner, id string) {
	previousOwner = strings.TrimSpace(previousOwner)
	nextOwner = strings.TrimSpace(nextOwner)
	if previousOwner != "" && previousOwner != nextOwner {
		delete(index[previousOwner], id)
		if len(index[previousOwner]) == 0 {
			delete(index, previousOwner)
		}
	}
	if index[nextOwner] == nil {
		index[nextOwner] = make(map[string]struct{})
	}
	index[nextOwner][id] = struct{}{}
}

func candidateDocumentOwner(document domain.CandidateDocument) string   { return document.CandidateID }
func claimOwner(claim domain.Claim) string                              { return claim.SolicitudID }
func notificationOwner(notification domain.Notification) string         { return notification.CandidateID }
func validateCandidateDocument(document domain.CandidateDocument) error { return document.Validate() }
func validateClaim(claim domain.Claim) error                            { return claim.Validate() }
func validateNotification(notification domain.Notification) error       { return notification.Validate() }

func copyClaim(claim domain.Claim) domain.Claim {
	claim.Documents = copyCandidateDocuments(claim.Documents)
	return claim
}

func copyNotification(notification domain.Notification) domain.Notification {
	notification.Receipts = append([]domain.NotificationReceipt(nil), notification.Receipts...)
	return notification
}

func copyAuditEntries(entries []domain.AuditEntry) []domain.AuditEntry {
	return append([]domain.AuditEntry(nil), entries...)
}

func copyCandidateDocuments(documents []domain.CandidateDocument) []domain.CandidateDocument {
	copied := make([]domain.CandidateDocument, 0, len(documents))
	for _, document := range documents {
		copied = append(copied, copyCandidateDocument(document))
	}
	return copied
}

func copyCandidateDocument(document domain.CandidateDocument) domain.CandidateDocument {
	document.Evidence.Signatures = append([]domain.SignatureEvidence(nil), document.Evidence.Signatures...)
	return document
}
