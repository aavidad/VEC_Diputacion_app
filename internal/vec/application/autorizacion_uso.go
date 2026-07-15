package application

import (
	"context"
	"errors"

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
	if err := decision.ValidarEvidenciaInstantanea(); err != nil || !decision.VigenteEn(instante) ||
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
	if err := decision.ValidarEvidenciaInstantanea(); err != nil || !decision.Concedida ||
		instante.IsZero() || !decision.VigenteEn(instante) {
		return errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida, err)
	}
	return nil
}
