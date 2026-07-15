package usecases

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	if err := command.RuleSet.Validate(); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	convocatoria, err := domain.NewConvocatoria(command.ID, command.Version)
	if err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	record := ports.ConvocatoriaRecord{Convocatoria: convocatoria, RuleSet: command.RuleSet}
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	if err := u.convocatorias.Save(ctx, record); err != nil {
		return ports.ConvocatoriaRecord{}, fmt.Errorf("save convocatoria: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	return record, nil
}

func (u *ProcedureUseCase) EnsureConvocatoria(
	ctx context.Context,
	command CrearConvocatoriaCommand,
) (ports.ConvocatoriaRecord, error) {
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	if err := command.RuleSet.Validate(); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	id := strings.TrimSpace(command.ID)
	if id == "" {
		return ports.ConvocatoriaRecord{}, ErrProcedureConvocatoriaRequired
	}
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	existing, err := u.convocatorias.GetByID(ctx, id)
	if contextErr := validateContext(ctx); contextErr != nil {
		return ports.ConvocatoriaRecord{}, contextErr
	}
	if err == nil &&
		existing.Convocatoria.Version == strings.TrimSpace(command.Version) &&
		reflect.DeepEqual(existing.RuleSet, command.RuleSet) {
		return existing, nil
	}
	if err != nil && !errors.Is(err, ports.ErrConvocatoriaNotFound) {
		return ports.ConvocatoriaRecord{}, err
	}
	return u.CrearConvocatoria(ctx, command)
}

func (u *ProcedureUseCase) RegistrarSolicitud(
	ctx context.Context,
	command RegistrarSolicitudCommand,
) (ports.SolicitudRecord, error) {
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	record, err := u.buildSolicitudRecord(ctx, command)
	if err != nil {
		return ports.SolicitudRecord{}, err
	}
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	if err := u.solicitudes.Save(ctx, record); err != nil {
		return ports.SolicitudRecord{}, fmt.Errorf("save solicitud: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	return record, nil
}

func (u *ProcedureUseCase) EnsureSolicitud(
	ctx context.Context,
	command RegistrarSolicitudCommand,
) (ports.SolicitudRecord, error) {
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	next, err := u.buildSolicitudRecord(ctx, command)
	if err != nil {
		return ports.SolicitudRecord{}, err
	}
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	existing, err := u.solicitudes.GetByID(ctx, next.ID)
	if contextErr := validateContext(ctx); contextErr != nil {
		return ports.SolicitudRecord{}, contextErr
	}
	if err == nil && sameSolicitudSeed(existing, next) {
		return existing, nil
	}
	if err != nil && !errors.Is(err, ports.ErrSolicitudNotFound) {
		return ports.SolicitudRecord{}, err
	}
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	if err := u.solicitudes.Save(ctx, next); err != nil {
		return ports.SolicitudRecord{}, fmt.Errorf("save solicitud: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	return next, nil
}

func (u *ProcedureUseCase) buildSolicitudRecord(
	ctx context.Context,
	command RegistrarSolicitudCommand,
) (ports.SolicitudRecord, error) {
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	if strings.TrimSpace(command.ID) == "" || strings.TrimSpace(command.CandidateID) == "" {
		return ports.SolicitudRecord{}, ErrProcedureSolicitudRequired
	}
	convocatoria, err := u.convocatoria(ctx, command.ConvocatoriaID)
	if err != nil {
		return ports.SolicitudRecord{}, err
	}
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	result, err := domain.CalcularBaremoOficial(command.Merits, convocatoria.RuleSet)
	if err != nil {
		return ports.SolicitudRecord{}, err
	}
	if err := validateContext(ctx); err != nil {
		return ports.SolicitudRecord{}, err
	}
	return ports.SolicitudRecord{
		ID:             strings.TrimSpace(command.ID),
		ConvocatoriaID: convocatoria.Convocatoria.ID,
		CandidateID:    strings.TrimSpace(command.CandidateID),
		SorteoKey:      strings.TrimSpace(command.SorteoKey),
		Estado:         domain.SolicitudStateInscrita,
		Result:         result,
	}, nil
}

func sameSolicitudSeed(existing ports.SolicitudRecord, next ports.SolicitudRecord) bool {
	existing.Estado = next.Estado
	return reflect.DeepEqual(existing, next)
}

func (u *ProcedureUseCase) PublicarListadoProvisional(
	ctx context.Context,
	convocatoriaID string,
	admitidas map[string]bool,
) (Listado, error) {
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	convocatoria, solicitudes, err := u.loadProcedure(ctx, convocatoriaID)
	if err != nil {
		return Listado{}, err
	}
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	for i := range solicitudes {
		if err := validateContext(ctx); err != nil {
			return Listado{}, err
		}
		next := domain.SolicitudStateExcluidaProvisional
		if isAdmitida(admitidas, solicitudes[i].ID) {
			next = domain.SolicitudStateAdmitidaProvisional
		}
		if solicitudes[i].Estado != next {
			if solicitudes[i].Estado, err = solicitudes[i].Estado.Transition(next); err != nil {
				return Listado{}, err
			}
			if err := validateContext(ctx); err != nil {
				return Listado{}, err
			}
			if err := u.solicitudes.Save(ctx, solicitudes[i]); err != nil {
				return Listado{}, fmt.Errorf("save provisional solicitud: %w", err)
			}
			if err := validateContext(ctx); err != nil {
				return Listado{}, err
			}
		}
	}
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	sortSolicitudes(solicitudes)
	return buildListado(convocatoria.Convocatoria, solicitudes, nil), nil
}

func (u *ProcedureUseCase) PublicarListadoDefinitivo(
	ctx context.Context,
	convocatoriaID string,
	admitidas map[string]bool,
) (Listado, error) {
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	convocatoria, solicitudes, err := u.loadProcedure(ctx, convocatoriaID)
	if err != nil {
		return Listado{}, err
	}
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	rankingInput := make([]domain.SolicitudRankingEntry, 0, len(solicitudes))
	for i := range solicitudes {
		if err := validateContext(ctx); err != nil {
			return Listado{}, err
		}
		next := domain.SolicitudStateExcluidaDefinitiva
		if isAdmitida(admitidas, solicitudes[i].ID) {
			next = domain.SolicitudStateAdmitidaDefinitiva
		}
		if solicitudes[i].Estado != next {
			if solicitudes[i].Estado, err = solicitudes[i].Estado.Transition(next); err != nil {
				return Listado{}, err
			}
			if err := validateContext(ctx); err != nil {
				return Listado{}, err
			}
			if err := u.solicitudes.Save(ctx, solicitudes[i]); err != nil {
				return Listado{}, fmt.Errorf("save definitive solicitud: %w", err)
			}
			if err := validateContext(ctx); err != nil {
				return Listado{}, err
			}
		}
		if solicitudes[i].Estado == domain.SolicitudStateAdmitidaDefinitiva {
			rankingInput = append(rankingInput, rankingEntry(solicitudes[i]))
		}
	}
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	ranked, err := domain.RankSolicitudes(rankingInput, convocatoria.RuleSet)
	if err != nil {
		return Listado{}, err
	}
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	sortSolicitudes(solicitudes)
	return buildListado(convocatoria.Convocatoria, solicitudes, ranked), nil
}

func (u *ProcedureUseCase) ListadoActual(
	ctx context.Context,
	convocatoriaID string,
) (Listado, error) {
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	convocatoria, solicitudes, err := u.loadProcedure(ctx, convocatoriaID)
	if err != nil {
		return Listado{}, err
	}
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	rankingInput := make([]domain.SolicitudRankingEntry, 0, len(solicitudes))
	for _, solicitud := range solicitudes {
		if err := validateContext(ctx); err != nil {
			return Listado{}, err
		}
		if solicitud.Estado == domain.SolicitudStateAdmitidaDefinitiva {
			rankingInput = append(rankingInput, rankingEntry(solicitud))
		}
	}
	var ranked []domain.RankedSolicitud
	if len(rankingInput) > 0 {
		if err := validateContext(ctx); err != nil {
			return Listado{}, err
		}
		ranked, err = domain.RankSolicitudes(rankingInput, convocatoria.RuleSet)
		if err != nil {
			return Listado{}, err
		}
	}
	if err := validateContext(ctx); err != nil {
		return Listado{}, err
	}
	sortSolicitudes(solicitudes)
	return buildListado(convocatoria.Convocatoria, solicitudes, ranked), nil
}

func (u *ProcedureUseCase) convocatoria(ctx context.Context, id string) (ports.ConvocatoriaRecord, error) {
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, err
	}
	if strings.TrimSpace(id) == "" {
		return ports.ConvocatoriaRecord{}, ErrProcedureConvocatoriaRequired
	}
	record, err := u.convocatorias.GetByID(ctx, strings.TrimSpace(id))
	if contextErr := validateContext(ctx); contextErr != nil {
		return ports.ConvocatoriaRecord{}, contextErr
	}
	return record, err
}

func (u *ProcedureUseCase) loadProcedure(
	ctx context.Context,
	convocatoriaID string,
) (ports.ConvocatoriaRecord, []ports.SolicitudRecord, error) {
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, nil, err
	}
	convocatoria, err := u.convocatoria(ctx, convocatoriaID)
	if err != nil {
		return ports.ConvocatoriaRecord{}, nil, err
	}
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, nil, err
	}
	solicitudes, err := u.solicitudes.ListByConvocatoria(ctx, convocatoria.Convocatoria.ID)
	if err != nil {
		return ports.ConvocatoriaRecord{}, nil, fmt.Errorf("list solicitudes: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return ports.ConvocatoriaRecord{}, nil, err
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
