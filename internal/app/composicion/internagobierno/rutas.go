package internagobierno

import (
	"context"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// AutoridadRutaSeguimiento limita la frontera HTTP a la ruta exacta de
// consulta. Comprueba sesión, vínculo F1 y vínculo corporativo vivos antes de entrar en el caso de
// uso. La concesión sobre el expediente concreto se evalúa después mediante
// V3 y se consume junto con la lectura PostgreSQL.
type AutoridadRutaSeguimiento struct{ fuente *FuenteF1 }

func NuevaAutoridadRutaSeguimiento(fuente *FuenteF1) (*AutoridadRutaSeguimiento, error) {
	if fuente == nil || fuente.identidad == nil || fuente.revalidador == nil || fuente.resolutor == nil ||
		fuente.corporativo == nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	return &AutoridadRutaSeguimiento{fuente: fuente}, nil
}

func (a *AutoridadRutaSeguimiento) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if a == nil || a.fuente == nil || ctx == nil || ctx.Err() != nil || ruta != httpct.RutaConsultaSeguimientoV2 {
		return httpapi.ErrAccesoRutaExactaDenegado
	}
	if _, err := a.fuente.ResolverContexto(ctx); err != nil {
		return httpapi.ErrAccesoRutaExactaDenegado
	}
	return nil
}

var _ httpapi.AutoridadRutasExactas = (*AutoridadRutaSeguimiento)(nil)
