package ports

import (
	"context"
	"errors"
	"time"
)

var ErrConsultaEstadoCeseNoDisponible = errors.New("bolsa: estado de cese no disponible")

// EstadoCese une la proyección efectiva B45 con la presencia de un cese B13
// aún sin vínculo. No traslada candidato_ref ni historia personal.
type EstadoCese struct {
	FechaEfecto     time.Time
	DisponibleDesde time.Time
	EnRestriccion   bool
	TrabajoCesado   bool
	// CesePendiente impide ofrecer la participación mientras el cese CT aún no
	// tiene vínculo B8. No implica una fecha de efecto ni de disponibilidad.
	CesePendiente bool
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
