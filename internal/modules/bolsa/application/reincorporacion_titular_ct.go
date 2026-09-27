package application

import (
	"context"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ListarReincorporacionesTitular usa el mismo contexto y consumo V3 que el
// histórico de contratos de la ficha RRHH. La base restringe la lectura a la
// participación autorizada y obtiene la fecha disponible de Bolsa 000045.
func (s *ServicioSituacionParticipacion) ListarReincorporacionesTitular(ctx context.Context, q ports.SolicitudCambiarSituacionParticipacion) ([]ports.ReincorporacionTitularFicha, error) {
	if s == nil || ctx == nil || q.Validar() != nil {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	repo, ok := s.repositorio.(ports.RepositorioReincorporacionesTitularCT)
	if !ok {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	_, _, _, material, err := s.autorizarOperacion(ctx, q)
	if err != nil {
		return nil, err
	}
	return repo.ListarReincorporacionesTitular(ctx, q.ParticipacionRef, q.ResultadoContexto.Contexto.PersonaRef, material)
}

type ResultadoEntregaReincorporacionesTitular struct {
	Nuevas, Reutilizadas, PendientesCese, CesesIncompatibles int
}

// ServicioRecepcionReincorporacionesTitular consume un feed paginado de CT130
// y entrega cada triple al verificador de origen de Bolsa 000046. No interpreta
// el evento de retorno como otro cese ni calcula meses de indisponibilidad.
type ServicioRecepcionReincorporacionesTitular struct {
	fuente ports.FuenteReincorporacionesTitularCT
	buzon  ports.BuzonReincorporacionesTitularCT
}

func NuevoServicioRecepcionReincorporacionesTitular(fuente ports.FuenteReincorporacionesTitularCT, buzon ports.BuzonReincorporacionesTitularCT) (*ServicioRecepcionReincorporacionesTitular, error) {
	if fuente == nil || buzon == nil {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	return &ServicioRecepcionReincorporacionesTitular{fuente: fuente, buzon: buzon}, nil
}

// Entregar hace hasta 20 páginas por pasada. El cursor durable está en la
// bandeja: un fallo tras una inserción se recupera al volver a empezar.
func (s *ServicioRecepcionReincorporacionesTitular) Entregar(ctx context.Context, limite int) (ResultadoEntregaReincorporacionesTitular, error) {
	var resultado ResultadoEntregaReincorporacionesTitular
	if s == nil || s.fuente == nil || s.buzon == nil || ctx == nil || limite < 1 || limite > 100 {
		return resultado, ports.ErrReincorporacionTitularNoDisponible
	}
	cursor, hay, err := s.buzon.CursorReincorporacionesTitular(ctx)
	if err != nil {
		return resultado, err
	}
	var desde *ports.CursorContratosParticipacion
	if hay {
		desde = &cursor
	}
	for pagina := 0; pagina < 20; pagina++ {
		eventos, err := s.fuente.LeerReincorporacionesTitular(ctx, desde, limite)
		if err != nil {
			return resultado, err
		}
		if len(eventos) > limite {
			return resultado, ports.ErrReincorporacionTitularNoDisponible
		}
		for _, evento := range eventos {
			if evento.EventoRef == "" || evento.EventoRef != evento.OrigenRef || evento.OrigenPosicion < 0 ||
				len(evento.HuellaSHA256) != 64 || strings.TrimSpace(evento.OrigenRef) != evento.OrigenRef ||
				(desde != nil && (evento.OrigenPosicion < desde.Posicion ||
					(evento.OrigenPosicion == desde.Posicion && evento.OrigenRef <= desde.OrigenRef))) {
				return resultado, ports.ErrReincorporacionTitularNoDisponible
			}
			r, err := s.buzon.RegistrarReincorporacionTitular(ctx, evento)
			if err != nil {
				return resultado, err
			}
			if r.Reutilizada {
				resultado.Reutilizadas++
			} else {
				resultado.Nuevas++
			}
			switch r.Estado {
			case "cese_aplicado":
			case "pendiente_cese":
				resultado.PendientesCese++
			case "cese_incompatible":
				resultado.CesesIncompatibles++
			default:
				return resultado, ports.ErrReincorporacionTitularNoDisponible
			}
			cursor = ports.CursorContratosParticipacion{Posicion: evento.OrigenPosicion, OrigenRef: evento.OrigenRef}
			desde = &cursor
		}
		if len(eventos) < limite {
			break
		}
	}
	return resultado, nil
}
