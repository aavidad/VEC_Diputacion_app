package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var ErrCambioSituacionParticipacionNoDisponible = errors.New("bolsa: cambio de situacion de participacion no disponible")

// AutorizadorCambioSituacionParticipacion representa la concesión V3 que la
// frontera interna resuelve con actor y ámbito de servidor, nunca desde HTTP.
type AutorizadorCambioSituacionParticipacion interface {
	AutorizarCambioSituacionParticipacion(context.Context, string, string) (string, error)
}

type SolicitudCambioSituacionParticipacion struct {
	ParticipacionRef, Destino, Motivo, ClaveIdempotencia string
	Desde                                                time.Time
	FechaDisponible                                      *time.Time
}

type ServicioSituacionParticipacion struct {
	autorizador AutorizadorCambioSituacionParticipacion
	repositorio puertosbolsa.RepositorioSituacionParticipacion
	reloj       func() time.Time
}

func NuevoServicioSituacionParticipacion(a AutorizadorCambioSituacionParticipacion, r puertosbolsa.RepositorioSituacionParticipacion, reloj func() time.Time) (*ServicioSituacionParticipacion, error) {
	if a == nil || r == nil || reloj == nil {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	return &ServicioSituacionParticipacion{autorizador: a, repositorio: r, reloj: reloj}, nil
}

func (s *ServicioSituacionParticipacion) Cambiar(ctx context.Context, solicitud SolicitudCambioSituacionParticipacion) (puertosbolsa.RegistroSituacionParticipacion, error) {
	if ctx == nil || s == nil || s.autorizador == nil || s.repositorio == nil || strings.TrimSpace(solicitud.ClaveIdempotencia) != solicitud.ClaveIdempotencia || solicitud.ClaveIdempotencia == "" {
		return puertosbolsa.RegistroSituacionParticipacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	actor, err := s.autorizador.AutorizarCambioSituacionParticipacion(ctx, solicitud.ParticipacionRef, solicitud.Destino)
	if err != nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, err
	}
	if actor == "" {
		return puertosbolsa.RegistroSituacionParticipacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	// La recuperación se hace después de la autorización positiva. Así una
	// clave conocida no permite consultar un recibo fuera de ámbito y el
	// reintento no vuelve a evaluar la transición ya registrada.
	previo, err := s.repositorio.BuscarRegistroSituacion(ctx, solicitud.ParticipacionRef, solicitud.ClaveIdempotencia)
	if err == nil {
		if previo.Situacion != solicitud.Destino || previo.ParticipacionRef != solicitud.ParticipacionRef ||
			previo.Motivo != solicitud.Motivo || !previo.Desde.Equal(solicitud.Desde.UTC().Truncate(time.Microsecond)) ||
			!mismaFechaDisponible(previo.FechaDisponible, solicitud.FechaDisponible) {
			return puertosbolsa.RegistroSituacionParticipacion{}, dominiobolsa.ErrCambioSituacionParticipacionInvalido
		}
		return previo, nil
	}
	if !errors.Is(err, puertosbolsa.ErrSituacionParticipacionNoEncontrada) {
		return puertosbolsa.RegistroSituacionParticipacion{}, err
	}
	vigente, err := s.repositorio.SituacionVigente(ctx, solicitud.ParticipacionRef)
	if err != nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, err
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	cambio := dominiobolsa.CambioSituacionParticipacion{ParticipacionRef: solicitud.ParticipacionRef, Origen: vigente.Situacion, Destino: solicitud.Destino, Desde: solicitud.Desde.UTC().Truncate(time.Microsecond), Motivo: solicitud.Motivo, FechaDisponible: solicitud.FechaDisponible, RegistradaEn: ahora}
	if cambio.Validar() != nil || cambio.Desde.Before(vigente.Desde) {
		return puertosbolsa.RegistroSituacionParticipacion{}, dominiobolsa.ErrCambioSituacionParticipacionInvalido
	}
	h := sha256.Sum256([]byte(solicitud.ParticipacionRef + "\x1f" + solicitud.ClaveIdempotencia))
	recibo := "recibo:situacion:" + hex.EncodeToString(h[:])
	return s.repositorio.RegistrarSituacion(ctx, solicitud.ParticipacionRef, solicitud.Destino, cambio.Desde, solicitud.FechaDisponible, solicitud.Motivo, actor, solicitud.ClaveIdempotencia, recibo, ahora)
}

func mismaFechaDisponible(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.UTC().Truncate(time.Microsecond).Equal(b.UTC().Truncate(time.Microsecond))
}
