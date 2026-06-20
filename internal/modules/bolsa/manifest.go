package bolsa

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID               = "vec.module.bolsa"
	PermissionRead         = "bolsa.solicitud.read"
	PermissionManage       = "bolsa.solicitud.manage"
	PermissionDocument     = "bolsa.document.read"
	PermissionClaim        = "bolsa.claim.read"
	PermissionNotification = "bolsa.notification.read"
	PermissionDemoAction   = "bolsa.demo.action"
	PermissionAudit        = "bolsa.audit.read"
	ActionDemoIntegration  = "bolsa.demo.integration"
)

func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.bolsa.name",
		DescriptionKey: "ui.vec.module.bolsa.description",
		Version:        "v0.1.0",
		Group:          "seleccion",
		BasePath:       "/modules/bolsa",
		Permissions: []domain.Permission{
			{Key: PermissionRead, LabelKey: "ui.permission.bolsa.read"},
			{Key: PermissionManage, LabelKey: "ui.permission.bolsa.manage"},
			{Key: PermissionDocument, LabelKey: "ui.permission.bolsa.document"},
			{Key: PermissionClaim, LabelKey: "ui.permission.bolsa.claim"},
			{Key: PermissionNotification, LabelKey: "ui.permission.bolsa.notification"},
			{Key: PermissionDemoAction, LabelKey: "ui.permission.bolsa.demo_action"},
			{Key: PermissionAudit, LabelKey: "ui.permission.bolsa.audit"},
		},
		Menu: []domain.MenuEntry{
			menu("bolsa.dashboard", "ui.vec.menu.bolsa.dashboard", "/modules/bolsa/dashboard", "layout-dashboard", 100, PermissionRead),
			menu("bolsa.convocatorias", "ui.vec.menu.bolsa.convocatorias", "/modules/bolsa/convocatorias", "calendar-days", 110, PermissionRead),
			menu("bolsa.solicitudes", "ui.vec.menu.bolsa.solicitudes", "/modules/bolsa/solicitudes", "file-text", 120, PermissionRead),
			menu("bolsa.meritos", "ui.vec.menu.bolsa.meritos", "/modules/bolsa/meritos", "graduation-cap", 130, PermissionRead),
			menu("bolsa.autobaremo", "ui.vec.menu.bolsa.autobaremo", "/modules/bolsa/autobaremo", "calculator", 140, PermissionRead),
			menu("bolsa.documentos", "ui.vec.menu.bolsa.documentos", "/modules/bolsa/documentos", "folder-check", 150, PermissionDocument),
			menu("bolsa.alegaciones", "ui.vec.menu.bolsa.alegaciones", "/modules/bolsa/alegaciones", "message-square-warning", 160, PermissionClaim),
			menu("bolsa.notificaciones", "ui.vec.menu.bolsa.notificaciones", "/modules/bolsa/notificaciones", "bell", 170, PermissionNotification),
			menu("bolsa.listados", "ui.vec.menu.bolsa.listados", "/modules/bolsa/listados", "list-ordered", 180, PermissionRead),
			menu("bolsa.auditoria", "ui.vec.menu.bolsa.auditoria", "/modules/bolsa/auditoria", "shield-check", 190, PermissionAudit),
			menu("bolsa.manifiestos", "ui.vec.menu.bolsa.manifiestos", "/modules/bolsa/manifiestos", "plug", 200, PermissionAudit),
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
		Group:               "modulo_bolsa",
		Order:               order,
		RequiredPermissions: []string{permission},
	}
}
