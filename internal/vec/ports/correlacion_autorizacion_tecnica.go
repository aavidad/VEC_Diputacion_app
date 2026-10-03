package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// ReferenciaCorrelacionAutorizacionV2DePeticion enlaza la correlación técnica
// privada de la petición con su referencia canónica V3. No lee cabeceras ni
// genera otra identidad. La cancelación no borra un identificador ya emitido.
func ReferenciaCorrelacionAutorizacionV2DePeticion(
	ctx context.Context,
) (domain.ReferenciaCorrelacionAutorizacionV2, error) {
	correlacion, ok := CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		return domain.ReferenciaCorrelacionAutorizacionV2{}, ErrCorrelacionIncidenciasNoDisponible
	}
	referencia, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.WithoutCancel(ctx), generadorCorrelacionTecnicaV3{valor: "correlacion_" + correlacion},
	)
	if err != nil {
		return domain.ReferenciaCorrelacionAutorizacionV2{}, ErrCorrelacionIncidenciasNoDisponible
	}
	return referencia, nil
}

type generadorCorrelacionTecnicaV3 struct{ valor string }

func (g generadorCorrelacionTecnicaV3) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}
