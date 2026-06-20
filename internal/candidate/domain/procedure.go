package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrProcedureInvalid    = errors.New("procedure is invalid")
	ErrProcedureTransition = errors.New("procedure transition is invalid")
	ErrProcedureRanking    = errors.New("procedure ranking is invalid")
)

type Convocatoria struct {
	ID      string
	Version string
	Estado  ProcedureState
}

func NewConvocatoria(id, version string) (Convocatoria, error) {
	call := Convocatoria{ID: strings.TrimSpace(id), Version: strings.TrimSpace(version), Estado: ProcedureStateBorrador}
	if err := call.Validate(); err != nil {
		return Convocatoria{}, err
	}
	return call, nil
}

func (c Convocatoria) Validate() error {
	switch {
	case strings.TrimSpace(c.ID) == "":
		return fmt.Errorf("%w: convocatoria id is required", ErrProcedureInvalid)
	case strings.TrimSpace(c.Version) == "":
		return fmt.Errorf("%w: convocatoria version is required", ErrProcedureInvalid)
	case !c.Estado.IsValid():
		return fmt.Errorf("%w: convocatoria state %q", ErrProcedureInvalid, c.Estado)
	default:
		return nil
	}
}

func (c Convocatoria) NewVersion(version string) (Convocatoria, error) {
	next := Convocatoria{ID: c.ID, Version: strings.TrimSpace(version), Estado: ProcedureStateBorrador}
	if next.Version == c.Version {
		return Convocatoria{}, fmt.Errorf("%w: duplicated convocatoria version", ErrProcedureInvalid)
	}
	if err := next.Validate(); err != nil {
		return Convocatoria{}, err
	}
	return next, nil
}

type ProcedureState string

const (
	ProcedureStateBorrador    ProcedureState = "Borrador"
	ProcedureStateInscripcion ProcedureState = "Inscripcion"
	ProcedureStateSubsanacion ProcedureState = "Subsanacion"
	ProcedureStateAlegaciones ProcedureState = "Alegaciones"
	ProcedureStateDefinitiva  ProcedureState = "Definitiva"
	ProcedureStateCerrada     ProcedureState = "Cerrada"
)

func (s ProcedureState) IsValid() bool {
	switch s {
	case ProcedureStateBorrador, ProcedureStateInscripcion, ProcedureStateSubsanacion,
		ProcedureStateAlegaciones, ProcedureStateDefinitiva, ProcedureStateCerrada:
		return true
	default:
		return false
	}
}

type SolicitudState string

const (
	SolicitudStateBorrador             SolicitudState = "Borrador"
	SolicitudStateInscrita             SolicitudState = "Inscrita"
	SolicitudStateSubsanacionRequerida SolicitudState = "SubsanacionRequerida"
	SolicitudStateSubsanada            SolicitudState = "Subsanada"
	SolicitudStateAdmitidaProvisional  SolicitudState = "AdmitidaProvisional"
	SolicitudStateExcluidaProvisional  SolicitudState = "ExcluidaProvisional"
	SolicitudStateAlegacionPresentada  SolicitudState = "AlegacionPresentada"
	SolicitudStateAdmitidaDefinitiva   SolicitudState = "AdmitidaDefinitiva"
	SolicitudStateExcluidaDefinitiva   SolicitudState = "ExcluidaDefinitiva"
)

func (s SolicitudState) IsValid() bool {
	switch s {
	case SolicitudStateBorrador, SolicitudStateInscrita, SolicitudStateSubsanacionRequerida,
		SolicitudStateSubsanada, SolicitudStateAdmitidaProvisional, SolicitudStateExcluidaProvisional,
		SolicitudStateAlegacionPresentada, SolicitudStateAdmitidaDefinitiva, SolicitudStateExcluidaDefinitiva:
		return true
	default:
		return false
	}
}

func (s SolicitudState) CanTransition(to SolicitudState) bool {
	if !s.IsValid() || !to.IsValid() {
		return false
	}
	for _, allowed := range allowedSolicitudTransitions[s] {
		if allowed == to {
			return true
		}
	}
	return false
}

func (s SolicitudState) Transition(to SolicitudState) (SolicitudState, error) {
	if !to.IsValid() {
		return "", fmt.Errorf("%w: solicitud state %q", ErrProcedureInvalid, to)
	}
	if !s.CanTransition(to) {
		return "", fmt.Errorf("%w: solicitud %s -> %s", ErrProcedureTransition, s, to)
	}
	return to, nil
}

type BolsaState string

const (
	BolsaStateSinConstituir BolsaState = "SinConstituir"
	BolsaStateProvisional   BolsaState = "Provisional"
	BolsaStateEnAlegaciones BolsaState = "EnAlegaciones"
	BolsaStateDefinitiva    BolsaState = "Definitiva"
	BolsaStateAgotada       BolsaState = "Agotada"
	BolsaStateCerrada       BolsaState = "Cerrada"
)

func (s BolsaState) IsValid() bool {
	switch s {
	case BolsaStateSinConstituir, BolsaStateProvisional, BolsaStateEnAlegaciones,
		BolsaStateDefinitiva, BolsaStateAgotada, BolsaStateCerrada:
		return true
	default:
		return false
	}
}

func (s BolsaState) CanTransition(to BolsaState) bool {
	if !s.IsValid() || !to.IsValid() {
		return false
	}
	for _, allowed := range allowedBolsaTransitions[s] {
		if allowed == to {
			return true
		}
	}
	return false
}

func (s BolsaState) Transition(to BolsaState) (BolsaState, error) {
	if !to.IsValid() {
		return "", fmt.Errorf("%w: bolsa state %q", ErrProcedureInvalid, to)
	}
	if !s.CanTransition(to) {
		return "", fmt.Errorf("%w: bolsa %s -> %s", ErrProcedureTransition, s, to)
	}
	return to, nil
}

