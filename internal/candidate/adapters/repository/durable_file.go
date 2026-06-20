package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

const durableFileSchemaVersion = 1

type DurableFileStore struct {
	path      string
	mu        sync.Mutex
	memory    *MemoryStore
	admin     *AdministrativeFlowMemoryStore
	baremo    *BaremoResultRepository
	procedure *ProcedureMemoryStore
}

type DurableCandidateRepository struct {
	*CandidateRepository
	store *DurableFileStore
}

type DurableMeritRepository struct {
	*MeritRepository
	store *DurableFileStore
}

type DurableBaremoResultRepository struct {
	*BaremoResultRepository
	store *DurableFileStore
}

type DurableCandidateDocumentRepository struct {
	*AdministrativeCandidateDocumentRepository
	store *DurableFileStore
}

type DurableClaimRepository struct {
	*AdministrativeClaimRepository
	store *DurableFileStore
}

type DurableNotificationRepository struct {
	*AdministrativeNotificationRepository
	store *DurableFileStore
}

type DurableAdministrativeAuditTrail struct {
	*AdministrativeAuditTrail
	store *DurableFileStore
}

type DurableProcedureConvocatoriaRepository struct {
	*ProcedureConvocatoriaRepository
	store *DurableFileStore
}

type DurableProcedureSolicitudRepository struct {
	*ProcedureSolicitudRepository
	store *DurableFileStore
}

type durableFileSnapshot struct {
	SchemaVersion int                         `json:"schema_version"`
	Candidates    []durableCandidateRecord    `json:"candidates"`
	Merits        []durableMeritRecord        `json:"merits"`
	BaremoResults []durableBaremoResultRecord `json:"baremo_results"`
	Convocatorias []durableConvocatoriaRecord `json:"convocatorias"`
	Solicitudes   []ports.SolicitudRecord     `json:"solicitudes"`
	Documents     []domain.CandidateDocument  `json:"documents"`
	Claims        []domain.Claim              `json:"claims"`
	Notifications []domain.Notification       `json:"notifications"`
	Audit         []durableAuditRecord        `json:"audit"`
}

type durableCandidateRecord struct {
	CallID    string           `json:"call_id"`
	Candidate domain.Candidate `json:"candidate"`
}

type durableMeritRecord struct {
	CandidateID string       `json:"candidate_id"`
	Merit       domain.Merit `json:"merit"`
}

type durableBaremoResultRecord struct {
	CandidateID string              `json:"candidate_id"`
	Result      domain.BaremoResult `json:"result"`
}

type durableConvocatoriaRecord struct {
	Convocatoria domain.Convocatoria        `json:"convocatoria"`
	RuleSet      domain.BaremoRuleSetConfig `json:"rule_set"`
}

type durableAuditRecord struct {
	Scope   string              `json:"scope"`
	Entries []domain.AuditEntry `json:"entries"`
}

func NewDurableFileStore(path string) (*DurableFileStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("durable file store: path is required")
	}
	store := &DurableFileStore{
		path:      path,
		memory:    NewMemoryStore(),
		admin:     NewAdministrativeFlowMemoryStore(),
		baremo:    NewBaremoResultRepository(),
		procedure: NewProcedureMemoryStore(),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *DurableFileStore) CandidateRepository() *DurableCandidateRepository {
	return &DurableCandidateRepository{CandidateRepository: NewCandidateRepository(s.memory), store: s}
}
func (s *DurableFileStore) MeritRepository() *DurableMeritRepository {
	return &DurableMeritRepository{MeritRepository: NewMeritRepository(s.memory), store: s}
}
func (s *DurableFileStore) BaremoResultRepository() *DurableBaremoResultRepository {
	return &DurableBaremoResultRepository{BaremoResultRepository: s.baremo, store: s}
}
func (s *DurableFileStore) CandidateDocumentRepository() *DurableCandidateDocumentRepository {
	return &DurableCandidateDocumentRepository{AdministrativeCandidateDocumentRepository: NewAdministrativeCandidateDocumentRepository(s.admin), store: s}
}
func (s *DurableFileStore) ClaimRepository() *DurableClaimRepository {
	return &DurableClaimRepository{AdministrativeClaimRepository: NewAdministrativeClaimRepository(s.admin), store: s}
}
func (s *DurableFileStore) NotificationRepository() *DurableNotificationRepository {
	return &DurableNotificationRepository{AdministrativeNotificationRepository: NewAdministrativeNotificationRepository(s.admin), store: s}
}
func (s *DurableFileStore) AdministrativeAuditTrail() *DurableAdministrativeAuditTrail {
	return &DurableAdministrativeAuditTrail{AdministrativeAuditTrail: NewAdministrativeAuditTrail(s.admin), store: s}
}
func (s *DurableFileStore) ProcedureConvocatoriaRepository() *DurableProcedureConvocatoriaRepository {
	return &DurableProcedureConvocatoriaRepository{ProcedureConvocatoriaRepository: NewProcedureConvocatoriaRepository(s.procedure), store: s}
}
func (s *DurableFileStore) ProcedureSolicitudRepository() *DurableProcedureSolicitudRepository {
	return &DurableProcedureSolicitudRepository{ProcedureSolicitudRepository: NewProcedureSolicitudRepository(s.procedure), store: s}
}

