package ports

import (
	"context"
	"errors"
)

var ErrGeneracionReferenciaAutorizacionV2 = errors.New(
	"vec: no se pudo generar una referencia de autorizacion V2",
)

// GeneradorReferenciasAutorizacionV2 crea identificadores opacos para dos
// finalidades distintas. La implementacion productiva debe usar un CSPRNG con
// al menos 128 bits y nunca derivarlos de identidad, expediente o texto libre.
// La correlacion en texto es solo la salida tecnica del adaptador: la frontera
// debe entregarla inmediatamente a domain.GenerarReferenciaCorrelacionAutorizacionV2
// y transportar desde entonces exclusivamente la capacidad nominal resultante.
type GeneradorReferenciasAutorizacionV2 interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
	NuevaClaveMotivoAutorizacionV2(context.Context) (string, error)
}
