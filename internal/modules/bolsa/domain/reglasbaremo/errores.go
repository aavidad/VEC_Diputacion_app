package reglasbaremo

import (
	"fmt"
)

// CodigoError identifica de forma estable por que se rechazo una
// configuracion. Los errores no incorporan los valores recibidos para evitar
// que una referencia sensible termine accidentalmente en un registro.
type CodigoError string

const (
	CodigoValorInvalido       CodigoError = "valor_invalido"
	CodigoValorNoCanonico     CodigoError = "valor_no_canonico"
	CodigoFueraDeLimites      CodigoError = "fuera_de_limites"
	CodigoValorDuplicado      CodigoError = "valor_duplicado"
	CodigoPoliticaIncompleta  CodigoError = "politica_incompleta"
	CodigoSeccionDesconocida  CodigoError = "seccion_desconocida"
	CodigoGrupoDesconocido    CodigoError = "grupo_desconocido"
	CodigoCoeficienteAusente  CodigoError = "coeficiente_ausente"
	CodigoInvarianteQuebrada  CodigoError = "invariante_quebrada"
	CodigoEsquemaIncompatible CodigoError = "esquema_incompatible"
	CodigoHuellaNoCoincide    CodigoError = "huella_no_coincide"
)

// ErrorModelo es un error de dominio clasificable con errors.Is.
type ErrorModelo struct {
	codigo CodigoError
	campo  string
}

func (e *ErrorModelo) Error() string {
	if e == nil {
		return "reglas de baremo: error de modelo"
	}
	if e.campo == "" {
		return fmt.Sprintf("reglas de baremo: %s", e.codigo)
	}
	return fmt.Sprintf("reglas de baremo: %s: %s", e.campo, e.codigo)
}

// Codigo devuelve la causa estable del rechazo.
func (e *ErrorModelo) Codigo() CodigoError {
	if e == nil {
		return ""
	}
	return e.codigo
}

// Campo devuelve el nombre tecnico del elemento rechazado, nunca su valor.
func (e *ErrorModelo) Campo() string {
	if e == nil {
		return ""
	}
	return e.campo
}

// Is permite clasificar errores sin depender del texto mostrado.
func (e *ErrorModelo) Is(objetivo error) bool {
	otro, ok := objetivo.(*ErrorModelo)
	if !ok || e == nil || otro == nil {
		return false
	}
	return (otro.codigo == "" || e.codigo == otro.codigo) &&
		(otro.campo == "" || e.campo == otro.campo)
}

var (
	ErrValorInvalido       = &ErrorModelo{codigo: CodigoValorInvalido}
	ErrValorNoCanonico     = &ErrorModelo{codigo: CodigoValorNoCanonico}
	ErrFueraDeLimites      = &ErrorModelo{codigo: CodigoFueraDeLimites}
	ErrValorDuplicado      = &ErrorModelo{codigo: CodigoValorDuplicado}
	ErrPoliticaIncompleta  = &ErrorModelo{codigo: CodigoPoliticaIncompleta}
	ErrSeccionDesconocida  = &ErrorModelo{codigo: CodigoSeccionDesconocida}
	ErrGrupoDesconocido    = &ErrorModelo{codigo: CodigoGrupoDesconocido}
	ErrCoeficienteAusente  = &ErrorModelo{codigo: CodigoCoeficienteAusente}
	ErrInvarianteQuebrada  = &ErrorModelo{codigo: CodigoInvarianteQuebrada}
	ErrEsquemaIncompatible = &ErrorModelo{codigo: CodigoEsquemaIncompatible}
	ErrHuellaNoCoincide    = &ErrorModelo{codigo: CodigoHuellaNoCoincide}
)

func nuevoError(campo string, codigo CodigoError) error {
	return &ErrorModelo{campo: campo, codigo: codigo}
}
