package bootstrap

import (
	"context"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// consultaBolsaVigenteDesarrollo pregunta a Bolsa si una referencia es la
// constitución vigente de su categoría.
type consultaBolsaVigenteDesarrollo interface {
	BolsaConstituidaVigente(context.Context, string) (bool, error)
}

// admitirBolsa decide si el perfil de RRHH de Bolsa puede operar sobre la
// bolsa. Valen las del manifiesto y las que Bolsa tiene constituidas y
// vigentes: todas pertenecen a la única unidad y ámbito de este perfil, los
// mismos con que se confirmó la carga. Una bolsa desconocida, sustituida o
// extinguida se deniega; si la base no responde, el acto no sigue.
func (p *preparadorBorradorLlamamientoDesarrollo) admitirBolsa(ctx context.Context, bolsaRef string) error {
	if _, admitida := p.soporte.bolsasRef[bolsaRef]; admitida {
		return nil
	}
	if p.vigentes == nil || ctx == nil {
		return dominiovec.ErrAutorizacionDenegada
	}
	vigente, err := p.vigentes.BolsaConstituidaVigente(ctx, bolsaRef)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errBorradorNoDisponibleEn()
	}
	if !vigente {
		return dominiovec.ErrAutorizacionDenegada
	}
	return nil
}
