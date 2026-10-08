package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La ficha recibe esta pista sólo después de la lectura CT ya autorizada y
// auditada. El montaje no prueba una concesión del perfil documental separado.
type consultorDetalleConDisponibilidadRRHH struct {
	lector            httpinterno.ConsultorDetalleRRHH
	documentalMontado bool
}

func (c consultorDetalleConDisponibilidadRRHH) Consultar(ctx context.Context, solicitud ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	if ctx == nil || ctx.Err() != nil || c.lector == nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	detalle, err := c.lector.Consultar(ctx, solicitud)
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	detalle.EstadoBorradoresPublicados = ports.BorradoresRRHHSinMontaje
	if c.documentalMontado {
		// Una ruta montada sin evaluación nominal F1 queda indeterminada.
		detalle.EstadoBorradoresPublicados = ports.BorradoresRRHHIndisponible
	}
	return detalle, nil
}
