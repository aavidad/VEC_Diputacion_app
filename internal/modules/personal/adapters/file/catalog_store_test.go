package file

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestCatalogStorePersisteRPTYCatalogosYConservaCategoriasHistoricasInertes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "personal_catalog.json")
	ctx := context.Background()
	historica := domain.ProfessionalCategory{
		Slug: "cat-durable", Name: "Categoria durable", Area: "administracion_general",
		Source: "inventario-historico", SourcePath: "legado/cat-durable", ModuleKey: "personal",
		State: "vigente", Usage: "Compatibilidad del snapshot v1",
	}
	categoriasHistoricas := json.RawMessage(`[{"slug":"cat-durable","name":"Categoria durable","area":"administracion_general","source":"inventario-historico","source_path":"legado/cat-durable","module_key":"personal","state":"vigente","usage":"Compatibilidad del snapshot v1","extension_legacy":{"revision":7,"coeficiente":1.2300,"etiquetas":["a","b"]}}]`)
	contenido, err := json.Marshal(catalogSnapshot{
		SchemaVersion: catalogStoreSchemaVersion,
		SavedAt:       time.Now().UTC(),
		Categories:    categoriasHistoricas,
	})
	if err != nil {
		t.Fatalf("codificar snapshot historico: %v", err)
	}
	if err := os.WriteFile(path, contenido, 0o600); err != nil {
		t.Fatalf("escribir snapshot historico: %v", err)
	}
	store, err := NewCatalogStore(path)
	if err != nil {
		t.Fatalf("NewCatalogStore() error = %v", err)
	}
	if err := store.UpsertPosition(ctx, domain.RPTPosition{Code: "100", Name: "Puesto durable", State: "vigente", Dot: 1, Group: "A1"}); err != nil {
		t.Fatalf("UpsertPosition() error = %v", err)
	}
	comprobarCategoriasHistoricasPreservadas(t, path, categoriasHistoricas, historica)
	if err := store.UpsertCatalogEntry(ctx, domain.CatalogEntry{Catalog: "rpt.provision", Code: "C", Label: "Concurso", State: "vigente"}); err != nil {
		t.Fatalf("UpsertCatalogEntry() error = %v", err)
	}
	comprobarCategoriasHistoricasPreservadas(t, path, categoriasHistoricas, historica)
	comprobarCategoriasHistoricasPreservadas(t, catalogBackupPath(path), categoriasHistoricas, historica)

	reopened, err := NewCatalogStore(path)
	if err != nil {
		t.Fatalf("NewCatalogStore(reopen) error = %v", err)
	}
	if position, ok, err := reopened.GetPosition(ctx, "100"); err != nil || !ok || position.Name != "Puesto durable" {
		t.Fatalf("GetPosition(reopen) = %#v, %v, %v", position, ok, err)
	}
	catalogs, err := reopened.ListCatalogEntries(ctx)
	if err != nil || len(catalogs) != 1 || catalogs[0].Code != "C" {
		t.Fatalf("ListCatalogEntries(reopen) = %#v, %v", catalogs, err)
	}
	stats, err := reopened.Stats(ctx)
	if err != nil || stats.Categories != 0 {
		t.Fatalf("Stats() categorias = %d, error = %v; el legado no debe ser autoridad", stats.Categories, err)
	}
}

func comprobarCategoriasHistoricasPreservadas(
	t *testing.T,
	path string,
	esperadasJSON json.RawMessage,
	esperada domain.ProfessionalCategory,
) {
	t.Helper()
	persistido, err := readCatalogSnapshot(path)
	if err != nil {
		t.Fatalf("releer snapshot %q: %v", path, err)
	}
	var categorias []domain.ProfessionalCategory
	if err := json.Unmarshal(persistido.Categories, &categorias); err != nil {
		t.Fatalf("decodificar categorias historicas de %q: %v", path, err)
	}
	if len(categorias) != 1 || categorias[0] != esperada {
		t.Fatalf("categorias historicas alteradas en %q: %#v", path, categorias)
	}

	var esperadasCompactas, persistidasCompactas bytes.Buffer
	if err := json.Compact(&esperadasCompactas, esperadasJSON); err != nil {
		t.Fatalf("compactar precondicion historica: %v", err)
	}
	if err := json.Compact(&persistidasCompactas, persistido.Categories); err != nil {
		t.Fatalf("compactar categorias persistidas: %v", err)
	}
	if !bytes.Equal(persistidasCompactas.Bytes(), esperadasCompactas.Bytes()) {
		t.Fatalf("subarbol historico alterado en %q:\nobtenido: %s\nesperado: %s", path, persistidasCompactas.Bytes(), esperadasCompactas.Bytes())
	}

	var categoriasExtendidas []map[string]json.RawMessage
	if err := json.Unmarshal(persistido.Categories, &categoriasExtendidas); err != nil {
		t.Fatalf("leer extensiones historicas: %v", err)
	}
	if len(categoriasExtendidas) != 1 || len(categoriasExtendidas[0]["extension_legacy"]) == 0 {
		t.Fatalf("extension heredada perdida en %q: %#v", path, categoriasExtendidas)
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
