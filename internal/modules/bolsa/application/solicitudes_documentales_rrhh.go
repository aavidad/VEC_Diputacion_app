package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ListarSolicitudesDocumentalesRRHH lee únicamente las pendientes de la
// participación autorizada. La autorización V3 se consume dentro de SQL.
func (s *ServicioSituacionParticipacion) ListarSolicitudesDocumentalesRRHH(ctx context.Context, q ports.SolicitudCambiarSituacionParticipacion) ([]ports.SolicitudDocumentalPendienteRRHH, error) {
	if s == nil || ctx == nil || q.Validar() != nil || s.contexto == nil || s.autorizador == nil || s.repositorio == nil {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	repo, ok := s.repositorio.(ports.ConsultaSolicitudesDocumentalesRRHH)
	if !ok {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, q.ResultadoContexto.Contexto, q.BolsaRef, q.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil {
		return nil, errorDependenciaSituacion(err)
	}
	pertenece, err := s.repositorio.ParticipacionPerteneceABolsa(ctx, q.BolsaRef, q.ParticipacionRef)
	if err != nil {
		return nil, err
	}
	if !pertenece {
		return nil, dominiovec.ErrAutorizacionDenegada
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: q.ParticipacionRef, ModuloID: ports.ModuloSituacionParticipacion, Tipo: ports.TipoRecursoSituacionParticipacion,
		Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion,
		Accion: ports.AccionConsultarSolicitudesDocumentalesRRHH, Recurso: recurso,
		Finalidad: ports.FinalidadCambiarSituacionParticipacion, Correlacion: q.Correlacion,
	})
	if err != nil {
		return nil, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return nil, dominiovec.ErrAutorizacionDenegada
		}
		return nil, errorDependenciaSituacion(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, q.ResultadoContexto, q.MotivoAutorizacion, material, ports.AudienciaConsultarSolicitudesDocumentalesRRHH) {
		return nil, errorDependenciaSituacion(err)
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	return repo.ListarSolicitudesDocumentalesPendientes(ctx, q.BolsaRef, q.ParticipacionRef, q.ResultadoContexto.Contexto.PersonaRef, ahora, material)
}
