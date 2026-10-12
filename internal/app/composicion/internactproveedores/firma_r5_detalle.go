package internactproveedores

import (
	"context"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type fuenteDetalleOriginalR5 struct{ fuente *internagobierno.FuenteF1 }

func (f fuenteDetalleOriginalR5) ResolverPeticionYContexto(ctx context.Context) (inc.PeticionAutoridad, ct.ContextoAutorizacionAltaV3, error) {
	if f.fuente == nil {
		return inc.PeticionAutoridad{}, ct.ContextoAutorizacionAltaV3{}, ErrProveedorFirmaR5NoDisponible
	}
	return f.fuente.PeticionYContextoVinculadoOriginalCT(ctx)
}

// AutoridadDetalleOriginalR5 reutiliza las comprobaciones de AutoridadDetalle
// y sustituye únicamente la fuente por el resultado F1 ya sellado.
func AutoridadDetalleOriginalR5(fuente *internagobierno.FuenteF1, reloj ct.Reloj) AutoridadDetalle {
	return AutoridadDetalle{Reloj: reloj, fuentePrueba: fuenteDetalleOriginalR5{fuente}}
}
