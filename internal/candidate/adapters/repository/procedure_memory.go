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
	_ ports.ConvocatoriaRepository = (*ProcedureConvocatoriaRepository)(nil)
	_ ports.SolicitudRepository    = (*ProcedureSolicitudRepository)(nil)
)

type ProcedureMemoryStore struct {
	mu sync.RWMutex

	convocatorias             map[string]ports.ConvocatoriaRecord
	solicitudes               map[string]ports.SolicitudRecord
	solicitudesByConvocatoria map[string]map[string]struct{}
}

type ProcedureConvocatoriaRepository struct {
	store *ProcedureMemoryStore
}

type ProcedureSolicitudRepository struct {
	store *ProcedureMemoryStore
}

func NewProcedureMemoryStore() *ProcedureMemoryStore {
	return &ProcedureMemoryStore{
		convocatorias:             make(map[string]ports.ConvocatoriaRecord),
		solicitudes:               make(map[string]ports.SolicitudRecord),
		solicitudesByConvocatoria: make(map[string]map[string]struct{}),
	}
}

func NewProcedureConvocatoriaRepository(store *ProcedureMemoryStore) *ProcedureConvocatoriaRepository {
	if store == nil {
		store = NewProcedureMemoryStore()
	}
	return &ProcedureConvocatoriaRepository{store: store}
}

func NewProcedureSolicitudRepository(store *ProcedureMemoryStore) *ProcedureSolicitudRepository {
	if store == nil {
		store = NewProcedureMemoryStore()
	}
	return &ProcedureSolicitudRepository{store: store}
}

func NewProcedureRepositories() (*ProcedureConvocatoriaRepository, *ProcedureSolicitudRepository) {
	store := NewProcedureMemoryStore()
	return NewProcedureConvocatoriaRepository(store), NewProcedureSolicitudRepository(store)
}

func (r *ProcedureConvocatoriaRepository) Save(
	ctx context.Context,
	convocatoria ports.ConvocatoriaRecord,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := convocatoria.Convocatoria.Validate(); err != nil {
		return err
	}
	if err := convocatoria.RuleSet.Validate(); err != nil {
		return err
	}

	store := r.memoryStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureMapsLocked()
	store.convocatorias[convocatoria.Convocatoria.ID] = convocatoria
	return nil
}

func (r *ProcedureConvocatoriaRepository) GetByID(
	ctx context.Context,
	id string,
) (ports.ConvocatoriaRecord, error) {
	if err := ctx.Err(); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}

	store := r.memoryStore()
	store.mu.RLock()
	defer store.mu.RUnlock()
	record, ok := store.convocatorias[strings.TrimSpace(id)]
	if !ok {
		return ports.ConvocatoriaRecord{}, ports.ErrConvocatoriaNotFound
	}
	return record, nil
}

func (r *ProcedureSolicitudRepository) Save(
	ctx context.Context,
	solicitud ports.SolicitudRecord,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	solicitud.ID = strings.TrimSpace(solicitud.ID)
	solicitud.ConvocatoriaID = strings.TrimSpace(solicitud.ConvocatoriaID)
	solicitud.CandidateID = strings.TrimSpace(solicitud.CandidateID)
	if err := validateSolicitudRecord(solicitud); err != nil {
		return err
	}

	store := r.memoryStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureMapsLocked()

	if previous, ok := store.solicitudes[solicitud.ID]; ok && previous.ConvocatoriaID != solicitud.ConvocatoriaID {
		delete(store.solicitudesByConvocatoria[previous.ConvocatoriaID], solicitud.ID)
	}
	store.solicitudes[solicitud.ID] = copySolicitudRecord(solicitud)
	if store.solicitudesByConvocatoria[solicitud.ConvocatoriaID] == nil {
		store.solicitudesByConvocatoria[solicitud.ConvocatoriaID] = make(map[string]struct{})
	}
	store.solicitudesByConvocatoria[solicitud.ConvocatoriaID][solicitud.ID] = struct{}{}
	return nil
}

func (r *ProcedureSolicitudRepository) GetByID(
	ctx context.Context,
	id string,
) (ports.SolicitudRecord, error) {
	if err := ctx.Err(); err != nil {
		return ports.SolicitudRecord{}, err
	}

	store := r.memoryStore()
	store.mu.RLock()
	defer store.mu.RUnlock()
	record, ok := store.solicitudes[strings.TrimSpace(id)]
	if !ok {
		return ports.SolicitudRecord{}, ports.ErrSolicitudNotFound
	}
	return copySolicitudRecord(record), nil
}

func (r *ProcedureSolicitudRepository) ListByConvocatoria(
	ctx context.Context,
	convocatoriaID string,
) ([]ports.SolicitudRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	store := r.memoryStore()
	store.mu.RLock()
	defer store.mu.RUnlock()

	ids := sortedProcedureKeys(store.solicitudesByConvocatoria[strings.TrimSpace(convocatoriaID)])
	records := make([]ports.SolicitudRecord, 0, len(ids))
	for _, id := range ids {
		records = append(records, copySolicitudRecord(store.solicitudes[id]))
	}
	return records, nil
}

func (r *ProcedureConvocatoriaRepository) memoryStore() *ProcedureMemoryStore {
	if r == nil || r.store == nil {
		return NewProcedureMemoryStore()
	}
	return r.store
}

func (r *ProcedureSolicitudRepository) memoryStore() *ProcedureMemoryStore {
	if r == nil || r.store == nil {
		return NewProcedureMemoryStore()
	}
	return r.store
}

func (s *ProcedureMemoryStore) ensureMapsLocked() {
	if s.convocatorias == nil {
		s.convocatorias = make(map[string]ports.ConvocatoriaRecord)
	}
	if s.solicitudes == nil {
		s.solicitudes = make(map[string]ports.SolicitudRecord)
	}
	if s.solicitudesByConvocatoria == nil {
		s.solicitudesByConvocatoria = make(map[string]map[string]struct{})
	}
}

func validateSolicitudRecord(record ports.SolicitudRecord) error {
	switch {
	case strings.TrimSpace(record.ID) == "":
		return domain.ErrProcedureInvalid
	case strings.TrimSpace(record.ConvocatoriaID) == "":
		return domain.ErrProcedureInvalid
	case strings.TrimSpace(record.CandidateID) == "":
		return domain.ErrProcedureRanking
	case !record.Estado.IsValid():
		return domain.ErrProcedureInvalid
	default:
		return nil
	}
}

func copySolicitudRecord(record ports.SolicitudRecord) ports.SolicitudRecord {
	record.Result.SectionPoints = copySectionPoints(record.Result.SectionPoints)
	record.Result.Details = append([]domain.BaremoMeritScore(nil), record.Result.Details...)
	return record
}

func copySectionPoints(points map[domain.BaremoSection]float64) map[domain.BaremoSection]float64 {
	copied := make(map[domain.BaremoSection]float64, len(points))
	for section, value := range points {
		copied[section] = value
	}
	return copied
}

func sortedProcedureKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
