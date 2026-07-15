package usecases

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
)

func TestExplicitBaremoRuleSetIsValidPrototypeRuleSet(t *testing.T) {
	ruleSet, err := BaremoRuleSetFor("convocatoria-prueba", "v1")
	if err != nil {
		t.Fatalf("BaremoRuleSetFor() error = %v", err)
	}

	result, err := domain.CalcularAutobaremo(
		[]domain.Merit{
			{
				ID:     "m1",
				Tipo:   domain.MeritTypeExperienciaMismaCategoria,
				Datos:  domain.MeritData{Meses: 12},
				Estado: domain.MeritStateValidado,
			},
			{
				ID:     "m2",
				Tipo:   domain.MeritTypeFormacionCurso,
				Datos:  domain.MeritData{Horas: 20},
				Estado: domain.MeritStateValidado,
			},
		},
		ruleSet,
	)
	if err != nil {
		t.Fatalf("CalcularAutobaremo() error = %v", err)
	}
	if result.RuleSetID != "convocatoria-prueba" || result.RuleSetVersion != "v1" {
		t.Fatalf("rule set identity = %q/%q, want explicit call/v1", result.RuleSetID, result.RuleSetVersion)
	}
	if result.TotalPoints != 3.4 {
		t.Fatalf("TotalPoints = %v, want 3.4", result.TotalPoints)
	}
}

func TestBaremoRuleSetForValidatesIdentity(t *testing.T) {
	_, err := BaremoRuleSetFor(" ", "v1")
	if !errors.Is(err, domain.ErrBaremoRuleSetInvalid) {
		t.Fatalf("BaremoRuleSetFor() error = %v, want %v", err, domain.ErrBaremoRuleSetInvalid)
	}
}
