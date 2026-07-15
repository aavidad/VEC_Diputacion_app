package administracion

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID                     = "vec.module.administracion"
	PermissionRolesManage        = "vec.roles.manage"
	PermissionCatalogsManage     = "vec.catalogs.manage"
	PermissionIntegrationsManage = "vec.integrations.manage"
	PermissionMonitoringRead     = "vec.monitoring.read"
	PermissionAuditRead          = "vec.audit.read"
	ActionAssignRole             = "vec.roles.assign"
	ActionPublishCatalog         = "vec.catalog.publish"
)

func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.administracion.name",
		DescriptionKey: "ui.vec.module.administracion.description",
		Version:        "v0.1.0",
		Group:          "administracion_vec",
		BasePath:       "/modules/administracion",
		Permissions: []domain.Permission{
			{Key: PermissionRolesManage, LabelKey: "ui.permission.administracion.roles_manage"},
			{Key: PermissionCatalogsManage, LabelKey: "ui.permission.administracion.catalogs_manage"},
			{Key: PermissionIntegrationsManage, LabelKey: "ui.permission.administracion.integrations_manage"},
			{Key: PermissionMonitoringRead, LabelKey: "ui.permission.administracion.monitoring_read"},
			{Key: PermissionAuditRead, LabelKey: "ui.permission.administracion.audit_read"},
		},
		Menu: []domain.MenuEntry{
			menu("admin.usuarios_roles", "ui.vec.menu.admin.usuarios_roles", "/modules/administracion/usuarios-roles", "shield-user", 500, PermissionRolesManage),
			menu("admin.catalogos", "ui.vec.menu.admin.catalogos", "/modules/administracion/catalogos", "list-checks", 510, PermissionCatalogsManage),
			menu("admin.integraciones", "ui.vec.menu.admin.integraciones", "/modules/administracion/integraciones", "plug-zap", 520, PermissionIntegrationsManage),
			menu("admin.monitorizacion", "ui.vec.menu.admin.monitorizacion", "/modules/administracion/monitorizacion", "activity", 530, PermissionMonitoringRead),
		},
	}
}

func menu(id, label, path, icon string, order int, permission string) domain.MenuEntry {
	return domain.MenuEntry{
		ID:                  id,
		ModuleID:            ModuleID,
		LabelKey:            label,
		Path:                path,
		Icon:                icon,
		Group:               "modulo_administracion",
		Order:               order,
		RequiredPermissions: []string{permission},
	}
}
