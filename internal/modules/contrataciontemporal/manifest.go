// Package contrataciontemporal declara el módulo coordinador de expedientes de
// contratación temporal. El manifiesto describe capacidades; no concede
// permisos ni sustituye la decisión del PDP.
package contrataciontemporal

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID = "vec.module.contratacion_temporal"

	PermisoConsultarCuadro       = "contratacion_temporal.cuadro.consultar"
	PermisoConsultarExpediente   = "contratacion_temporal.expediente.consultar"
	PermisoCrearSolicitud        = "contratacion_temporal.solicitud.crear"
	PermisoAnalizarSolicitud     = "contratacion_temporal.analisis.validar"
	PermisoDecidirViaCobertura   = "contratacion_temporal.cobertura.decidir"
	PermisoAsignarUnidad         = "contratacion_temporal.unidad.asignar"
	PermisoConfigurarFlujos      = "contratacion_temporal.flujo.configurar"
	PermisoConsultarTrazabilidad = "contratacion_temporal.auditoria.consultar"
)

// Manifest registra exclusivamente rutas de navegación y capacidades
// conocidas por esta versión del módulo.
func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.contratacion_temporal.name",
		DescriptionKey: "ui.vec.module.contratacion_temporal.description",
		Version:        "v0.1.0",
		Group:          "recursos_humanos",
		BasePath:       "/modules/contratacion-temporal",
		Permissions: []domain.Permission{
			{Key: PermisoConsultarCuadro, LabelKey: "ui.permission.contratacion_temporal.cuadro"},
			{Key: PermisoConsultarExpediente, LabelKey: "ui.permission.contratacion_temporal.expediente"},
			{Key: PermisoCrearSolicitud, LabelKey: "ui.permission.contratacion_temporal.solicitud"},
			{Key: PermisoAnalizarSolicitud, LabelKey: "ui.permission.contratacion_temporal.analisis"},
			{Key: PermisoDecidirViaCobertura, LabelKey: "ui.permission.contratacion_temporal.cobertura"},
			{Key: PermisoAsignarUnidad, LabelKey: "ui.permission.contratacion_temporal.asignacion"},
			{Key: PermisoConfigurarFlujos, LabelKey: "ui.permission.contratacion_temporal.flujos"},
			{Key: PermisoConsultarTrazabilidad, LabelKey: "ui.permission.contratacion_temporal.auditoria"},
		},
		Menu: []domain.MenuEntry{
			entradaMenu("contratacion_temporal.cuadro", "ui.vec.menu.contratacion_temporal.cuadro",
				"/modules/contratacion-temporal/cuadro", "layout-dashboard", 100, PermisoConsultarCuadro),
			entradaMenu("contratacion_temporal.expedientes", "ui.vec.menu.contratacion_temporal.expedientes",
				"/modules/contratacion-temporal/expedientes", "folder-kanban", 110, PermisoConsultarExpediente),
			entradaMenu("contratacion_temporal.tareas", "ui.vec.menu.contratacion_temporal.tareas",
				"/modules/contratacion-temporal/tareas", "list-checks", 120, PermisoConsultarExpediente),
			entradaMenu("contratacion_temporal.flujos", "ui.vec.menu.contratacion_temporal.flujos",
				"/modules/contratacion-temporal/configuracion", "workflow", 190, PermisoConfigurarFlujos),
			entradaMenu("contratacion_temporal.auditoria", "ui.vec.menu.contratacion_temporal.auditoria",
				"/modules/contratacion-temporal/auditoria", "shield-check", 200, PermisoConsultarTrazabilidad),
		},
	}
}

func entradaMenu(id, etiqueta, ruta, icono string, orden int, permiso string) domain.MenuEntry {
	return domain.MenuEntry{
		ID: id, ModuleID: ModuleID, LabelKey: etiqueta, Path: ruta, Icon: icono,
		Group: "modulo_contratacion_temporal", Order: orden,
		RequiredPermissions: []string{permiso},
	}
}
