package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
)

// cargarFixturesCatalogoPersonalPrueba mantiene los datos demostrativos fuera
// del arranque productivo. Cada prueba que usa newTestHandler los carga de
// forma expresa sobre su almacen aislado.
func cargarFixturesCatalogoPersonalPrueba(t *testing.T, service *personalapp.CatalogService) {
	t.Helper()
	if service == nil {
		t.Fatal("servicio de catalogo personal de prueba requerido")
	}

	command := personaldomain.RPTImportCommand{
		Source:    "fixture sintetico en memoria",
		Version:   "prueba-v1",
		Replace:   true,
		Positions: posicionesRPTSinteticasPrueba(),
	}
	if _, err := service.ImportPositions(context.Background(), command); err != nil {
		t.Fatalf("importar fixture RPT: %v", err)
	}

	for _, raw := range workspaceProfessionalCategories() {
		if err := service.UpsertCategory(context.Background(), professionalCategoryFromMap(raw)); err != nil {
			t.Fatalf("cargar categoria profesional de prueba: %v", err)
		}
	}
	for _, raw := range workspaceRPTContractTypes() {
		if err := service.UpsertCatalogEntry(context.Background(), catalogEntryFromMap(raw)); err != nil {
			t.Fatalf("cargar entrada de catalogo de prueba: %v", err)
		}
	}
}

// posicionesRPTSinteticasPrueba contiene solo los registros necesarios para
// probar filtros, lectura y estadísticas. No deriva de la RPT real ni contiene
// datos de personas, unidades o importes de producción.
func posicionesRPTSinteticasPrueba() []personaldomain.RPTPosition {
	return []personaldomain.RPTPosition{
		{
			Code: "8", Name: "Administrativo/a sintético", Dot: 2, Group: "C1",
			Provision: "C", State: "Vigente", Source: "fixture sintético",
		},
		{
			Code: "118-DEMO", Name: "Puesto técnico sintético", Dot: 1, Group: "A1",
			Provision: "L", State: "Vigente", Source: "fixture sintético",
		},
		{
			Code: "344-DEMO", Name: "Puesto de gestión sintético", Dot: 1, Group: "A2",
			Provision: "C", State: "Vigente", Source: "fixture sintético",
		},
	}
}

func TestArranqueCatalogoPersonalConRutaDurableVaciaNoCreaFichero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalogo", "personal.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("precondicion: el snapshot no debe existir: %v", err)
	}

	service, err := newWorkspacePersonalCatalogService(path)
	if err != nil {
		t.Fatalf("arrancar catalogo vacio: %v", err)
	}
	stats, err := service.Stats(context.Background())
	if err != nil {
		t.Fatalf("consultar catalogo vacio: %v", err)
	}
	if stats.Positions != 0 || stats.Categories != 0 || stats.CatalogEntries != 0 {
		t.Fatalf("el arranque creo datos implicitos: %+v", stats)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("el arranque creo o modifico el snapshot vacio: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("el arranque creo una copia de respaldo implicita: %v", err)
	}
}

func TestArranqueCatalogoPersonalIgnoraImportacionHostilDeEntorno(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "personal.json")
	service, err := newWorkspacePersonalCatalogService(path)
	if err != nil {
		t.Fatalf("crear catalogo durable: %v", err)
	}
	original := personaldomain.RPTPosition{
		Code: "RPT-ORIGINAL-001", Name: "Puesto original autorizado", Dot: 1,
		State: "Vigente", Source: "prueba de snapshot preexistente",
	}
	if _, err := service.UpsertPosition(ctx, original); err != nil {
		t.Fatalf("preparar snapshot preexistente: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer snapshot preexistente: %v", err)
	}
	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat snapshot preexistente: %v", err)
	}

	hostilePath := filepath.Join(dir, "replace-hostil.json")
	hostile := personaldomain.RPTImportCommand{
		Source:  "entrada de entorno no autorizada",
		Version: "hostil-v1",
		Replace: true,
		Positions: []personaldomain.RPTPosition{{
			Code: "RPT-HOSTIL-999", Name: "Sustitucion hostil", Dot: 1,
			State: "Vigente", Source: "entorno",
		}},
	}
	hostileData, err := json.Marshal(hostile)
	if err != nil {
		t.Fatalf("codificar replace hostil: %v", err)
	}
	if err := os.WriteFile(hostilePath, hostileData, 0o600); err != nil {
		t.Fatalf("escribir replace hostil: %v", err)
	}
	t.Setenv("VEC_RPT_IMPORT_JSON", hostilePath)

	reopened, err := newWorkspacePersonalCatalogService(path)
	if err != nil {
		t.Fatalf("reabrir snapshot con entorno hostil: %v", err)
	}
	stored, err := reopened.GetPosition(ctx, original.Code)
	if err != nil || stored.Name != original.Name {
		t.Fatalf("el arranque no preservo el registro original: puesto=%+v error=%v", stored, err)
	}
	if _, err := reopened.GetPosition(ctx, "RPT-HOSTIL-999"); err == nil {
		t.Fatal("el arranque aplico el replace indicado por el entorno")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("releer snapshot: %v", err)
	}
	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat snapshot tras arranque: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("el arranque modifico el contenido del snapshot preexistente")
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) || afterInfo.Mode() != beforeInfo.Mode() || afterInfo.Size() != beforeInfo.Size() {
		t.Fatalf("el arranque modifico metadatos del snapshot: antes=%+v despues=%+v", beforeInfo, afterInfo)
	}
}
