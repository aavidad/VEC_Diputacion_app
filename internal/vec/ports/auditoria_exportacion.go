package ports

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
)

// FuenteCapturaExportacionAuditoria está ligada por composición a una operación
// nominal concreta. Debe consumir su permiso, leer una instantánea coherente y
// escribir el acuse en la auditoría común dentro de la misma transacción. Sólo
// devuelve datos tras COMMIT confirmado. Un archivo local no implementa este
// contrato ni concede permisos; aún no hay adaptador runtime de esta capacidad.
type FuenteCapturaExportacionAuditoria interface {
	CapturarAuditoriaParaExportacion(context.Context) (CapturaParaExportacionAuditoria, error)
}

type CapturaParaExportacionAuditoria struct {
	Captura   domain.CapturaExportacionAuditoria
	Cobertura domain.CoberturaCheckpoint
	Documento []byte
}

type FirmadorExportacionAuditoriaDesarrollo interface {
	FirmarExportacionAuditoria(context.Context, domain.ReciboExportacionAuditoriaDesarrollo) (domain.ReciboExportacionAuditoriaDesarrollo, error)
	PinExportacionAuditoria() string
}
type SelladorExportacionAuditoriaDesarrollo interface {
	SellarExportacionAuditoria(context.Context, domain.ManifiestoExportacionAuditoriaDesarrollo) (domain.ReciboTSACheckpoint, error)
}
type VerificadorExportacionAuditoriaDesarrollo interface {
	VerificarExportacionAuditoria(context.Context, domain.ReciboExportacionAuditoriaDesarrollo) error
}
