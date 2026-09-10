package historiaincorporacion

import (
	"context"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type EvaluacionPropietaria struct{ lector vp.LectorEvaluacionOriginalV3 }

var _ LectorEvaluacion = (*EvaluacionPropietaria)(nil)

func NuevoEvaluacionPropietaria(l vp.LectorEvaluacionOriginalV3) (*EvaluacionPropietaria, error) {
	if contextoPropietarioNulo(l) {
		return nil, ErrHistoria
	}
	return &EvaluacionPropietaria{l}, nil
}

func (p *EvaluacionPropietaria) LeerEvaluacionOriginal(ctx context.Context, ref ReferenciaEvaluacion) (core.InstantaneaAutorizacion, error) {
	var cero core.InstantaneaAutorizacion
	if contextoPropietarioNulo(ctx) {
		return cero, ErrHistoria
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if p == nil || contextoPropietarioNulo(p.lector) {
		return cero, ErrHistoria
	}
	s := vp.SolicitudLecturaEvaluacionOriginalV3{DecisionRef: ref.DecisionRef, DecisionSHA256: ref.DecisionSHA256, SolicitudSHA256: ref.SolicitudSHA256}
	if s.Validar() != nil {
		return cero, ErrHistoria
	}
	r, e := p.lector.LeerEvaluacionOriginalV3(ctx, s)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if e != nil {
		return cero, ErrHistoria
	}
	c, e := vp.CopiarInstantaneaEvaluacionOriginalV3(r)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if e != nil {
		return cero, ErrHistoria
	}
	return c, nil
}
