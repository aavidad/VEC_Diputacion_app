package contrataciontemporal

import (
	"strings"
	"testing"
)

func TestManifestRegistraModuloSinConcederPermisosImplicitos(t *testing.T) {
	manifiesto := Manifest()
	if manifiesto.ID != ModuleID || manifiesto.Version != "v0.1.0" {
		t.Fatalf("manifiesto inesperado: %#v", manifiesto)
	}
	if len(manifiesto.Permissions) != 8 || len(manifiesto.Menu) != 5 {
		t.Fatalf("capacidades=%d menú=%d", len(manifiesto.Permissions), len(manifiesto.Menu))
	}
	for _, entrada := range manifiesto.Menu {
		if entrada.ModuleID != ModuleID || len(entrada.RequiredPermissions) != 1 ||
			!strings.HasPrefix(entrada.RequiredPermissions[0], "contratacion_temporal.") {
			t.Fatalf("entrada no cerrada: %#v", entrada)
		}
	}
}
