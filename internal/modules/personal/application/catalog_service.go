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
	ErrCategoryNotFound     = errors.New("personal professional category not found")
)

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
	if err := s.store.UpsertPosition(ctx, normalizePosition(position)); err != nil {
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
		position := normalizePosition(cmd.Positions[i])
		if err := position.Validate(); err != nil {
			return domain.RPTImportReceipt{}, err
		}
		cmd.Positions[i] = position
	}
	return s.store.ImportPositions(ctx, cmd)
}

func (s *CatalogService) UpsertCategory(ctx context.Context, category domain.ProfessionalCategory) error {
	if err := category.Validate(); err != nil {
		return err
	}
	return s.store.UpsertCategory(ctx, normalizeCategory(category))
}

func (s *CatalogService) GetCategory(ctx context.Context, slug string) (domain.ProfessionalCategory, error) {
	category, ok, err := s.store.GetCategory(ctx, strings.TrimSpace(slug))
	if err != nil {
		return domain.ProfessionalCategory{}, err
	}
	if !ok {
		return domain.ProfessionalCategory{}, ErrCategoryNotFound
	}
	return category, nil
}

func (s *CatalogService) DeleteCategory(ctx context.Context, slug string) (bool, error) {
	return s.store.DeleteCategory(ctx, strings.TrimSpace(slug))
}

func (s *CatalogService) ListCategories(ctx context.Context, filter domain.ProfessionalCategoryFilter) (domain.ProfessionalCategoryPage, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Area = strings.TrimSpace(filter.Area)
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.store.ListCategories(ctx, filter)
}

func (s *CatalogService) UpsertCatalogEntry(ctx context.Context, entry domain.CatalogEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	return s.store.UpsertCatalogEntry(ctx, normalizeCatalogEntry(entry))
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
	if position.State == "" {
		position.State = "Vigente"
	}
	return position
}

func normalizePositionFilter(filter domain.RPTPositionFilter) domain.RPTPositionFilter {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Group = strings.TrimSpace(filter.Group)
	filter.CenterCode = strings.TrimSpace(filter.CenterCode)
	filter.Provision = strings.TrimSpace(filter.Provision)
	filter.State = strings.TrimSpace(filter.State)
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return filter
}

func normalizeCategory(category domain.ProfessionalCategory) domain.ProfessionalCategory {
	category.Slug = strings.TrimSpace(category.Slug)
	category.Name = strings.TrimSpace(category.Name)
	category.Area = strings.TrimSpace(category.Area)
	category.State = strings.TrimSpace(category.State)
	if category.State == "" {
		category.State = "Vigente"
	}
	return category
}

func normalizeCatalogEntry(entry domain.CatalogEntry) domain.CatalogEntry {
	entry.Catalog = strings.TrimSpace(entry.Catalog)
	entry.Code = strings.TrimSpace(entry.Code)
	entry.Label = strings.TrimSpace(entry.Label)
	entry.State = strings.TrimSpace(entry.State)
	if entry.State == "" {
		entry.State = "Vigente"
	}
	return entry
}
