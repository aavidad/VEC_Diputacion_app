package memory

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestCatalogStoreListsFiltersAndStats(t *testing.T) {
	store := NewCatalogStore()
	ctx := context.Background()
	for _, position := range []domain.RPTPosition{
		{Code: "10", Name: "Administrativo", Dot: 1, Group: "C1", Provision: "C", State: "Importado demo"},
		{Code: "2", Name: "Jefatura Servicio", Dot: 1, Group: "A1", Provision: "L", State: "Pendiente leyenda RPT"},
	} {
		if err := store.UpsertPosition(ctx, position); err != nil {
			t.Fatalf("UpsertPosition() error = %v", err)
		}
	}
	if err := store.UpsertCategory(ctx, domain.ProfessionalCategory{Slug: "administrativo", Name: "Administrativo", Area: "administracion_general"}); err != nil {
		t.Fatalf("UpsertCategory() error = %v", err)
	}
	page, err := store.ListPositions(ctx, domain.RPTPositionFilter{Query: "admin", Limit: 10})
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if page.Total != 1 || page.Items[0].Code != "10" {
		t.Fatalf("ListPositions() = %#v, want Administrativo", page)
	}
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Positions != 2 || stats.Categories != 1 || stats.PendingLegend != 1 || stats.PositionsByGroup["A1"] != 1 {
		t.Fatalf("Stats() = %#v, want grouped counts", stats)
	}
}

func TestCatalogStoreImportReplace(t *testing.T) {
	store := NewCatalogStore()
	ctx := context.Background()
	if err := store.UpsertPosition(ctx, domain.RPTPosition{Code: "old", Name: "Old", Dot: 1}); err != nil {
		t.Fatalf("UpsertPosition() error = %v", err)
	}
	receipt, err := store.ImportPositions(ctx, domain.RPTImportCommand{
		Source:  "test",
		Version: "v1",
		Replace: true,
		Positions: []domain.RPTPosition{
			{Code: "new", Name: "New", Dot: 1},
		},
	})
	if err != nil {
		t.Fatalf("ImportPositions() error = %v", err)
	}
	if receipt.Imported != 1 || !receipt.Replaced {
		t.Fatalf("ImportPositions() receipt = %#v", receipt)
	}
	if _, ok, err := store.GetPosition(ctx, "old"); err != nil || ok {
		t.Fatalf("old position = ok %v err %v, want removed", ok, err)
	}
	if _, ok, err := store.GetPosition(ctx, "new"); err != nil || !ok {
		t.Fatalf("new position = ok %v err %v, want stored", ok, err)
	}
}

func TestCatalogStoreDeletesPositionsAndCategories(t *testing.T) {
	store := NewCatalogStore()
	ctx := context.Background()
	if err := store.UpsertPosition(ctx, domain.RPTPosition{Code: "1", Name: "Puesto", Dot: 1}); err != nil {
		t.Fatalf("UpsertPosition() error = %v", err)
	}
	deleted, err := store.DeletePosition(ctx, "1")
	if err != nil || !deleted {
		t.Fatalf("DeletePosition() = %v, %v; want deleted", deleted, err)
	}
	if _, ok, err := store.GetPosition(ctx, "1"); err != nil || ok {
		t.Fatalf("GetPosition() after delete = ok %v err %v", ok, err)
	}
	if err := store.UpsertCategory(ctx, domain.ProfessionalCategory{Slug: "cat", Name: "Categoria"}); err != nil {
		t.Fatalf("UpsertCategory() error = %v", err)
	}
	deleted, err = store.DeleteCategory(ctx, "cat")
	if err != nil || !deleted {
		t.Fatalf("DeleteCategory() = %v, %v; want deleted", deleted, err)
	}
	if _, ok, err := store.GetCategory(ctx, "cat"); err != nil || ok {
		t.Fatalf("GetCategory() after delete = ok %v err %v", ok, err)
	}
}
