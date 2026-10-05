package ports

import (
	"context"

	vd "vec-diputacion-granada/internal/vec/domain"
)

// DescriptorPlanFijadoFirmaV2 conserva por separado el descriptor nominal
// original y la entrada publicada del plan que lo seleccionó. No modifica
// el contrato de once claves que consumen AD170, CT172 y AUT35.
type DescriptorPlanFijadoFirmaV2 struct {
	Descriptor DescriptorConstructorFirmaV2
	Plan       vd.ReferenciaEntradaCatalogo
}

// FuenteDescriptorPlanFijadoFirmaV2 sólo prepara la solicitud. El consumidor
// durable debe revalidar la entrada del plan en la transacción del efecto.
type FuenteDescriptorPlanFijadoFirmaV2 interface {
	DescriptorPlanFijadoFirmaV2(context.Context, MaterialFirmaVerificadaV2) (DescriptorPlanFijadoFirmaV2, error)
}
