package ports

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
)

// FirmadorCheckpointDesarrollo nunca acepta bytes arbitrarios ni claves.
type FirmadorCheckpointDesarrollo interface {
	FirmarCheckpoint(context.Context, domain.ReciboCheckpointDesarrollo) (domain.ReciboCheckpointDesarrollo, error)
	PinCheckpoint() string
}
type SelladorCheckpointDesarrollo interface {
	SellarCheckpoint(context.Context, domain.CheckpointDesarrollo) (domain.ReciboTSACheckpoint, error)
}
type VerificadorCheckpointDesarrollo interface {
	VerificarCheckpoint(context.Context, domain.ReciboCheckpointDesarrollo) error
}
