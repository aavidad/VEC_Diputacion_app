package repository

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
)

func TestBaremoResultRepositorySavesCopyByCandidate(t *testing.T) {
	ctx := context.Background()
	repository := NewBaremoResultRepository()
	result := baremoResultFixture()

	if err := repository.Save(ctx, " cand-1 ", result); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	result.SectionPoints[domain.BaremoSectionExperiencia] = 99
	result.Details[0].AppliedPoints = 99

	got, ok, err := repository.GetByCandidate(ctx, "cand-1")
	if err != nil {
		t.Fatalf("GetByCandidate() error = %v", err)
	}
	if !ok {
		t.Fatalf("GetByCandidate() ok = false, want true")
	}
	if got.SectionPoints[domain.BaremoSectionExperiencia] != 2 {
		t.Fatalf("experience points = %v, want 2", got.SectionPoints[domain.BaremoSectionExperiencia])
	}
	if got.Details[0].AppliedPoints != 2 {
		t.Fatalf("detail points = %v, want 2", got.Details[0].AppliedPoints)
	}
}

func TestBaremoResultRepositoryValidatesCandidateID(t *testing.T) {
	err := NewBaremoResultRepository().Save(context.Background(), " ", baremoResultFixture())
	if !errors.Is(err, domain.ErrCandidateIDRequired) {
		t.Fatalf("Save() error = %v, want %v", err, domain.ErrCandidateIDRequired)
	}
}

func baremoResultFixture() domain.BaremoResult {
	return domain.BaremoResult{
		TotalPoints:    2,
		RuleSetID:      "conv-1",
		RuleSetVersion: "v1",
		SectionPoints:  map[domain.BaremoSection]float64{domain.BaremoSectionExperiencia: 2},
		Details: []domain.BaremoMeritScore{
			{
				MeritID:       "m1",
				MeritType:     domain.MeritTypeExperienciaMismaCategoria,
				Section:       domain.BaremoSectionExperiencia,
				RawPoints:     2,
				AppliedPoints: 2,
			},
		},
	}
}
