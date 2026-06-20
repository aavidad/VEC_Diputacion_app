package usecases

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

var (
	ErrProcedureRepositoryRequired   = errors.New("procedure usecase: repositories are required")
	ErrProcedureConvocatoriaRequired = errors.New("procedure usecase: convocatoria is required")
	ErrProcedureSolicitudRequired    = errors.New("procedure usecase: solicitud is required")
)

type ProcedureUseCase struct {
	convocatorias ports.ConvocatoriaRepository
	solicitudes   ports.SolicitudRepository
}

type CrearConvocatoriaCommand struct {
	ID      string
	Version string
	RuleSet domain.BaremoRuleSet
}

type RegistrarSolicitudCommand struct {
	ID             string
	ConvocatoriaID string
	CandidateID    string
	SorteoKey      string
	Merits         []domain.Merit
}

type ListadoItem struct {
	SolicitudID string
	CandidateID string
	Estado      domain.SolicitudState
	Result      domain.BaremoResult
	Rank        int
}

type Listado struct {
	ConvocatoriaID string
	Version        string
	Items          []ListadoItem
}

func NewProcedureUseCase(
	convocatorias ports.ConvocatoriaRepository,
	solicitudes ports.SolicitudRepository,
) (*ProcedureUseCase, error) {
	if convocatorias == nil || solicitudes == nil {
		return nil, ErrProcedureRepositoryRequired
	}
	return &ProcedureUseCase{convocatorias: convocatorias, solicitudes: solicitudes}, nil
}

func (u *ProcedureUseCase) CrearConvocatoria(
	ctx context.Context,
	command CrearConvocatoriaCommand,
) (ports.ConvocatoriaRecord, error) {
	ctx = normalizeContext(ctx)
	if err := command.RuleSet.Validate(); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	convocatoria, err := domain.NewConvocatoria(command.ID, command.Version)
	if err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	record := ports.ConvocatoriaRecord{Convocatoria: convocatoria, RuleSet: command.RuleSet}
	if err := u.convocatorias.Save(ctx, record); err != nil {
		return ports.ConvocatoriaRecord{}, fmt.Errorf("save convocatoria: %w", err)
	}
	return record, nil
}

func (u *ProcedureUseCase) RegistrarSolicitud(
	ctx context.Context,
	command RegistrarSolicitudCommand,
) (ports.SolicitudRecord, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(command.ID) == "" || strings.TrimSpace(command.CandidateID) == "" {
		return ports.SolicitudRecord{}, ErrProcedureSolicitudRequired
	}
	convocatoria, err := u.convocatoria(ctx, command.ConvocatoriaID)
	if err != nil {
		return ports.SolicitudRecord{}, err
	}
	result, err := domain.CalcularAutobaremo(command.Merits, convocatoria.RuleSet)
	if err != nil {
		return ports.SolicitudRecord{}, err
	}
	record := ports.SolicitudRecord{
		ID: strings.TrimSpace(command.ID), ConvocatoriaID: convocatoria.Convocatoria.ID,
		CandidateID: strings.TrimSpace(command.CandidateID), SorteoKey: strings.TrimSpace(command.SorteoKey),
		Estado: domain.SolicitudStateInscrita, Result: result,
	}
	if err := u.solicitudes.Save(ctx, record); err != nil {
		return ports.SolicitudRecord{}, fmt.Errorf("save solicitud: %w", err)
	}
	return record, nil
}

func (u *ProcedureUseCase) PublicarListadoProvisional(
	ctx context.Context,
	convocatoriaID string,
	admitidas map[string]bool,
) (Listado, error) {
	ctx = normalizeContext(ctx)
	convocatoria, solicitudes, err := u.loadProcedure(ctx, convocatoriaID)
	if err != nil {
		return Listado{}, err
	}
	for i := range solicitudes {
		next := domain.SolicitudStateExcluidaProvisional
		if isAdmitida(admitidas, solicitudes[i].ID) {
			next = domain.SolicitudStateAdmitidaProvisional
		}
		if solicitudes[i].Estado != next {
			if solicitudes[i].Estado, err = solicitudes[i].Estado.Transition(next); err != nil {
				return Listado{}, err
			}
			if err := u.solicitudes.Save(ctx, solicitudes[i]); err != nil {
				return Listado{}, fmt.Errorf("save provisional solicitud: %w", err)
			}
		}
	}
	sortSolicitudes(solicitudes)
	return buildListado(convocatoria.Convocatoria, solicitudes, nil), nil
}

