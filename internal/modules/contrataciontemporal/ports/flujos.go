package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrFlujoNoDisponible = errors.New("contratacion temporal: flujo no disponible")

type SolicitudResolverFlujo struct {
	OrganizacionRef string
	CentroRef       string
	CategoriaRef    string
	MotivoClave     domain.ClaveCatalogo
	Instante        time.Time
}

func (s SolicitudResolverFlujo) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.CentroRef) ||
		!domain.ReferenciaOpacaValida(s.CategoriaRef) ||
		!s.MotivoClave.Valida() || !domain.InstanteUTCCanonico(s.Instante) {
		return ErrFlujoNoDisponible
	}
	return nil
}

type ConfiguracionAltaFlujo struct {
	Flujo            domain.ReferenciaFlujo
	FaseInicial      domain.ClaveFase
	UnidadInicialRef string
	AccionInicial    domain.ClaveCatalogo
}

func (c ConfiguracionAltaFlujo) Validar() error {
	if c.Flujo.Validar() != nil || !c.FaseInicial.Valida() ||
		!domain.ReferenciaOpacaValida(c.UnidadInicialRef) ||
		!c.AccionInicial.Valida() {
		return ErrFlujoNoDisponible
	}
	return nil
}

type ResolutorFlujoAlta interface {
	ResolverFlujoAlta(context.Context, SolicitudResolverFlujo) (ConfiguracionAltaFlujo, error)
}
