package ports

import (
	"context"
	"errors"
	"time"
)

var ErrConsultaRestriccionCeseNoDisponible = errors.New("bolsa: restriccion de cese no disponible")

// RestriccionCese es la lectura mínima B45 para una participación en un corte.
// La restricción pertenece al candidato y no se copia a sus participaciones.
type RestriccionCese struct {
	DisponibleDesde time.Time
	FechaEfecto     time.Time
	ReciboRef       string
	PoliticaVersion uint64
}

type ConsultaRestriccionCese interface {
	ConsultarRestriccionCese(context.Context, string, time.Time) (RestriccionCese, bool, error)
}
