package contrataciontemporal

import (
	"context"
	"net/http"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// NuevaRutaIncorporacionV2 monta el único contrato GET/POST existente. Cada
// petición conserva su ejecutor nominal propio; nunca se comparte entre HTTP.
func NuevaRutaIncorporacionV2(s *inc.ServidorV2PostgreSQL) (httpapi.RutaExacta, error) {
	if s == nil {
		return httpapi.RutaExacta{}, ErrRutasContratacionTemporalInvalidas
	}
	h := manejadorIncorporacionV2{nueva: func(ctx context.Context) (ct.ServicioIncorporacionAplicacionV2, error) { return s.NuevaPeticion(ctx) }}
	return httpapi.RutaExacta{Ruta: httpinterno.RutaIncorporacionEjercicioV2, Manejador: h}, nil
}

type manejadorIncorporacionV2 struct {
	nueva func(context.Context) (ct.ServicioIncorporacionAplicacionV2, error)
}

func (m manejadorIncorporacionV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := &peticionIncorporacionV2{nueva: m.nueva}
	h, err := httpinterno.NuevoManejadorIncorporacionEjercicioV2(p, p)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	h.ServeHTTP(w, r)
}

type peticionIncorporacionV2 struct {
	nueva    func(context.Context) (ct.ServicioIncorporacionAplicacionV2, error)
	servicio ct.ServicioIncorporacionAplicacionV2
}

func (p *peticionIncorporacionV2) ResolverContextoIncorporacionEjercicioV2(ctx context.Context) error {
	if p.nueva == nil || p.servicio != nil {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	s, err := p.nueva(ctx)
	if err != nil {
		return err
	}
	if s == nil {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	p.servicio = s
	return nil
}
func (p *peticionIncorporacionV2) Consultar(ctx context.Context, exp string) (ct.ProyeccionIncorporacionAplicacionV2, error) {
	if p.servicio == nil {
		return ct.ProyeccionIncorporacionAplicacionV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	return p.servicio.Consultar(ctx, exp)
}
func (p *peticionIncorporacionV2) Confirmar(ctx context.Context, i ct.IntencionIncorporacionAplicacionV2) (ct.ReciboIncorporacionAplicacionV2, error) {
	if p.servicio == nil {
		return ct.ReciboIncorporacionAplicacionV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	return p.servicio.Confirmar(ctx, i)
}
