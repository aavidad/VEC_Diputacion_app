package bootstrap

import (
	"context"
	"errors"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// consultaBolsaVigenteDesarrollo pregunta a Bolsa si una referencia es la
// constitución vigente de su categoría y con qué ámbito se gobierna.
type consultaBolsaVigenteDesarrollo interface {
	BolsaConstituidaVigente(context.Context, string) (bool, error)
	AmbitoBolsa(context.Context, string) (puertosbolsa.AmbitoBolsa, error)
}

// admitirBolsa decide si el perfil de RRHH de Bolsa puede operar sobre la
// bolsa. Valen las del manifiesto y las que Bolsa tiene constituidas y
// vigentes. Una bolsa desconocida, sustituida o extinguida se deniega; si la
// base no responde, el acto no sigue.
func (p *preparadorBorradorLlamamientoDesarrollo) admitirBolsa(ctx context.Context, bolsaRef string) error {
	if _, admitida := p.soporte.bolsasRef[bolsaRef]; admitida {
		return nil
	}
	if p.vigentes == nil || ctx == nil {
		return dominiovec.ErrAutorizacionDenegada
	}
	vigente, err := p.vigentes.BolsaConstituidaVigente(ctx, bolsaRef)
	if err != nil {
		return errorConsultaBolsaDesarrollo(ctx)
	}
	if !vigente {
		return dominiovec.ErrAutorizacionDenegada
	}
	return nil
}

// contextoBolsaAdmitida resuelve la unidad gestora y el ámbito de una bolsa
// admitida (F1-60). Salen de la tabla gobernada de Bolsa (B144), no del
// manifiesto ni de la composición: la decisión V3 los contrasta después con la
// asignación real del actor, de modo que una bolsa de otra unidad se deniega
// aunque el actor tenga permiso en la suya. Una bolsa sin ámbito declarado se
// deniega siempre, también si figura en el manifiesto.
func (p *preparadorBorradorLlamamientoDesarrollo) contextoBolsaAdmitida(ctx context.Context, bolsaRef string) (puertosbolsa.ContextoSituacionParticipacionResuelto, error) {
	var vacio puertosbolsa.ContextoSituacionParticipacionResuelto
	if err := p.admitirBolsa(ctx, bolsaRef); err != nil {
		return vacio, err
	}
	if p.vigentes == nil || ctx == nil {
		return vacio, dominiovec.ErrAutorizacionDenegada
	}
	ambito, err := p.vigentes.AmbitoBolsa(ctx, bolsaRef)
	if errors.Is(err, puertosbolsa.ErrAmbitoBolsaNoDeclarado) {
		return vacio, dominiovec.ErrAutorizacionDenegada
	}
	if err != nil {
		return vacio, errorConsultaBolsaDesarrollo(ctx)
	}
	resuelto := puertosbolsa.ContextoSituacionParticipacionResuelto{UnidadRef: ambito.UnidadRef, AmbitoRef: ambito.AmbitoRef}
	if resuelto.Validar() != nil {
		return vacio, errBorradorNoDisponibleEn()
	}
	return resuelto, nil
}

// errorConsultaBolsaDesarrollo distingue la cancelación de la indisponibilidad;
// ninguna de las dos se presenta como permiso.
func errorConsultaBolsaDesarrollo(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return errBorradorNoDisponibleEn()
}
