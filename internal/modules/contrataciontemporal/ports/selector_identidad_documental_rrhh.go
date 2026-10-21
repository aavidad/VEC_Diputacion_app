package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Consulta la identidad opaca de la aceptación nativa conservada en la
// propuesta original v7. Solo se ofrece al consumidor interno documental.
type ConsultaSelectorIdentidadDocumentalRRHH interface {
	Consultar(context.Context, string) (domain.SelectorBolsaPlanB2Nativo, error)
}

// La consulta y el recibo de acceso a detalle se confirman juntos.
type SesionSelectorIdentidadDocumentalRRHH interface {
	SesionConsultaRRHH
	ConsultarSelectorYRegistrarAcceso(context.Context, OrdenConsultaDetalleRRHH) (DetalleExpedienteRRHH, domain.SelectorBolsaPlanB2Nativo, error)
}
