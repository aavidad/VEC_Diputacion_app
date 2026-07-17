package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// exigirDecisionAutorizacionVinculadaSolicitudLigadaV2 es el PEP exclusivo de
// V2. Construye una solicitud completa con capacidades confiables y coteja sus
// dos compromisos; no acepta ni convierte decisiones V1.
func exigirDecisionAutorizacionVinculadaSolicitudLigadaV2(
	ctx context.Context,
	autorizador ports.AutorizadorSolicitudLigadaV2,
	reloj ports.Reloj,
	actor domain.ContextoActor,
	vinculo domain.VinculoAutenticacionActorV1,
	accion string,
	recurso domain.RecursoAutorizable,
	finalidad, correlacionRef string,
	motivo domain.ReferenciaEntradaCatalogo,
	usoCampos usoCamposDecisionAutorizacion,
) (domain.DecisionAutorizacion, error) {
	if ctx == nil || dependenciaAutorizacionNula(autorizador) || dependenciaAutorizacionNula(reloj) ||
		usoCampos == usoCamposDecisionNoDeclarado || usoCampos > usoCamposDecisionConsumidos {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrConfiguracionAccesoInvalida,
		)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	actorCanonico, err := actor.Clonar()
	if err != nil || vinculo.ValidarPara(actorCanonico) != nil {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrVinculoAutenticacionActorInvalido,
			err,
		)
	}
	instante := reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instante.IsZero() || !vinculo.VigenteEn(instante, actorCanonico) {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrVinculoAutenticacionActorInvalido,
		)
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV2(
		domain.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: actorCanonico, VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: motivo, Accion: accion,
			Recurso: clonarRecursoUsoAutorizacion(recurso), Finalidad: finalidad,
			CorrelacionRef: correlacionRef,
		},
	)
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	huellaContexto, err := datosSolicitud.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	huellaSolicitud, err := domain.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	decision, err := autorizador.ExigirSolicitudLigadaV2(ctx, solicitud)
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	instante = reloj.Ahora().UTC()
	if decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil ||
		!decision.VigenteParaEfectoEn(instante) ||
		!vinculo.CoincideExactamenteCon(decision.VinculoAutenticacionActor) ||
		decision.PrincipalID != actorCanonico.Principal.ID ||
		decision.PerfilActivoRef != actorCanonico.PerfilActivoRef ||
		decision.Accion != datosSolicitud.Accion || decision.RecursoRef != datosSolicitud.Recurso.Referencia ||
		decision.ModuloID != datosSolicitud.Recurso.ModuloID || decision.TipoRecurso != datosSolicitud.Recurso.Tipo ||
		decision.ContextoRecursoHuellaSHA256 != huellaContexto ||
		decision.Finalidad != datosSolicitud.Finalidad || decision.CorrelacionRef != datosSolicitud.CorrelacionRef ||
		decision.EsquemaHuellaSolicitud != domain.EsquemaHuellaSolicitudAutorizacionV2 ||
		decision.SolicitudHuellaSHA256 != huellaSolicitud ||
		decision.EsquemaHuellaMotivo != domain.EsquemaHuellaMotivoAutorizacionV2 ||
		decision.MotivoHuellaSHA256 != huellaMotivo ||
		!domain.CumpleGarantiaAutenticacion(actorCanonico.Principal.AuthAssurance, decision.GarantiaMinima) ||
		len(decision.Obligaciones) != 0 ||
		(usoCampos == usoCamposDecisionNoAplicables && len(decision.CamposPermitidos) != 0) {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrDecisionAutorizacionInvalida,
		)
	}
	return decision, nil
}
