package calculoexperienciaoficial

import "fmt"

// CodigoError identifica de forma estable un rechazo del contrato oficial.
// Los errores no incorporan valores recibidos para evitar filtraciones.
type CodigoError string

const (
	CodigoValorInvalido       CodigoError = "valor_invalido"
	CodigoValorNoCanonico     CodigoError = "valor_no_canonico"
	CodigoFueraDeLimites      CodigoError = "fuera_de_limites"
	CodigoEsquemaIncompatible CodigoError = "esquema_incompatible"
	CodigoHuellaNoCoincide    CodigoError = "huella_no_coincide"
	CodigoEstadoIncompatible  CodigoError = "estado_incompatible"
	CodigoSecretoInvalido     CodigoError = "secreto_invalido"
	CodigoEntradaNoPermitida  CodigoError = "entrada_no_permitida"
)

// ErrorDominio permite clasificar fallos sin depender de su texto.
type ErrorDominio struct {
	codigo CodigoError
	campo  string
}

func (e *ErrorDominio) Error() string {
	if e == nil {
		return "calculo oficial de experiencia: error"
	}
	if e.campo == "" {
		return fmt.Sprintf("calculo oficial de experiencia: %s", e.codigo)
	}
	return fmt.Sprintf("calculo oficial de experiencia: %s: %s", e.campo, e.codigo)
}

func (e *ErrorDominio) Codigo() CodigoError {
	if e == nil {
		return ""
	}
	return e.codigo
}

// Campo es una etiqueta técnica fija, nunca el valor rechazado.
func (e *ErrorDominio) Campo() string {
	if e == nil {
		return ""
	}
	return e.campo
}

func (e *ErrorDominio) Is(objetivo error) bool {
	otro, ok := objetivo.(*ErrorDominio)
	if !ok || e == nil || otro == nil {
		return false
	}
	return (otro.codigo == "" || e.codigo == otro.codigo) &&
		(otro.campo == "" || e.campo == otro.campo)
}

var (
	ErrValorInvalido       = &ErrorDominio{codigo: CodigoValorInvalido}
	ErrValorNoCanonico     = &ErrorDominio{codigo: CodigoValorNoCanonico}
	ErrFueraDeLimites      = &ErrorDominio{codigo: CodigoFueraDeLimites}
	ErrEsquemaIncompatible = &ErrorDominio{codigo: CodigoEsquemaIncompatible}
	ErrHuellaNoCoincide    = &ErrorDominio{codigo: CodigoHuellaNoCoincide}
	ErrEstadoIncompatible  = &ErrorDominio{codigo: CodigoEstadoIncompatible}
	ErrSecretoInvalido     = &ErrorDominio{codigo: CodigoSecretoInvalido}
	ErrEntradaNoPermitida  = &ErrorDominio{codigo: CodigoEntradaNoPermitida}
)

func nuevoError(campo string, codigo CodigoError) error {
	return &ErrorDominio{codigo: codigo, campo: campo}
}
