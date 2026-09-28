package application

import (
	"context"
	"errors"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// ListarReincorporacionesTitular consume una decisión V3 de lectura nominal.
// La pertenencia se contrasta antes de emitir material; la función SQL vuelve
// a verificar recurso, actor, ámbito, vigencia y consumo dentro de la lectura.
func (s *ServicioSituacionParticipacion) ListarReincorporacionesTitular(ctx context.Context, q ports.SolicitudConsultarReincorporacionesTitular) ([]ports.ReincorporacionTitularFicha, error) {
	if s == nil || ctx == nil || q.Validar() != nil {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	repo, ok := s.repositorio.(ports.RepositorioReincorporacionesTitularCT)
	if !ok || s.contexto == nil || s.autorizador == nil || s.repositorio == nil {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	actor := q.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, actor, q.BolsaRef, q.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil || actor.PersonaRef == "" {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
			return nil, err
		}
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	pertenece, err := s.repositorio.ParticipacionPerteneceABolsa(ctx, q.BolsaRef, q.ParticipacionRef)
	if err != nil {
		return nil, err
	}
	if !pertenece {
		return nil, dominiovec.ErrAutorizacionDenegada
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: q.ParticipacionRef, ModuloID: ports.ModuloSituacionParticipacion,
		Tipo:    ports.TipoRecursoSituacionParticipacion,
		Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion,
		Accion: ports.AccionConsultarReincorporacionTitular, Recurso: recurso,
		Finalidad: ports.FinalidadConsultarReincorporacionTitular, Correlacion: q.Correlacion,
	})
	if err != nil {
		return nil, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, q.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(solicitud) != nil {
		return nil, errorDependenciaSituacion(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(solicitud, decision, confirmacion, q.ResultadoContexto,
		q.MotivoAutorizacion, material, ports.AudienciaConsultarReincorporacionTitular) {
		return nil, errorDependenciaSituacion(err)
	}
	return repo.ListarReincorporacionesTitular(ctx, q.ParticipacionRef, actor.Principal.ID, material)
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
