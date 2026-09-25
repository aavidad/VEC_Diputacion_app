package domain

import (
	"errors"
	"testing"
)

func TestCircuitoAvanzaPorCuatroAutoridadesYDevuelve(t *testing.T) {
	pasos := []struct {
		etapa          EtapaCircuito
		antes, despues string
	}{
		{EtapaRevision, EstadoEnviadoPendienteRevision, EstadoPendienteAutorizacion},
		{EtapaAutorizacion, EstadoPendienteAutorizacion, EstadoPendienteLiquidacion},
		{EtapaLiquidacion, EstadoPendienteLiquidacion, EstadoPendienteFiscalizacion},
		{EtapaFiscalizacion, EstadoPendienteFiscalizacion, EstadoFiscalizada},
	}
	previos := []string{}
	for i, p := range pasos {
		actor := "per_actor_" + string(rune('a'+i))
		despues, err := ResolverDecisionCircuito(p.antes, p.etapa, DecisionAprobar, "", actor, "per_solicitante", previos)
		if err != nil || despues != p.despues {
			t.Fatalf("paso %s: %q %v", p.etapa, despues, err)
		}
		devuelta, err := ResolverDecisionCircuito(p.antes, p.etapa, DecisionDevolver, "Falta justificante", actor, "per_solicitante", previos)
		if err != nil || devuelta != EstadoDevuelta {
			t.Fatalf("devolución %s: %q %v", p.etapa, devuelta, err)
		}
		previos = append(previos, actor)
	}
}

func TestCircuitoRechazaSaltoYSuperposicionDeActores(t *testing.T) {
	if _, err := ResolverDecisionCircuito(EstadoEnviadoPendienteRevision, EtapaFiscalizacion, DecisionAprobar, "", "per_interventor", "per_solicitante", nil); !errors.Is(err, ErrTransicionCircuitoInvalida) {
		t.Fatalf("salto aceptado: %v", err)
	}
	if _, err := ResolverDecisionCircuito(EstadoPendienteAutorizacion, EtapaAutorizacion, DecisionAprobar, "", "per_revisor", "per_solicitante", []string{"per_revisor"}); !errors.Is(err, ErrSeparacionCircuitoIncumplida) {
		t.Fatalf("mismo actor aceptado: %v", err)
	}
	if _, err := ResolverDecisionCircuito(EstadoPendienteLiquidacion, EtapaLiquidacion, DecisionDevolver, "", "per_rrhh", "per_solicitante", nil); !errors.Is(err, ErrDecisionCircuitoInvalida) {
		t.Fatalf("devolución sin motivo aceptada: %v", err)
	}
}
