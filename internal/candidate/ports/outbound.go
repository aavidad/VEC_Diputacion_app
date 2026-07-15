package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/candidate/domain"
)

var (
	// ErrCandidateNotFound signals that no candidate exists for the requested ID.
	ErrCandidateNotFound = errors.New("candidate repository: candidate not found")
	// ErrCandidateCallInvalid impide interpretar una convocatoria ausente,
	// no canonica o con comodines como un ambito valido.
	ErrCandidateCallInvalid = errors.New("candidate repository: candidate call is invalid")
	// ErrMeritNotFound signals that no merit exists for the requested repository query.
	ErrMeritNotFound = errors.New("merit repository: merit not found")
)

// CandidateRepository is the outbound persistence port for candidates.
type CandidateRepository interface {
	Save(ctx context.Context, callID string, candidate domain.Candidate) error
	// GetByID devuelve tambien la convocatoria duradera exacta. El caso de uso
	// no puede reconstruirla mediante memoria de proceso ni un valor por defecto.
	GetByID(ctx context.Context, id string) (domain.Candidate, string, error)
	ListByCall(ctx context.Context, callID string) ([]domain.Candidate, error)
}

// MeritRepository is the outbound persistence port for candidate merits.
type MeritRepository interface {
	Save(ctx context.Context, candidateID string, merit domain.Merit) error
	ListByCandidate(ctx context.Context, candidateID string) ([]domain.Merit, error)
}
