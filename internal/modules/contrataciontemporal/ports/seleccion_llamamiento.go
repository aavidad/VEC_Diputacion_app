package ports

import "context"

// PreparadorSeleccionLlamamiento resuelve desde una referencia de intención
// no autoritativa los contextos, la política y los límites gobernados de cada
// paso. No selecciona una posición ni ejecuta efectos en Bolsa.
type PreparadorSeleccionLlamamiento interface {
	PrepararConsultaDisponibilidad(
		context.Context,
		string,
	) (SolicitudDisponibilidadBolsa, error)
	PrepararOrdenCompleto(
		context.Context,
		string,
		ResultadoDisponibilidadBolsa,
	) (ComandoPrepararOrdenBolsa, error)
	PrepararContextoLlamamiento(
		context.Context,
		string,
		ReciboOrdenBolsa,
	) (ContextoPeticionIntegracionBolsa, error)
}
