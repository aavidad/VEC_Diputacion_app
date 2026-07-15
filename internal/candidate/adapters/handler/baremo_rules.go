package handler

import (
	"strings"

	"vec-diputacion-granada/internal/candidate/domain"
)

func ruleSetFor(convocatoriaID string, version string) (domain.BaremoRuleSet, error) {
	return domain.NewBaremoRuleSet(domain.BaremoRuleSetConfig{
		ConvocatoriaID: strings.TrimSpace(convocatoriaID),
		Version:        strings.TrimSpace(version),
		SorteoLetra:    "A",
		MeritRules: []domain.BaremoMeritRule{
			{
				MeritType:     domain.MeritTypeExperienciaMismaCategoria,
				Section:       domain.BaremoSectionExperiencia,
				Unit:          domain.BaremoUnitMeses,
				PointsPerUnit: 0.2,
			},
			{
				MeritType:     domain.MeritTypeExperienciaOtraCategoria,
				Section:       domain.BaremoSectionExperiencia,
				Unit:          domain.BaremoUnitMeses,
				PointsPerUnit: 0.1,
			},
			{
				MeritType:     domain.MeritTypeFormacionTitulo,
				Section:       domain.BaremoSectionFormacion,
				Unit:          domain.BaremoUnitPuntosDeclarado,
				PointsPerUnit: 1,
			},
			{
				MeritType:     domain.MeritTypeFormacionCurso,
				Section:       domain.BaremoSectionFormacion,
				Unit:          domain.BaremoUnitHoras,
				PointsPerUnit: 0.05,
			},
			{
				MeritType:     domain.MeritTypeOtros,
				Section:       domain.BaremoSectionOtros,
				Unit:          domain.BaremoUnitPuntosDeclarado,
				PointsPerUnit: 1,
			},
		},
		SectionCaps: []domain.BaremoSectionCap{
			{Section: domain.BaremoSectionExperiencia, MaxPoints: 50},
			{Section: domain.BaremoSectionFormacion, MaxPoints: 30},
		},
		TieBreakRules: []domain.BaremoTieBreakRule{
			domain.BaremoTieMayorExperiencia,
			domain.BaremoTieMayorFormacion,
			domain.BaremoTieLetraSorteo,
		},
	})
}
