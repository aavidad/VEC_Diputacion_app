// Package nodisponible es el adaptador de los servicios externos de
// Selección que aún no tienen proveedor (firma, registro en sede, tasas y
// notificación): responde siempre «no disponible», nunca un éxito aparente.
package nodisponible

import (
	"context"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// Servicios implementa los cuatro puertos externos sin proveedor.
type Servicios struct{}

var (
	_ ports.FirmaSolicitud         = Servicios{}
	_ ports.RegistroSede           = Servicios{}
	_ ports.PasarelaTasas          = Servicios{}
	_ ports.NotificadorSolicitudes = Servicios{}
)

func (Servicios) FirmarPresentacion(context.Context, ports.PresentacionExterna) error {
	return ports.ErrServicioExternoNoDisponible
}

func (Servicios) RegistrarPresentacion(context.Context, ports.PresentacionExterna) error {
	return ports.ErrServicioExternoNoDisponible
}

func (Servicios) ComprobarTasa(context.Context, ports.PresentacionExterna) error {
	return ports.ErrServicioExternoNoDisponible
}

func (Servicios) NotificarPresentacion(context.Context, ports.PresentacionExterna) error {
	return ports.ErrServicioExternoNoDisponible
}
