package documentos

import (
	"vec-diputacion-granada/internal/vec/documentos/ports"
	"vec-diputacion-granada/internal/vec/domain"
)

const ModuleID = "vec.module.documentos"

// Manifest declara capacidades nominales. Registrar el manifiesto no concede
// permisos ni activa por si solo rutas, autorizacion o custodia.
func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "modules.documentos.name",
		DescriptionKey: "modules.documentos.description",
		Version:        "v1.0.0",
		Group:          "rrhh",
		BasePath:       "/modules/documentos",
		Permissions: []domain.Permission{
			{Key: ports.AccionAlta, LabelKey: "modules.documentos.permissions.alta"},
			{Key: ports.AccionListar, LabelKey: "modules.documentos.permissions.listar"},
			{Key: ports.AccionDescargar, LabelKey: "modules.documentos.permissions.descargar"},
			{Key: ports.AccionPrepararNotificacion, LabelKey: "modules.documentos.permissions.preparar_notificacion"},
		},
		Menu: []domain.MenuEntry{{
			ID:                  "documentos.expedientes",
			ModuleID:            ModuleID,
			LabelKey:            "modules.documentos.menu.expedientes",
			Path:                "/modules/documentos/expedientes",
			Icon:                "file-text",
			Group:               "rrhh",
			Order:               50,
			RequiredPermissions: []string{ports.AccionListar},
		}},
	}
}
