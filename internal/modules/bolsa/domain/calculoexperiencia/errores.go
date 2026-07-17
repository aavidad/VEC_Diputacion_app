package calculoexperiencia

import (
	"errors"
	"fmt"
)

// CodigoError identifica de forma estable un fallo del calculo exacto. Los
// errores nunca incorporan los valores operados ni referencias del expediente.
type CodigoError string

const (
	CodigoValorInvalido        CodigoError = "valor_invalido"
	CodigoResultadoNegativo    CodigoError = "resultado_negativo"
	CodigoDivisionPorCero      CodigoError = "division_por_cero"
	CodigoResultadoNoExacto    CodigoError = "resultado_no_exacto"
	CodigoDesbordamiento       CodigoError = "desbordamiento"
	CodigoLimiteOperaciones    CodigoError = "limite_operaciones"
	CodigoContextoIncompatible CodigoError = "contexto_incompatible"
	CodigoModoRedondeoInvalido CodigoError = "modo_redondeo_invalido"
)

// ErrorCalculo permite clasificar un fallo sin depender de su texto. Campo es
// siempre una etiqueta tecnica fija y nunca el valor de entrada rechazado.
type ErrorCalculo struct {
	codigo CodigoError
	campo  string
}

func (e *ErrorCalculo) Error() string {
	if e == nil {
		return "calculo de experiencia: error"
	}
	if e.campo == "" {
		return fmt.Sprintf("calculo de experiencia: %s", e.codigo)
	}
	return fmt.Sprintf("calculo de experiencia: %s: %s", e.campo, e.codigo)
}

// Codigo devuelve la clasificacion estable del error.
func (e *ErrorCalculo) Codigo() CodigoError {
	if e == nil {
		return ""
	}
	return e.codigo
}

// Campo devuelve la etiqueta tecnica del elemento rechazado.
func (e *ErrorCalculo) Campo() string {
	if e == nil {
		return ""
	}
	return e.campo
}

// Is permite clasificar el error mediante errors.Is.
func (e *ErrorCalculo) Is(objetivo error) bool {
	otro, ok := objetivo.(*ErrorCalculo)
	if !ok || e == nil || otro == nil {
		return false
	}
	return (otro.codigo == "" || e.codigo == otro.codigo) &&
		(otro.campo == "" || e.campo == otro.campo)
}

var (
	ErrValorInvalido        = &ErrorCalculo{codigo: CodigoValorInvalido}
	ErrResultadoNegativo    = &ErrorCalculo{codigo: CodigoResultadoNegativo}
	ErrDivisionPorCero      = &ErrorCalculo{codigo: CodigoDivisionPorCero}
	ErrResultadoNoExacto    = &ErrorCalculo{codigo: CodigoResultadoNoExacto}
	ErrDesbordamiento       = &ErrorCalculo{codigo: CodigoDesbordamiento}
	ErrLimiteOperaciones    = &ErrorCalculo{codigo: CodigoLimiteOperaciones}
	ErrContextoIncompatible = &ErrorCalculo{codigo: CodigoContextoIncompatible}
	ErrModoRedondeoInvalido = &ErrorCalculo{codigo: CodigoModoRedondeoInvalido}
)

func nuevoError(campo string, codigo CodigoError) error {
	return &ErrorCalculo{campo: campo, codigo: codigo}
}

func codigoError(err error) CodigoError {
	var errorCalculo *ErrorCalculo
	if errors.As(err, &errorCalculo) {
		return errorCalculo.Codigo()
	}
	return CodigoValorInvalido
}
