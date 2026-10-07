package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ErrPoliticaCreditoCoberturaInvalida: la política de crédito no puede ser nula.
var ErrPoliticaCreditoCoberturaInvalida = errors.New(
	"contratacion temporal: politica de credito de cobertura invalida",
)

// politicaCreditoPredeterminada conserva la regla del flujo anterior: sin
// retención, el motivo sobre las partidas va con el coste aproximado. El
// circuito nuevo exige además acreditar esa constancia antes de ofrecer.
var politicaCreditoPredeterminada = domain.PoliticaCreditoOferta{ExigeCosteConPartidas: true}

type politicaCreditoConfigurada struct{ politica ports.PoliticaCreditoOferta }

func (c *politicaCreditoConfigurada) obtener() ports.PoliticaCreditoOferta {
	if c == nil {
		return nil
	}
	return c.politica
}

// ConfigurarPoliticaCredito conecta, una sola vez, la política del catálogo
// de reglas.
func (s *ServicioPresentacionPropuestaCobertura) ConfigurarPoliticaCredito(p ports.PoliticaCreditoOferta) error {
	if s == nil || dependenciaNula(p) ||
		!s.politicaCredito.CompareAndSwap(nil, &politicaCreditoConfigurada{politica: p}) {
		return ErrPoliticaCreditoCoberturaInvalida
	}
	return nil
}

// ConfigurarPoliticaCredito conecta, una sola vez, la política del catálogo
// de reglas.
func (s *ServicioConfirmacionDecisionCobertura) ConfigurarPoliticaCredito(p ports.PoliticaCreditoOferta) error {
	if s == nil || dependenciaNula(p) ||
		!s.politicaCredito.CompareAndSwap(nil, &politicaCreditoConfigurada{politica: p}) {
		return ErrPoliticaCreditoCoberturaInvalida
	}
	return nil
}

// motivoSegunPoliticaCredito resuelve la política vigente y devuelve el motivo
// que impide ofrecer, o "". Una política no disponible es un error: nunca se
// toma como «no exige nada».
func motivoSegunPoliticaCredito(
	ctx context.Context,
	politica ports.PoliticaCreditoOferta,
	expediente domain.Expediente,
) (domain.MotivoSinCredito, error) {
	// La política solo alcanza a la decisión inicial: una vía ya decidida se
	// rectifica sin volver a pedir el coste (el análisis ya no se puede
	// rectificar) y una asignación hecha no se reabre aquí.
	if expediente.Analisis == nil || expediente.ViaCobertura != nil ||
		len(expediente.DecisionesCobertura) != 0 || expediente.Asignacion != nil {
		return "", nil
	}
	// Solo se consulta el catálogo cuando importa: con la retención validada
	// o rechazada, un catálogo caído no bloquea nada que no dependa de él.
	if motivo := expediente.Analisis.MotivoSinCreditoParaOferta(); motivo != "" ||
		expediente.Analisis.ValidacionRC.Resultado != domain.RCNoRequerida {
		return motivo, nil
	}
	if expediente.Circuito != nil {
		return domain.SinCreditoPartidasNoAcreditadas, nil
	}
	vigente := politicaCreditoPredeterminada
	if !dependenciaNula(politica) {
		var err error
		if vigente, err = politica.PoliticaCreditoOferta(ctx); err != nil {
			return "", err
		}
	}
	return expediente.Analisis.MotivoSinCreditoSegunPolitica(vigente), nil
}

// La propuesta ya está autorizada aquí: puede decir el motivo.
func (s *ServicioPresentacionPropuestaCobertura) comprobarPoliticaCredito(
	ctx context.Context,
	expediente domain.Expediente,
) error {
	motivo, err := motivoSegunPoliticaCredito(ctx, s.politicaCredito.Load().obtener(), expediente)
	if err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return errContexto
		}
		return ErrPresentacionPropuestaCoberturaNoDisponible
	}
	if motivo != "" {
		return errors.Join(ErrPresentacionPropuestaCoberturaEstadoNoAdmite, domain.NuevoErrorSinCredito(motivo))
	}
	return nil
}

// La decisión aún no ha autorizado el expediente: rechaza sin decir el motivo.
func (s *ServicioConfirmacionDecisionCobertura) comprobarPoliticaCredito(
	ctx context.Context,
	expediente domain.Expediente,
) error {
	motivo, err := motivoSegunPoliticaCredito(ctx, s.politicaCredito.Load().obtener(), expediente)
	if err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return errContexto
		}
		return ErrConfirmacionDecisionCoberturaNoDisponible
	}
	if motivo != "" {
		return ErrConfirmacionDecisionCoberturaEnConflicto
	}
	return nil
}
