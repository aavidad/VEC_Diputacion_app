package application

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ServicioRecepcionNoIncorporaciones es el consumidor de las no
// incorporaciones de Contratación temporal (CT124): valida el evento, calcula
// con el calendario del catálogo de Bolsa las fechas de su consecuencia y lo
// entrega a la bandeja, que comprueba el origen con CT, resuelve la
// consecuencia con la política publicada y la aplica una sola vez.
type ServicioRecepcionNoIncorporaciones struct {
	buzon    ports.BuzonNoIncorporaciones
	catalogo ports.ResolvedorSancionNoIncorporacion
}

func NuevoServicioRecepcionNoIncorporaciones(buzon ports.BuzonNoIncorporaciones, catalogo ports.ResolvedorSancionNoIncorporacion) (*ServicioRecepcionNoIncorporaciones, error) {
	if buzon == nil || catalogo == nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	return &ServicioRecepcionNoIncorporaciones{buzon: buzon, catalogo: catalogo}, nil
}

// Cursor devuelve el último origen recibido; false si la bandeja está vacía.
func (s *ServicioRecepcionNoIncorporaciones) Cursor(ctx context.Context) (ports.CursorContratosParticipacion, bool, error) {
	if s == nil || ctx == nil {
		return ports.CursorContratosParticipacion{}, false, ports.ErrContratosParticipacionNoDisponible
	}
	return s.buzon.CursorNoIncorporaciones(ctx)
}

// Recibir registra una sola vez el evento. Si el catálogo no reconoce la
// clave, el evento se entrega sin plazos y queda sin efecto; si el catálogo
// no está disponible, no se entrega nada y el relevo reintentará.
func (s *ServicioRecepcionNoIncorporaciones) Recibir(ctx context.Context, contenido []byte, huellaSHA256 string, origenCreadaEn time.Time, origenPosicion int64) (ports.ResultadoRegistroNoIncorporacion, error) {
	if s == nil || ctx == nil || origenCreadaEn.IsZero() || origenPosicion < 0 {
		return ports.ResultadoRegistroNoIncorporacion{}, ports.ErrContratosParticipacionNoDisponible
	}
	evento, err := dominiobolsa.DecodificarEventoNoIncorporacion(contenido, huellaSHA256)
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, err
	}
	notificada, err := evento.Notificacion()
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, err
	}
	plazos, err := s.plazos(ctx, evento.ConsecuenciaClave, notificada)
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, err
	}
	copia := append([]byte(nil), contenido...)
	return s.buzon.RegistrarNoIncorporacion(ctx, ports.EventoNoIncorporacionRecibido{Evento: evento, Contenido: copia, HuellaSHA256: huellaSHA256,
		OrigenCreadaEn: origenCreadaEn.UTC(), OrigenPosicion: origenPosicion, Plazos: plazos})
}

// ReevaluarPendientes vuelve a evaluar lo que la bandeja aún no ha aplicado
// (aceptación posterior, política nueva o situación corregida por RRHH).
// Devuelve cuántos quedaron aplicados en esta pasada.
func (s *ServicioRecepcionNoIncorporaciones) ReevaluarPendientes(ctx context.Context, limite int) (int, error) {
	if s == nil || ctx == nil || limite < 1 || limite > ports.LimitePendientesNoIncorporacion {
		return 0, ports.ErrContratosParticipacionNoDisponible
	}
	pendientes, err := s.buzon.PendientesNoIncorporacion(ctx, limite)
	if err != nil {
		return 0, err
	}
	aplicadas := 0
	for _, p := range pendientes {
		notificada, err := time.Parse(time.DateOnly, p.FechaNotificacion)
		if err != nil || notificada.Format(time.DateOnly) != p.FechaNotificacion {
			return aplicadas, ports.ErrContratosParticipacionNoDisponible
		}
		plazos, err := s.plazos(ctx, p.ConsecuenciaClave, notificada.UTC())
		if err != nil {
			return aplicadas, err
		}
		r, err := s.buzon.ReevaluarNoIncorporacion(ctx, p.EventoRef, plazos)
		if err != nil {
			return aplicadas, err
		}
		if r.Estado == ports.EstadoNoIncorporacionAplicada {
			aplicadas++
		}
	}
	return aplicadas, nil
}

// plazos calcula con el calendario del catálogo las fechas de la
// consecuencia; nil si el catálogo no reconoce la clave.
func (s *ServicioRecepcionNoIncorporaciones) plazos(ctx context.Context, clave string, notificada time.Time) (*ports.PlazosNoIncorporacion, error) {
	r, err := s.catalogo.ResolverSancion(ctx, clave, notificada)
	switch {
	case errors.Is(err, dominiobolsa.ErrSancionParticipacionInvalida):
		return nil, nil
	case err != nil:
		return nil, err
	}
	plazos := &ports.PlazosNoIncorporacion{RecursoVence: r.Recurso.UltimoDia, RecursoReglaRef: r.Recurso.ReglaRef,
		RecursoReglaHuellaSHA256: r.Recurso.Huella}
	if r.SuspensionHasta != "" {
		hasta := r.SuspensionHasta
		plazos.SuspensionHasta = &hasta
	}
	return plazos, nil
}
