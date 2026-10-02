package registrocopias

import (
	"context"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

// AceptarOrden persists an order already authenticated and revalidated upstream.
// It does not itself validate authority and is deliberately absent from the CLI.
func (f *Fichero) AceptarOrden(ctx context.Context, d port.Declaracion, o port.RecepcionOrden) (port.Aceptacion, error) {
	r, err := f.ejecutar(ctx, peticion{Accion: "aceptar_orden", Declaracion: d, Operacion: o.Operacion, Orden: &o})
	return port.Aceptacion{Orden: o, Recibo: r.Recibo, Replay: r.Replay, Auditoria: r.Auditoria}, err
}

func validarOrden(o port.RecepcionOrden) error {
	if !referencia.MatchString(o.Orden) || !referencia.MatchString(o.Operacion) || !referencia.MatchString(o.Destino) || !referencia.MatchString(o.Epoca) || !huella.MatchString(o.SHA256) || !huella.MatchString(o.SolicitudSHA256) || o.Fence == 0 {
		return port.ErrEntrada
	}
	return nil
}

func (m *motor) aceptarOrden(o port.RecepcionOrden, a port.Auditoria) (port.Recibo, bool, error) {
	if anterior, ok := m.ordenes[o.Orden]; ok {
		if anterior.Orden != o {
			return port.Recibo{}, false, port.ErrOrdenConflicto
		}
		return anterior.Recibo, true, nil
	}
	s, ok := m.operaciones[o.Operacion]
	if !ok {
		return port.Recibo{}, false, port.ErrNoExiste
	}
	if s.Solicitud.Destino != o.Destino || s.Solicitud.SHA256 != o.SolicitudSHA256 {
		return port.Recibo{}, false, operacionescopias.ErrVinculo
	}
	if _, ok := m.ordenPorOperacion[o.Operacion]; ok {
		return port.Recibo{}, false, port.ErrOrdenConflicto
	}
	if anterior, ok := m.fences[o.Destino]; ok {
		if anterior.Epoca != o.Epoca {
			return port.Recibo{}, false, operacionescopias.ErrVinculo
		}
		if o.Fence <= anterior.Fence {
			return port.Recibo{}, false, port.ErrFence
		}
	}
	r := port.Recibo{Referencia: identidad("aceptacion", a), Instante: a.Instante, Version: s.Operacion.Version(), Estado: s.Operacion.Estado()}
	m.ordenes[o.Orden] = port.Aceptacion{Orden: o, Recibo: r}
	m.ordenPorOperacion[o.Operacion], m.fences[o.Destino] = o.Orden, o
	return r, false, nil
}
