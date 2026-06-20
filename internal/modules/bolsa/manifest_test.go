package bolsa

import (
	"strings"
	"testing"
)

func TestManifestRegistersBolsaAsVECModule(t *testing.T) {
	manifest := Manifest()
	if manifest.ID != ModuleID {
		t.Fatalf("module id = %q, want %q", manifest.ID, ModuleID)
	}
	if len(manifest.Menu) == 0 {
		t.Fatalf("manifest menu is empty")
	}
	for _, entry := range manifest.Menu {
		if entry.ModuleID != ModuleID {
			t.Fatalf("entry %s module = %q", entry.ID, entry.ModuleID)
		}
		if len(entry.RequiredPermissions) == 0 || !strings.HasPrefix(entry.RequiredPermissions[0], "bolsa.") {
			t.Fatalf("entry %s permissions = %#v, want bolsa.*", entry.ID, entry.RequiredPermissions)
		}
	}
}
