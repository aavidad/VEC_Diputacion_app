package application

import dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"

// TransicionarLlamamientoConOrdenTerminalAutorizadaV2 deriva en memoria el
// agregado terminal ligado por una orden PRE-CAP valida.
func TransicionarLlamamientoConOrdenTerminalAutorizadaV2(
	orden OrdenTerminalLlamamientoAutorizadaV2,
) (dominiobolsa.LlamamientoAbierto, error) {
	proyeccion, err := orden.ReacreditarYProyectar()
	if err != nil {
		return dominiobolsa.LlamamientoAbierto{}, ErrOrdenTerminalLlamamientoInvalida
	}
	return transicionarLlamamientoProyectadoAutorizadoV2(proyeccion)
}

func transicionarLlamamientoProyectadoAutorizadoV2(
	proyeccion ProyeccionOrdenTerminalLlamamientoAutorizadaV2,
) (dominiobolsa.LlamamientoAbierto, error) {
	llamamiento, versionEsperada, terminal, err := proyeccion.Datos()
	if err != nil {
		return dominiobolsa.LlamamientoAbierto{}, ErrOrdenTerminalLlamamientoInvalida
	}
	derivado, err := llamamiento.TransicionarATerminal(versionEsperada, &terminal)
	if err != nil || derivado.Validar() != nil {
		return dominiobolsa.LlamamientoAbierto{}, ErrOrdenTerminalLlamamientoInvalida
	}
	return derivado, nil
}
