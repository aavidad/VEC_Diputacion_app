package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

type CatalogStore struct {
	mu         sync.RWMutex
	positions  map[string]domain.RPTPosition
	catalogs   map[string]domain.CatalogEntry
	lastImport domain.RPTImportReceipt
}

func NewCatalogStore() *CatalogStore {
	return &CatalogStore{
		positions: map[string]domain.RPTPosition{},
		catalogs:  map[string]domain.CatalogEntry{},
	}
}

func (s *CatalogStore) UpsertPosition(ctx context.Context, position domain.RPTPosition) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := position.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLocked()
	s.positions[position.Code] = position
	return nil
}

func (s *CatalogStore) GetPosition(ctx context.Context, code string) (domain.RPTPosition, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.RPTPosition{}, false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	position, ok := s.positions[strings.TrimSpace(code)]
	return position, ok, nil
}

func (s *CatalogStore) DeletePosition(ctx context.Context, code string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	code = strings.TrimSpace(code)
	if _, ok := s.positions[code]; !ok {
		return false, nil
	}
	delete(s.positions, code)
	return true, nil
}

func (s *CatalogStore) ListPositions(ctx context.Context, filter domain.RPTPositionFilter) (domain.RPTPositionPage, error) {
	if err := ctx.Err(); err != nil {
		return domain.RPTPositionPage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.RPTPosition, 0, len(s.positions))
	for _, position := range s.positions {
		if matchesPosition(position, filter) {
			items = append(items, position)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return naturalPositionLess(items[i].Code, items[j].Code)
	})
	total := len(items)
	items = pagePositions(items, filter.Offset, filter.Limit)
	return domain.RPTPositionPage{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (s *CatalogStore) ImportPositions(ctx context.Context, cmd domain.RPTImportCommand) (domain.RPTImportReceipt, error) {
	if err := ctx.Err(); err != nil {
		return domain.RPTImportReceipt{}, err
	}
	for _, position := range cmd.Positions {
		if err := position.Validate(); err != nil {
			return domain.RPTImportReceipt{}, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLocked()
	if cmd.Replace {
		s.positions = map[string]domain.RPTPosition{}
	}
	for _, position := range cmd.Positions {
		s.positions[position.Code] = position
	}
	receipt := domain.RPTImportReceipt{
		Source:   cmd.Source,
		Version:  cmd.Version,
		Imported: len(cmd.Positions),
		Replaced: cmd.Replace,
	}
	s.lastImport = receipt
	return receipt, nil
}

func (s *CatalogStore) UpsertCatalogEntry(ctx context.Context, entry domain.CatalogEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLocked()
	s.catalogs[entry.Catalog+"|"+entry.Code] = entry
	return nil
}

func (s *CatalogStore) ListCatalogEntries(ctx context.Context) ([]domain.CatalogEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.CatalogEntry, 0, len(s.catalogs))
	for _, entry := range s.catalogs {
		items = append(items, entry)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Catalog == items[j].Catalog {
			return items[i].Code < items[j].Code
		}
		return items[i].Catalog < items[j].Catalog
	})
	return items, nil
}

func (s *CatalogStore) Stats(ctx context.Context) (domain.CatalogStats, error) {
	if err := ctx.Err(); err != nil {
		return domain.CatalogStats{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := domain.CatalogStats{
		Positions:        len(s.positions),
		CatalogEntries:   len(s.catalogs),
		PositionsByGroup: map[string]int{},
		CategoriesByArea: map[string]int{},
	}
	for _, position := range s.positions {
		group := valueOr(position.Group, "sin_grupo")
		stats.PositionsByGroup[group]++
		if strings.Contains(strings.ToLower(position.State), "pendiente leyenda") {
			stats.PendingLegend++
		}
	}
	return stats, nil
}

func (s *CatalogStore) ensureLocked() {
	if s.positions == nil {
		s.positions = map[string]domain.RPTPosition{}
	}
	if s.catalogs == nil {
		s.catalogs = map[string]domain.CatalogEntry{}
	}
}

func matchesPosition(position domain.RPTPosition, filter domain.RPTPositionFilter) bool {
	if filter.Group != "" && !strings.EqualFold(position.Group, filter.Group) {
		return false
	}
	if filter.CenterCode != "" && !strings.EqualFold(position.CenterCode, filter.CenterCode) {
		return false
	}
	if filter.Provision != "" && !strings.EqualFold(position.Provision, filter.Provision) {
		return false
	}
	if filter.State != "" && !strings.Contains(strings.ToLower(position.State), strings.ToLower(filter.State)) {
		return false
	}
	if filter.Query != "" {
		q := strings.ToLower(filter.Query)
		haystack := strings.ToLower(strings.Join([]string{
			position.Code, position.Name, position.Group, position.Area,
			position.Scale, position.CategoryCode, position.CategorySlug,
			position.Delegation, position.CenterCode, position.CenterName,
			position.Coverage, position.Observations,
		}, " "))
		return strings.Contains(haystack, q)
	}
	return true
}

func pagePositions(items []domain.RPTPosition, offset, limit int) []domain.RPTPosition {
	if offset >= len(items) {
		return []domain.RPTPosition{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]domain.RPTPosition(nil), items[offset:end]...)
}

func naturalPositionLess(a, b string) bool {
	ai, aok := atoi(a)
	bi, bok := atoi(b)
	if aok && bok {
		return ai < bi
	}
	return a < b
}

func atoi(value string) (int, bool) {
	var out int
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
		out = out*10 + int(r-'0')
	}
	return out, true
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
