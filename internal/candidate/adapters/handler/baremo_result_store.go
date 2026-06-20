package handler

import (
	"context"
	"sync"

	"vec-diputacion-granada/internal/candidate/domain"
)

type baremoResultStore struct {
	mu      sync.Mutex
	results map[string]domain.BaremoResult
}

func newBaremoResultStore() *baremoResultStore {
	return &baremoResultStore{results: map[string]domain.BaremoResult{}}
}

func (s *baremoResultStore) Save(
	ctx context.Context,
	candidateID string,
	result domain.BaremoResult,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[candidateID] = result
	return nil
}
