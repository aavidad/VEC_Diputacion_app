package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// seleccionSoporteSeguimientoRRHH elige desde la capacidad sellada por la
// frontera mTLS. El segundo técnico sólo alcanza la no incorporación.
// Cada soporte conserva su comprobación nominal de sujeto, hoja y ruta.
type seleccionSoporteSeguimientoRRHH struct {
	original  *soporteAltaContratacionTemporalDesarrollo
	adicional *soporteAltaContratacionTemporalDesarrollo
}

func (s seleccionSoporteSeguimientoRRHH) elegir(ctx context.Context) (*soporteAltaContratacionTemporalDesarrollo, bool) {
	if s.original == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return nil, false
	}
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || capacidad.sello == nil || capacidad.principal.ID == "" {
		return nil, false
	}
	if capacidad.principal.ID == s.original.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.original.certificadoSHA256 {
		if _, valido := s.original.capacidadValida(ctx); valido {
			return s.original, true
		}
		return nil, false
	}
	if s.adicional != nil && capacidad.ruta == httpinterno.RutaNoIncorporaciones &&
		capacidad.principal.ID == s.adicional.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.adicional.certificadoSHA256 {
		if _, valido := s.adicional.capacidadValida(ctx); valido {
			return s.adicional, true
		}
	}
	return nil, false
}

func (s seleccionSoporteSeguimientoRRHH) ResolverContextoAutorizacionAltaV3(ctx context.Context, solicitud ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	soporte, ok := s.elegir(ctx)
	if !ok {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return soporte.ResolverContextoAutorizacionAltaV3(ctx, solicitud)
}
