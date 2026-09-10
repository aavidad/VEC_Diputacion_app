package historiaincorporacion

import (
	"context"
	"reflect"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type ContextoPropietario struct{ lector vp.LectorContextoOriginalV2 }

var _ LectorContexto = (*ContextoPropietario)(nil)

func contextoPropietarioNulo(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Chan, reflect.Slice:
		return x.IsNil()
	}
	return false
}
func NuevoContextoPropietario(l vp.LectorContextoOriginalV2) (*ContextoPropietario, error) {
	if contextoPropietarioNulo(l) {
		return nil, ErrHistoria
	}
	return &ContextoPropietario{l}, nil
}

// Traducción exacta, sin resolver actores actuales ni emitir autoridad.
func (p *ContextoPropietario) LeerContextoOriginal(ctx context.Context, ref ReferenciaContexto) (core.ResultadoContextoActorRegistradoV2, error) {
	var cero core.ResultadoContextoActorRegistradoV2
	if ctx == nil {
		return cero, ErrHistoria
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if p == nil || contextoPropietarioNulo(p.lector) {
		return cero, ErrHistoria
	}
	s := vp.SolicitudLecturaContextoOriginalV2{RegistroContextoRef: ref.RegistroRef, HuellaSHA256: ref.HuellaSHA256, ManifiestoProcedenciaHuellaSHA256: ref.ProcedenciaSHA256}
	if s.Validar() != nil {
		return cero, ErrHistoria
	}
	r, e := p.lector.LeerContextoOriginalV2(ctx, s)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if e != nil || s.ValidarResultado(r) != nil {
		return cero, ErrHistoria
	}
	copia, e := r.Clonar()
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if e != nil {
		return cero, ErrHistoria
	}
	return copia, nil
}
