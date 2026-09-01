package contrataciontemporal

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestManifestRegistraModuloSinConcederPermisosImplicitos(t *testing.T) {
	manifiesto := Manifest()
	if manifiesto.ID != ModuleID || manifiesto.Version != "v0.3.0" {
		t.Fatalf("manifiesto inesperado: %#v", manifiesto)
	}
	if err := manifiesto.Validate(); err != nil {
		t.Fatalf("manifiesto inválido: %v", err)
	}
	if len(manifiesto.Permissions) != 26 || len(manifiesto.Menu) != 6 {
		t.Fatalf("capacidades=%d menú=%d", len(manifiesto.Permissions), len(manifiesto.Menu))
	}
	if PermisoAsignarUnidad != ports.AccionRegistrarAsignacion ||
		PermisoReasignarUnidad != ports.AccionRegistrarReasignacion ||
		PermisoAsignarUnidad == PermisoReasignarUnidad {
		t.Fatalf("capacidades de asignación discordantes: asignar=%q reasignar=%q", PermisoAsignarUnidad, PermisoReasignarUnidad)
	}
	vistas := make(map[string]struct{}, len(manifiesto.Permissions))
	apariciones := make(map[string]int, 2)
	for _, permiso := range manifiesto.Permissions {
		if permiso.Key == "contratacion_temporal.expediente.gestionar" ||
			strings.Contains(permiso.Key, "*") {
			t.Fatalf("capacidad genérica prohibida: %q", permiso.Key)
		}
		if _, repetida := vistas[permiso.Key]; repetida {
			t.Fatalf("capacidad repetida: %q", permiso.Key)
		}
		vistas[permiso.Key] = struct{}{}
		if permiso.Key == ports.AccionRegistrarAsignacion || permiso.Key == ports.AccionRegistrarReasignacion {
			apariciones[permiso.Key]++
		}
		if permiso.Key == ports.AccionRegistrarReasignacion &&
			permiso.LabelKey != "ui.permission.contratacion_temporal.reasignacion" {
			t.Fatalf("etiqueta de reasignación inesperada: %q", permiso.LabelKey)
		}
	}
	for _, accion := range []string{ports.AccionRegistrarAsignacion, ports.AccionRegistrarReasignacion} {
		if apariciones[accion] != 1 {
			t.Fatalf("apariciones de %q: %d", accion, apariciones[accion])
		}
	}
	for _, entrada := range manifiesto.Menu {
		if entrada.ModuleID != ModuleID || len(entrada.RequiredPermissions) != 1 ||
			!strings.HasPrefix(entrada.RequiredPermissions[0], "contratacion_temporal.") {
			t.Fatalf("entrada no cerrada: %#v", entrada)
		}
		if entrada.RequiredPermissions[0] == PermisoAsignarUnidad ||
			entrada.RequiredPermissions[0] == PermisoReasignarUnidad {
			t.Fatalf("el catálogo concedió acceso implícito desde el menú: %#v", entrada)
		}
	}
}
