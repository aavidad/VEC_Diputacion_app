package ports

import "errors"

// La selección se suspende hasta que RRHH compruebe los hechos contextuales
// de una persona situada antes del siguiente ofrecimiento.
var ErrCriteriosOfrecimientoPendientesRRHH = errors.New("contratacion temporal: comprobacion de ofrecimiento pendiente por RRHH")

type ComprobacionOfrecimientoPendiente struct {
	BolsaRef               string `json:"bolsa_ref"`
	OrdenActa              uint64 `json:"orden_acta"`
	VersionPoliticaOfertas int64  `json:"version_politica_ofertas,omitempty"`
	MotivoClave            string `json:"motivo_clave"`
}

type ErrorCriteriosOfrecimientoPendientesRRHH struct {
	Comprobacion ComprobacionOfrecimientoPendiente
}

func (e *ErrorCriteriosOfrecimientoPendientesRRHH) Error() string {
	return ErrCriteriosOfrecimientoPendientesRRHH.Error()
}

func (e *ErrorCriteriosOfrecimientoPendientesRRHH) Unwrap() error {
	return ErrCriteriosOfrecimientoPendientesRRHH
}
