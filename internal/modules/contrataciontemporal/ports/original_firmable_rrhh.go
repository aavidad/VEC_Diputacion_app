package ports

import "context"

// ConsultorOriginalFirmableRRHH reutiliza la lectura V3 que consume una
// concesión y registra la auditoría de acceso en la misma transacción.
type ConsultorOriginalFirmableRRHH interface {
	Consultar(context.Context, SolicitudDetalleRRHH) (DetalleExpedienteRRHH, error)
}

// TiposOriginalFirmableRRHH traduce la clave CT a la referencia del tipo
// documental publicada por el catálogo gobernado. No toma referencias del
// navegador ni fabrica una política documental alternativa.
type TiposOriginalFirmableRRHH interface {
	ResolverTipoOriginalRRHH(context.Context, TipoBorradorRRHH) (string, error)
}
