package personal

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID                       = "vec.module.personal"
	PermissionEmployeeRead         = "personal.empleado.read"
	PermissionEmployeeManage       = "personal.empleado.manage"
	PermissionPositionRead         = "personal.puesto.read"
	PermissionPositionManage       = "personal.puesto.manage"
	PermissionPayrollRead          = "personal.nomina.read"
	PermissionPayrollManage        = "personal.nomina.manage"
	PermissionSeniorityRead        = "personal.antiguedad.read"
	PermissionCertificateManage    = "personal.certificado.manage"
	PermissionAdministrativeManage = "personal.situacion.manage"
	PermissionAudit                = "personal.audit.read"
	ActionReviewPayrollIncident    = "personal.nomina.incidencia.review"
	ActionIssueServiceCertificate  = "personal.certificado.servicios.issue"
)

func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.personal.name",
		DescriptionKey: "ui.vec.module.personal.description",
		Version:        "v0.1.0",
		Group:          "gestion_personal",
		BasePath:       "/modules/personal",
		Permissions: []domain.Permission{
			{Key: PermissionEmployeeRead, LabelKey: "ui.permission.personal.employee_read"},
			{Key: PermissionEmployeeManage, LabelKey: "ui.permission.personal.employee_manage"},
			{Key: PermissionPositionRead, LabelKey: "ui.permission.personal.position_read"},
			{Key: PermissionPositionManage, LabelKey: "ui.permission.personal.position_manage"},
			{Key: PermissionPayrollRead, LabelKey: "ui.permission.personal.payroll_read"},
			{Key: PermissionPayrollManage, LabelKey: "ui.permission.personal.payroll_manage"},
			{Key: PermissionSeniorityRead, LabelKey: "ui.permission.personal.seniority_read"},
			{Key: PermissionCertificateManage, LabelKey: "ui.permission.personal.certificate_manage"},
			{Key: PermissionAdministrativeManage, LabelKey: "ui.permission.personal.admin_status_manage"},
			{Key: PermissionAudit, LabelKey: "ui.permission.personal.audit"},
		},
		Menu: []domain.MenuEntry{
			menu("personal.dashboard", "ui.vec.menu.personal.dashboard", "/modules/personal/dashboard", "users", 5, PermissionEmployeeRead),
			menu("personal.expedientes", "ui.vec.menu.personal.expedientes", "/modules/personal/expedientes", "id-card", 6, PermissionEmployeeRead),
			menu("personal.puestos", "ui.vec.menu.personal.puestos", "/modules/personal/puestos", "building-2", 7, PermissionPositionRead),
			menu("personal.situaciones", "ui.vec.menu.personal.situaciones", "/modules/personal/situaciones", "folder-sync", 8, PermissionAdministrativeManage),
			menu("personal.antiguedad", "ui.vec.menu.personal.antiguedad", "/modules/personal/antiguedad", "history", 9, PermissionSeniorityRead),
			menu("personal.servicios", "ui.vec.menu.personal.servicios", "/modules/personal/servicios-prestados", "scroll-text", 10, PermissionSeniorityRead),
			menu("personal.certificados", "ui.vec.menu.personal.certificados", "/modules/personal/certificados", "file-badge", 11, PermissionCertificateManage),
			menu("personal.nominas", "ui.vec.menu.personal.nominas", "/modules/personal/nominas", "banknote", 12, PermissionPayrollRead),
			menu("personal.retribuciones", "ui.vec.menu.personal.retribuciones", "/modules/personal/retribuciones", "calculator", 13, PermissionPayrollManage),
			menu("personal.incidencias", "ui.vec.menu.personal.incidencias", "/modules/personal/incidencias", "circle-alert", 14, PermissionPayrollManage),
			menu("personal.integraciones", "ui.vec.menu.personal.integraciones", "/modules/personal/integraciones", "workflow", 15, PermissionEmployeeManage),
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
		Group:               "modulo_personal",
		Order:               order,
		RequiredPermissions: []string{permission},
	}
}
