package domain

import (
	"errors"
	"time"
)

var ErrAvisoRRHHInvalido = errors.New("bolsa: aviso RRHH invalido")

const (
	AvisoSaltoOrden = "salto_orden"
	AvisoTresAnos   = "tres_anos"
)

// AvisoRRHH es una proyeccion derivada. Solo contiene referencias opacas y los
// datos imprescindibles para que RRHH compruebe el hecho que origina el aviso.
type AvisoRRHH struct {
	Tipo       string
	BolsaRef   string
	Referencia string
	Detalle    map[string]any
	Fecha      time.Time
}

func (a AvisoRRHH) Validar() error {
	if (a.Tipo != AvisoSaltoOrden && a.Tipo != AvisoTresAnos) || a.BolsaRef == "" ||
		a.Referencia == "" || a.Fecha.IsZero() || len(a.Detalle) == 0 {
		return ErrAvisoRRHHInvalido
	}
	return nil
}
