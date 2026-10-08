package ports

import (
	"context"
	"errors"
	"time"
)

var ErrConsultaEstadoCeseNoDisponible = errors.New("bolsa: estado de cese no disponible")

// EstadoCese une la proyección efectiva B45 con un cese B13 aún sin resolver.
// PendienteDesde es el instante real de recepción B13, no una fecha de plazo.
type EstadoCese struct {
	FechaEfecto     time.Time
	DisponibleDesde time.Time
	EnRestriccion   bool
	TrabajoCesado   bool
	// CesePendiente impide ofrecer la participación mientras el cese B13
	// sigue sin restricción B45. No implica fecha de disponibilidad.
	CesePendiente  bool
	PendienteDesde time.Time
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
