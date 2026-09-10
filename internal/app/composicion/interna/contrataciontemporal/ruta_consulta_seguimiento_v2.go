package contrataciontemporal

import (
	"context"
	"net/http"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// Consulta exclusivamente el estado original de la incorporación. La petición
// nominal se crea después de validar el transporte y revalida la lectura actual.
func NuevaRutaConsultaSeguimientoV2(s *inc.ServidorV2PostgreSQL) (httpapi.RutaExacta, error) {
	if s == nil {
		return httpapi.RutaExacta{}, ErrRutasContratacionTemporalInvalidas
	}
	h := manejadorConsultaSeguimientoV2{nueva: func(ctx context.Context) (consultaSeguimientoV2, error) {
		return s.NuevaPeticion(ctx)
	}}
	return httpapi.RutaExacta{Ruta: httpinterno.RutaConsultaSeguimientoV2, Manejador: h}, nil
}

type consultaSeguimientoV2 interface {
	ConsultarSeguimientoIncorporacionV2(context.Context, string) (ct.VistaSeguimientoIncorporacionV2, error)
}

type manejadorConsultaSeguimientoV2 struct {
	nueva func(context.Context) (consultaSeguimientoV2, error)
}

func (m manejadorConsultaSeguimientoV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := &peticionConsultaSeguimientoV2{nueva: m.nueva}
	h, err := httpinterno.NuevoManejadorConsultaSeguimientoV2(p, p)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	h.ServeHTTP(w, r)
}

type peticionConsultaSeguimientoV2 struct {
	nueva    func(context.Context) (consultaSeguimientoV2, error)
	consulta consultaSeguimientoV2
}

func (p *peticionConsultaSeguimientoV2) ResolverContextoIncorporacionEjercicioV2(ctx context.Context) error {
	if p.nueva == nil || p.consulta != nil {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	consulta, err := p.nueva(ctx)
	if err != nil {
		return err
	}
	if consulta == nil {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	p.consulta = consulta
	return nil
}

func (p *peticionConsultaSeguimientoV2) ConsultarSeguimientoIncorporacionV2(ctx context.Context, exp string) (ct.VistaSeguimientoIncorporacionV2, error) {
	if p.consulta == nil {
		return ct.VistaSeguimientoIncorporacionV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	return p.consulta.ConsultarSeguimientoIncorporacionV2(ctx, exp)
}
