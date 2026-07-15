package domain

import (
	"errors"
	"reflect"
	"testing"
)

func TestCalcularAutobaremoAppliesCapsAndOrdersDeterministically(t *testing.T) {
	ruleSet := mustBaremoRuleSet(t, []BaremoTieBreakRule{BaremoTieMayorExperiencia})
	merits := []Merit{
		{ID: "m3", Tipo: MeritTypeFormacionCurso, Datos: MeritData{Horas: 100}, Estado: MeritStateValidado},
		{ID: "m2", Tipo: MeritTypeExperienciaOtraCategoria, Datos: MeritData{Meses: 20}, Estado: MeritStateValidado},
		{ID: "m1", Tipo: MeritTypeExperienciaMismaCategoria, Datos: MeritData{Meses: 20}, Estado: MeritStateValidado},
	}

	got, err := CalcularAutobaremo(merits, ruleSet)
	if err != nil {
		t.Fatalf("CalcularAutobaremo() error = %v", err)
	}
	reversed := []Merit{merits[2], merits[1], merits[0]}
	gotAgain, err := CalcularAutobaremo(reversed, ruleSet)
	if err != nil {
		t.Fatalf("CalcularAutobaremo(reversed) error = %v", err)
	}
	if !reflect.DeepEqual(got, gotAgain) {
		t.Fatalf("result should be deterministic\nfirst: %#v\nsecond: %#v", got, gotAgain)
	}

	if got.TotalPoints != 5 {
		t.Fatalf("TotalPoints = %v, want 5", got.TotalPoints)
	}
	if got.SectionPoints[BaremoSectionExperiencia] != 3 {
		t.Fatalf("experience points = %v, want 3", got.SectionPoints[BaremoSectionExperiencia])
	}
	if got.SectionPoints[BaremoSectionFormacion] != 2 {
		t.Fatalf("training points = %v, want 2", got.SectionPoints[BaremoSectionFormacion])
	}

	wantDetails := []BaremoMeritScore{
		{MeritID: "m1", MeritType: MeritTypeExperienciaMismaCategoria, Section: BaremoSectionExperiencia, RawPoints: 4, AppliedPoints: 3, Capped: true},
		{MeritID: "m2", MeritType: MeritTypeExperienciaOtraCategoria, Section: BaremoSectionExperiencia, RawPoints: 2, AppliedPoints: 0, Capped: true},
		{MeritID: "m3", MeritType: MeritTypeFormacionCurso, Section: BaremoSectionFormacion, RawPoints: 2, AppliedPoints: 2, Capped: false},
	}
	if !reflect.DeepEqual(got.Details, wantDetails) {
		t.Fatalf("Details = %#v, want %#v", got.Details, wantDetails)
	}
}

func TestCalcularAutobaremoRejectsMeritWithoutRule(t *testing.T) {
	ruleSet, err := NewBaremoRuleSet(BaremoRuleSetConfig{
		ConvocatoriaID: "conv-1",
		Version:        "v1",
		MeritRules: []BaremoMeritRule{
			{MeritType: MeritTypeFormacionCurso, Section: BaremoSectionFormacion, Unit: BaremoUnitHoras, PointsPerUnit: 0.02},
		},
	})
	if err != nil {
		t.Fatalf("NewBaremoRuleSet() error = %v", err)
	}

	_, err = CalcularAutobaremo([]Merit{
		{ID: "m1", Tipo: MeritTypeExperienciaMismaCategoria, Datos: MeritData{Meses: 1}, Estado: MeritStateValidado},
	}, ruleSet)
	if !errors.Is(err, ErrBaremoMeritNoRule) {
		t.Fatalf("CalcularAutobaremo() error = %v, want %v", err, ErrBaremoMeritNoRule)
	}
}

func TestBaremoSeparaAutobaremoDeValoracionOficial(t *testing.T) {
	ruleSet := mustBaremoRuleSet(t, nil)
	meritos := []Merit{
		{ID: "borrador", Tipo: MeritTypeFormacionTitulo, Datos: MeritData{PuntosFijos: 1}, Estado: MeritStateBorrador},
		{ID: "presentado", Tipo: MeritTypeFormacionTitulo, Datos: MeritData{PuntosFijos: 1}, Estado: MeritStatePresentado},
		{ID: "validado", Tipo: MeritTypeFormacionTitulo, Datos: MeritData{PuntosFijos: 1}, Estado: MeritStateValidado},
		{ID: "rechazado", Tipo: MeritTypeFormacionTitulo, Datos: MeritData{PuntosFijos: 100}, Estado: MeritStateRechazado},
		{ID: "subsanacion", Tipo: MeritTypeFormacionTitulo, Datos: MeritData{PuntosFijos: 100}, Estado: MeritStateSubsanacion},
	}

	autobaremo, err := CalcularAutobaremo(meritos, ruleSet)
	if err != nil {
		t.Fatalf("CalcularAutobaremo() error = %v", err)
	}
	if autobaremo.TotalPoints != 3 || len(autobaremo.Details) != 3 {
		t.Fatalf("autobaremo puntua estados no admisibles: %+v", autobaremo)
	}

	oficial, err := CalcularBaremoOficial(meritos, ruleSet)
	if err != nil {
		t.Fatalf("CalcularBaremoOficial() error = %v", err)
	}
	if oficial.TotalPoints != 1 || len(oficial.Details) != 1 || oficial.Details[0].MeritID != "validado" {
		t.Fatalf("baremo oficial no se limita a meritos validados: %+v", oficial)
	}
}

