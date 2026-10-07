package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var ErrFuenteCreditoCircuitoRRHHInvalida = errors.New(
	"contratacion temporal: fuente de credito del circuito RRHH invalida",
)

type fuenteCreditoCircuitoConfigurada struct {
	fuente ports.FuenteEvidenciaCreditoCircuitoRRHH
}

// ConfigurarFuenteCreditoCircuitoRRHH admite solo la autoridad documental y
// el catálogo publicado compuestos por el servidor. Su ausencia cierra la
// decisión de cobertura del circuito nuevo.
func (s *ServicioConfirmacionDecisionCobertura) ConfigurarFuenteCreditoCircuitoRRHH(
	fuente ports.FuenteEvidenciaCreditoCircuitoRRHH,
) error {
	if s == nil || dependenciaNula(fuente) ||
		!s.fuenteCreditoCircuito.CompareAndSwap(nil, &fuenteCreditoCircuitoConfigurada{fuente: fuente}) {
		return ErrFuenteCreditoCircuitoRRHHInvalida
	}
	return nil
}

func (s *ServicioConfirmacionDecisionCobertura) acreditarCreditoCircuito(
	ctx context.Context,
	expediente domain.Expediente,
	tipo domain.TipoDecisionCoberturaGobernada,
	contexto ports.ContextoAutorizacionAltaV3,
) ([]ports.EvidenciaCreditoCircuitoRRHH, error) {
	if expediente.Circuito == nil || tipo == domain.DecisionCoberturaRectificacion {
		return nil, nil
	}
	if tipo != domain.DecisionCoberturaInicial {
		return nil, ErrConfirmacionDecisionCoberturaEnConflicto
	}
	configurada := s.fuenteCreditoCircuito.Load()
	if configurada == nil || dependenciaNula(configurada.fuente) {
		return nil, ErrConfirmacionDecisionCoberturaNoDisponible
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return nil, ErrConfirmacionDecisionCoberturaDenegada
	}
	credito, err := expediente.DatosCreditoCircuitoRRHH()
	if err != nil {
		return nil, ErrConfirmacionDecisionCoberturaEnConflicto
	}
	solicitud, err := ports.NuevaSolicitudEvidenciaCreditoCircuitoRRHH(
		credito, vinculo.PrincipalID, vinculo.PerfilActivoRef,
	)
	if err != nil {
		return nil, ErrConfirmacionDecisionCoberturaNoConfiable
	}
	evidencia, err := configurada.fuente.AcreditarCreditoCircuitoRRHH(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return nil, errContexto
	}
	if err != nil {
		return nil, ErrConfirmacionDecisionCoberturaNoDisponible
	}
	if evidencia.ValidarPara(solicitud) != nil {
		return nil, ErrConfirmacionDecisionCoberturaNoConfiable
	}
	return []ports.EvidenciaCreditoCircuitoRRHH{evidencia.Clonar()}, nil
}
