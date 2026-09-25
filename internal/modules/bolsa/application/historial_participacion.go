package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ListarHistorial devuelve las operaciones B8 y la traza de valores anterior y
// nuevo (petición RRHH p.4) con el mismo permiso que el historial B8.
func (s *ServicioSituacionParticipacion) ListarHistorial(ctx context.Context, q ports.SolicitudCambiarSituacionParticipacion) (ports.HistorialParticipacion, error) {
	if s == nil || ctx == nil || q.Validar() != nil {
		return ports.HistorialParticipacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	repo, ok := s.repositorio.(ports.RepositorioHistorialParticipacion)
	if !ok {
		return ports.HistorialParticipacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	_, _, _, material, err := s.autorizarOperacion(ctx, q)
	if err != nil {
		return ports.HistorialParticipacion{}, err
	}
	return repo.ListarHistorial(ctx, q.ParticipacionRef, q.ResultadoContexto.Contexto.PersonaRef, material)
}
