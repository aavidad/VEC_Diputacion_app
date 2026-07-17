package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// usoCamposDecisionAutorizacion obliga a que cada caso de uso declare si
// interpreta la lista positiva de campos de la decision. No existe un valor
// predeterminado util: el cero es invalido para que una ampliacion futura no
// termine aceptada por olvido.
type usoCamposDecisionAutorizacion uint8

const (
	usoCamposDecisionNoDeclarado usoCamposDecisionAutorizacion = iota
	usoCamposDecisionNoAplicables
	usoCamposDecisionConsumidos
)

// autorizadorConVinculo inserta las capacidades de identidad resueltas por la
// frontera confiable. Ningun DTO puede aportar estos campos al PDP y una
// decision que responda con otro vinculo se descarta antes de llegar al caso
// de uso.
type autorizadorConVinculo struct {
	base    ports.Autorizador
	actor   domain.ContextoActor
	vinculo domain.VinculoAutenticacionActorV1
}

func (a autorizadorConVinculo) Exigir(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	solicitud.ContextoActor = a.actor
	solicitud.VinculoAutenticacionActor = a.vinculo
	if err := solicitud.ValidarVinculoAutenticacionActor(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	decision, err := a.base.Exigir(ctx, solicitud)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	if !a.vinculo.CoincideExactamenteCon(decision.VinculoAutenticacionActor) {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrDecisionAutorizacionInvalida,
		)
	}
	return decision, nil
}

// exigirDecisionAutorizacionVinculada es la variante productiva para casos de
// uso migrados al contexto de actor canonico. La identidad, el perfil activo y
// la garantia proceden exclusivamente de esa capacidad interna.
func exigirDecisionAutorizacionVinculada(
	ctx context.Context,
	autorizador ports.Autorizador,
	reloj ports.Reloj,
	actor domain.ContextoActor,
	vinculo domain.VinculoAutenticacionActorV1,
	accion string,
	recurso domain.RecursoAutorizable,
	finalidad, correlacionRef, motivo string,
	usoCampos usoCamposDecisionAutorizacion,
) (domain.DecisionAutorizacion, error) {
	if ctx == nil || dependenciaAutorizacionNula(autorizador) || dependenciaAutorizacionNula(reloj) {
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
	return exigirDecisionAutorizacion(
		ctx,
		autorizadorConVinculo{base: autorizador, actor: actorCanonico, vinculo: vinculo},
		reloj,
		actorCanonico.Principal,
		actorCanonico.PerfilActivoRef,
		accion,
		recurso,
		finalidad,
		correlacionRef,
		motivo,
		usoCampos,
	)
}

// exigirDecisionAutorizacion aplica defensa en profundidad sobre el puerto:
// una implementacion defectuosa no puede conceder una decision vencida o
// perteneciente a otra solicitud.
func exigirDecisionAutorizacion(
	ctx context.Context,
	autorizador ports.Autorizador,
	reloj ports.Reloj,
	principal domain.Principal,
	perfilActivo, accion string,
	recurso domain.RecursoAutorizable,
	finalidad, correlacionRef, motivo string,
	usoCampos usoCamposDecisionAutorizacion,
) (domain.DecisionAutorizacion, error) {
	if ctx == nil || autorizador == nil || reloj == nil || usoCampos == usoCamposDecisionNoDeclarado ||
		usoCampos > usoCamposDecisionConsumidos {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	solicitud := domain.SolicitudAutorizacion{
		Principal: principal,
		// La autorizacion no normaliza entradas para convertirlas en una
		// concesion valida. Perfil, accion, finalidad y referencias deben llegar
		// ya en su representacion canonica; cualquier diferencia se deniega.
		PerfilActivoRef: perfilActivo,
		Accion:          accion,
		Recurso:         recurso,
		Finalidad:       finalidad,
		CorrelacionRef:  correlacionRef,
		Motivo:          motivo,
	}
	if err := solicitud.Validar(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	huellaContextoRecurso, err := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	decision, err := autorizador.Exigir(ctx, solicitud)
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	instante := reloj.Ahora().UTC()
	if err := decision.ValidarEvidenciaInstantanea(); err != nil || decision.TieneSolicitudLigadaV2() ||
		!decision.VigenteEn(instante) ||
		decision.PrincipalID != principal.ID ||
		decision.PerfilActivoRef != solicitud.PerfilActivoRef ||
		decision.Accion != solicitud.Accion ||
		decision.RecursoRef != solicitud.Recurso.Referencia ||
		decision.ModuloID != solicitud.Recurso.ModuloID ||
		decision.TipoRecurso != solicitud.Recurso.Tipo ||
		decision.ContextoRecursoHuellaSHA256 != huellaContextoRecurso ||
		decision.Finalidad != solicitud.Finalidad ||
		decision.CorrelacionRef != solicitud.CorrelacionRef ||
		!domain.CumpleGarantiaAutenticacion(principal.AuthAssurance, decision.GarantiaMinima) {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida)
	}
	// Una obligacion solo puede admitirse cuando el caso de uso posee una
	// implementacion positiva y comprobable para ella. Actualmente ningun caso
	// de uso general declara obligaciones soportadas: ignorarlas ampliaria la
	// concesion que realmente emitio la politica.
	if len(decision.Obligaciones) != 0 {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida)
	}
	// Para una operacion atomica, los campos no son aplicables. Si la decision
	// intenta restringirlos y el caso no los interpreta, se deniega en vez de
	// ejecutar la operacion completa.
	if usoCampos == usoCamposDecisionNoAplicables && len(decision.CamposPermitidos) != 0 {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida)
	}
	return decision, nil
}

// revalidarDecisionAutorizacionEnUso reduce la ventana entre el PEP y un
// efecto lento o externo. No sustituye el consumo atomico con el efecto en el
// adaptador duradero: esa segunda barrera sigue siendo obligatoria antes del
// COMMIT productivo.
func revalidarDecisionAutorizacionEnUso(decision domain.DecisionAutorizacion, reloj ports.Reloj) error {
	if reloj == nil {
		return errors.Join(domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida)
	}
	instante := reloj.Ahora().UTC()
	if err := decision.ValidarEvidenciaInstantanea(); err != nil || decision.TieneSolicitudLigadaV2() ||
		!decision.Concedida || instante.IsZero() || !decision.VigenteEn(instante) {
		return errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida, err)
	}
	return nil
}
