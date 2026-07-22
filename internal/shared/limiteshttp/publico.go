// Package limiteshttp centraliza presupuestos de transporte que deben
// coincidir entre la composición, el servidor y los adaptadores HTTP.
package limiteshttp

import "time"

const (
	DuracionMaximaPeticionPublica  = 8 * time.Second
	PresupuestoEscrituraPublica    = 2 * time.Second
	ReservaLimpiezaPublica         = 2 * time.Second
	DuracionMaximaRetencionCupo    = DuracionMaximaPeticionPublica - PresupuestoEscrituraPublica
	DuracionMaximaOperacionPublica = DuracionMaximaRetencionCupo - ReservaLimpiezaPublica
)
