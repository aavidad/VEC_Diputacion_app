package handler

import (
	"vec-diputacion-granada/internal/candidate/application"
	"vec-diputacion-granada/internal/candidate/domain"
)

type Service = application.Service
type CandidateApplicationService = application.CandidateApplicationService
type CreateCandidateCommand = application.CreateCandidateCommand
type AddMeritCommand = application.AddMeritCommand
type MeritDataCommand = application.MeritDataCommand
type CandidateView = application.CandidateView
type MeritView = application.MeritView
type BaremoView = application.BaremoView
type BaremoDetailView = application.BaremoDetailView
type ExpedienteView = application.ExpedienteView

type ConvocatoriaView struct {
	ID      string                `json:"id"`
	Version string                `json:"version"`
	Estado  domain.ProcedureState `json:"estado"`
}

type ListadoView struct {
	ConvocatoriaID string            `json:"convocatoria_id"`
	Version        string            `json:"version"`
	Items          []ListadoItemView `json:"items"`
}

type ListadoItemView struct {
	ConvocatoriaID string                `json:"convocatoria_id"`
	SolicitudID    string                `json:"solicitud_id"`
	CandidateID    string                `json:"candidate_id"`
	Estado         domain.SolicitudState `json:"estado"`
	TotalPoints    float64               `json:"total_points"`
	SectionPoints  map[string]float64    `json:"section_points,omitempty"`
	RuleSetID      string                `json:"rule_set_id,omitempty"`
	RuleSetVersion string                `json:"rule_set_version,omitempty"`
	Details        []BaremoDetailView    `json:"details,omitempty"`
	Rank           int                   `json:"rank,omitempty"`
}

type ProcedureDemoView struct {
	Convocatoria ConvocatoriaView `json:"convocatoria"`
	Provisional  ListadoView      `json:"provisional"`
	Definitivo   ListadoView      `json:"definitivo"`
}

type responseEnvelope struct {
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}
