package administracion

import (
	"strings"
	"testing"
)

func TestManifestRegistersAdministracionAsVECModule(t *testing.T) {
	manifest := Manifest()
	if manifest.ID != ModuleID {
		t.Fatalf("module id = %q, want %q", manifest.ID, ModuleID)
	}
	if manifest.Group != "administracion_vec" {
		t.Fatalf("module group = %q, want administracion_vec", manifest.Group)
	}
	if len(manifest.Menu) < 4 {
		t.Fatalf("manifest menu has %d entries, want at least 4", len(manifest.Menu))
	}
	var hasCatalogs bool
	for _, entry := range manifest.Menu {
		if entry.ModuleID != ModuleID {
			t.Fatalf("entry %s module = %q", entry.ID, entry.ModuleID)
		}
		if len(entry.RequiredPermissions) == 0 || !strings.HasPrefix(entry.RequiredPermissions[0], "vec.") {
			t.Fatalf("entry %s permissions = %#v, want vec.*", entry.ID, entry.RequiredPermissions)
		}
		if entry.ID == "admin.catalogos" {
			hasCatalogs = true
		}
	}
	if !hasCatalogs {
		t.Fatalf("manifest menu missing admin.catalogos: %#v", manifest.Menu)
	}
}
