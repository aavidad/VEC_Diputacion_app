package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Reutiliza la lectura nominal y auditada del acto (CT69 ampliada por CT261),
// válida desde la versión 8 en adelante; no abre el pool de efectos ni
// convierte una fecha o secuencia del historial en un número de resolución.
func fuenteResolucionDocumentoCT(consulta any) ports.FuenteResolucionDocumentoRRHH {
	acto, _ := consulta.(ports.ConsultorActoResolucionFormalizacion)
	return func(ctx context.Context, expediente string) (*ports.ResultadoResolucionFormalizacion, error) {
		if acto == nil {
			return nil, ports.ErrResolucionFormalizacionNoDisponible
		}
		r, err := acto.ConsultarActoResolucionFormalizacion(ctx, expediente)
		if err != nil {
			return nil, err
		}
		return &r, nil
	}
}
