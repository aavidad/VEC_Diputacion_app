package composicion

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type ExportadorServiciosPropios interface {
	Exportar(context.Context, domain.SolicitudExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error)
}
type exportadorServiciosPropiosConIdentidad struct {
	siguiente ExportadorServiciosPropios
	identidad ResolutorIdentidadFichaPropia
	limite    time.Duration
}

func NuevoExportadorServiciosPropiosConIdentidad(e ExportadorServiciosPropios, i ResolutorIdentidadFichaPropia, limite time.Duration) (ExportadorServiciosPropios, error) {
	if dependenciaNula(e) || dependenciaNula(i) || limite <= 0 || limite > 30*time.Second {
		return nil, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	return exportadorServiciosPropiosConIdentidad{e, i, limite}, nil
}
func (e exportadorServiciosPropiosConIdentidad) Exportar(ctx context.Context, in domain.SolicitudExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error) {
	captura, err := PrepararContextoIntentoFichaPropia(ctx, e.identidad, e.limite)
	if err != nil || dependenciaNula(e.siguiente) {
		return ports.ResultadoExportacionServiciosPropios{}, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	return e.siguiente.Exportar(captura, in)
}
