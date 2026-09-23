package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var ErrClaveOperacionReutilizada = errors.New("bolsa: clave de operacion reutilizada")

type SolicitudOperacionSituacion struct {
	SolicitudCambiarSituacionParticipacion
	Operacion    string
	Justificante dominiobolsa.JustificanteOperacionSituacion
	Validador    string
}

type ComandoOperacionSituacion struct {
	ComandoCambiarSituacionParticipacion
	Operacion    string
	Justificante dominiobolsa.JustificanteOperacionSituacion
	Validador    string
	ValidadaEn   time.Time
}

type RegistroOperacionSituacion struct {
	RegistroSituacionParticipacion
	Operacion    string
	Justificante dominiobolsa.JustificanteOperacionSituacion
	Actor        string
	Validador    string
	ValidadaEn   time.Time
}

type RepositorioOperacionSituacion interface {
	BuscarOperacion(context.Context, string, string) (RegistroOperacionSituacion, error)
	RegistrarOperacion(context.Context, ComandoOperacionSituacion) (RegistroSituacionParticipacion, error)
	ListarOperaciones(context.Context, string) ([]RegistroOperacionSituacion, error)
}
