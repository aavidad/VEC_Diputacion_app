package domain

import "errors"

// Sin crédito no se tramita nada (RRHH, 02/10/2026): el expediente no pasa a
// la oferta ni al llamamiento (vía de cobertura) sin uno de estos respaldos:
//
//   - la retención de crédito validada, con su número y su documento
//     (ValidacionRC.Validar ya exige ambos cuando el resultado es «validada»);
//   - si la retención no llega, la constancia del estado de las partidas en
//     SICAL, que la fuente presupuestaria devuelve como «no_requerida» con su
//     motivo (ValidacionRC.Validar exige el motivo).
//
// Validar ya impide guardar una vía de cobertura sin este respaldo; aquí se
// nombra el motivo para que la pantalla pueda explicarlo.

// ErrSinCreditoParaOferta agrupa los motivos por los que el expediente no
// puede pasar a la oferta. Cada motivo concreto lo envuelve para que la
// frontera HTTP pueda explicarlo sin conocer la regla.
var ErrSinCreditoParaOferta = errors.New("contratacion temporal: sin credito para ofrecer")

// MotivoSinCredito es un código estable; el texto visible vive en los
// catálogos de idioma.
type MotivoSinCredito string

const (
	// SinCreditoAnalisisPendiente: falta el análisis de RRHH registrado.
	SinCreditoAnalisisPendiente MotivoSinCredito = "analisis_pendiente"
	// SinCreditoRetencionRechazada: la fuente presupuestaria rechazó la retención.
	SinCreditoRetencionRechazada MotivoSinCredito = "retencion_rechazada"
)

type errorSinCredito struct{ motivo MotivoSinCredito }

func (e errorSinCredito) Error() string {
	return ErrSinCreditoParaOferta.Error() + ": " + string(e.motivo)
}

func (e errorSinCredito) Unwrap() error { return ErrSinCreditoParaOferta }

// NuevoErrorSinCredito construye el error de un motivo conocido.
func NuevoErrorSinCredito(motivo MotivoSinCredito) error {
	return errorSinCredito{motivo: motivo}
}

// MotivoSinCreditoDe devuelve el motivo que lleva un error, si lo hay.
func MotivoSinCreditoDe(err error) (MotivoSinCredito, bool) {
	var concreto errorSinCredito
	if errors.As(err, &concreto) {
		return concreto.motivo, true
	}
	return "", false
}

// MotivoSinCreditoParaOferta devuelve "" si el crédito permite ofrecer el
// puesto o el motivo por el que no lo permite.
func (a *AnalisisRRHH) MotivoSinCreditoParaOferta() MotivoSinCredito {
	if a == nil || a.ActuacionRegistro == nil || a.Validar() != nil {
		return SinCreditoAnalisisPendiente
	}
	switch a.ValidacionRC.Resultado {
	case RCValidada, RCNoRequerida:
		return ""
	case RCRechazada:
		return SinCreditoRetencionRechazada
	default:
		return SinCreditoAnalisisPendiente
	}
}

// ErrorSinCreditoParaOferta devuelve nil si el expediente puede pasar a la
// oferta, o el error con su motivo.
func (e Expediente) ErrorSinCreditoParaOferta() error {
	if motivo := e.Analisis.MotivoSinCreditoParaOferta(); motivo != "" {
		return NuevoErrorSinCredito(motivo)
	}
	return nil
}
