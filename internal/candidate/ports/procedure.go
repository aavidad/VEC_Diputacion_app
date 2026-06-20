package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/candidate/domain"
)

var (
	ErrConvocatoriaNotFound = errors.New("procedure repository: convocatoria not found")
	ErrSolicitudNotFound    = errors.New("procedure repository: solicitud not found")
)

type ConvocatoriaRecord struct {
	Convocatoria domain.Convocatoria
	RuleSet      domain.BaremoRuleSet
}

type SolicitudRecord struct {
	ID             string
	ConvocatoriaID string
	CandidateID    string
	SorteoKey      string
	Estado         domain.SolicitudState
	Result         domain.BaremoResult
}

type ConvocatoriaRepository interface {
	Save(ctx context.Context, convocatoria ConvocatoriaRecord) error
	GetByID(ctx context.Context, id string) (ConvocatoriaRecord, error)
}

type SolicitudRepository interface {
	Save(ctx context.Context, solicitud SolicitudRecord) error
	GetByID(ctx context.Context, id string) (SolicitudRecord, error)
	ListByConvocatoria(ctx context.Context, convocatoriaID string) ([]SolicitudRecord, error)
}
