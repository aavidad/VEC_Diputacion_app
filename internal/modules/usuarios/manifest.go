package usuarios

import "vec-diputacion-granada/internal/vec/domain"

// El manifiesto identifica al propietario del contacto VEC. Su registro no
// concede permisos ni publica políticas de autorización.
const (
	ModuleID                     = "vec.module.usuarios"
	PermissionContactoAlta       = "vec.contacto_usuario.alta"
	PermissionContactoActualizar = "vec.contacto_usuario.actualizar"
	PermissionContactoConsultar  = "vec.contacto_usuario.consultar"
)

func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.usuarios.name",
		DescriptionKey: "ui.vec.module.usuarios.description",
		Version:        "v0.1.0",
		Group:          "usuarios_vec",
		BasePath:       "/modules/usuarios",
		Permissions: []domain.Permission{
			{Key: PermissionContactoAlta, LabelKey: "ui.permission.usuarios.contacto_alta"},
			{Key: PermissionContactoActualizar, LabelKey: "ui.permission.usuarios.contacto_actualizar"},
			{Key: PermissionContactoConsultar, LabelKey: "ui.permission.usuarios.contacto_consultar"},
		},
	}
}
