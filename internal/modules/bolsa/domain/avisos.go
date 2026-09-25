package domain

import (
	"errors"
	"time"
)

var ErrAvisoRRHHInvalido = errors.New("bolsa: aviso RRHH invalido")

const (
	AvisoSaltoOrden = "salto_orden"
	AvisoTresAnos   = "tres_anos"
	// Portal del candidato: solicitudes pendientes de validar y respuestas.
	AvisoSolicitudPortal = "solicitud_portal"
	AvisoRespuestaPortal = "respuesta_portal"
)

// TiposAvisoRRHH enumera los avisos que puede recibir la bandeja de RRHH.
func TiposAvisoRRHH() []string {
	return []string{AvisoSaltoOrden, AvisoTresAnos, AvisoSolicitudPortal, AvisoRespuestaPortal}
}

func tipoAvisoRRHHValido(tipo string) bool {
	for _, t := range TiposAvisoRRHH() {
		if t == tipo {
			return true
		}
	}
	return false
}

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
	if !tipoAvisoRRHHValido(a.Tipo) || a.BolsaRef == "" ||
		a.Referencia == "" || a.Fecha.IsZero() || len(a.Detalle) == 0 {
		return ErrAvisoRRHHInvalido
	}
	return nil
}
