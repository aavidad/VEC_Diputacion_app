package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var ErrConsultaAvisosNoDisponible = errors.New("bolsa: consulta de avisos no disponible")

type ConsultaAvisosRRHH interface {
	ListarAvisosRRHH(context.Context, time.Time, int, int) ([]dominiobolsa.AvisoRRHH, error)
	ContarAvisosRRHH(context.Context, time.Time) (map[string]int, error)
}
