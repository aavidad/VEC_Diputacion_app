package seguridad

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// GenerarReferenciasSubsanacionReparo reutiliza el CSPRNG privado de la
// autoridad de referencias; los prefijos no contienen identidad funcional.
func (g *GeneradorReferenciasAltaCriptografico) GenerarReferenciasSubsanacionReparo(ctx context.Context) (ports.ReferenciasEfectoSubsanacionReparo, error) {
	if !generadorValido(g) || ctx == nil {
		return ports.ReferenciasEfectoSubsanacionReparo{}, ErrGeneracionReferenciaAlta
	}
	prefijos := []string{"reserva:ct-subsanacion-reparo:", "recibo:ct-subsanacion-reparo:", "evento:ct-subsanacion-reparo:"}
	valores := make([]string, len(prefijos))
	for i, prefijo := range prefijos {
		v, err := g.generar(ctx, prefijo)
		if err != nil {
			return ports.ReferenciasEfectoSubsanacionReparo{}, err
		}
		valores[i] = v
	}
	r := ports.ReferenciasEfectoSubsanacionReparo{ReservaRef: valores[0], ReciboRef: valores[1], EventoRef: valores[2]}
	if !r.Validas() {
		return ports.ReferenciasEfectoSubsanacionReparo{}, ErrGeneracionReferenciaAlta
	}
	return r, nil
}

var _ ports.GeneradorReferenciasSubsanacionReparo = (*GeneradorReferenciasAltaCriptografico)(nil)
