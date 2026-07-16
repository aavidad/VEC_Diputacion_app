package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	personalmemory "vec-diputacion-granada/internal/modules/personal/adapters/memory"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

const catalogStoreSchemaVersion = 1

type CatalogStore struct {
	path                 string
	mu                   sync.Mutex
	memory               *personalmemory.CatalogStore
	categoriasHistoricas json.RawMessage
}

type catalogSnapshot struct {
	SchemaVersion int                   `json:"schema_version"`
	SavedAt       time.Time             `json:"saved_at"`
	Positions     []domain.RPTPosition  `json:"positions"`
	Categories    json.RawMessage       `json:"categories"`
	Catalogs      []domain.CatalogEntry `json:"catalogs"`
}

func NewCatalogStore(path string) (*CatalogStore, error) {
	if path == "" {
		return nil, errors.New("personal catalog durable path required")
	}
	store := &CatalogStore{
		path:   path,
		memory: personalmemory.NewCatalogStore(),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *CatalogStore) UpsertPosition(ctx context.Context, position domain.RPTPosition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.memory.UpsertPosition(ctx, position); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *CatalogStore) GetPosition(ctx context.Context, code string) (domain.RPTPosition, bool, error) {
	return s.memory.GetPosition(ctx, code)
}

func (s *CatalogStore) DeletePosition(ctx context.Context, code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted, err := s.memory.DeletePosition(ctx, code)
	if err != nil || !deleted {
		return deleted, err
	}
	return deleted, s.persistLocked(ctx)
}

func (s *CatalogStore) ListPositions(ctx context.Context, filter domain.RPTPositionFilter) (domain.RPTPositionPage, error) {
	return s.memory.ListPositions(ctx, filter)
}

func (s *CatalogStore) ImportPositions(ctx context.Context, cmd domain.RPTImportCommand) (domain.RPTImportReceipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, err := s.memory.ImportPositions(ctx, cmd)
	if err != nil {
		return domain.RPTImportReceipt{}, err
	}
	return receipt, s.persistLocked(ctx)
}

func (s *CatalogStore) UpsertCatalogEntry(ctx context.Context, entry domain.CatalogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.memory.UpsertCatalogEntry(ctx, entry); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *CatalogStore) ListCatalogEntries(ctx context.Context) ([]domain.CatalogEntry, error) {
	return s.memory.ListCatalogEntries(ctx)
}

func (s *CatalogStore) Stats(ctx context.Context) (domain.CatalogStats, error) {
	return s.memory.Stats(ctx)
}

func (s *CatalogStore) load() error {
	snapshot, err := readCatalogSnapshot(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		if backup, backupErr := readCatalogSnapshot(catalogBackupPath(s.path)); backupErr == nil {
			return s.applySnapshot(context.Background(), backup)
		}
		return err
	}
	return s.applySnapshot(context.Background(), snapshot)
}

func readCatalogSnapshot(path string) (catalogSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return catalogSnapshot{}, fmt.Errorf("read personal catalog %q: %w", path, err)
	}
	var snapshot catalogSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return catalogSnapshot{}, fmt.Errorf("decode personal catalog %q: %w", path, err)
	}
	if snapshot.SchemaVersion != catalogStoreSchemaVersion {
		return catalogSnapshot{}, fmt.Errorf("personal catalog schema %d is unsupported", snapshot.SchemaVersion)
	}
	return snapshot, nil
}

func (s *CatalogStore) applySnapshot(ctx context.Context, snapshot catalogSnapshot) error {
	if _, err := s.memory.ImportPositions(ctx, domain.RPTImportCommand{
		Source:    s.path,
		Version:   fmt.Sprintf("schema-%d", snapshot.SchemaVersion),
		Replace:   true,
		Positions: snapshot.Positions,
	}); err != nil {
		return err
	}
	// Categories pertenecia al esquema historico de este snapshot. Se valida una
	// proyeccion tipada, pero se conserva el subarbol JSON original para que una
	// escritura de RPT no elimine extensiones heredadas desconocidas. Nunca se
	// carga como autoridad consultable ni admite mutaciones nuevas.
	categorias, err := decodificarCategoriasHistoricas(snapshot.Categories)
	if err != nil {
		return err
	}
	for _, category := range categorias {
		if err := category.Validate(); err != nil {
			return err
		}
	}
	s.categoriasHistoricas = clonarMensajeJSON(snapshot.Categories)
	for _, entry := range snapshot.Catalogs {
		if err := s.memory.UpsertCatalogEntry(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *CatalogStore) persistLocked(ctx context.Context) error {
	snapshot, err := s.snapshot(ctx)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode personal catalog: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create personal catalog dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create personal catalog temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write personal catalog temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync personal catalog temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close personal catalog temp file: %w", err)
	}
	if err := s.copyBackup(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace personal catalog file: %w", err)
	}
	return syncDir(dir)
}

func (s *CatalogStore) snapshot(ctx context.Context) (catalogSnapshot, error) {
	positions, err := s.memory.ListPositions(ctx, domain.RPTPositionFilter{Limit: maxSnapshotRows})
	if err != nil {
		return catalogSnapshot{}, err
	}
	catalogs, err := s.memory.ListCatalogEntries(ctx)
	if err != nil {
		return catalogSnapshot{}, err
	}
	return catalogSnapshot{
		SchemaVersion: catalogStoreSchemaVersion,
		SavedAt:       time.Now().UTC(),
		Positions:     positions.Items,
		Categories:    clonarMensajeJSON(s.categoriasHistoricas),
		Catalogs:      catalogs,
	}, nil
}

func decodificarCategoriasHistoricas(contenido json.RawMessage) ([]domain.ProfessionalCategory, error) {
	if len(contenido) == 0 {
		return nil, nil
	}
	var categorias []domain.ProfessionalCategory
	if err := json.Unmarshal(contenido, &categorias); err != nil {
		return nil, fmt.Errorf("decode personal legacy categories: %w", err)
	}
	return categorias, nil
}

func clonarMensajeJSON(contenido json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), contenido...)
}

func (s *CatalogStore) copyBackup() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read personal catalog backup source: %w", err)
	}
	if !json.Valid(data) {
		return nil
	}
	if err := os.WriteFile(catalogBackupPath(s.path), data, 0o600); err != nil {
		return fmt.Errorf("write personal catalog backup: %w", err)
	}
	return nil
}

func catalogBackupPath(path string) string {
	return path + ".bak"
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open dir for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync dir: %w", err)
	}
	return nil
}

const maxSnapshotRows = int(^uint(0) >> 2)