func (u *ProcedureUseCase) PublicarListadoDefinitivo(
	ctx context.Context,
	convocatoriaID string,
	admitidas map[string]bool,
) (Listado, error) {
	ctx = normalizeContext(ctx)
	convocatoria, solicitudes, err := u.loadProcedure(ctx, convocatoriaID)
	if err != nil {
		return Listado{}, err
	}
	rankingInput := make([]domain.SolicitudRankingEntry, 0, len(solicitudes))
	for i := range solicitudes {
		next := domain.SolicitudStateExcluidaDefinitiva
		if isAdmitida(admitidas, solicitudes[i].ID) {
			next = domain.SolicitudStateAdmitidaDefinitiva
		}
		if solicitudes[i].Estado != next {
			if solicitudes[i].Estado, err = solicitudes[i].Estado.Transition(next); err != nil {
				return Listado{}, err
			}
			if err := u.solicitudes.Save(ctx, solicitudes[i]); err != nil {
				return Listado{}, fmt.Errorf("save definitive solicitud: %w", err)
			}
		}
		if solicitudes[i].Estado == domain.SolicitudStateAdmitidaDefinitiva {
			rankingInput = append(rankingInput, rankingEntry(solicitudes[i]))
		}
	}
	ranked, err := domain.RankSolicitudes(rankingInput, convocatoria.RuleSet)
	if err != nil {
		return Listado{}, err
	}
	sortSolicitudes(solicitudes)
	return buildListado(convocatoria.Convocatoria, solicitudes, ranked), nil
}

func (u *ProcedureUseCase) convocatoria(ctx context.Context, id string) (ports.ConvocatoriaRecord, error) {
	if strings.TrimSpace(id) == "" {
		return ports.ConvocatoriaRecord{}, ErrProcedureConvocatoriaRequired
	}
	return u.convocatorias.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProcedureUseCase) loadProcedure(
	ctx context.Context,
	convocatoriaID string,
) (ports.ConvocatoriaRecord, []ports.SolicitudRecord, error) {
	convocatoria, err := u.convocatoria(ctx, convocatoriaID)
	if err != nil {
		return ports.ConvocatoriaRecord{}, nil, err
	}
	solicitudes, err := u.solicitudes.ListByConvocatoria(ctx, convocatoria.Convocatoria.ID)
	if err != nil {
		return ports.ConvocatoriaRecord{}, nil, fmt.Errorf("list solicitudes: %w", err)
	}
	return convocatoria, solicitudes, nil
}

func rankingEntry(record ports.SolicitudRecord) domain.SolicitudRankingEntry {
	return domain.SolicitudRankingEntry{
		SolicitudID: record.ID, CandidateID: record.CandidateID,
		Estado: record.Estado, Result: record.Result, SorteoKey: record.SorteoKey,
	}
}

func sortSolicitudes(records []ports.SolicitudRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].CandidateID == records[j].CandidateID {
			return records[i].ID < records[j].ID
		}
		return records[i].CandidateID < records[j].CandidateID
	})
}

func isAdmitida(admitidas map[string]bool, solicitudID string) bool {
	return admitidas[strings.TrimSpace(solicitudID)]
}

func buildListado(
	convocatoria domain.Convocatoria,
	solicitudes []ports.SolicitudRecord,
	ranked []domain.RankedSolicitud,
) Listado {
	ranks := make(map[string]int, len(ranked))
	for _, item := range ranked {
		ranks[item.SolicitudID] = item.Position
	}
	listado := Listado{ConvocatoriaID: convocatoria.ID, Version: convocatoria.Version}
	for _, solicitud := range solicitudes {
		listado.Items = append(listado.Items, ListadoItem{
			SolicitudID: solicitud.ID, CandidateID: solicitud.CandidateID,
			Estado: solicitud.Estado, Result: solicitud.Result, Rank: ranks[solicitud.ID],
		})
	}
	return listado
}
