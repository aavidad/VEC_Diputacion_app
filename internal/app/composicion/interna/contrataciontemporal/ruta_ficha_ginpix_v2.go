package contrataciontemporal

import (
	"context"
	"errors"
	"net/http"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// NuevaRutaFichaGINPIXV2 crea una petición nominal independiente para cada GET.
// Reutiliza el servidor V2; no tiene acceso a Confirmar ni al emisor GINPIX V1.
func NuevaRutaFichaGINPIXV2(s *inc.ServidorV2PostgreSQL, m inc.ProveedorMapeoFichaGINPIXV2) (httpapi.RutaExacta, error) {
	if s == nil || m == nil {
		return httpapi.RutaExacta{}, ErrRutasContratacionTemporalInvalidas
	}
	h := manejadorFichaGINPIXV2{mapeos: m, nueva: func(ctx context.Context) (inc.FuenteFichaGINPIXV2, error) { return s.NuevaPeticion(ctx) }}
	return httpapi.RutaExacta{Ruta: httpinterno.RutaFichaGINPIXV2, Manejador: h}, nil
}

type manejadorFichaGINPIXV2 struct {
	nueva  func(context.Context) (inc.FuenteFichaGINPIXV2, error)
	mapeos inc.ProveedorMapeoFichaGINPIXV2
}

func (m manejadorFichaGINPIXV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := &peticionFichaGINPIXV2{nueva: m.nueva, mapeos: m.mapeos}
	h, err := httpinterno.NuevoManejadorFichaGINPIXV2(p, p)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	h.ServeHTTP(w, r)
}

type peticionFichaGINPIXV2 struct {
	nueva  func(context.Context) (inc.FuenteFichaGINPIXV2, error)
	mapeos inc.ProveedorMapeoFichaGINPIXV2
	ficha  *inc.FichaGINPIXV2
}

func (p *peticionFichaGINPIXV2) ResolverContextoFichaGINPIXV2(ctx context.Context) error {
	if p.nueva == nil || p.ficha != nil {
		return inc.ErrFichaGINPIXV2NoDisponible
	}
	fuente, err := p.nueva(ctx)
	if errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || errors.Is(err, ct.ErrAutorizacionDenegada) {
		return inc.ErrFichaGINPIXV2Denegada
	}
	if err != nil {
		return err
	}
	p.ficha, err = inc.NuevaFichaGINPIXV2(fuente, p.mapeos)
	return err
}
func (p *peticionFichaGINPIXV2) Preparar(ctx context.Context, exp string) (ginpixfichero.PreparacionExportacion, error) {
	if p.ficha == nil {
		return ginpixfichero.PreparacionExportacion{}, inc.ErrFichaGINPIXV2NoDisponible
	}
	return p.ficha.Preparar(ctx, exp)
}
