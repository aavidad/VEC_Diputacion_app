package cronos

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID                    = "vec.module.cronos"
	PermissionTimeRead          = "cronos.fichaje.read"
	PermissionTimeManage        = "cronos.fichaje.manage"
	PermissionScheduleRead      = "cronos.horario.read"
	PermissionScheduleManage    = "cronos.horario.manage"
	PermissionLeaveRead         = "cronos.permiso.read"
	PermissionLeaveManage       = "cronos.permiso.manage"
	PermissionApprovalManage    = "cronos.aprobacion.manage"
	PermissionAudit             = "cronos.audit.read"
	ActionReviewJustification   = "cronos.jornada.justificacion.review"
	ActionReviewLeaveAndHoliday = "cronos.permiso.vacacion.review"
)

func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.cronos.name",
		DescriptionKey: "ui.vec.module.cronos.description",
		Version:        "v0.1.0",
		Group:          "gestion_tiempo",
		BasePath:       "/modules/cronos",
		Permissions: []domain.Permission{
			{Key: PermissionTimeRead, LabelKey: "ui.permission.cronos.time_read"},
			{Key: PermissionTimeManage, LabelKey: "ui.permission.cronos.time_manage"},
			{Key: PermissionScheduleRead, LabelKey: "ui.permission.cronos.schedule_read"},
			{Key: PermissionScheduleManage, LabelKey: "ui.permission.cronos.schedule_manage"},
			{Key: PermissionLeaveRead, LabelKey: "ui.permission.cronos.leave_read"},
			{Key: PermissionLeaveManage, LabelKey: "ui.permission.cronos.leave_manage"},
			{Key: PermissionApprovalManage, LabelKey: "ui.permission.cronos.approval_manage"},
			{Key: PermissionAudit, LabelKey: "ui.permission.cronos.audit"},
		},
		Menu: []domain.MenuEntry{
			menu("cronos.dashboard", "ui.vec.menu.cronos.dashboard", "/modules/cronos/dashboard", "activity", 10, PermissionTimeRead),
			menu("cronos.fichajes", "ui.vec.menu.cronos.fichajes", "/modules/cronos/fichajes", "clock", 20, PermissionTimeRead),
			menu("cronos.horarios", "ui.vec.menu.cronos.horarios", "/modules/cronos/horarios", "calendar-clock", 30, PermissionScheduleRead),
			menu("cronos.incidencias", "ui.vec.menu.cronos.incidencias", "/modules/cronos/incidencias", "triangle-alert", 40, PermissionTimeManage),
			menu("cronos.permisos", "ui.vec.menu.cronos.permisos", "/modules/cronos/permisos", "calendar-check", 50, PermissionLeaveRead),
			menu("cronos.vacaciones", "ui.vec.menu.cronos.vacaciones", "/modules/cronos/vacaciones", "calendar-days", 60, PermissionLeaveRead),
			menu("cronos.reducciones", "ui.vec.menu.cronos.reducciones", "/modules/cronos/reducciones", "timer-off", 70, PermissionScheduleManage),
			menu("cronos.aprobaciones", "ui.vec.menu.cronos.aprobaciones", "/modules/cronos/aprobaciones", "check-check", 80, PermissionApprovalManage),
			menu("cronos.saldos", "ui.vec.menu.cronos.saldos", "/modules/cronos/saldos", "gauge", 90, PermissionLeaveManage),
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
		Group:               "modulo_cronos",
		Order:               order,
		RequiredPermissions: []string{permission},
	}
}
