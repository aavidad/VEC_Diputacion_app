package domain

import "errors"

// Cancelación del expediente antes de la fiscalización (duda 12 de RRHH).
// El centro solicitante sobre su petición o RRHH sobre cualquier expediente
// de su ámbito lo dan por terminado con un motivo del catálogo gobernado y,
// si quieren, una observación. El expediente queda en estado `cancelado`,
// terminal, en la misma fase en que estaba; la historia anterior se conserva.
//
// Las fases desde las que se admite, los motivos y quién puede usar cada uno
// los fija el catálogo (regla c20 y catálogo de motivos de cancelación). La
// barrera de la fiscalización es una invariante técnica: tras ella existen
// llamamientos y efectos en Bolsa que esta operación no deshace.
const AccionCancelarExpediente ClaveCatalogo = "contratacion_temporal.expediente.cancelar"

// CanalCancelacion identifica quién cancela; cada canal se autoriza aparte.
type CanalCancelacion string

const (
	CanalCancelacionRRHH   CanalCancelacion = "rrhh"
	CanalCancelacionCentro CanalCancelacion = "centro"

	maximoFasesCancelacion = 16
)

var ErrCancelacionInvalida = errors.New("contratacion temporal: cancelacion de expediente invalida")

func (c CanalCancelacion) Valido() bool {
	return c == CanalCancelacionRRHH || c == CanalCancelacionCentro
}

// DatosCancelacion es la intención exacta: motivo del catálogo, observación
// opcional, canal de quien cancela y fases que la regla vigente admite.
type DatosCancelacion struct {
	MotivoClave    ClaveCatalogo
	Observaciones  string
	Canal          CanalCancelacion
	FasesAdmitidas []ClaveFase
}

func (d DatosCancelacion) Validar() error {
	if !d.MotivoClave.Valida() || !textoValido(d.Observaciones, 2000, true) || !d.Canal.Valido() ||
		!FasesCancelacionValidas(d.FasesAdmitidas) {
		return ErrCancelacionInvalida
	}
	return nil
}

// FasesCancelacionValidas exige una lista no vacía, acotada y sin repetir.
func FasesCancelacionValidas(fases []ClaveFase) bool {
	if len(fases) == 0 || len(fases) > maximoFasesCancelacion {
		return false
	}
	vistas := make(map[ClaveFase]struct{}, len(fases))
	for _, f := range fases {
		if _, repetida := vistas[f]; repetida || !f.Valida() {
			return false
		}
		vistas[f] = struct{}{}
	}
	return true
}

// Fiscalizado indica si el expediente llegó a fiscalizarse alguna vez. Una
// modificación tras el nombramiento retira la fiscalización de la proyección
// vigente, pero su actuación permanece en la historia.
func (e Expediente) Fiscalizado() bool {
	return e.Fiscalizacion != nil || e.TieneAccion(AccionRegistrarFiscalizacion)
}

// CancelableEn indica si el expediente puede cancelarse con esas fases:
// en curso, en una fase admitida y sin fiscalización en su historia.
func (e Expediente) CancelableEn(fases []ClaveFase) bool {
	if e.EstadoActual != EstadoEnCurso || e.Fiscalizado() || len(e.Actuaciones) == 0 {
		return false
	}
	for _, f := range fases {
		if f == e.FaseActual {
			return true
		}
	}
	return false
}

// UnidadActual es la unidad de la última actuación: quien tiene el expediente.
func (e Expediente) UnidadActual() string {
	if len(e.Actuaciones) == 0 {
		return ""
	}
	return e.Actuaciones[len(e.Actuaciones)-1].UnidadRef
}

// Cancelar deja el expediente cancelado en su fase actual con la actuación de
// cancelación. La observación de la actuación es la de la cancelación; el
// motivo y el canal viajan en el registro durable y en el evento.
func (e Expediente) Cancelar(versionEsperada uint64, datos DatosCancelacion, actuacion DatosActuacion) (Expediente, error) {
	if e.Validar() != nil || datos.Validar() != nil || actuacion.validar() != nil ||
		!e.CancelableEn(datos.FasesAdmitidas) ||
		actuacion.AccionClave != AccionCancelarExpediente ||
		actuacion.FaseDestino != e.FaseActual || actuacion.EstadoDestino != EstadoCancelado ||
		actuacion.UnidadRef != e.UnidadActual() || actuacion.Observaciones != datos.Observaciones ||
		len(actuacion.DocumentosRef) != 0 || actuacion.RetornoRef != "" {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	return siguiente.confirmarTransicion(actuacion)
}
