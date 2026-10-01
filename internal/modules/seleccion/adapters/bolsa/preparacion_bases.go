package bolsa

import (
	"context"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

type CanonizadorBases struct{}

var _ ports.CanonizadorBases = CanonizadorBases{}

func (CanonizadorBases) ComprobarReferencia(r dominio.ReferenciaConfiguracionConvocatoria) error {
	return r.Validar()
}

func (CanonizadorBases) CanonizarContenido(ctx context.Context, c dominio.ContenidoPublicableConvocatoria) (dominio.ContenidoPublicableConvocatoria, error) {
	if err := ctx.Err(); err != nil {
		return dominio.ContenidoPublicableConvocatoria{}, err
	}
	return c.ClonarCanonico()
}
