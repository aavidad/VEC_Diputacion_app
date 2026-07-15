package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestCandidateApplicationServiceRequiresExplicitCall(t *testing.T) {
	candidates := newFakeCandidateRepository()
	service := mustService(t, candidates, newFakeMeritRepository(), fakeBaremoCalculator{})

	_, err := service.CreateCandidate(context.Background(), CreateCandidateCommand{
		ID: "cand-1", DNI: "12345678A", Nombre: "Ana Perez", Email: "ana@example.test",
	})
	if !errors.Is(err, ErrCallIDRequired) {
		t.Fatalf("CreateCandidate() error = %v, want %v", err, ErrCallIDRequired)
	}
	if len(candidates.byID) != 0 {
		t.Fatal("a candidate without an explicit call was persisted")
	}
}

func TestCandidateApplicationServiceAddsMeritAndExportsExpediente(t *testing.T) {
	candidates := newFakeCandidateRepository()
	merits := newFakeMeritRepository()
	service := mustService(t, candidates, merits, fakeBaremoCalculator{})

	if _, err := service.CreateCandidate(context.Background(), CreateCandidateCommand{
		ID: "cand-1", DNI: "12345678A", Nombre: "Ana Perez", Email: "ana@example.test",
		CallID: "conv-1",
	}); err != nil {
		t.Fatalf("CreateCandidate() error = %v", err)
	}
	merit, err := service.AddMerit(context.Background(), " cand-1 ", AddMeritCommand{
		ID: "merit-1", Tipo: domain.MeritTypeExperienciaMismaCategoria,
		Datos: MeritDataCommand{Meses: 12},
	})
	if err != nil {
		t.Fatalf("AddMerit() error = %v", err)
	}
	if merit.ID != "merit-1" || merits.savedCandidateID != "cand-1" {
		t.Fatalf("merit = %#v, saved candidate = %q", merit, merits.savedCandidateID)
	}
	if merit.Estado != domain.MeritStateBorrador {
		t.Fatalf("estado del merito ciudadano = %q, esperado %q", merit.Estado, domain.MeritStateBorrador)
	}
	expediente, err := service.ExportExpediente(context.Background(), "cand-1")
	if err != nil {
		t.Fatalf("ExportExpediente() error = %v", err)
	}
	if expediente.Candidate.ID != "cand-1" || len(expediente.Merits) != 1 || expediente.Baremo.TotalPoints != 2.4 {
		t.Fatalf("expediente = %#v, want candidate, merit and baremo", expediente)
	}
}

func TestCandidateApplicationServiceRehydratesExactCandidateCall(t *testing.T) {
	candidates := newFakeCandidateRepository()
	service := mustService(t, candidates, newFakeMeritRepository(), fakeBaremoCalculator{})

	if _, err := service.CreateCandidate(context.Background(), CreateCandidateCommand{
		ID: "cand-1", DNI: "12345678A", Nombre: "Ana Perez", Email: "ana@example.test",
		CallID: "conv-1",
	}); err != nil {
		t.Fatalf("CreateCandidate() error = %v", err)
	}
	// Simula un reinicio: la vinculacion se recupera del repositorio, no de
	// memoria volatil del servicio ni de una convocatoria predeterminada.
	service = mustService(t, candidates, newFakeMeritRepository(), fakeBaremoCalculator{})
	expediente, err := service.ExportExpediente(context.Background(), "cand-1")
	if err != nil {
		t.Fatalf("ExportExpediente() error = %v", err)
	}
	if expediente.Candidate.CallID != "conv-1" {
		t.Fatalf("expediente call = %q, want conv-1", expediente.Candidate.CallID)
	}
}

func TestCandidateApplicationServiceRejectsUnconfiguredCall(t *testing.T) {
	service := mustService(t, newFakeCandidateRepository(), newFakeMeritRepository(), fakeBaremoCalculator{})
	for _, callID := range []string{"conv-2", " conv-1 ", "conv-*"} {
		_, err := service.CreateCandidate(context.Background(), CreateCandidateCommand{
			ID: "cand-" + callID, DNI: "12345678A", Nombre: "Ana Perez", Email: "ana@example.test", CallID: callID,
		})
		if !errors.Is(err, ErrCallNotConfigured) {
			t.Fatalf("CreateCandidate(call=%q) error = %v, want %v", callID, err, ErrCallNotConfigured)
		}
	}
}

func TestCandidateApplicationServiceRejectsLostOrDifferentCallBinding(t *testing.T) {
	candidates := newFakeCandidateRepository()
	candidates.byID["cand-1"] = domain.Candidate{ID: "cand-1", DNI: "12345678A", Nombre: "Ana Perez", Email: "ana@example.test"}
	service := mustService(t, candidates, newFakeMeritRepository(), fakeBaremoCalculator{})

	for _, callID := range []string{"", "conv-2", "conv-*"} {
		candidates.callByID["cand-1"] = callID
		if _, err := service.ExportExpediente(context.Background(), "cand-1"); !errors.Is(err, ErrCandidateCallBindingInvalid) {
			t.Fatalf("ExportExpediente(call=%q) error = %v, want %v", callID, err, ErrCandidateCallBindingInvalid)
		}
	}
}

