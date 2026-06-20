package dietas

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID                  = "vec.module.dietas"
	PermissionExpenseRead     = "dietas.gasto.read"
	PermissionExpenseManage   = "dietas.gasto.manage"
	PermissionRouteRead       = "dietas.ruta.read"
	PermissionRouteManage     = "dietas.ruta.manage"
	PermissionApprovalManage  = "dietas.aprobacion.manage"
	PermissionAudit           = "dietas.audit.read"
	ActionReviewTravelExpense = "dietas.comision.review"
	ActionReviewRouteKM       = "dietas.ruta.km.review"
)

func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.dietas.name",
		DescriptionKey: "ui.vec.module.dietas.description",
		Version:        "v0.1.0",
		Group:          "gestion_gastos",
		BasePath:       "/modules/dietas",
		Permissions: []domain.Permission{
			{Key: PermissionExpenseRead, LabelKey: "ui.permission.dietas.expense_read"},
			{Key: PermissionExpenseManage, LabelKey: "ui.permission.dietas.expense_manage"},
			{Key: PermissionRouteRead, LabelKey: "ui.permission.dietas.route_read"},
			{Key: PermissionRouteManage, LabelKey: "ui.permission.dietas.route_manage"},
			{Key: PermissionApprovalManage, LabelKey: "ui.permission.dietas.approval_manage"},
			{Key: PermissionAudit, LabelKey: "ui.permission.dietas.audit"},
		},
		Menu: []domain.MenuEntry{
			menu("dietas.dashboard", "ui.vec.menu.dietas.dashboard", "/modules/dietas/dashboard", "receipt", 80, PermissionExpenseRead),
			menu("dietas.comisiones", "ui.vec.menu.dietas.comisiones", "/modules/dietas/comisiones", "briefcase-business", 90, PermissionExpenseRead),
			menu("dietas.kilometraje", "ui.vec.menu.dietas.kilometraje", "/modules/dietas/kilometraje", "route", 100, PermissionRouteRead),
			menu("dietas.mapa_provincia", "ui.vec.menu.dietas.mapa_provincia", "/modules/dietas/mapa-provincia", "map", 110, PermissionRouteRead),
			menu("dietas.dietas", "ui.vec.menu.dietas.dietas", "/modules/dietas/dietas", "utensils", 120, PermissionExpenseRead),
			menu("dietas.justificantes", "ui.vec.menu.dietas.justificantes", "/modules/dietas/justificantes", "file-check", 130, PermissionExpenseManage),
			menu("dietas.aprobaciones", "ui.vec.menu.dietas.aprobaciones", "/modules/dietas/aprobaciones", "badge-check", 140, PermissionApprovalManage),
			menu("dietas.liquidaciones", "ui.vec.menu.dietas.liquidaciones", "/modules/dietas/liquidaciones", "landmark", 150, PermissionExpenseManage),
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
		Group:               "modulo_dietas",
		Order:               order,
		RequiredPermissions: []string{permission},
	}
}
