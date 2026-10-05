package application

import "errors"

// El marcador sólo se emite después del retorno del repositorio y la
// validación de su evidencia. Un error de entrada no acredita ese COMMIT.
type errorAccesoConfirmadoPreparacionBasesV3 struct{ causa error }

func (e errorAccesoConfirmadoPreparacionBasesV3) Error() string { return e.causa.Error() }
func (e errorAccesoConfirmadoPreparacionBasesV3) Unwrap() error { return e.causa }

func AccesoConfirmadoPreparacionBasesV3(err error) bool {
	var confirmado errorAccesoConfirmadoPreparacionBasesV3
	return errors.As(err, &confirmado) && confirmado.causa != nil
}
