package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
)

func TestCalcularAutobaremoUsesPortsAndPersistsResult(t *testing.T) {
	merits := &fakeMeritRepository{
		byCandidate: map[string][]domain.Merit{
			"cand-1": {
				{ID: "m2", Tipo: domain.MeritTypeFormacionCurso, Datos: domain.MeritData{Horas: 50}, Estado: domain.MeritStateValidado},
				{ID: "m1", Tipo: domain.MeritTypeExperienciaMismaCategoria, Datos: domain.MeritData{Meses: 10}, Estado: domain.MeritStateValidado},
			},
		},
	}
	results := newFakeBaremoResultRepository()
	useCase, err := NewBaremoUseCase(merits, results)
	if err != nil {
		t.Fatalf("NewBaremoUseCase() error = %v", err)
	}

	got, err := useCase.CalcularAutobaremo(context.Background(), " cand-1 ", testBaremoRuleSet(t))
	if err != nil {
		t.Fatalf("CalcularAutobaremo() error = %v", err)
	}

	if merits.lastListCandidateID != "cand-1" {
		t.Fatalf("ListByCandidate candidateID = %q, want %q", merits.lastListCandidateID, "cand-1")
	}
	if got.TotalPoints != 3 {
		t.Fatalf("TotalPoints = %v, want 3", got.TotalPoints)
	}
	if got.SectionPoints[domain.BaremoSectionExperiencia] != 2 {
		t.Fatalf("experience points = %v, want 2", got.SectionPoints[domain.BaremoSectionExperiencia])
	}
	if got.SectionPoints[domain.BaremoSectionFormacion] != 1 {
		t.Fatalf("training points = %v, want 1", got.SectionPoints[domain.BaremoSectionFormacion])
	}
	saved, ok := results.savedByCandidate["cand-1"]
	if !ok {
		t.Fatalf("result was not saved for candidate")
	}
	if !reflect.DeepEqual(saved, got) {
		t.Fatalf("saved result = %#v, want %#v", saved, got)
	}
}

func TestPresentarSolicitudTransitionsDraftMeritsBeforeScoring(t *testing.T) {
	merits := &fakeMeritRepository{
		byCandidate: map[string][]domain.Merit{
			"cand-1": {
				{ID: "m1", Tipo: domain.MeritTypeExperienciaMismaCategoria, Datos: domain.MeritData{Meses: 10}, Estado: domain.MeritStateBorrador},
				{ID: "m2", Tipo: domain.MeritTypeFormacionCurso, Datos: domain.MeritData{Horas: 50}, Estado: domain.MeritStateSubsanacion},
			},
		},
	}
	results := newFakeBaremoResultRepository()
	useCase, err := NewBaremoUseCase(merits, results)
	if err != nil {
		t.Fatalf("NewBaremoUseCase() error = %v", err)
	}

	got, err := useCase.PresentarSolicitud(context.Background(), "cand-1", testBaremoRuleSet(t))
	if err != nil {
		t.Fatalf("PresentarSolicitud() error = %v", err)
	}

	if got.TotalPoints != 3 {
		t.Fatalf("TotalPoints = %v, want 3", got.TotalPoints)
	}
	if len(merits.saved) != 2 {
		t.Fatalf("saved merits = %d, want 2", len(merits.saved))
	}
	for _, saved := range merits.saved {
		if saved.candidateID != "cand-1" {
			t.Fatalf("saved candidateID = %q, want %q", saved.candidateID, "cand-1")
		}
		if saved.merit.Estado != domain.MeritStatePresentado {
			t.Fatalf("saved merit state = %q, want %q", saved.merit.Estado, domain.MeritStatePresentado)
		}
	}
	if _, ok := results.savedByCandidate["cand-1"]; !ok {
		t.Fatalf("result was not saved for candidate")
	}
}

func TestBaremoUseCaseReturnsSaveResultError(t *testing.T) {
	wantErr := errors.New("store down")
	useCase, err := NewBaremoUseCase(
		&fakeMeritRepository{byCandidate: map[string][]domain.Merit{"cand-1": {validBaremoMerit()}}},
		&fakeBaremoResultRepository{err: wantErr},
	)
	if err != nil {
		t.Fatalf("NewBaremoUseCase() error = %v", err)
	}

	_, err = useCase.CalcularAutobaremo(context.Background(), "cand-1", testBaremoRuleSet(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("CalcularAutobaremo() error = %v, want %v", err, wantErr)
	}
}

type fakeMeritRepository struct {
	byCandidate         map[string][]domain.Merit
	lastListCandidateID string
	saved               []savedMerit
}

type savedMerit struct {
	candidateID string
	merit       domain.Merit
}

func (r *fakeMeritRepository) Save(ctx context.Context, candidateID string, merit domain.Merit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.saved = append(r.saved, savedMerit{candidateID: candidateID, merit: merit})
	r.byCandidate[candidateID] = upsertMerit(r.byCandidate[candidateID], merit)
	return nil
}

func (r *fakeMeritRepository) ListByCandidate(ctx context.Context, candidateID string) ([]domain.Merit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.lastListCandidateID = candidateID
	return append([]domain.Merit(nil), r.byCandidate[candidateID]...), nil
}

type fakeBaremoResultRepository struct {
	savedByCandidate map[string]domain.BaremoResult
	err              error
}

func newFakeBaremoResultRepository() *fakeBaremoResultRepository {
	return &fakeBaremoResultRepository{savedByCandidate: map[string]domain.BaremoResult{}}
}

func (r *fakeBaremoResultRepository) Save(
	ctx context.Context,
	candidateID string,
	result domain.BaremoResult,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.err != nil {
		return r.err
	}
	r.savedByCandidate[candidateID] = result
	return nil
}

func upsertMerit(merits []domain.Merit, merit domain.Merit) []domain.Merit {
	for i := range merits {
		if merits[i].ID == merit.ID {
			updated := append([]domain.Merit(nil), merits...)
			updated[i] = merit
			return updated
		}
	}
	return append(append([]domain.Merit(nil), merits...), merit)
}

func testBaremoRuleSet(t *testing.T) domain.BaremoRuleSet {
	t.Helper()
	ruleSet, err := domain.NewBaremoRuleSet(domain.BaremoRuleSetConfig{
		ConvocatoriaID: "conv-1",
		Version:        "v1",
		MeritRules: []domain.BaremoMeritRule{
			{MeritType: domain.MeritTypeExperienciaMismaCategoria, Section: domain.BaremoSectionExperiencia, Unit: domain.BaremoUnitMeses, PointsPerUnit: 0.2},
			{MeritType: domain.MeritTypeFormacionCurso, Section: domain.BaremoSectionFormacion, Unit: domain.BaremoUnitHoras, PointsPerUnit: 0.02},
		},
		SectionCaps: []domain.BaremoSectionCap{
			{Section: domain.BaremoSectionExperiencia, MaxPoints: 5},
			{Section: domain.BaremoSectionFormacion, MaxPoints: 5},
		},
	})
	if err != nil {
		t.Fatalf("NewBaremoRuleSet() error = %v", err)
	}
	return ruleSet
}

func validBaremoMerit() domain.Merit {
	return domain.Merit{
		ID:     "m1",
		Tipo:   domain.MeritTypeExperienciaMismaCategoria,
		Datos:  domain.MeritData{Meses: 10},
		Estado: domain.MeritStateValidado,
	}
}
