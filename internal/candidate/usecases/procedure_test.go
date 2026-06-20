package usecases

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestProcedureUseCaseRunsRecommendedFlow(t *testing.T) {
	ctx := context.Background()
	convocatorias := newFakeConvocatoriaRepository()
	solicitudes := newFakeSolicitudRepository()
	useCase, err := NewProcedureUseCase(convocatorias, solicitudes)
	if err != nil {
		t.Fatalf("NewProcedureUseCase() error = %v", err)
	}

	convocatoria, err := useCase.CrearConvocatoria(ctx, CrearConvocatoriaCommand{
		ID: " conv-1 ", Version: " v1 ", RuleSet: testBaremoRuleSet(t),
	})
	if err != nil {
		t.Fatalf("CrearConvocatoria() error = %v", err)
	}
	if convocatoria.Convocatoria.ID != "conv-1" || convocatoria.Convocatoria.Version != "v1" {
		t.Fatalf("convocatoria = %#v, want trimmed v1", convocatoria.Convocatoria)
	}

	registerSolicitud(t, ctx, useCase, "sol-2", "cand-b", "Navas", 10, 50)
	registerSolicitud(t, ctx, useCase, "sol-1", "cand-a", "Lopez", 20, 25)

	provisional, err := useCase.PublicarListadoProvisional(ctx, "conv-1", map[string]bool{
		"sol-1": true,
		"sol-2": true,
	})
	if err != nil {
		t.Fatalf("PublicarListadoProvisional() error = %v", err)
	}
	if got := itemKeys(provisional.Items); !reflect.DeepEqual(got, []string{"sol-1:cand-a:0", "sol-2:cand-b:0"}) {
		t.Fatalf("provisional items = %#v, want reproducible candidate order", got)
	}

	definitivo, err := useCase.PublicarListadoDefinitivo(ctx, "conv-1", map[string]bool{
		"sol-1": true,
		"sol-2": true,
	})
	if err != nil {
		t.Fatalf("PublicarListadoDefinitivo() error = %v", err)
	}
	if got := itemKeys(definitivo.Items); !reflect.DeepEqual(got, []string{"sol-1:cand-a:1", "sol-2:cand-b:2"}) {
		t.Fatalf("definitive items = %#v, want ranked candidate order", got)
	}
	if definitivo.Version != "v1" || definitivo.Items[0].Result.RuleSetVersion != "v1" {
		t.Fatalf("frozen baremo version = listing %q result %q, want v1", definitivo.Version, definitivo.Items[0].Result.RuleSetVersion)
	}
}

func TestProcedureUseCaseRejectsDefinitiveBeforeProvisional(t *testing.T) {
	ctx := context.Background()
	useCase, err := NewProcedureUseCase(newFakeConvocatoriaRepository(), newFakeSolicitudRepository())
	if err != nil {
		t.Fatalf("NewProcedureUseCase() error = %v", err)
	}
	if _, err := useCase.CrearConvocatoria(ctx, CrearConvocatoriaCommand{ID: "conv-1", Version: "v1", RuleSet: testBaremoRuleSet(t)}); err != nil {
		t.Fatalf("CrearConvocatoria() error = %v", err)
	}
	registerSolicitud(t, ctx, useCase, "sol-1", "cand-a", "Lopez", 12, 0)
	if _, err := useCase.PublicarListadoDefinitivo(ctx, "conv-1", map[string]bool{"sol-1": true}); err == nil {
		t.Fatalf("PublicarListadoDefinitivo() error = nil, want transition error")
	}
}

