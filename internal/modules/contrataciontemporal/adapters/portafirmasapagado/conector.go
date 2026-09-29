// Package portafirmasapagado es el conector de Firmadoc mientras no exista
// su interfaz: siempre «no conectado» y ningún envío. No simula respuestas.
package portafirmasapagado

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Conector no envía nada y lo dice.
type Conector struct{}

var _ ports.ConectorPortafirmas = Conector{}

// EstadoConexion responde siempre que falta la conexión.
func (Conector) EstadoConexion(ctx context.Context) (ports.EstadoConexionPortafirmas, error) {
	if ctx == nil {
		return ports.EstadoConexionPortafirmas{}, ports.ErrPortafirmasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.EstadoConexionPortafirmas{}, err
	}
	return ports.EstadoConexionPortafirmas{Motivo: ports.MotivoPortafirmasConexionPendiente}, nil
}

// Enviar rechaza siempre: no hay portafirmas al que enviar.
func (Conector) Enviar(context.Context, ports.SolicitudEnvioPortafirmas) (ports.ReciboEnvioPortafirmas, error) {
	return ports.ReciboEnvioPortafirmas{}, ports.ErrPortafirmasNoDisponible
}
