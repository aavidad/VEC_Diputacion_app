package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestCatalogStorePersistsCategoriesPositionsAndCatalogs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "personal_catalog.json")
	ctx := context.Background()
	store, err := NewCatalogStore(path)
	if err != nil {
		t.Fatalf("NewCatalogStore() error = %v", err)
	}
	if err := store.UpsertPosition(ctx, domain.RPTPosition{Code: "100", Name: "Puesto durable", State: "vigente", Dot: 1, Group: "A1"}); err != nil {
		t.Fatalf("UpsertPosition() error = %v", err)
	}
	if err := store.UpsertCategory(ctx, domain.ProfessionalCategory{Slug: "cat-durable", Name: "Categoria durable", Area: "administracion_general", State: "vigente"}); err != nil {
		t.Fatalf("UpsertCategory() error = %v", err)
	}
	if err := store.UpsertCatalogEntry(ctx, domain.CatalogEntry{Catalog: "rpt.provision", Code: "C", Label: "Concurso", State: "vigente"}); err != nil {
		t.Fatalf("UpsertCatalogEntry() error = %v", err)
	}

	reopened, err := NewCatalogStore(path)
	if err != nil {
		t.Fatalf("NewCatalogStore(reopen) error = %v", err)
	}
	if position, ok, err := reopened.GetPosition(ctx, "100"); err != nil || !ok || position.Name != "Puesto durable" {
		t.Fatalf("GetPosition(reopen) = %#v, %v, %v", position, ok, err)
	}
	if category, ok, err := reopened.GetCategory(ctx, "cat-durable"); err != nil || !ok || category.Name != "Categoria durable" {
		t.Fatalf("GetCategory(reopen) = %#v, %v, %v", category, ok, err)
	}
	catalogs, err := reopened.ListCatalogEntries(ctx)
	if err != nil || len(catalogs) != 1 || catalogs[0].Code != "C" {
		t.Fatalf("ListCatalogEntries(reopen) = %#v, %v", catalogs, err)
	}
}

func TestCatalogStorePersistsDeletesAndUsesBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "personal_catalog.json")
	ctx := context.Background()
	store, err := NewCatalogStore(path)
	if err != nil {
		t.Fatalf("NewCatalogStore() error = %v", err)
	}
	if err := store.UpsertPosition(ctx, domain.RPTPosition{Code: "100", Name: "Puesto durable", State: "vigente", Dot: 1}); err != nil {
		t.Fatalf("UpsertPosition() error = %v", err)
	}
	if err := store.UpsertPosition(ctx, domain.RPTPosition{Code: "200", Name: "Puesto backup", State: "vigente", Dot: 1}); err != nil {
		t.Fatalf("UpsertPosition(second) error = %v", err)
	}
	deleted, err := store.DeletePosition(ctx, "100")
	if err != nil || !deleted {
		t.Fatalf("DeletePosition() = %v, %v; want deleted", deleted, err)
	}
	reopened, err := NewCatalogStore(path)
	if err != nil {
		t.Fatalf("NewCatalogStore(reopen) error = %v", err)
	}
	if _, ok, err := reopened.GetPosition(ctx, "100"); err != nil || ok {
		t.Fatalf("deleted position after reopen = ok %v err %v", ok, err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":`), 0o600); err != nil {
		t.Fatalf("corrupt primary: %v", err)
	}
	recovered, err := NewCatalogStore(path)
	if err != nil {
		t.Fatalf("NewCatalogStore(corrupt primary) error = %v", err)
	}
	if _, ok, err := recovered.GetPosition(ctx, "200"); err != nil || !ok {
		t.Fatalf("backup recovery position = ok %v err %v", ok, err)
	}
}
