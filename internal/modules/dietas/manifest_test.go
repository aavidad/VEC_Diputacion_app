package dietas

import (
	"encoding/json"
	"os"
	"testing"
)

func TestManifestRegistersDietasAsVECModule(t *testing.T) {
	manifest := Manifest()
	if manifest.ID != ModuleID {
		t.Fatalf("module id = %q, want %q", manifest.ID, ModuleID)
	}
	if manifest.Group != "gestion_gastos" {
		t.Fatalf("module group = %q, want gestion_gastos", manifest.Group)
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("manifiesto Dietas inválido: %v", err)
	}
	if manifest.BasePath != "/modules/dietas" || len(manifest.Menu) != 0 {
		t.Fatal("el manifiesto publicó enlaces de Dietas sin ruta web real")
	}
	esperados := map[string]bool{
		PermissionDraftCreate: true, PermissionDraftRead: true, PermissionDraftEdit: true,
		PermissionDraftDelete: true, PermissionDraftSend: true, PermissionDocumentRead: true,
		PermissionRouteCatalog: true, PermissionRouteCalculate: true,
		PermissionReview: true, PermissionAuthorize: true, PermissionLiquidate: true,
		PermissionAuditDecision: true, PermissionReviewInbox: true,
		PermissionAuthorizationInbox: true, PermissionLiquidationInbox: true,
		PermissionAuditInbox: true,
	}
	if len(manifest.Permissions) != len(esperados) {
		t.Fatalf("permisos Dietas = %d; se esperaban %d", len(manifest.Permissions), len(esperados))
	}
	for _, permiso := range manifest.Permissions {
		if !esperados[permiso.Key] {
			t.Fatalf("permiso ajeno o legado publicado: %q", permiso.Key)
		}
		delete(esperados, permiso.Key)
	}
	if len(esperados) != 0 {
		t.Fatalf("faltan permisos nominales: %#v", esperados)
	}
	bruto, err := os.ReadFile("../../../locales/es.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalogo map[string]string
	if err := json.Unmarshal(bruto, &catalogo); err != nil {
		t.Fatal(err)
	}
	for _, permiso := range manifest.Permissions {
		if catalogo[permiso.LabelKey] == "" {
			t.Fatalf("etiqueta castellana ausente para %q", permiso.Key)
		}
	}
}
