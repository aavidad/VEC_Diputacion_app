package application

import (
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// errorSinCreditoCobertura une el error público del servicio con el motivo de
// crédito del dominio, si la causa lo trae. Solo viaja el motivo (un código
// estable), nunca el resto de la causa interna.
func errorSinCreditoCobertura(publico, causa error) (error, bool) {
	motivo, ok := domain.MotivoSinCreditoDe(causa)
	if !ok {
		return nil, false
	}
	return errors.Join(publico, domain.NuevoErrorSinCredito(motivo)), true
}
