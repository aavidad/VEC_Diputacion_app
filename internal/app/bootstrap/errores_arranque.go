package bootstrap

import "errors"

// falloComponenteArranque conserva la causa y añade la etapa de composición
// conocida por el punto de llamada. La etiqueta procede solo del código.
type falloComponenteArranque struct {
	componente string
	causa      error
}

func (f *falloComponenteArranque) Error() string { return f.causa.Error() }
func (f *falloComponenteArranque) Unwrap() error { return f.causa }

func marcarFalloComponenteArranque(componente string, err error) error {
	if err == nil {
		return nil
	}
	var previo *falloComponenteArranque
	if errors.As(err, &previo) {
		return err // Conservar la etapa más precisa de un constructor interior.
	}
	if !componenteArranqueValido(componente) {
		componente = "sin_etiqueta"
	}
	return &falloComponenteArranque{componente: componente, causa: err}
}

// ComponenteFalloArranque entrega una etiqueta cerrada para el log fatal.
// Un error no etiquetado queda visible como una deuda de composición.
func ComponenteFalloArranque(err error) string {
	var fallo *falloComponenteArranque
	if errors.As(err, &fallo) && fallo != nil && componenteArranqueValido(fallo.componente) {
		return fallo.componente
	}
	return "sin_etiqueta"
}

func componenteArranqueValido(valor string) bool {
	if len(valor) == 0 || len(valor) > 64 {
		return false
	}
	for _, c := range valor {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' {
			continue
		}
		return false
	}
	return true
}
