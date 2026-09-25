package ports

import (
	"context"
	"errors"
	"time"
)

// ErrPublicacionContratosBolsaNoDisponible cubre cualquier fallo de lectura:
// la indisponibilidad nunca se interpreta como ausencia de contratos.
var ErrPublicacionContratosBolsaNoDisponible = errors.New("contratacion temporal: publicacion de contratos a Bolsa no disponible")

// LimiteLecturaContratosBolsa es el máximo de eventos por página que admite
// la función SQL CT113.
const LimiteLecturaContratosBolsa = 100

// CursorPublicacionContratosBolsa marca el último evento consumido por su
// posición de publicación (la transacción CT que lo escribió) y su origen. El
// valor cero lee desde el principio. CT solo publica eventos de transacciones
// ya terminadas (marca de agua), así que nada puede aparecer después por
// detrás del cursor. Lo conserva el consumidor, no CT.
type CursorPublicacionContratosBolsa struct {
	Posicion  int64
	OrigenRef string
}

// Vacio indica que no hay cursor y la lectura empieza por el principio.
func (c CursorPublicacionContratosBolsa) Vacio() bool {
	return c.OrigenRef == ""
}

// EventoContratoBolsaPublicado es el evento de integración que CT proyecta
// desde su outbox de incorporación. Contenido es el JSON canónico exacto
// cuya huella SHA-256 se publica; solo lleva referencias opacas, fechas y
// claves de catálogo, sin datos personales.
type EventoContratoBolsaPublicado struct {
	EventoRef      string
	Contenido      []byte
	HuellaSHA256   string
	OrigenRef      string
	OrigenPosicion int64
	OrigenCreadaEn time.Time
}

// LectorPublicacionContratosBolsa lee eventos posteriores al cursor, en orden.
type LectorPublicacionContratosBolsa interface {
	LeerContratosBolsa(ctx context.Context, desde CursorPublicacionContratosBolsa, limite int) ([]EventoContratoBolsaPublicado, error)
}
