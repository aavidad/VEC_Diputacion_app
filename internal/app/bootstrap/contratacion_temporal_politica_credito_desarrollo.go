package bootstrap

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Valores del atributo «coste_con_partidas» de la regla c25.
const (
	costeConPartidasExigido   = "exigido"
	costeConPartidasNoExigido = "no_exigido"
)

// politicaCreditoOfertaDesarrollo lee la regla c25 vigente en cada consulta.
// Sin catálogo o sin la regla rige la decisión de RRHH del 02/10/2026 (se
// exige el coste con la constancia de las partidas). Un catálogo declarado
// que no se puede leer, o un valor que no se reconoce, falla cerrado.
type politicaCreditoOfertaDesarrollo struct {
	reglas *reglas.Resolutor
}

func (p politicaCreditoOfertaDesarrollo) PoliticaCreditoOferta(ctx context.Context) (domain.PoliticaCreditoOferta, error) {
	exige := domain.PoliticaCreditoOferta{ExigeCosteConPartidas: true}
	if p.reglas == nil {
		return exige, nil
	}
	regla, err := p.reglas.Regla(ctx, reglas.CTCreditoOferta)
	switch {
	case errors.Is(err, reglas.ErrReglasNoConfiguradas), errors.Is(err, reglas.ErrReglaNoEncontrada):
		return exige, nil
	case err != nil:
		return domain.PoliticaCreditoOferta{}, ports.ErrPoliticaCreditoOfertaNoDisponible
	}
	switch regla.Atributos[reglas.AtributoCosteConPartidas] {
	case costeConPartidasExigido, "":
		return exige, nil
	case costeConPartidasNoExigido:
		return domain.PoliticaCreditoOferta{}, nil
	default:
		return domain.PoliticaCreditoOferta{}, ports.ErrPoliticaCreditoOfertaNoDisponible
	}
}
