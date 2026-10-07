package ports

import (
	"context"
	"errors"
	"time"
)

var ErrConsultaEstadoCeseNoDisponible = errors.New("bolsa: estado de cese no disponible")

// EstadoCese es la proyección efectiva B45: fecha del último cese acreditado,
// máximo de disponibilidad entre todos sus ceses y guardia de otra relación.
// No traslada candidato_ref ni historia personal a la participación.
type EstadoCese struct {
	FechaEfecto     time.Time
	DisponibleDesde time.Time
	EnRestriccion   bool
	TrabajoCesado   bool
}

type ConsultaEstadoCese interface {
	ConsultarEstadoCese(context.Context, string, time.Time) (EstadoCese, bool, error)
}

// ConsultaEstadosCese consulta el estado de cese de varias participaciones en
// una sola ida y vuelta, con la misma fachada y guardas que ConsultarEstadoCese.
// El mapa solo contiene las participaciones con estado presente.
type ConsultaEstadosCese interface {
	ConsultarEstadosCese(context.Context, []string, time.Time) (map[string]EstadoCese, error)
}
