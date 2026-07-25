package application

import (
	"errors"
	"strings"
	"sync"
)

var errReferenciaDecisionVECReservadaNoDisponible = errors.New(
	"contratacion temporal: referencia reservada de decision VEC no disponible",
)

// generadorDecisionVECReservada entrega exactamente una vez la referencia
// preasignada por la reserva durable. No genera, deriva ni acepta otra.
type generadorDecisionVECReservada struct {
	mu         sync.Mutex
	referencia string
	consumida  bool
}

func nuevoGeneradorDecisionVECReservada(
	referencia string,
) (*generadorDecisionVECReservada, error) {
	if !referenciaDecisionVECReservadaValida(referencia) {
		return nil, errReferenciaDecisionVECReservadaNoDisponible
	}
	return &generadorDecisionVECReservada{referencia: referencia}, nil
}

func (g *generadorDecisionVECReservada) NuevaReferenciaDecisionAutorizacion() (
	string,
	error,
) {
	if g == nil {
		return "", errReferenciaDecisionVECReservadaNoDisponible
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.consumida || !referenciaDecisionVECReservadaValida(g.referencia) {
		return "", errReferenciaDecisionVECReservadaNoDisponible
	}
	g.consumida = true
	return g.referencia, nil
}

func referenciaDecisionVECReservadaValida(referencia string) bool {
	if referencia == "" || referencia != strings.TrimSpace(referencia) ||
		len(referencia) > 512 {
		return false
	}
	for _, caracter := range referencia {
		if caracter < 0x21 || caracter > 0x7e || caracter == '*' {
			return false
		}
	}
	return true
}
