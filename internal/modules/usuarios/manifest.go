package usuarios

import "vec-diputacion-granada/internal/vec/domain"

// El manifiesto identifica al propietario del contacto VEC. Su registro no
// concede permisos ni publica políticas de autorización.
const (
	ModuleID                         = "vec.module.usuarios"
	PermissionContactoAlta           = "vec.contacto_usuario.alta"
	PermissionContactoActualizar     = "vec.contacto_usuario.actualizar"
	PermissionContactoConsultar      = "vec.contacto_usuario.consultar"
	PermissionPreferenciasConsultar  = "vec.preferencias.consultar"
	PermissionPreferenciasActualizar = "vec.preferencias.actualizar"
	PermissionCorreosConsultar       = "vec.correos.consultar"
	PermissionCorreosAnadir          = "vec.correos.anadir"
	PermissionCorreosReenviar        = "vec.correos.reenviar"
	PermissionCorreosVerificar       = "vec.correos.verificar"
	PermissionCorreosActivar         = "vec.correos.activar"
	PermissionCorreosRetirar         = "vec.correos.retirar"
	PermissionImagenConsultar        = "vec.imagen.consultar"
	PermissionImagenActualizar       = "vec.imagen.actualizar"
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
			{Key: PermissionPreferenciasConsultar, LabelKey: "ui.permission.usuarios.preferencias_consultar"},
			{Key: PermissionPreferenciasActualizar, LabelKey: "ui.permission.usuarios.preferencias_actualizar"},
			{Key: PermissionCorreosConsultar, LabelKey: "ui.permission.usuarios.correos_consultar"},
			{Key: PermissionCorreosAnadir, LabelKey: "ui.permission.usuarios.correos_anadir"},
			{Key: PermissionCorreosReenviar, LabelKey: "ui.permission.usuarios.correos_reenviar"},
			{Key: PermissionCorreosVerificar, LabelKey: "ui.permission.usuarios.correos_verificar"},
			{Key: PermissionCorreosActivar, LabelKey: "ui.permission.usuarios.correos_activar"},
			{Key: PermissionCorreosRetirar, LabelKey: "ui.permission.usuarios.correos_retirar"},
			{Key: PermissionImagenConsultar, LabelKey: "ui.permission.usuarios.imagen_consultar"},
			{Key: PermissionImagenActualizar, LabelKey: "ui.permission.usuarios.imagen_actualizar"},
		},
	}
}
