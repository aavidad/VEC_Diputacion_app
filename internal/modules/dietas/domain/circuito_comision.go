package domain

import (
	"errors"
	"time"
)

var (
	ErrDecisionCircuitoInvalida     = errors.New("dietas: decision de circuito invalida")
	ErrTransicionCircuitoInvalida   = errors.New("dietas: transicion de circuito invalida")
	ErrSeparacionCircuitoIncumplida = errors.New("dietas: separacion de funciones incumplida")
)

type EtapaCircuito string
type DecisionCircuito string

const (
	EtapaRevision      EtapaCircuito    = "revision"
	EtapaAutorizacion  EtapaCircuito    = "autorizacion"
	EtapaLiquidacion   EtapaCircuito    = "liquidacion"
	EtapaFiscalizacion EtapaCircuito    = "fiscalizacion"
	DecisionAprobar    DecisionCircuito = "aprobar"
	DecisionDevolver   DecisionCircuito = "devolver"
)

const (
	EstadoEnviadoPendienteRevision = "enviado_pendiente_revision"
	EstadoPendienteAutorizacion    = "pendiente_autorizacion"
	EstadoPendienteLiquidacion     = "pendiente_liquidacion"
	EstadoPendienteFiscalizacion   = "pendiente_fiscalizacion"
	EstadoFiscalizada              = "fiscalizada"
	EstadoDevuelta                 = "devuelta"
)

// EstadoPendiente relaciona una bandeja con el único estado que la alimenta.
func (e EtapaCircuito) EstadoPendiente() string {
	switch e {
	case EtapaRevision:
		return EstadoEnviadoPendienteRevision
	case EtapaAutorizacion:
		return EstadoPendienteAutorizacion
	case EtapaLiquidacion:
		return EstadoPendienteLiquidacion
	case EtapaFiscalizacion:
		return EstadoPendienteFiscalizacion
	default:
		return ""
	}
}

func (e EtapaCircuito) EstadoTrasAprobar() string {
	switch e {
	case EtapaRevision:
		return EstadoPendienteAutorizacion
	case EtapaAutorizacion:
		return EstadoPendienteLiquidacion
	case EtapaLiquidacion:
		return EstadoPendienteFiscalizacion
	case EtapaFiscalizacion:
		return EstadoFiscalizada
	default:
		return ""
	}
}

// ResolverDecisionCircuito es una regla pura. actoresOtrosPasos contiene solo
// quienes decidieron en una etapa distinta de esta comisión; una devolución
// no impide que el mismo administrativo vuelva a revisar la nueva versión.
// El repositorio aplica la regla sobre la versión bloqueada con el efecto.
func ResolverDecisionCircuito(estado string, etapa EtapaCircuito, decision DecisionCircuito, motivo, actorRef, solicitanteActorRef string, actoresOtrosPasos []string) (string, error) {
	if etapa.EstadoPendiente() == "" ||
		(decision != DecisionAprobar && decision != DecisionDevolver) ||
		actorRef == "" || solicitanteActorRef == "" ||
		len(motivo) > 600 || !textoVisible(motivo) || !TextoSinBordes(motivo) ||
		(decision == DecisionDevolver && len(motivo) < 3) {
		return "", ErrDecisionCircuitoInvalida
	}
	if estado != etapa.EstadoPendiente() {
		return "", ErrTransicionCircuitoInvalida
	}
	if actorRef == solicitanteActorRef {
		return "", ErrSeparacionCircuitoIncumplida
	}
	for _, previo := range actoresOtrosPasos {
		if previo == actorRef {
			return "", ErrSeparacionCircuitoIncumplida
		}
	}
	if decision == DecisionDevolver {
		return EstadoDevuelta, nil
	}
	return etapa.EstadoTrasAprobar(), nil
}

// Fecha civil inclusiva, sin convertirla en un instante UTC.
func FechaCircuitoValida(fecha string) bool {
	if len(fecha) != 10 {
		return false
	}
	v, err := time.Parse("2006-01-02", fecha)
	return err == nil && v.Format("2006-01-02") == fecha
}
