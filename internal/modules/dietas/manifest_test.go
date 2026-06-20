package dietas

import (
	"strings"
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
	if len(manifest.Menu) < 6 {
		t.Fatalf("manifest menu has %d entries, want at least 6", len(manifest.Menu))
	}
	for _, entry := range manifest.Menu {
		if entry.ModuleID != ModuleID {
			t.Fatalf("entry %s module = %q", entry.ID, entry.ModuleID)
		}
		if len(entry.RequiredPermissions) == 0 || !strings.HasPrefix(entry.RequiredPermissions[0], "dietas.") {
			t.Fatalf("entry %s permissions = %#v, want dietas.*", entry.ID, entry.RequiredPermissions)
		}
	}
}