func TestCandidateApplicationServiceRequiresExistingCandidateForMerit(t *testing.T) {
	service := mustService(t, newFakeCandidateRepository(), newFakeMeritRepository(), fakeBaremoCalculator{})

	_, err := service.AddMerit(context.Background(), "missing", AddMeritCommand{
		ID: "merit-1", Tipo: domain.MeritTypeExperienciaMismaCategoria, Datos: MeritDataCommand{Meses: 1},
	})
	if !errors.Is(err, ports.ErrCandidateNotFound) {
		t.Fatalf("AddMerit() error = %v, want %v", err, ports.ErrCandidateNotFound)
	}
}

func TestCandidateApplicationServiceImpideAutovalidarMerito(t *testing.T) {
	candidates := newFakeCandidateRepository()
	service := mustService(t, candidates, newFakeMeritRepository(), fakeBaremoCalculator{})
	if _, err := service.CreateCandidate(context.Background(), CreateCandidateCommand{
		ID: "cand-1", DNI: "12345678A", Nombre: "Ana Perez", Email: "ana@example.test",
		CallID: "conv-1",
	}); err != nil {
		t.Fatalf("CreateCandidate() error = %v", err)
	}

	for _, estado := range []domain.MeritState{
		domain.MeritStatePresentado,
		domain.MeritStateValidado,
		domain.MeritStateRechazado,
		domain.MeritStateSubsanacion,
	} {
		_, err := service.AddMerit(context.Background(), "cand-1", AddMeritCommand{
			ID: "merit-" + string(estado), Tipo: domain.MeritTypeFormacionTitulo,
			Datos: MeritDataCommand{PuntosFijos: 1}, Estado: estado,
		})
		if !errors.Is(err, domain.ErrMeritTransition) {
			t.Fatalf("AddMerit(estado=%q) error = %v, esperado %v", estado, err, domain.ErrMeritTransition)
		}
	}
}

func mustService(
	t *testing.T,
	candidates *fakeCandidateRepository,
	merits *fakeMeritRepository,
	baremo fakeBaremoCalculator,
) *CandidateApplicationService {
	t.Helper()
	service, err := NewCandidateApplicationService(candidates, merits, baremo, testRuleSet(t))
	if err != nil {
		t.Fatalf("NewCandidateApplicationService() error = %v", err)
	}
	return service
}

func testRuleSet(t *testing.T) domain.BaremoRuleSet {
	t.Helper()
	ruleSet, err := domain.NewBaremoRuleSet(domain.BaremoRuleSetConfig{
		ConvocatoriaID: "conv-1", Version: "v1", SorteoLetra: "A",
		MeritRules: []domain.BaremoMeritRule{
			{
				MeritType:     domain.MeritTypeExperienciaMismaCategoria,
				Section:       domain.BaremoSectionExperiencia,
				Unit:          domain.BaremoUnitMeses,
				PointsPerUnit: 0.2,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewBaremoRuleSet() error = %v", err)
	}
	return ruleSet
}

type fakeCandidateRepository struct {
	byID     map[string]domain.Candidate
	callByID map[string]string
}

func newFakeCandidateRepository() *fakeCandidateRepository {
	return &fakeCandidateRepository{
		byID:     map[string]domain.Candidate{},
		callByID: map[string]string{},
	}
}

func (r *fakeCandidateRepository) Save(_ context.Context, callID string, candidate domain.Candidate) error {
	if err := candidate.Validate(); err != nil {
		return err
	}
	r.byID[candidate.ID] = candidate
	r.callByID[candidate.ID] = callID
	return nil
}

func (r *fakeCandidateRepository) GetByID(_ context.Context, id string) (domain.Candidate, string, error) {
	candidate, ok := r.byID[id]
	if !ok {
		return domain.Candidate{}, "", ports.ErrCandidateNotFound
	}
	return candidate, r.callByID[id], nil
}

func (r *fakeCandidateRepository) ListByCall(_ context.Context, callID string) ([]domain.Candidate, error) {
	var result []domain.Candidate
	for id, candidate := range r.byID {
		if r.callByID[id] == callID {
			result = append(result, candidate)
		}
	}
	return result, nil
}

type fakeMeritRepository struct {
	byCandidate      map[string][]domain.Merit
	savedCandidateID string
}

func newFakeMeritRepository() *fakeMeritRepository {
	return &fakeMeritRepository{byCandidate: map[string][]domain.Merit{}}
}

func (r *fakeMeritRepository) Save(_ context.Context, candidateID string, merit domain.Merit) error {
	if err := merit.Validate(); err != nil {
		return err
	}
	r.savedCandidateID = candidateID
	r.byCandidate[candidateID] = append(r.byCandidate[candidateID], merit)
	return nil
}

func (r *fakeMeritRepository) ListByCandidate(_ context.Context, candidateID string) ([]domain.Merit, error) {
	return append([]domain.Merit(nil), r.byCandidate[candidateID]...), nil
}

type fakeBaremoCalculator struct{}

func (fakeBaremoCalculator) CalcularAutobaremo(
	_ context.Context,
	_ string,
	_ domain.BaremoRuleSet,
) (domain.BaremoResult, error) {
	return domain.BaremoResult{
		RuleSetID: "conv-1", RuleSetVersion: "v1",
		TotalPoints: 2.4,
		SectionPoints: map[domain.BaremoSection]float64{
			domain.BaremoSectionExperiencia: 2.4,
		},
		Details: []domain.BaremoMeritScore{
			{
				MeritID: "merit-1", MeritType: domain.MeritTypeExperienciaMismaCategoria,
				Section: domain.BaremoSectionExperiencia, RawPoints: 2.4, AppliedPoints: 2.4,
			},
		},
	}, nil
}
