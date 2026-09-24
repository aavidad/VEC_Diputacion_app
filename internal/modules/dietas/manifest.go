package dietas

import "vec-diputacion-granada/internal/vec/domain"

const (
	ModuleID = "vec.module.dietas"

	PermissionDraftCreate        = "dietas.borrador.propio.crear"
	PermissionDraftRead          = "dietas.borrador.propio.consultar"
	PermissionDraftEdit          = "dietas.borrador.propio.editar"
	PermissionDraftDelete        = "dietas.borrador.propio.borrar"
	PermissionDraftSend          = "dietas.borrador.propio.enviar"
	PermissionDocumentRead       = "dietas.documento.propio.consultar"
	PermissionRouteCatalog       = "dietas.ruta.catalogo.consultar"
	PermissionRouteCalculate     = "dietas.ruta.calculo.solicitar"
	PermissionReview             = "dietas.documento.revisar"
	PermissionAuthorize          = "dietas.documento.autorizar"
	PermissionLiquidate          = "dietas.documento.liquidar"
	PermissionAuditDecision      = "dietas.documento.fiscalizar"
	PermissionReviewInbox        = "dietas.bandeja.revision.consultar"
	PermissionAuthorizationInbox = "dietas.bandeja.autorizacion.consultar"
	PermissionLiquidationInbox   = "dietas.bandeja.liquidacion.consultar"
	PermissionAuditInbox         = "dietas.bandeja.fiscalizacion.consultar"

	// Permiso de la ruta por carretera de Cartografía; no se anuncia como
	// permiso del módulo interno.
	PermissionRouteRead = "dietas.ruta.read"
)

func Manifest() domain.ModuleManifest {
	return domain.ModuleManifest{
		ID:             ModuleID,
		NameKey:        "ui.vec.module.dietas.name",
		DescriptionKey: "ui.vec.module.dietas.description",
		Version:        "v0.2.0",
		Group:          "gestion_gastos",
		// El portal interno navega por #dietas tras consultar el catálogo.
		// ModuleManifest prohíbe fragmentos en rutas de menú; no se publican
		// aquí enlaces a /modules/dietas/* que no existen en el servidor.
		BasePath: "/portal-empleado/",
		Permissions: []domain.Permission{
			{Key: PermissionDraftCreate, LabelKey: "ui.permission.dietas.borrador_crear"},
			{Key: PermissionDraftRead, LabelKey: "ui.permission.dietas.borrador_consultar"},
			{Key: PermissionDraftEdit, LabelKey: "ui.permission.dietas.borrador_editar"},
			{Key: PermissionDraftDelete, LabelKey: "ui.permission.dietas.borrador_borrar"},
			{Key: PermissionDraftSend, LabelKey: "ui.permission.dietas.borrador_enviar"},
			{Key: PermissionDocumentRead, LabelKey: "ui.permission.dietas.documento_consultar"},
			{Key: PermissionRouteCatalog, LabelKey: "ui.permission.dietas.ruta_catalogo"},
			{Key: PermissionRouteCalculate, LabelKey: "ui.permission.dietas.ruta_calculo"},
			{Key: PermissionReview, LabelKey: "ui.permission.dietas.documento_revisar"},
			{Key: PermissionAuthorize, LabelKey: "ui.permission.dietas.documento_autorizar"},
			{Key: PermissionLiquidate, LabelKey: "ui.permission.dietas.documento_liquidar"},
			{Key: PermissionAuditDecision, LabelKey: "ui.permission.dietas.documento_fiscalizar"},
			{Key: PermissionReviewInbox, LabelKey: "ui.permission.dietas.bandeja_revision"},
			{Key: PermissionAuthorizationInbox, LabelKey: "ui.permission.dietas.bandeja_autorizacion"},
			{Key: PermissionLiquidationInbox, LabelKey: "ui.permission.dietas.bandeja_liquidacion"},
			{Key: PermissionAuditInbox, LabelKey: "ui.permission.dietas.bandeja_fiscalizacion"},
		},
		Menu: []domain.MenuEntry{},
	}
}
