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

// CursorPublicacionContratosBolsa marca el último evento consumido. El valor
// cero lee desde el principio; un instante con OrigenRef vacío lee desde ese
// instante incluido (relectura). Lo conserva el consumidor, no CT.
type CursorPublicacionContratosBolsa struct {
	CreadaEn  time.Time
	OrigenRef string
}

// Vacio indica que no hay cursor y la lectura empieza por el principio.
func (c CursorPublicacionContratosBolsa) Vacio() bool {
	return c.CreadaEn.IsZero()
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
	OrigenCreadaEn time.Time
}

// LectorPublicacionContratosBolsa lee eventos posteriores al cursor, en orden.
type LectorPublicacionContratosBolsa interface {
	LeerContratosBolsa(ctx context.Context, desde CursorPublicacionContratosBolsa, limite int) ([]EventoContratoBolsaPublicado, error)
}
