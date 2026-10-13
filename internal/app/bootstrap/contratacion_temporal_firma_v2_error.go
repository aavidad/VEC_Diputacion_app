package bootstrap

import "fmt"

type errorSesionFirmanteV2Opaco struct{ causa error }

func (e *errorSesionFirmanteV2Opaco) Error() string    { return errSesionFirmanteV2Denegada.Error() }
func (e *errorSesionFirmanteV2Opaco) String() string   { return e.Error() }
func (e *errorSesionFirmanteV2Opaco) GoString() string { return e.Error() }
func (e *errorSesionFirmanteV2Opaco) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(e.Error()))
}
func (e *errorSesionFirmanteV2Opaco) Unwrap() []error {
	return []error{errSesionFirmanteV2Denegada, e.causa}
}

func falloSesionFirmanteV2(causa error) error {
	if causa == nil {
		return errSesionFirmanteV2Denegada
	}
	return &errorSesionFirmanteV2Opaco{causa: causa}
}
