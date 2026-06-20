package application

import "vec-diputacion-granada/internal/candidate/domain"

type CandidateView struct {
	ID     string `json:"id"`
	DNI    string `json:"dni"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	CallID string `json:"call_id"`
}

type MeritView struct {
	ID     string            `json:"id"`
	Tipo   domain.MeritType  `json:"tipo"`
	Datos  MeritDataCommand  `json:"datos"`
	Estado domain.MeritState `json:"estado"`
}

type BaremoView struct {
	TotalPoints    float64            `json:"total_points"`
	SectionPoints  map[string]float64 `json:"section_points"`
	RuleSetID      string             `json:"rule_set_id"`
	RuleSetVersion string             `json:"rule_set_version"`
	Details        []BaremoDetailView `json:"details"`
}

type BaremoDetailView struct {
	MeritID       string  `json:"merit_id"`
	MeritType     string  `json:"merit_type"`
	Section       string  `json:"section"`
	RawPoints     float64 `json:"raw_points"`
	AppliedPoints float64 `json:"applied_points"`
	Capped        bool    `json:"capped"`
}

type ExpedienteView struct {
	Candidate CandidateView `json:"candidate"`
	Merits    []MeritView   `json:"merits"`
	Baremo    BaremoView    `json:"baremo"`
}
