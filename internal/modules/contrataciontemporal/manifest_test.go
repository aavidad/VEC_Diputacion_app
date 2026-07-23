package contrataciontemporal

import (
	"strings"
	"testing"
)

func TestManifestRegistraModuloSinConcederPermisosImplicitos(t *testing.T) {
	manifiesto := Manifest()
	if manifiesto.ID != ModuleID || manifiesto.Version != "v0.2.0" {
		t.Fatalf("manifiesto inesperado: %#v", manifiesto)
	}
	if len(manifiesto.Permissions) != 25 || len(manifiesto.Menu) != 6 {
		t.Fatalf("capacidades=%d menú=%d", len(manifiesto.Permissions), len(manifiesto.Menu))
	}
	vistas := make(map[string]struct{}, len(manifiesto.Permissions))
	for _, permiso := range manifiesto.Permissions {
		if permiso.Key == "contratacion_temporal.expediente.gestionar" ||
			strings.Contains(permiso.Key, "*") {
			t.Fatalf("capacidad genérica prohibida: %q", permiso.Key)
		}
		if _, repetida := vistas[permiso.Key]; repetida {
			t.Fatalf("capacidad repetida: %q", permiso.Key)
		}
		vistas[permiso.Key] = struct{}{}
	}
	for _, entrada := range manifiesto.Menu {
		if entrada.ModuleID != ModuleID || len(entrada.RequiredPermissions) != 1 ||
			!strings.HasPrefix(entrada.RequiredPermissions[0], "contratacion_temporal.") {
			t.Fatalf("entrada no cerrada: %#v", entrada)
		}
	}
}
