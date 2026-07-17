package reglasbaremo

import "fmt"

// CodigoErrorGobierno clasifica rechazos del ciclo de gobierno sin incorporar
// referencias, actores ni otros valores de entrada a los mensajes de error.
type CodigoErrorGobierno string

const (
	CodigoGobiernoValorInvalido       CodigoErrorGobierno = "valor_invalido"
	CodigoGobiernoEstadoInvalido      CodigoErrorGobierno = "estado_invalido"
	CodigoGobiernoRevisionConflicto   CodigoErrorGobierno = "revision_conflicto"
	CodigoGobiernoTransicionProhibida CodigoErrorGobierno = "transicion_prohibida"
	CodigoGobiernoEvidenciaInvalida   CodigoErrorGobierno = "evidencia_invalida"
	CodigoGobiernoVinculoInexacto     CodigoErrorGobierno = "vinculo_inexacto"
	CodigoGobiernoInstanteInvalido    CodigoErrorGobierno = "instante_invalido"
	CodigoGobiernoInvarianteQuebrada  CodigoErrorGobierno = "invariante_quebrada"
)

// ErrorGobierno es un error de dominio estable y deliberadamente exento de
// valores potencialmente sensibles.
type ErrorGobierno struct {
	codigo CodigoErrorGobierno
}

func (e *ErrorGobierno) Error() string {
	if e == nil {
		return "gobierno de reglas de baremo: error"
	}
	return fmt.Sprintf("gobierno de reglas de baremo: %s", e.codigo)
}

func (e *ErrorGobierno) Codigo() CodigoErrorGobierno {
	if e == nil {
		return ""
	}
	return e.codigo
}

func (e *ErrorGobierno) Is(objetivo error) bool {
	otro, ok := objetivo.(*ErrorGobierno)
	return ok && e != nil && otro != nil &&
		(otro.codigo == "" || e.codigo == otro.codigo)
}

var (
	ErrGobiernoValorInvalido       = &ErrorGobierno{codigo: CodigoGobiernoValorInvalido}
	ErrGobiernoEstadoInvalido      = &ErrorGobierno{codigo: CodigoGobiernoEstadoInvalido}
	ErrGobiernoRevisionConflicto   = &ErrorGobierno{codigo: CodigoGobiernoRevisionConflicto}
	ErrGobiernoTransicionProhibida = &ErrorGobierno{codigo: CodigoGobiernoTransicionProhibida}
	ErrGobiernoEvidenciaInvalida   = &ErrorGobierno{codigo: CodigoGobiernoEvidenciaInvalida}
	ErrGobiernoVinculoInexacto     = &ErrorGobierno{codigo: CodigoGobiernoVinculoInexacto}
	ErrGobiernoInstanteInvalido    = &ErrorGobierno{codigo: CodigoGobiernoInstanteInvalido}
	ErrGobiernoInvarianteQuebrada  = &ErrorGobierno{codigo: CodigoGobiernoInvarianteQuebrada}
)

func nuevoErrorGobierno(codigo CodigoErrorGobierno) error {
	return &ErrorGobierno{codigo: codigo}
}
