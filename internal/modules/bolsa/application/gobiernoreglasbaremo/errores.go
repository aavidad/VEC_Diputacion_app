package gobiernoreglasbaremo

import "errors"

var (
	ErrOperacionInvalida            = errors.New("bolsa: operacion de gobierno de reglas de baremo invalida")
	ErrContratoAutorizacionInvalido = errors.New(
		"bolsa: contrato de autorizacion de reglas de baremo invalido",
	)
	ErrPlanCambioInvalido          = errors.New("bolsa: plan de cambio de reglas de baremo invalido")
	ErrConsultaExactaInvalida      = errors.New("bolsa: consulta exacta de reglas de baremo invalida")
	ErrSujetoSeudonimoHMACInvalido = errors.New(
		"bolsa: sujeto seudonimo HMAC de reglas de baremo invalido",
	)
	ErrSerializacionProhibida = errors.New(
		"bolsa: serializacion generica de gobierno de reglas de baremo prohibida",
	)
)
