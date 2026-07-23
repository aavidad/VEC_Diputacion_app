package ports

import (
	"fmt"
	"log/slog"
)

// ExportadorCapacidadAtestacionAutorizacionV3 es la unica frontera que el
// futuro consumidor SQL necesita conocer. La exportacion no concede autoridad
// por su tipo Go: el consumidor independiente debe verificar de nuevo formato
// canonico, MAC, clave gobernada, rotacion, revocacion, audiencia, vigencia y
// ligaduras exactas antes del efecto.
//
// El contrato es neutral al cliente. No recibe HTTP, cookies, almacenamiento
// de navegador ni cabeceras de identidad.
type ExportadorCapacidadAtestacionAutorizacionV3 interface {
	fmt.Stringer
	slog.LogValuer
	ExportacionCanonicaParaConsumidor() ([]byte, error)
}
