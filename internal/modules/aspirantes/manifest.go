package aspirantes

import "vec-diputacion-granada/internal/vec/domain"

// Aspirantes es el dueño de los datos personales de la población externa.
// El registro del manifiesto no concede permisos ni publica políticas.
const (
	ModuleID                  = "vec.module.aspirantes"
	PermissionFichaConsultar  = "vec.aspirantes.ficha.consultar"
	PermissionFichaAlta       = "vec.aspirantes.ficha.alta"
	PermissionFichaRectificar = "vec.aspirantes.ficha.rectificar"
)

func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.aspirantes.name",
		DescriptionKey: "ui.vec.module.aspirantes.description",
		Version:        "v0.1.0",
		Group:          "aspirantes",
		BasePath:       "/modules/aspirantes",
		Permissions: []domain.Permission{
			{Key: PermissionFichaConsultar, LabelKey: "ui.permission.aspirantes.ficha_consultar"},
			{Key: PermissionFichaAlta, LabelKey: "ui.permission.aspirantes.ficha_alta"},
			{Key: PermissionFichaRectificar, LabelKey: "ui.permission.aspirantes.ficha_rectificar"},
		},
	}
}
