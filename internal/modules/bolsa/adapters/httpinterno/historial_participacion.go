package httpinterno

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ListadorHistorialParticipacion lo satisface el servicio de situación cuando
// la traza de valores (petición RRHH p.4) está disponible.
type ListadorHistorialParticipacion interface {
	ListarHistorial(context.Context, ports.SolicitudCambiarSituacionParticipacion) (ports.HistorialParticipacion, error)
}

// listarHistorialOperaciones usa el historial completo si el operador lo
// ofrece; si no, conserva la respuesta anterior sin la clave "cambios".
func listarHistorialOperaciones(ctx context.Context, o OperadorOperacionesSituacion, q ports.SolicitudCambiarSituacionParticipacion) ([]ports.RegistroOperacionSituacion, []map[string]any, error) {
	l, ok := o.(ListadorHistorialParticipacion)
	if !ok {
		items, err := o.ListarOperaciones(ctx, q)
		return items, nil, err
	}
	h, err := l.ListarHistorial(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	cambios := make([]map[string]any, 0, len(h.Cambios))
	for _, c := range h.Cambios {
		cambios = append(cambios, map[string]any{"instante": c.Instante.UTC().Format(time.RFC3339Nano), "recibo_ref": c.ReciboRef, "campo": c.Campo, "valor_anterior": c.ValorAnterior, "valor_nuevo": c.ValorNuevo, "actor": c.Actor})
	}
	return h.Operaciones, cambios, nil
}
