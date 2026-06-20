package cronos

import (
	"strings"
	"testing"
)

func TestManifestRegistersCronosAsVECModule(t *testing.T) {
	manifest := Manifest()
	if manifest.ID != ModuleID {
		t.Fatalf("module id = %q, want %q", manifest.ID, ModuleID)
	}
	if manifest.Group != "gestion_tiempo" {
		t.Fatalf("module group = %q, want gestion_tiempo", manifest.Group)
	}
	if len(manifest.Menu) < 5 {
		t.Fatalf("manifest menu has %d entries, want at least 5", len(manifest.Menu))
	}
	for _, entry := range manifest.Menu {
		if entry.ModuleID != ModuleID {
			t.Fatalf("entry %s module = %q", entry.ID, entry.ModuleID)
		}
		if len(entry.RequiredPermissions) == 0 || !strings.HasPrefix(entry.RequiredPermissions[0], "cronos.") {
			t.Fatalf("entry %s permissions = %#v, want cronos.*", entry.ID, entry.RequiredPermissions)
		}
	}
}
