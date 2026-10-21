package application

import "log/slog"

// Conserva la causa para el diagnóstico mediante errors.Is/As; el texto y
// el registro estructurado solo contienen el estado público de la operación.
type falloSeleccionIndeterminada struct{ causa error }

func (e falloSeleccionIndeterminada) Error() string {
	return ErrEjecucionSeleccionLlamamientoIndeterminada.Error()
}

func (e falloSeleccionIndeterminada) Is(objetivo error) bool {
	return objetivo == ErrEjecucionSeleccionLlamamientoIndeterminada
}

func (e falloSeleccionIndeterminada) Unwrap() error { return e.causa }

func (e falloSeleccionIndeterminada) LogValue() slog.Value {
	return slog.StringValue(e.Error())
}
