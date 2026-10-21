package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Reutiliza la lectura nominal y auditada del recibo; no abre el pool de efectos
// ni convierte una fecha o secuencia del historial en un número de resolución.
func fuenteResolucionDocumentoCT(consulta ports.ConsultorPreparacionResolucionFormalizacion) ports.FuenteResolucionDocumentoRRHH {
	return func(ctx context.Context, expediente string) (*ports.ResultadoResolucionFormalizacion, error) {
		if consulta == nil {
			return nil, ports.ErrResolucionFormalizacionNoDisponible
		}
		p, err := consulta.ConsultarPreparacionResolucionFormalizacion(ctx, expediente)
		if err != nil {
			return nil, err
		}
		if p.ValidarPara(expediente) != nil {
			return nil, ports.ErrResultadoResolucionFormalizacionNoConfiable
		}
		if p.Recibo == nil {
			return nil, nil
		}
		r := *p.Recibo
		return &r, nil
	}
}
