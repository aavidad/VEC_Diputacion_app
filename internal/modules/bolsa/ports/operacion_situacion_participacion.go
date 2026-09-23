package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
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
	RegistrarOperacion(context.Context, ComandoOperacionSituacion) (RegistroSituacionParticipacion, error)
	ListarOperaciones(context.Context, string, string, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]RegistroOperacionSituacion, error)
}