func TestBaremoRuleSetValidation(t *testing.T) {
	tests := []struct {
		name   string
		config BaremoRuleSetConfig
	}{
		{
			name: "missing identity",
			config: BaremoRuleSetConfig{
				Version: "v1",
				MeritRules: []BaremoMeritRule{
					{MeritType: MeritTypeFormacionCurso, Section: BaremoSectionFormacion, Unit: BaremoUnitHoras, PointsPerUnit: 0.02},
				},
			},
		},
		{
			name: "duplicate merit rule",
			config: BaremoRuleSetConfig{
				ConvocatoriaID: "conv-1",
				Version:        "v1",
				MeritRules: []BaremoMeritRule{
					{MeritType: MeritTypeFormacionCurso, Section: BaremoSectionFormacion, Unit: BaremoUnitHoras, PointsPerUnit: 0.02},
					{MeritType: MeritTypeFormacionCurso, Section: BaremoSectionFormacion, Unit: BaremoUnitHoras, PointsPerUnit: 0.03},
				},
			},
		},
		{
			name: "lottery letter required",
			config: BaremoRuleSetConfig{
				ConvocatoriaID: "conv-1",
				Version:        "v1",
				MeritRules: []BaremoMeritRule{
					{MeritType: MeritTypeFormacionCurso, Section: BaremoSectionFormacion, Unit: BaremoUnitHoras, PointsPerUnit: 0.02},
				},
				TieBreakRules: []BaremoTieBreakRule{BaremoTieLetraSorteo},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewBaremoRuleSet(tt.config); !errors.Is(err, ErrBaremoRuleSetInvalid) {
				t.Fatalf("NewBaremoRuleSet() error = %v, want %v", err, ErrBaremoRuleSetInvalid)
			}
		})
	}
}

func TestDesempateUsesConfiguredRulesAndFallback(t *testing.T) {
	t.Run("greater experience wins before later rules", func(t *testing.T) {
		ruleSet := mustBaremoRuleSet(t, []BaremoTieBreakRule{BaremoTieMayorExperiencia, BaremoTieMayorFormacion})
		result, err := Desempate(
			tieCandidate("cand-a", "Lopez", 10, 1),
			tieCandidate("cand-b", "Navas", 8, 4),
			ruleSet,
		)
		if err != nil {
			t.Fatalf("Desempate() error = %v", err)
		}
		if result.WinnerID != "cand-a" || len(result.Decisions) != 1 {
			t.Fatalf("result = %#v, want cand-a after first decision", result)
		}
	})

	t.Run("lottery letter breaks equal section points", func(t *testing.T) {
		ruleSet := mustBaremoRuleSet(t, []BaremoTieBreakRule{BaremoTieMayorExperiencia, BaremoTieMayorFormacion, BaremoTieLetraSorteo})
		result, err := Desempate(
			tieCandidate("cand-a", "Lopez", 5, 2),
			tieCandidate("cand-b", "Navas", 5, 2),
			ruleSet,
		)
		if err != nil {
			t.Fatalf("Desempate() error = %v", err)
		}
		if result.WinnerID != "cand-b" || len(result.Decisions) != 3 {
			t.Fatalf("result = %#v, want cand-b by lottery letter", result)
		}
	})

	t.Run("candidate id fallback breaks unresolved tie", func(t *testing.T) {
		ruleSet := mustBaremoRuleSet(t, nil)
		result, err := Desempate(
			tieCandidate("cand-b", "Garcia", 5, 2),
			tieCandidate("cand-a", "Garcia", 5, 2),
			ruleSet,
		)
		if err != nil {
			t.Fatalf("Desempate() error = %v", err)
		}
		if result.WinnerID != "cand-a" || len(result.Decisions) != 1 {
			t.Fatalf("result = %#v, want lower candidate id fallback", result)
		}
	})
}

func mustBaremoRuleSet(t *testing.T, tieBreakRules []BaremoTieBreakRule) BaremoRuleSet {
	t.Helper()
	ruleSet, err := NewBaremoRuleSet(BaremoRuleSetConfig{
		ConvocatoriaID: "conv-1",
		Version:        "v1",
		SorteoLetra:    "M",
		MeritRules: []BaremoMeritRule{
			{MeritType: MeritTypeExperienciaMismaCategoria, Section: BaremoSectionExperiencia, Unit: BaremoUnitMeses, PointsPerUnit: 0.2},
			{MeritType: MeritTypeExperienciaOtraCategoria, Section: BaremoSectionExperiencia, Unit: BaremoUnitMeses, PointsPerUnit: 0.1},
			{MeritType: MeritTypeFormacionTitulo, Section: BaremoSectionFormacion, Unit: BaremoUnitPuntosDeclarado, PointsPerUnit: 1},
			{MeritType: MeritTypeFormacionCurso, Section: BaremoSectionFormacion, Unit: BaremoUnitHoras, PointsPerUnit: 0.02},
			{MeritType: MeritTypeOtros, Section: BaremoSectionOtros, Unit: BaremoUnitPuntosDeclarado, PointsPerUnit: 1},
		},
		SectionCaps: []BaremoSectionCap{
			{Section: BaremoSectionExperiencia, MaxPoints: 3},
			{Section: BaremoSectionFormacion, MaxPoints: 5},
		},
		TieBreakRules: tieBreakRules,
	})
	if err != nil {
		t.Fatalf("NewBaremoRuleSet() error = %v", err)
	}
	return ruleSet
}

func tieCandidate(id, sorteoKey string, experiencia, formacion float64) BaremoTieCandidate {
	return BaremoTieCandidate{
		CandidateID: id,
		SorteoKey:   sorteoKey,
		Result: BaremoResult{SectionPoints: map[BaremoSection]float64{
			BaremoSectionExperiencia: experiencia,
			BaremoSectionFormacion:   formacion,
		}},
	}
}
