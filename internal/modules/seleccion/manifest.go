// Package seleccion es el módulo Selección de VEC: convocatorias selectivas
// y solicitudes de participación («Convoca integrado»). En la fase 1 publica
// convocatorias gobernadas, recibe solicitudes de personas (también ajenas a
// la Diputación) y las muestra a RRHH. Bolsa conserva la constitución, el
// orden y los llamamientos.
package seleccion

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID = "vec.module.seleccion"

	PermisoSolicitudPropia      = "seleccion.solicitud.propia.gestionar"
	PermisoConsultarSolicitudes = "seleccion.solicitudes.consultar"
)

// Manifest declara el módulo en el catálogo /api/vec/modules. No publica
// entradas de menú: el portal de RRHH lo muestra solo si el módulo aparece y
// su consulta responde.
func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.seleccion.name",
		DescriptionKey: "ui.vec.module.seleccion.description",
		Version:        "v0.1.0",
		Group:          "seleccion",
		BasePath:       "/modules/seleccion",
		Permissions: []domain.Permission{
			{Key: PermisoSolicitudPropia, LabelKey: "ui.permission.seleccion.solicitud_propia"},
			{Key: PermisoConsultarSolicitudes, LabelKey: "ui.permission.seleccion.solicitudes_consultar"},
		},
		Menu: []domain.MenuEntry{},
	}
}
