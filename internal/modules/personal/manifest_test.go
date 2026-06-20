package personal

import (
	"strings"
	"testing"
)

func TestManifestRegistersPersonalAsVECModule(t *testing.T) {
	manifest := Manifest()
	if manifest.ID != ModuleID {
		t.Fatalf("module id = %q, want %q", manifest.ID, ModuleID)
	}
	if manifest.Group != "gestion_personal" {
		t.Fatalf("module group = %q, want gestion_personal", manifest.Group)
	}
	if len(manifest.Menu) < 7 {
		t.Fatalf("manifest menu has %d entries, want at least 7", len(manifest.Menu))
	}
	for _, entry := range manifest.Menu {
		if entry.ModuleID != ModuleID {
			t.Fatalf("entry %s module = %q", entry.ID, entry.ModuleID)
		}
		if len(entry.RequiredPermissions) == 0 || !strings.HasPrefix(entry.RequiredPermissions[0], "personal.") {
			t.Fatalf("entry %s permissions = %#v, want personal.*", entry.ID, entry.RequiredPermissions)
		}
	}
}