func TestProcedureUseCaseNormalizesNilContextBeforeCallingPorts(t *testing.T) {
	useCase, err := NewProcedureUseCase(newFakeConvocatoriaRepository(), newFakeSolicitudRepository())
	if err != nil {
		t.Fatalf("NewProcedureUseCase() error = %v", err)
	}
	if _, err := useCase.CrearConvocatoria(nil, CrearConvocatoriaCommand{ID: "conv-1", Version: "v1", RuleSet: testBaremoRuleSet(t)}); err != nil {
		t.Fatalf("CrearConvocatoria(nil context) error = %v", err)
	}
	registerSolicitud(t, nil, useCase, "sol-1", "cand-a", "Lopez", 12, 0)
	if _, err := useCase.PublicarListadoProvisional(nil, "conv-1", map[string]bool{"sol-1": true}); err != nil {
		t.Fatalf("PublicarListadoProvisional(nil context) error = %v", err)
	}
	if _, err := useCase.PublicarListadoDefinitivo(nil, "conv-1", map[string]bool{"sol-1": true}); err != nil {
		t.Fatalf("PublicarListadoDefinitivo(nil context) error = %v", err)
	}
}

func registerSolicitud(
	t *testing.T,
	ctx context.Context,
	useCase *ProcedureUseCase,
	solicitudID string,
	candidateID string,
	sorteoKey string,
	meses int,
	horas int,
) {
	t.Helper()
	_, err := useCase.RegistrarSolicitud(ctx, RegistrarSolicitudCommand{
		ID: solicitudID, ConvocatoriaID: "conv-1", CandidateID: candidateID, SorteoKey: sorteoKey,
		Merits: []domain.Merit{
			{ID: solicitudID + "-exp", Tipo: domain.MeritTypeExperienciaMismaCategoria, Datos: domain.MeritData{Meses: meses}, Estado: domain.MeritStateValidado},
			{ID: solicitudID + "-cur", Tipo: domain.MeritTypeFormacionCurso, Datos: domain.MeritData{Horas: horas}, Estado: domain.MeritStateValidado},
		},
	})
	if err != nil {
		t.Fatalf("RegistrarSolicitud(%s) error = %v", solicitudID, err)
	}
}

func itemKeys(items []ListadoItem) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.SolicitudID+":"+item.CandidateID+":"+strconv.Itoa(item.Rank))
	}
	return keys
}

type fakeConvocatoriaRepository struct {
	byID map[string]ports.ConvocatoriaRecord
}

func newFakeConvocatoriaRepository() *fakeConvocatoriaRepository {
	return &fakeConvocatoriaRepository{byID: map[string]ports.ConvocatoriaRecord{}}
}

func (r *fakeConvocatoriaRepository) Save(ctx context.Context, record ports.ConvocatoriaRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.byID[record.Convocatoria.ID] = record
	return nil
}

func (r *fakeConvocatoriaRepository) GetByID(ctx context.Context, id string) (ports.ConvocatoriaRecord, error) {
	if err := ctx.Err(); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	record, ok := r.byID[id]
	if !ok {
		return ports.ConvocatoriaRecord{}, ports.ErrConvocatoriaNotFound
	}
	return record, nil
}

type fakeSolicitudRepository struct {
	byID map[string]ports.SolicitudRecord
}

func newFakeSolicitudRepository() *fakeSolicitudRepository {
	return &fakeSolicitudRepository{byID: map[string]ports.SolicitudRecord{}}
}

func (r *fakeSolicitudRepository) Save(ctx context.Context, record ports.SolicitudRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.byID[record.ID] = record
	return nil
}

func (r *fakeSolicitudRepository) GetByID(ctx context.Context, id string) (ports.SolicitudRecord, error) {
	if err := ctx.Err(); err != nil {
		return ports.SolicitudRecord{}, err
	}
	record, ok := r.byID[id]
	if !ok {
		return ports.SolicitudRecord{}, ports.ErrSolicitudNotFound
	}
	return record, nil
}

func (r *fakeSolicitudRepository) ListByConvocatoria(ctx context.Context, convocatoriaID string) ([]ports.SolicitudRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	records := make([]ports.SolicitudRecord, 0, len(r.byID))
	for _, record := range r.byID {
		if record.ConvocatoriaID == convocatoriaID {
			records = append(records, record)
		}
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].ID > records[j].ID })
	return records, nil
}
