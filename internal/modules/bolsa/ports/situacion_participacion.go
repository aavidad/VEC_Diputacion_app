package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSituacionParticipacionNoDisponible = errors.New("bolsa: situacion de participacion no disponible")
	ErrSituacionParticipacionNoEncontrada = errors.New("bolsa: participacion no encontrada")
)

type SituacionParticipacion struct {
	ParticipacionRef, Situacion string
	Desde                       time.Time
	FechaDisponible             *time.Time
}
type RegistroSituacionParticipacion struct {
	Reutilizada bool
	ReciboRef   string
	Motivo      string
	SituacionParticipacion
}

type RepositorioSituacionParticipacion interface {
	SituacionVigente(context.Context, string) (SituacionParticipacion, error)
	BuscarRegistroSituacion(context.Context, string, string) (RegistroSituacionParticipacion, error)
	RegistrarSituacion(context.Context, string, string, time.Time, *time.Time, string, string, string, string, time.Time) (RegistroSituacionParticipacion, error)
}
