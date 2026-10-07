package domain

import "errors"

// RRHH (02/10/2026) exige crédito antes de ofrecer. Una RC validada conserva
// número y documento. La alternativa del estado de las partidas necesita una
// constancia acreditada: el resultado «no_requerida» actual solo conserva un
// motivo y no la acredita en el circuito nuevo. La regla previa sigue vigente
// en los expedientes anteriores; los motivos permiten explicarlo en pantalla.

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
	// SinCreditoPartidasSinCoste: consta el estado de las partidas, pero no el
	// coste aproximado que RRHH pide con él (decisión del 02/10/2026). Lo
	// exige la política de crédito del catálogo de reglas, que RRHH puede
	// desactivar; la regla base no lo comprueba.
	SinCreditoPartidasSinCoste MotivoSinCredito = "partidas_sin_coste"
	// SinCreditoPartidasNoAcreditadas: el motivo «no requerida» no contiene
	// una constancia documental del estado de las partidas para este circuito.
	SinCreditoPartidasNoAcreditadas MotivoSinCredito = "partidas_no_acreditadas"
)

// PoliticaCreditoOferta son las exigencias de crédito que fija el catálogo de
// reglas versionado. El valor cero no exige nada más que la regla base.
type PoliticaCreditoOferta struct {
	// ExigeCosteConPartidas: sin retención, la constancia de las partidas
	// debe ir con el coste aproximado del análisis.
	ExigeCosteConPartidas bool
}

// MotivoSinCreditoSegunPolitica añade a la regla base lo que exige la
// política vigente. Devuelve "" si el expediente puede pasar a la oferta.
func (a *AnalisisRRHH) MotivoSinCreditoSegunPolitica(p PoliticaCreditoOferta) MotivoSinCredito {
	if motivo := a.MotivoSinCreditoParaOferta(); motivo != "" {
		return motivo
	}
	if p.ExigeCosteConPartidas && a.ValidacionRC.Resultado == RCNoRequerida && a.CostePrevisto == nil {
		return SinCreditoPartidasSinCoste
	}
	return ""
}

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
	if e.Circuito != nil && e.Analisis.ValidacionRC.Resultado == RCNoRequerida {
		return NuevoErrorSinCredito(SinCreditoPartidasNoAcreditadas)
	}
	return nil
}
