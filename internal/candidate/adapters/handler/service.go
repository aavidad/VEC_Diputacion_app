package handler

import (
	"vec-diputacion-granada/internal/candidate/application"
	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

type baremoCalculator = application.BaremoCalculator

func NewCandidateApplicationService(
	candidates ports.CandidateRepository,
	merits ports.MeritRepository,
	baremo baremoCalculator,
	ruleSet domain.BaremoRuleSet,
) (*CandidateApplicationService, error) {
	return application.NewCandidateApplicationService(candidates, merits, baremo, ruleSet)
}
