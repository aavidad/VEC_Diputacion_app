// Package capturafisica contiene el contrato de plataforma para captura fría.
package capturafisica

import "context"

// Control observa la exclusión y el estado real de PostgreSQL. La aplicación
// que posee la ventana conserva la exclusión durante todas estas operaciones.
// Ningún valor declarado en el inventario sustituye estas comprobaciones.
type Control interface {
	ComprobarExclusion(context.Context) error
	DetenerPostgreSQL(context.Context) error
	ComprobarFrio(context.Context) error
	ReanudarPostgreSQL(context.Context) error
}
