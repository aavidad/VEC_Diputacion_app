package historiaincorporacion

import (
	"context"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type AutenticacionPropietaria struct {
	lector vp.LectorAutenticacionOriginalV1
}

var _ LectorAutenticacion = (*AutenticacionPropietaria)(nil)

func NuevoAutenticacionPropietaria(l vp.LectorAutenticacionOriginalV1) (*AutenticacionPropietaria, error) {
	if contextoPropietarioNulo(l) {
		return nil, ErrHistoria
	}
	return &AutenticacionPropietaria{l}, nil
}

// Sólo adapta el selector y coteja el original. No fabrica un vínculo opaco ni
// invoca el revalidador vivo. El resultado tiene exclusivamente campos de valor.
func (p *AutenticacionPropietaria) LeerAutenticacionOriginal(ctx context.Context, ref ReferenciaAutenticacion) (core.AutenticacionRevalidadaV1, error) {
	var cero core.AutenticacionRevalidadaV1
	if contextoPropietarioNulo(ctx) {
		return cero, ErrHistoria
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if p == nil || contextoPropietarioNulo(p.lector) {
		return cero, ErrHistoria
	}
	s := vp.SolicitudLecturaAutenticacionOriginalV1{AutenticacionRef: ref.AutenticacionRef, SesionRef: ref.SesionRef, HuellaSHA256: ref.HuellaSHA256}
	if s.Validar() != nil {
		return cero, ErrHistoria
	}
	r, e := p.lector.LeerAutenticacionOriginalV1(ctx, s)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if e != nil || s.ValidarResultado(r) != nil {
		return cero, ErrHistoria
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	return r, nil
}
