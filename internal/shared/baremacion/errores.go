package baremacion

import (
	"errors"
	"fmt"
)

// CodigoError identifica de forma estable la causa de rechazo de un valor.
type CodigoError string

const (
	CodigoFueraDeLimites    CodigoError = "fuera_de_limites"
	CodigoDenominadorCero   CodigoError = "denominador_cero"
	CodigoValorNoCanonico   CodigoError = "valor_no_canonico"
	CodigoValorInvalido     CodigoError = "valor_invalido"
	CodigoResultadoNegativo CodigoError = "resultado_negativo"
	CodigoDesbordamiento    CodigoError = "desbordamiento"
	CodigoDivisionPorCero   CodigoError = "division_por_cero"
	CodigoResultadoNoExacto CodigoError = "resultado_no_exacto"
	CodigoFechaInvalida     CodigoError = "fecha_invalida"
	CodigoIntervaloVacio    CodigoError = "intervalo_vacio"
)

// ErrorValor es un error de dominio sin datos de entrada potencialmente
// sensibles. Codigo permite clasificarlo sin depender del texto de Error.
type ErrorValor struct {
	tipo   string
	codigo CodigoError
}

func (e *ErrorValor) Error() string {
	if e == nil {
		return "baremacion: error de valor"
	}
	if e.tipo == "" {
		return fmt.Sprintf("baremacion: %s", e.codigo)
	}
	return fmt.Sprintf("baremacion: %s: %s", e.tipo, e.codigo)
}

// Codigo devuelve la clasificacion estable del error.
func (e *ErrorValor) Codigo() CodigoError {
	if e == nil {
		return ""
	}
	return e.codigo
}

// Tipo devuelve el valor que no pudo construirse u operar.
func (e *ErrorValor) Tipo() string {
	if e == nil {
		return ""
	}
	return e.tipo
}

// Is permite usar errors.Is con los errores centinela del paquete.
func (e *ErrorValor) Is(objetivo error) bool {
	otro, ok := objetivo.(*ErrorValor)
	if !ok || e == nil || otro == nil {
		return false
	}
	return (otro.codigo == "" || e.codigo == otro.codigo) &&
		(otro.tipo == "" || e.tipo == otro.tipo)
}

var (
	ErrFueraDeLimites    = &ErrorValor{codigo: CodigoFueraDeLimites}
	ErrDenominadorCero   = &ErrorValor{codigo: CodigoDenominadorCero}
	ErrValorNoCanonico   = &ErrorValor{codigo: CodigoValorNoCanonico}
	ErrValorInvalido     = &ErrorValor{codigo: CodigoValorInvalido}
	ErrResultadoNegativo = &ErrorValor{codigo: CodigoResultadoNegativo}
	ErrDesbordamiento    = &ErrorValor{codigo: CodigoDesbordamiento}
	ErrDivisionPorCero   = &ErrorValor{codigo: CodigoDivisionPorCero}
	ErrResultadoNoExacto = &ErrorValor{codigo: CodigoResultadoNoExacto}
	ErrFechaInvalida     = &ErrorValor{codigo: CodigoFechaInvalida}
	ErrIntervaloVacio    = &ErrorValor{codigo: CodigoIntervaloVacio}
)

func nuevoError(tipo string, codigo CodigoError) error {
	return &ErrorValor{tipo: tipo, codigo: codigo}
}

// remapearTipoError conserva el codigo estable y atribuye el rechazo al valor
// publico que se estaba construyendo, no a un detalle interno de composicion.
func remapearTipoError(tipo string, err error) error {
	var errorValor *ErrorValor
	if errors.As(err, &errorValor) {
		return nuevoError(tipo, errorValor.Codigo())
	}
	return nuevoError(tipo, CodigoValorInvalido)
}
