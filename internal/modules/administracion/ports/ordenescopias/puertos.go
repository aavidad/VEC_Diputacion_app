package ordenescopias

import (
	"context"
	"time"
	dominio "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ConsumidorV3 is private infrastructure, never an HTTP DTO factory. It must
// reread current V3/session/person/policy state, validate the exact action,
// resource, purpose, fields and obligations, CAS and consume UNIQUE, and append
// the exact order, audit and outbox in ONE transaction. Registering a candidate
// with AlmacenAutorizacion is not this consumption. Success requires confirmed
// COMMIT with central V3 audit and outbox; a local receipt is not that evidence.
// The optional PostgreSQL adapter requires separately reviewed SQL functions
// and a central material provider. This delivery neither installs nor wires them.
type ConsumidorV3 interface {
	ComprometerOrden(context.Context, dominio.Orden, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) error
}

// LectorCommit must observe the committed order (no dirty/uncommitted reads).
// It must not reconstruct a missing record from the requested commitment.
type LectorCommit interface {
	LeerOrdenComprometida(context.Context, string) (dominio.Orden, error)
}

// Autenticador is bound by trusted composition to a managed key, fixed suite,
// audience, domain and key revision. No key or trust anchor comes from a packet.
// Production must reuse the existing KMS authority. A local synthetic adapter
// does not meet this contract for production.
type Autenticador interface {
	Sellar(context.Context, dominio.Orden) ([]byte, error)
	Verificar(context.Context, dominio.Orden, []byte) error
}

// Anclaje verifies the CURRENT external revocation/epoch/fence authority.
// After restore it denies all new orders until current authority is reconciled.
// An exact maintenance delegation is bounded by order, expiry and fence; restored
// SQL history or a historical signature is never sufficient.
type Anclaje interface {
	ValidarActual(context.Context, dominio.Orden, time.Time) error
}

// AceptadorExterno appends a UNIQUE acceptance and fsyncs BEFORE PostgreSQL stop
// or any destructive action. Same exact order returns its original receipt; an
// altered order or stale fence fails. Its journal is outside restored roots.
// Acceptance does not execute the platform operation.
type AceptadorExterno interface {
	AceptarOrden(context.Context, dominio.Orden) (Aceptacion, error)
}
type Aceptacion struct {
	Orden, SHA256, Recibo string
	Replay                bool
}
type Sobre struct {
	Orden dominio.Orden
	Firma []byte
}
