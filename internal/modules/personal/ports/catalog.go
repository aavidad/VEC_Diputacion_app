package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

type CatalogStore interface {
	UpsertPosition(context.Context, domain.RPTPosition) error
	GetPosition(context.Context, string) (domain.RPTPosition, bool, error)
	DeletePosition(context.Context, string) (bool, error)
	ListPositions(context.Context, domain.RPTPositionFilter) (domain.RPTPositionPage, error)
	ImportPositions(context.Context, domain.RPTImportCommand) (domain.RPTImportReceipt, error)

	UpsertCategory(context.Context, domain.ProfessionalCategory) error
	GetCategory(context.Context, string) (domain.ProfessionalCategory, bool, error)
	DeleteCategory(context.Context, string) (bool, error)
	ListCategories(context.Context, domain.ProfessionalCategoryFilter) (domain.ProfessionalCategoryPage, error)
	UpsertCatalogEntry(context.Context, domain.CatalogEntry) error
	ListCatalogEntries(context.Context) ([]domain.CatalogEntry, error)
	Stats(context.Context) (domain.CatalogStats, error)
}
