package application

import (
	"context"
	"errors"
	"strings"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

var (
	ErrCatalogStoreRequired = errors.New("personal catalog store required")
	ErrRPTPositionNotFound  = errors.New("personal rpt position not found")
)

const maxRPTPositionPageSize = 2000

type CatalogService struct {
	store ports.CatalogStore
}

func NewCatalogService(store ports.CatalogStore) (*CatalogService, error) {
	if store == nil {
		return nil, ErrCatalogStoreRequired
	}
	return &CatalogService{store: store}, nil
}

func (s *CatalogService) UpsertPosition(ctx context.Context, position domain.RPTPosition) (domain.RPTPosition, error) {
	if err := position.Validate(); err != nil {
		return domain.RPTPosition{}, err
	}
	position = normalizePosition(position)
	if err := s.store.UpsertPosition(ctx, position); err != nil {
		return domain.RPTPosition{}, err
	}
	stored, ok, err := s.store.GetPosition(ctx, position.Code)
	if err != nil {
		return domain.RPTPosition{}, err
	}
	if !ok {
		return domain.RPTPosition{}, ErrRPTPositionNotFound
	}
	return stored, nil
}

func (s *CatalogService) GetPosition(ctx context.Context, code string) (domain.RPTPosition, error) {
	position, ok, err := s.store.GetPosition(ctx, strings.TrimSpace(code))
	if err != nil {
		return domain.RPTPosition{}, err
	}
	if !ok {
		return domain.RPTPosition{}, ErrRPTPositionNotFound
	}
	return position, nil
}

func (s *CatalogService) DeletePosition(ctx context.Context, code string) (bool, error) {
	return s.store.DeletePosition(ctx, strings.TrimSpace(code))
}

func (s *CatalogService) ListPositions(ctx context.Context, filter domain.RPTPositionFilter) (domain.RPTPositionPage, error) {
	filter = normalizePositionFilter(filter)
	return s.store.ListPositions(ctx, filter)
}

func (s *CatalogService) ImportPositions(ctx context.Context, cmd domain.RPTImportCommand) (domain.RPTImportReceipt, error) {
	cmd.Source = strings.TrimSpace(cmd.Source)
	cmd.Version = strings.TrimSpace(cmd.Version)
	for i := range cmd.Positions {
		if err := cmd.Positions[i].Validate(); err != nil {
			return domain.RPTImportReceipt{}, err
		}
		cmd.Positions[i] = normalizePosition(cmd.Positions[i])
	}
	return s.store.ImportPositions(ctx, cmd)
}

func (s *CatalogService) UpsertCatalogEntry(ctx context.Context, entry domain.CatalogEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	entry = normalizeCatalogEntry(entry)
	return s.store.UpsertCatalogEntry(ctx, entry)
}

func (s *CatalogService) ListCatalogEntries(ctx context.Context) ([]domain.CatalogEntry, error) {
	return s.store.ListCatalogEntries(ctx)
}

func (s *CatalogService) Stats(ctx context.Context) (domain.CatalogStats, error) {
	return s.store.Stats(ctx)
}

func normalizePosition(position domain.RPTPosition) domain.RPTPosition {
	position.Code = strings.TrimSpace(position.Code)
	position.Name = strings.TrimSpace(position.Name)
	position.Type = strings.TrimSpace(position.Type)
	position.Administration = strings.TrimSpace(position.Administration)
	position.Provision = strings.TrimSpace(position.Provision)
	position.Group = strings.TrimSpace(position.Group)
	position.Area = strings.TrimSpace(position.Area)
	position.Scale = strings.TrimSpace(position.Scale)
	position.CategoryCode = strings.TrimSpace(position.CategoryCode)
	position.CategorySlug = strings.TrimSpace(position.CategorySlug)
	position.State = strings.TrimSpace(position.State)
	return position
}

func normalizePositionFilter(filter domain.RPTPositionFilter) domain.RPTPositionFilter {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Group = strings.TrimSpace(filter.Group)
	filter.CenterCode = strings.TrimSpace(filter.CenterCode)
	filter.Provision = strings.TrimSpace(filter.Provision)
	filter.State = strings.TrimSpace(filter.State)
	if filter.Limit <= 0 || filter.Limit > maxRPTPositionPageSize {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return filter
}

func normalizeCatalogEntry(entry domain.CatalogEntry) domain.CatalogEntry {
	entry.Catalog = strings.TrimSpace(entry.Catalog)
	entry.Code = strings.TrimSpace(entry.Code)
	entry.Label = strings.TrimSpace(entry.Label)
	entry.State = strings.TrimSpace(entry.State)
	return entry
}
