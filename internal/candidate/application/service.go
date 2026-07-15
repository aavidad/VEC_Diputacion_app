package application

import (
	"context"
	"errors"
	"strings"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

var (
	ErrServiceDependenciesRequired = errors.New("candidate application service: repositories and baremo usecase are required")
	ErrCallIDRequired              = errors.New("candidate application service: an explicit call is required")
	ErrCallNotConfigured           = errors.New("candidate application service: call is not configured")
	ErrCandidateCallBindingInvalid = errors.New("candidate application service: candidate call binding is invalid")
)

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
		candidates: candidates,
		merits:     merits,
		baremo:     baremo,
		ruleSet:    ruleSet,
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
	callID := command.CallID
	if callID == "" {
		return CandidateView{}, ErrCallIDRequired
	}
	if callID != strings.TrimSpace(callID) || strings.Contains(callID, "*") ||
		callID != s.ruleSet.Config().ConvocatoriaID {
		return CandidateView{}, ErrCallNotConfigured
	}
	if err := s.candidates.Save(ctx, callID, candidate); err != nil {
		return CandidateView{}, err
	}
	return candidateView(candidate, callID), nil
}

func (s *CandidateApplicationService) AddMerit(
	ctx context.Context,
	candidateID string,
	command AddMeritCommand,
) (MeritView, error) {
	candidateID = strings.TrimSpace(candidateID)
	if _, _, err := s.candidateInConfiguredCall(ctx, candidateID); err != nil {
		return MeritView{}, err
	}
	merit, err := domain.NewMerit(command.ID, command.Tipo, domain.MeritData(command.Datos))
	if err != nil {
		return MeritView{}, err
	}
	// El estado administrativo no es un dato declarable por el candidato.
	// Admitirlo aqui permitia crear un merito ya "Validado" desde la API
	// ciudadana. La presentacion y la revision deben pasar por casos de uso
	// distintos, autorizados y auditados.
	if command.Estado != "" && command.Estado != domain.MeritStateBorrador {
		return MeritView{}, domain.ErrMeritTransition
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
	if _, _, err := s.candidateInConfiguredCall(ctx, candidateID); err != nil {
		return BaremoView{}, err
	}
	result, err := s.baremo.CalcularAutobaremo(ctx, candidateID, s.ruleSet)
	if err != nil {
		return BaremoView{}, err
	}
	configuracion := s.ruleSet.Config()
	if result.RuleSetID != configuracion.ConvocatoriaID || result.RuleSetVersion != configuracion.Version {
		return BaremoView{}, ErrCandidateCallBindingInvalid
	}
	return baremoView(result), nil
}

func (s *CandidateApplicationService) ExportExpediente(
	ctx context.Context,
	candidateID string,
) (ExpedienteView, error) {
	candidateID = strings.TrimSpace(candidateID)
	candidate, callID, err := s.candidateInConfiguredCall(ctx, candidateID)
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
		Candidate: candidateView(candidate, callID),
		Baremo:    baremo,
		Merits:    make([]MeritView, 0, len(merits)),
	}
	for _, merit := range merits {
		view.Merits = append(view.Merits, meritView(merit))
	}
	return view, nil
}

func (s *CandidateApplicationService) candidateInConfiguredCall(
	ctx context.Context,
	candidateID string,
) (domain.Candidate, string, error) {
	candidate, callID, err := s.candidates.GetByID(ctx, candidateID)
	if err != nil {
		return domain.Candidate{}, "", err
	}
	configuredCallID := s.ruleSet.Config().ConvocatoriaID
	if callID == "" || callID != strings.TrimSpace(callID) || strings.Contains(callID, "*") ||
		callID != configuredCallID {
		return domain.Candidate{}, "", ErrCandidateCallBindingInvalid
	}
	return candidate, callID, nil
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