type SolicitudRankingEntry struct {
	SolicitudID string
	CandidateID string
	Estado      SolicitudState
	Result      BaremoResult
	SorteoKey   string
}

type RankedSolicitud struct {
	Position             int
	SolicitudID          string
	CandidateID          string
	TotalPoints          float64
	PreviousOrderReason  string
	PreviousTieDecisions []BaremoTieBreakDecision
}

func RankSolicitudes(entries []SolicitudRankingEntry, ruleSet BaremoRuleSet) ([]RankedSolicitud, error) {
	if err := ruleSet.Validate(); err != nil {
		return nil, err
	}
	ordered := append([]SolicitudRankingEntry(nil), entries...)
	for _, entry := range ordered {
		if err := entry.Validate(); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		less, _ := solicitudRanksBefore(ordered[i], ordered[j], ruleSet)
		return less
	})
	return explainRanking(ordered, ruleSet)
}

func (e SolicitudRankingEntry) Validate() error {
	switch {
	case strings.TrimSpace(e.SolicitudID) == "":
		return fmt.Errorf("%w: solicitud id is required", ErrProcedureRanking)
	case strings.TrimSpace(e.CandidateID) == "":
		return fmt.Errorf("%w: candidate id is required", ErrProcedureRanking)
	case !e.Estado.IsValid():
		return fmt.Errorf("%w: solicitud state %q", ErrProcedureRanking, e.Estado)
	case e.Estado != SolicitudStateAdmitidaDefinitiva:
		return fmt.Errorf("%w: only admitted definitive solicitudes can be ranked", ErrProcedureRanking)
	default:
		return nil
	}
}

func solicitudRanksBefore(a, b SolicitudRankingEntry, ruleSet BaremoRuleSet) (bool, []BaremoTieBreakDecision) {
	if a.Result.TotalPoints != b.Result.TotalPoints {
		return a.Result.TotalPoints > b.Result.TotalPoints, nil
	}
	result, err := Desempate(toTieCandidate(a), toTieCandidate(b), ruleSet)
	if err != nil {
		return a.CandidateID < b.CandidateID, nil
	}
	switch result.WinnerID {
	case a.CandidateID:
		return true, result.Decisions
	case b.CandidateID:
		return false, result.Decisions
	default:
		return a.SolicitudID < b.SolicitudID, result.Decisions
	}
}

func explainRanking(ordered []SolicitudRankingEntry, ruleSet BaremoRuleSet) ([]RankedSolicitud, error) {
	ranked := make([]RankedSolicitud, 0, len(ordered))
	for i, entry := range ordered {
		item := RankedSolicitud{
			Position: i + 1, SolicitudID: entry.SolicitudID,
			CandidateID: entry.CandidateID, TotalPoints: roundPoints(entry.Result.TotalPoints),
			PreviousOrderReason: "highest ranked solicitud",
		}
		if i > 0 {
			reason, decisions, err := previousOrderReason(ordered[i-1], entry, ruleSet)
			if err != nil {
				return nil, err
			}
			item.PreviousOrderReason, item.PreviousTieDecisions = reason, decisions
		}
		ranked = append(ranked, item)
	}
	return ranked, nil
}

func previousOrderReason(prev, current SolicitudRankingEntry, ruleSet BaremoRuleSet) (string, []BaremoTieBreakDecision, error) {
	if prev.Result.TotalPoints > current.Result.TotalPoints {
		return "previous has greater total points", nil, nil
	}
	result, err := Desempate(toTieCandidate(prev), toTieCandidate(current), ruleSet)
	if err != nil {
		return "", nil, err
	}
	if result.WinnerID == "" {
		return "previous has lower solicitud id fallback", result.Decisions, nil
	}
	return "previous wins configured tie break", result.Decisions, nil
}

func toTieCandidate(entry SolicitudRankingEntry) BaremoTieCandidate {
	return BaremoTieCandidate{CandidateID: entry.CandidateID, SorteoKey: entry.SorteoKey, Result: entry.Result}
}

var allowedSolicitudTransitions = map[SolicitudState][]SolicitudState{
	SolicitudStateBorrador:             {SolicitudStateInscrita},
	SolicitudStateInscrita:             {SolicitudStateAdmitidaProvisional, SolicitudStateSubsanacionRequerida, SolicitudStateExcluidaProvisional},
	SolicitudStateSubsanacionRequerida: {SolicitudStateSubsanada, SolicitudStateExcluidaProvisional},
	SolicitudStateSubsanada:            {SolicitudStateAdmitidaProvisional, SolicitudStateExcluidaProvisional},
	SolicitudStateAdmitidaProvisional:  {SolicitudStateAlegacionPresentada, SolicitudStateAdmitidaDefinitiva},
	SolicitudStateExcluidaProvisional:  {SolicitudStateAlegacionPresentada, SolicitudStateExcluidaDefinitiva},
	SolicitudStateAlegacionPresentada:  {SolicitudStateAdmitidaDefinitiva, SolicitudStateExcluidaDefinitiva},
}

var allowedBolsaTransitions = map[BolsaState][]BolsaState{
	BolsaStateSinConstituir: {BolsaStateProvisional},
	BolsaStateProvisional:   {BolsaStateEnAlegaciones, BolsaStateDefinitiva},
	BolsaStateEnAlegaciones: {BolsaStateDefinitiva},
	BolsaStateDefinitiva:    {BolsaStateAgotada, BolsaStateCerrada},
	BolsaStateAgotada:       {BolsaStateCerrada},
}
