package application

import (
	"context"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ListarContratos devuelve el histórico B13 de una participación. Exige la
// misma autorización V3 que el historial B8 de la ficha, consumida en SQL.
func (s *ServicioSituacionParticipacion) ListarContratos(ctx context.Context, q ports.SolicitudCambiarSituacionParticipacion) ([]ports.ContratoParticipacion, error) {
	if s == nil || ctx == nil || q.Validar() != nil {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	repo, ok := s.repositorio.(ports.RepositorioContratosParticipacion)
	if !ok {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	_, _, _, material, err := s.autorizarOperacion(ctx, q)
	if err != nil {
		return nil, err
	}
	return repo.ListarContratos(ctx, q.ParticipacionRef, q.ResultadoContexto.Contexto.PersonaRef, material)
}

// ServicioRecepcionContratos es el consumidor B13 de los eventos de contrato
// de Contratación temporal. Valida el evento completo antes del inbox.
type ServicioRecepcionContratos struct {
	buzon ports.BuzonContratosParticipacion
}

func NuevoServicioRecepcionContratos(buzon ports.BuzonContratosParticipacion) (*ServicioRecepcionContratos, error) {
	if buzon == nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	return &ServicioRecepcionContratos{buzon: buzon}, nil
}

// Cursor devuelve el último origen recibido; false si el inbox está vacío.
func (s *ServicioRecepcionContratos) Cursor(ctx context.Context) (ports.CursorContratosParticipacion, bool, error) {
	if s == nil || ctx == nil {
		return ports.CursorContratosParticipacion{}, false, ports.ErrContratosParticipacionNoDisponible
	}
	return s.buzon.CursorContratos(ctx)
}

// Recibir registra una sola vez el evento. Una reentrega idéntica devuelve
// Reutilizado; un contenido distinto para el mismo evento es un error.
func (s *ServicioRecepcionContratos) Recibir(ctx context.Context, contenido []byte, huellaSHA256 string, origenCreadaEn time.Time) (ports.ResultadoRegistroContrato, error) {
	if s == nil || ctx == nil || origenCreadaEn.IsZero() {
		return ports.ResultadoRegistroContrato{}, ports.ErrContratosParticipacionNoDisponible
	}
	evento, err := dominiobolsa.DecodificarEventoContratoParticipacion(contenido, huellaSHA256)
	if err != nil {
		return ports.ResultadoRegistroContrato{}, err
	}
	copia := append([]byte(nil), contenido...)
	return s.buzon.RegistrarContrato(ctx, ports.EventoContratoRecibido{Evento: evento, Contenido: copia, HuellaSHA256: huellaSHA256, OrigenCreadaEn: origenCreadaEn.UTC()})
}
