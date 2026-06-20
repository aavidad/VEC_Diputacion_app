package application

import (
	"context"
	"errors"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

const DefaultCallID = "convocatoria-default"

var ErrServiceDependenciesRequired = errors.New("candidate application service: repositories and baremo usecase are required")

type Service interface {
	CreateCandidate(context.Context, CreateCandidateCommand) (CandidateView, error)
	AddMerit(context.Context, string, AddMeritCommand) (MeritView, error)
	CalculateBaremo(context.Context, string) (BaremoView, error)
	ExportExpediente(context.Context, string) (ExpedienteView, error)
}

type BaremoCalculator interface {
	CalcularAutobaremo(context.Context, string, domain.BaremoRuleSet) (domain.BaremoResult, error)
}

type CandidateApplicationService struct {
	candidates ports.CandidateRepository
	merits     ports.MeritRepository
	baremo     BaremoCalculator
	ruleSet    domain.BaremoRuleSet

	mu                  sync.RWMutex
	callIDByCandidateID map[string]string
}

func NewCandidateApplicationService(
	candidates ports.CandidateRepository,
	merits ports.MeritRepository,
	baremo BaremoCalculator,
	ruleSet domain.BaremoRuleSet,
) (*CandidateApplicationService, error) {
	if candidates == nil || merits == nil || baremo == nil {
		return nil, ErrServiceDependenciesRequired
	}
	if err := ruleSet.Validate(); err != nil {
		return nil, err
	}
	return &CandidateApplicationService{
		candidates:          candidates,
		merits:              merits,
		baremo:              baremo,
		ruleSet:             ruleSet,
		callIDByCandidateID: make(map[string]string),
	}, nil
}

func (s *CandidateApplicationService) CreateCandidate(
	ctx context.Context,
	command CreateCandidateCommand,
) (CandidateView, error) {
	candidate, err := domain.NewCandidate(command.ID, command.DNI, command.Nombre, command.Email)
	if err != nil {
		return CandidateView{}, err
	}
	callID := strings.TrimSpace(command.CallID)
	if callID == "" {
		callID = DefaultCallID
	}
	if err := s.candidates.Save(ctx, callID, candidate); err != nil {
		return CandidateView{}, err
	}
	s.rememberCandidateCallID(candidate.ID, callID)
	return candidateView(candidate, callID), nil
}

func (s *CandidateApplicationService) AddMerit(
	ctx context.Context,
	candidateID string,
	command AddMeritCommand,
) (MeritView, error) {
	candidateID = strings.TrimSpace(candidateID)
	if _, err := s.candidates.GetByID(ctx, candidateID); err != nil {
		return MeritView{}, err
	}
	merit, err := domain.NewMerit(command.ID, command.Tipo, domain.MeritData(command.Datos))
	if err != nil {
		return MeritView{}, err
	}
	if command.Estado != "" {
		merit.Estado = command.Estado
		if err := merit.Validate(); err != nil {
			return MeritView{}, err
		}
	}
	if err := s.merits.Save(ctx, candidateID, merit); err != nil {
		return MeritView{}, err
	}
	return meritView(merit), nil
}

func (s *CandidateApplicationService) CalculateBaremo(
	ctx context.Context,
	candidateID string,
) (BaremoView, error) {
	candidateID = strings.TrimSpace(candidateID)
	if _, err := s.candidates.GetByID(ctx, candidateID); err != nil {
		return BaremoView{}, err
	}
	result, err := s.baremo.CalcularAutobaremo(ctx, candidateID, s.ruleSet)
	if err != nil {
		return BaremoView{}, err
	}
	return baremoView(result), nil
}

func (s *CandidateApplicationService) ExportExpediente(
	ctx context.Context,
	candidateID string,
) (ExpedienteView, error) {
	candidateID = strings.TrimSpace(candidateID)
	candidate, err := s.candidates.GetByID(ctx, candidateID)
	if err != nil {
		return ExpedienteView{}, err
	}
	merits, err := s.merits.ListByCandidate(ctx, candidateID)
	if err != nil {
		return ExpedienteView{}, err
	}
	baremo, err := s.CalculateBaremo(ctx, candidateID)
	if err != nil {
		return ExpedienteView{}, err
	}
	view := ExpedienteView{
		Candidate: candidateView(candidate, s.candidateCallID(candidate.ID)),
		Baremo:    baremo,
		Merits:    make([]MeritView, 0, len(merits)),
	}
	for _, merit := range merits {
		view.Merits = append(view.Merits, meritView(merit))
	}
	return view, nil
}

func (s *CandidateApplicationService) rememberCandidateCallID(candidateID, callID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callIDByCandidateID[candidateID] = callID
}

func (s *CandidateApplicationService) candidateCallID(candidateID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	callID := strings.TrimSpace(s.callIDByCandidateID[candidateID])
	if callID == "" {
		return DefaultCallID
	}
	return callID
}

func candidateView(candidate domain.Candidate, callID string) CandidateView {
	return CandidateView{
		ID:     candidate.ID,
		DNI:    candidate.DNI,
		Nombre: candidate.Nombre,
		Email:  candidate.Email,
		CallID: callID,
	}
}

func meritView(merit domain.Merit) MeritView {
	return MeritView{
		ID:     merit.ID,
		Tipo:   merit.Tipo,
		Estado: merit.Estado,
		Datos:  MeritDataCommand(merit.Datos),
	}
}

func baremoView(result domain.BaremoResult) BaremoView {
	view := BaremoView{
		TotalPoints:    result.TotalPoints,
		RuleSetID:      result.RuleSetID,
		RuleSetVersion: result.RuleSetVersion,
		SectionPoints:  map[string]float64{},
		Details:        make([]BaremoDetailView, 0, len(result.Details)),
	}
	for section, points := range result.SectionPoints {
		view.SectionPoints[string(section)] = points
	}
	for _, detail := range result.Details {
		view.Details = append(view.Details, BaremoDetailView{
			MeritID:       detail.MeritID,
			MeritType:     string(detail.MeritType),
			Section:       string(detail.Section),
			RawPoints:     detail.RawPoints,
			AppliedPoints: detail.AppliedPoints,
			Capped:        detail.Capped,
		})
	}
	return view
}
