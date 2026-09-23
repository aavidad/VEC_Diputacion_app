package ports

import (
	"context"
	"errors"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var ErrConsultaOrdenVigenteNoDisponible = errors.New("bolsa: consulta de orden vigente no disponible")

type ConsultaOrdenVigente interface {
	ConsultarOrdenVigente(context.Context, string) (dominiobolsa.OrdenVigenteBolsa, error)
}
