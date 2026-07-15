package application_test

import (
	"context"
	"errors"
	"testing"

	personalmemory "vec-diputacion-granada/internal/modules/personal/adapters/memory"
	"vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestCatalogServiceNoFabricaEstadoHabilitante(t *testing.T) {
	t.Parallel()

	store := personalmemory.NewCatalogStore()
	service, err := application.NewCatalogService(store)
	if err != nil {
		t.Fatalf("NewCatalogService() error = %v", err)
	}
	ctx := context.Background()

	for _, test := range []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "puesto sin estado",
			run: func() error {
				_, err := service.UpsertPosition(ctx, domain.RPTPosition{Code: "P-1", Name: "Puesto"})
				return err
			},
			want: domain.ErrRPTPositionInvalid,
		},
		{
			name: "puesto con estado no canonico",
			run: func() error {
				_, err := service.UpsertPosition(ctx, domain.RPTPosition{Code: "P-2", Name: "Puesto", State: " vigente "})
				return err
			},
			want: domain.ErrRPTPositionInvalid,
		},
		{
			name: "categoria sin estado",
			run: func() error {
				return service.UpsertCategory(ctx, domain.ProfessionalCategory{Slug: "categoria", Name: "Categoria"})
			},
			want: domain.ErrProfessionalCategoryInvalid,
		},
		{
			name: "entrada con estado no canonico",
			run: func() error {
				return service.UpsertCatalogEntry(ctx, domain.CatalogEntry{Catalog: "tipos", Code: "A", Label: "Alta", State: " vigente "})
			},
			want: domain.ErrCatalogEntryInvalid,
		},
		{
			name: "importacion con estado no canonico",
			run: func() error {
				_, err := service.ImportPositions(ctx, domain.RPTImportCommand{
					Source: "prueba", Version: "v1",
					Positions: []domain.RPTPosition{{Code: "P-3", Name: "Puesto", State: " vigente "}},
				})
				return err
			},
			want: domain.ErrRPTPositionInvalid,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, test.want) {
				t.Fatalf("operacion error = %v; esperado %v", err, test.want)
			}
		})
	}

	stats, err := service.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Positions != 0 || stats.Categories != 0 || stats.CatalogEntries != 0 {
		t.Fatalf("datos rechazados persistidos: %+v", stats)
	}
}
