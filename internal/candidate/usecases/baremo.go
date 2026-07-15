package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

var (
	ErrBaremoMeritRepositoryRequired  = errors.New("baremo usecase: merit repository is required")
	ErrBaremoResultRepositoryRequired = errors.New("baremo usecase: result repository is required")
)

type BaremoResultRepository interface {
	Save(ctx context.Context, candidateID string, result domain.BaremoResult) error
}

type BaremoUseCase struct {
	merits  ports.MeritRepository
	results BaremoResultRepository
}

func NewBaremoUseCase(
	merits ports.MeritRepository,
	results BaremoResultRepository,
) (BaremoUseCase, error) {
	useCase := BaremoUseCase{merits: merits, results: results}
	if err := useCase.validate(); err != nil {
		return BaremoUseCase{}, err
	}
	return useCase, nil
}

func (u BaremoUseCase) PresentarSolicitud(
	ctx context.Context,
	candidateID string,
	ruleSet domain.BaremoRuleSet,
) (domain.BaremoResult, error) {
	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	candidateID = strings.TrimSpace(candidateID)
	if err := u.validateCandidate(candidateID); err != nil {
		return domain.BaremoResult{}, err
	}
	if err := ruleSet.Validate(); err != nil {
		return domain.BaremoResult{}, err
	}

	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	merits, err := u.merits.ListByCandidate(ctx, candidateID)
	if err != nil {
		return domain.BaremoResult{}, fmt.Errorf("list candidate merits: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	merits, err = u.presentMerits(ctx, candidateID, merits)
	if err != nil {
		return domain.BaremoResult{}, err
	}
	return u.calculateAndSave(ctx, candidateID, merits, ruleSet)
}

func (u BaremoUseCase) CalcularAutobaremo(
	ctx context.Context,
	candidateID string,
	ruleSet domain.BaremoRuleSet,
) (domain.BaremoResult, error) {
	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	candidateID = strings.TrimSpace(candidateID)
	if err := u.validateCandidate(candidateID); err != nil {
		return domain.BaremoResult{}, err
	}
	if err := ruleSet.Validate(); err != nil {
		return domain.BaremoResult{}, err
	}

	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	merits, err := u.merits.ListByCandidate(ctx, candidateID)
	if err != nil {
		return domain.BaremoResult{}, fmt.Errorf("list candidate merits: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	return u.calculateAndSave(ctx, candidateID, merits, ruleSet)
}

func (u BaremoUseCase) PuntuacionProvisional(
	ctx context.Context,
	candidateID string,
	ruleSet domain.BaremoRuleSet,
) (float64, error) {
	result, err := u.CalcularAutobaremo(ctx, candidateID, ruleSet)
	if err != nil {
		return 0, err
	}
	return result.TotalPoints, nil
}

func (u BaremoUseCase) validateCandidate(candidateID string) error {
	if err := u.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(candidateID) == "" {
		return domain.ErrCandidateIDRequired
	}
	return nil
}

func (u BaremoUseCase) validate() error {
	if u.merits == nil {
		return ErrBaremoMeritRepositoryRequired
	}
	if u.results == nil {
		return ErrBaremoResultRepositoryRequired
	}
	return nil
}

func (u BaremoUseCase) presentMerits(
	ctx context.Context,
	candidateID string,
	merits []domain.Merit,
) ([]domain.Merit, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	presented := append([]domain.Merit(nil), merits...)
	for i := range presented {
		if err := validateContext(ctx); err != nil {
			return nil, err
		}
		switch presented[i].Estado {
		case domain.MeritStateBorrador, domain.MeritStateSubsanacion:
			if err := presented[i].Transition(domain.MeritStatePresentado); err != nil {
				return nil, err
			}
			if err := validateContext(ctx); err != nil {
				return nil, err
			}
			if err := u.merits.Save(ctx, candidateID, presented[i]); err != nil {
				return nil, fmt.Errorf("present candidate merit: %w", err)
			}
			if err := validateContext(ctx); err != nil {
				return nil, err
			}
		default:
			if err := presented[i].Validate(); err != nil {
				return nil, err
			}
		}
	}
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	return presented, nil
}

func (u BaremoUseCase) calculateAndSave(
	ctx context.Context,
	candidateID string,
	merits []domain.Merit,
	ruleSet domain.BaremoRuleSet,
) (domain.BaremoResult, error) {
	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	result, err := domain.CalcularAutobaremo(merits, ruleSet)
	if err != nil {
		return domain.BaremoResult{}, err
	}
	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	if err := u.results.Save(ctx, candidateID, result); err != nil {
		return domain.BaremoResult{}, fmt.Errorf("save baremo result: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return domain.BaremoResult{}, err
	}
	return result, nil
}
