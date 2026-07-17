package calculoexperienciaoficial

import "errors"

var (
	ErrServicioInvalido = errors.New(
		"bolsa: servicio de calculo oficial de experiencia invalido",
	)
	ErrOrdenInvalida = errors.New(
		"bolsa: orden confiable de calculo oficial de experiencia invalida",
	)
	ErrSesionNoApta = errors.New(
		"bolsa: sesion no apta para calculo oficial de experiencia",
	)
	ErrFuenteNoConfiable = errors.New(
		"bolsa: fuente de calculo oficial de experiencia no confiable",
	)
	ErrMotorNoCoincide = errors.New(
		"bolsa: motor de calculo oficial de experiencia no coincide",
	)
	ErrResultadoNoConfiable = errors.New(
		"bolsa: resultado de calculo oficial de experiencia no confiable",
	)
	ErrConfirmacionInvalida = errors.New(
		"bolsa: confirmacion durable de calculo oficial de experiencia invalida",
	)
	ErrReciboNoConfiable = errors.New(
		"bolsa: recibo de calculo oficial de experiencia no confiable",
	)
	ErrResultadoConfirmacionIndeterminado = errors.New(
		"bolsa: resultado de confirmacion de calculo oficial indeterminado",
	)
	ErrReconciliacionRequerida = errors.New(
		"bolsa: reconciliacion de calculo oficial requerida",
	)
	ErrSerializacionProhibida = errors.New(
		"bolsa: serializacion de capacidad de calculo oficial prohibida",
	)
)
