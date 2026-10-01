package httpcopias

import (
	"context"
	"strings"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

// OpcionesConfiguradas is a configuration-backed source, not an identity or
// authorization authority. The HTTP operation separately requires current permission.
type OpcionesConfiguradas struct{ opciones p.OpcionesRestauracion }

func NuevasOpcionesConfiguradas(v p.OpcionesRestauracion) (*OpcionesConfiguradas, error) {
	for _, group := range [][]p.Opcion{v.Destinos, v.Motivos, v.Ventanas} {
		if len(group) > 64 {
			return nil, p.ErrSolicitud
		}
		seen := map[string]bool{}
		for _, o := range group {
			if o.Ref == "" || len(o.Ref) > 192 || strings.ContainsAny(o.Ref, "/\\ \t\r\n") || o.ClaveI18N == "" || len(o.ClaveI18N) > 160 || strings.ContainsAny(o.ClaveI18N, "/\\ \t\r\n") || seen[o.Ref] {
				return nil, p.ErrSolicitud
			}
			seen[o.Ref] = true
		}
	}
	return &OpcionesConfiguradas{clonarOpciones(v)}, nil
}
func (o *OpcionesConfiguradas) Opciones(ctx context.Context, _ p.Sesion) (p.OpcionesRestauracion, error) {
	if o == nil || ctx.Err() != nil {
		return p.OpcionesRestauracion{}, p.ErrNoDisponible
	}
	return clonarOpciones(o.opciones), nil
}
func clonarOpciones(v p.OpcionesRestauracion) p.OpcionesRestauracion {
	return p.OpcionesRestauracion{Destinos: append([]p.Opcion{}, v.Destinos...), Motivos: append([]p.Opcion{}, v.Motivos...), Ventanas: append([]p.Opcion{}, v.Ventanas...)}
}
