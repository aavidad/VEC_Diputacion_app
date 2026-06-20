package handler

import (
	"context"

	"vec-diputacion-granada/internal/candidate/ports"
	"vec-diputacion-granada/internal/candidate/usecases"
)

type recordingDemoRunner struct {
	view  ProcedureDemoView
	err   error
	calls int
}

func (r *recordingDemoRunner) Run(ctx context.Context) (ProcedureDemoView, error) {
	r.calls++
	if r.err != nil {
		return ProcedureDemoView{}, r.err
	}
	return r.view, ctx.Err()
}

func newTestProcedureUseCase() (*usecases.ProcedureUseCase, error) {
	return usecases.NewProcedureUseCase(
		&memoryConvocatoriaRepository{records: map[string]ports.ConvocatoriaRecord{}},
		&memorySolicitudRepository{records: map[string]ports.SolicitudRecord{}},
	)
}

type memoryConvocatoriaRepository struct {
	records map[string]ports.ConvocatoriaRecord
}

func (r *memoryConvocatoriaRepository) Save(_ context.Context, record ports.ConvocatoriaRecord) error {
	r.records[record.Convocatoria.ID] = record
	return nil
}

func (r *memoryConvocatoriaRepository) GetByID(_ context.Context, id string) (ports.ConvocatoriaRecord, error) {
	record, ok := r.records[id]
	if !ok {
		return ports.ConvocatoriaRecord{}, ports.ErrConvocatoriaNotFound
	}
	return record, nil
}

type memorySolicitudRepository struct {
	records map[string]ports.SolicitudRecord
}

func (r *memorySolicitudRepository) Save(_ context.Context, record ports.SolicitudRecord) error {
	r.records[record.ID] = record
	return nil
}

func (r *memorySolicitudRepository) GetByID(_ context.Context, id string) (ports.SolicitudRecord, error) {
	record, ok := r.records[id]
	if !ok {
		return ports.SolicitudRecord{}, ports.ErrSolicitudNotFound
	}
	return record, nil
}

func (r *memorySolicitudRepository) ListByConvocatoria(_ context.Context, convocatoriaID string) ([]ports.SolicitudRecord, error) {
	result := make([]ports.SolicitudRecord, 0, len(r.records))
	for _, record := range r.records {
		if record.ConvocatoriaID == convocatoriaID {
			result = append(result, record)
		}
	}
	return result, nil
}
