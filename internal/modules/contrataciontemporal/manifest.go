// Package contrataciontemporal declara el módulo coordinador de expedientes de
// contratación temporal. El manifiesto describe capacidades; no concede
// permisos ni sustituye la decisión del PDP.
package contrataciontemporal

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID = "vec.module.contratacion_temporal"

	PermisoConsultarCuadro               = "contratacion_temporal.cuadro.consultar"
	PermisoConsultarExpediente           = "contratacion_temporal.expediente.consultar"
	PermisoConsultarDocumentos           = "contratacion_temporal.documentos.consultar"
	PermisoCrearSolicitud                = "contratacion_temporal.solicitud.crear"
	PermisoEnviarAnalisis                = "contratacion_temporal.solicitud.enviar_analisis"
	PermisoAnalizarSolicitud             = "contratacion_temporal.analisis.validar"
	PermisoDecidirViaCobertura           = "contratacion_temporal.cobertura.decidir"
	PermisoAsignarUnidad                 = "contratacion_temporal.unidad.asignar"
	PermisoPrepararInforme               = "contratacion_temporal.informe.preparar"
	PermisoFirmarInforme                 = "contratacion_temporal.informe.firmar"
	PermisoSolicitarFiscalizacion        = "contratacion_temporal.fiscalizacion.solicitar"
	PermisoRegistrarFiscalizacion        = "contratacion_temporal.fiscalizacion.registrar"
	PermisoRegistrarSubsanacion          = "contratacion_temporal.subsanacion.registrar"
	PermisoPrepararLlamamiento           = "contratacion_temporal.llamamiento.preparar"
	PermisoSeleccionarCandidatura        = "contratacion_temporal.llamamiento.seleccionar"
	PermisoRegistrarResultadoLlamamiento = "contratacion_temporal.llamamiento.registrar_resultado"
	PermisoPrepararFormalizacion         = "contratacion_temporal.formalizacion.preparar"
	PermisoFirmarFormalizacion           = "contratacion_temporal.formalizacion.firmar"
	PermisoConfirmarIncorporacion        = "contratacion_temporal.incorporacion.confirmar"
	PermisoExportarGINPIX                = "contratacion_temporal.ginpix.exportar"
	PermisoEnviarGINPIX                  = "contratacion_temporal.ginpix.enviar"
	PermisoRegistrarSeguimiento          = "contratacion_temporal.seguimiento.registrar"
	PermisoCerrarExpediente              = "contratacion_temporal.expediente.cerrar"
	PermisoConfigurarFlujos              = "contratacion_temporal.flujo.configurar"
	PermisoConsultarTrazabilidad         = "contratacion_temporal.auditoria.consultar"
)

// Manifest registra exclusivamente rutas de navegación y capacidades
// conocidas por esta versión del módulo.
func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.contratacion_temporal.name",
		DescriptionKey: "ui.vec.module.contratacion_temporal.description",
		Version:        "v0.2.0",
		Group:          "recursos_humanos",
		BasePath:       "/modules/contratacion-temporal",
		Permissions: []domain.Permission{
			{Key: PermisoConsultarCuadro, LabelKey: "ui.permission.contratacion_temporal.cuadro"},
			{Key: PermisoConsultarExpediente, LabelKey: "ui.permission.contratacion_temporal.expediente"},
			{Key: PermisoConsultarDocumentos, LabelKey: "ui.permission.contratacion_temporal.documentos"},
			{Key: PermisoCrearSolicitud, LabelKey: "ui.permission.contratacion_temporal.solicitud"},
			{Key: PermisoEnviarAnalisis, LabelKey: "ui.permission.contratacion_temporal.enviar_analisis"},
			{Key: PermisoAnalizarSolicitud, LabelKey: "ui.permission.contratacion_temporal.analisis"},
			{Key: PermisoDecidirViaCobertura, LabelKey: "ui.permission.contratacion_temporal.cobertura"},
			{Key: PermisoAsignarUnidad, LabelKey: "ui.permission.contratacion_temporal.asignacion"},
			{Key: PermisoPrepararInforme, LabelKey: "ui.permission.contratacion_temporal.informe_preparar"},
			{Key: PermisoFirmarInforme, LabelKey: "ui.permission.contratacion_temporal.informe_firmar"},
			{Key: PermisoSolicitarFiscalizacion, LabelKey: "ui.permission.contratacion_temporal.fiscalizacion_solicitar"},
			{Key: PermisoRegistrarFiscalizacion, LabelKey: "ui.permission.contratacion_temporal.fiscalizacion_registrar"},
			{Key: PermisoRegistrarSubsanacion, LabelKey: "ui.permission.contratacion_temporal.subsanacion"},
			{Key: PermisoPrepararLlamamiento, LabelKey: "ui.permission.contratacion_temporal.llamamiento_preparar"},
			{Key: PermisoSeleccionarCandidatura, LabelKey: "ui.permission.contratacion_temporal.llamamiento_seleccionar"},
			{Key: PermisoRegistrarResultadoLlamamiento, LabelKey: "ui.permission.contratacion_temporal.llamamiento_resultado"},
			{Key: PermisoPrepararFormalizacion, LabelKey: "ui.permission.contratacion_temporal.formalizacion_preparar"},
			{Key: PermisoFirmarFormalizacion, LabelKey: "ui.permission.contratacion_temporal.formalizacion_firmar"},
			{Key: PermisoConfirmarIncorporacion, LabelKey: "ui.permission.contratacion_temporal.incorporacion"},
			{Key: PermisoExportarGINPIX, LabelKey: "ui.permission.contratacion_temporal.ginpix_exportar"},
			{Key: PermisoEnviarGINPIX, LabelKey: "ui.permission.contratacion_temporal.ginpix_enviar"},
			{Key: PermisoRegistrarSeguimiento, LabelKey: "ui.permission.contratacion_temporal.seguimiento"},
			{Key: PermisoCerrarExpediente, LabelKey: "ui.permission.contratacion_temporal.cerrar"},
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
			entradaMenu("contratacion_temporal.documentos", "ui.vec.menu.contratacion_temporal.documentos",
				"/modules/contratacion-temporal/documentos", "files", 130, PermisoConsultarDocumentos),
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
