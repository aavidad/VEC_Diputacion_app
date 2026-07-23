package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type SolicitudSellarAmbitoIdempotencia struct {
	ClaveIdempotencia string
	OrganizacionRef   string
	ActorRef          string
	PerfilRef         string
}

func (s SolicitudSellarAmbitoIdempotencia) Validar() error {
	if !claveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// SelladorAmbitoIdempotencia deriva un identificador HMAC sin persistir ni
// exponer la clave aportada. La clave criptográfica procede del gestor de
// secretos y su identificador/versionado pertenece al adaptador concreto.
type SelladorAmbitoIdempotencia interface {
	SellarAmbitoIdempotencia(
		context.Context,
		SolicitudSellarAmbitoIdempotencia,
	) (string, error)
}

// GeneradorReferenciasAlta acuña candidatos opacos. PostgreSQL decide cuáles
// prevalecen ante dos preparaciones concurrentes del mismo ámbito.
type GeneradorReferenciasAlta interface {
	GenerarReferenciasAlta(context.Context) (ReferenciasAlta, error)
	NuevaReferenciaReservaAlta(context.Context) (string, error)
}
