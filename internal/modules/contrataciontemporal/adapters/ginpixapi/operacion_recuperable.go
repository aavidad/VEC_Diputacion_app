package ginpixapi

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	_ ports.EmisorOperacionGINPIX    = (*Adaptador)(nil)
	_ ports.ConsultorOperacionGINPIX = (*Adaptador)(nil)
)

// EmitirOperacionGINPIX traduce la reserva durable al envio HTTP ya
// implementado por Adaptador. La reserva debe autorizar exactamente este
// intento; cualquier otra situacion se rechaza antes de preparar o autenticar.
func (a *Adaptador) EmitirOperacionGINPIX(
	ctx context.Context,
	solicitud ports.SolicitudOperacionGINPIX,
	reserva ports.ReservaOperacionGINPIX,
) (ports.ReciboExternoOperacionGINPIX, error) {
	if validarEntradaOperacionRecuperable(
		solicitud,
		reserva,
		ports.ReservaOperacionGINPIXEmisionAutorizada,
	) != nil || ctx == nil {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrEmisionOperacionGINPIXNoIniciada
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			errorEmisionOperacionRecuperable(err, false)
	}
	preparacion, err := prepararOperacionRecuperable(solicitud)
	if err != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrEmisionOperacionGINPIXNoIniciada
	}

	reciboExterno, err := a.Enviar(ctx, preparacion)
	_, hayReciboCompleto := datosReciboExternoCompletos(reciboExterno)
	if err != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			errorEmisionOperacionRecuperable(err, hayReciboCompleto)
	}
	if !hayReciboCompleto {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrEmisionOperacionGINPIXIndeterminada
	}
	recibo, err := traducirReciboOperacionRecuperable(reciboExterno, solicitud)
	if err != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrEmisionOperacionGINPIXIndeterminada
	}
	return recibo, nil
}

// ConsultarOperacionGINPIX usa exclusivamente Adaptador.Consultar. La
// situacion pendiente acredita que hubo una emision anterior y evita que esta
// ruta pueda activar una operacion nueva.
func (a *Adaptador) ConsultarOperacionGINPIX(
	ctx context.Context,
	solicitud ports.SolicitudOperacionGINPIX,
	reserva ports.ReservaOperacionGINPIX,
) (ports.ReciboExternoOperacionGINPIX, error) {
	if validarEntradaOperacionRecuperable(
		solicitud,
		reserva,
		ports.ReservaOperacionGINPIXPendienteConciliacion,
	) != nil || ctx == nil {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrConsultaOperacionGINPIXNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			errorConsultaOperacionRecuperable(err)
	}
	preparacion, err := prepararOperacionRecuperable(solicitud)
	if err != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrConsultaOperacionGINPIXNoDisponible
	}

	reciboExterno, err := a.Consultar(ctx, preparacion)
	_, hayReciboCompleto := datosReciboExternoCompletos(reciboExterno)
	if err != nil || !hayReciboCompleto {
		return ports.ReciboExternoOperacionGINPIX{},
			errorConsultaOperacionRecuperable(err)
	}
	recibo, err := traducirReciboOperacionRecuperable(reciboExterno, solicitud)
	if err != nil {
		return ports.ReciboExternoOperacionGINPIX{},
			ports.ErrConsultaOperacionGINPIXNoDisponible
	}
	return recibo, nil
}

func validarEntradaOperacionRecuperable(
	solicitud ports.SolicitudOperacionGINPIX,
	reserva ports.ReservaOperacionGINPIX,
	situacion ports.SituacionReservaOperacionGINPIX,
) error {
	if solicitud.Validar() != nil || reserva.ValidarPara(solicitud) != nil ||
		reserva.Situacion != situacion {
		return ports.ErrReservaOperacionGINPIXInvalida
	}
	return nil
}