func (r *DurableCandidateRepository) Save(ctx context.Context, callID string, candidate domain.Candidate) error {
	return r.store.after(r.CandidateRepository.Save(ctx, callID, candidate))
}

func (r *DurableMeritRepository) Save(ctx context.Context, candidateID string, merit domain.Merit) error {
	return r.store.after(r.MeritRepository.Save(ctx, candidateID, merit))
}

func (r *DurableBaremoResultRepository) Save(ctx context.Context, candidateID string, result domain.BaremoResult) error {
	return r.store.after(r.BaremoResultRepository.Save(ctx, candidateID, result))
}

func (r *DurableCandidateDocumentRepository) Save(ctx context.Context, document domain.CandidateDocument) error {
	return r.store.after(r.AdministrativeCandidateDocumentRepository.Save(ctx, document))
}

func (r *DurableClaimRepository) Save(ctx context.Context, claim domain.Claim) error {
	return r.store.after(r.AdministrativeClaimRepository.Save(ctx, claim))
}

func (r *DurableNotificationRepository) Save(ctx context.Context, notification domain.Notification) error {
	return r.store.after(r.AdministrativeNotificationRepository.Save(ctx, notification))
}

func (r *DurableAdministrativeAuditTrail) Append(ctx context.Context, scope string, envelope domain.AuditEnvelope) (domain.AuditEntry, error) {
	entry, err := r.AdministrativeAuditTrail.Append(ctx, scope, envelope)
	if err != nil {
		return domain.AuditEntry{}, err
	}
	if err := r.store.persist(); err != nil {
		return entry, err
	}
	return entry, nil
}
func (r *DurableAdministrativeAuditTrail) ListByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error) {
	return r.AdministrativeAuditTrail.ListByScope(ctx, scope)
}

func (r *DurableProcedureConvocatoriaRepository) Save(ctx context.Context, convocatoria ports.ConvocatoriaRecord) error {
	return r.store.after(r.ProcedureConvocatoriaRepository.Save(ctx, convocatoria))
}

func (r *DurableProcedureConvocatoriaRepository) GetByID(ctx context.Context, id string) (ports.ConvocatoriaRecord, error) {
	return r.ProcedureConvocatoriaRepository.GetByID(ctx, id)
}

func (r *DurableProcedureSolicitudRepository) Save(ctx context.Context, solicitud ports.SolicitudRecord) error {
	return r.store.after(r.ProcedureSolicitudRepository.Save(ctx, solicitud))
}

func (r *DurableProcedureSolicitudRepository) GetByID(ctx context.Context, id string) (ports.SolicitudRecord, error) {
	return r.ProcedureSolicitudRepository.GetByID(ctx, id)
}

func (r *DurableProcedureSolicitudRepository) ListByConvocatoria(ctx context.Context, convocatoriaID string) ([]ports.SolicitudRecord, error) {
	return r.ProcedureSolicitudRepository.ListByConvocatoria(ctx, convocatoriaID)
}

func (s *DurableFileStore) after(err error) error {
	if err != nil {
		return err
	}
	return s.persist()
}

func (s *DurableFileStore) load() error {
	snapshot, err := readDurableSnapshot(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		if backup, backupErr := readDurableSnapshot(durableBackupPath(s.path)); backupErr == nil {
			return s.applySnapshot(backup)
		}
		return err
	}
	return s.applySnapshot(snapshot)
}

func readDurableSnapshot(path string) (durableFileSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return durableFileSnapshot{}, fmt.Errorf("read durable file %q: %w", path, err)
	}
	var snapshot durableFileSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return durableFileSnapshot{}, fmt.Errorf("decode durable file %q: %w", path, err)
	}
	if snapshot.SchemaVersion != durableFileSchemaVersion {
		return durableFileSnapshot{}, fmt.Errorf("durable file schema %d is unsupported", snapshot.SchemaVersion)
	}
	return snapshot, nil
}

func (s *DurableFileStore) persist() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.snapshot(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode durable file: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create durable dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create durable temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write durable temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync durable temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close durable temp file: %w", err)
	}
	if err := s.copyBackup(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace durable file: %w", err)
	}
	return syncDir(dir)
}

func (s *DurableFileStore) copyBackup() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read durable backup source: %w", err)
	}
	if !json.Valid(data) {
		return nil
	}
	if err := os.WriteFile(durableBackupPath(s.path), data, 0o600); err != nil {
		return fmt.Errorf("write durable backup: %w", err)
	}
	return nil
}

func durableBackupPath(path string) string {
	return path + ".bak"
}
