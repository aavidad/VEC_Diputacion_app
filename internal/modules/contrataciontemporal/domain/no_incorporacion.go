package domain

import (
	"errors"
	"regexp"
	"time"
)

// No incorporación (duda 12 de RRHH, respuesta de ejemplo: «no incorporación
// = baja y siguiente»): la persona aceptada y nombrada no llega a
// incorporarse. RRHH lo registra con la resolución (referencia y huella). El
// expediente vuelve a la fiscalización en curso: la necesidad sigue
// fiscalizada y el llamamiento continúa con el siguiente candidato.
//
// Si el catálogo exige segunda persona (c22), el control es de cuatro ojos:
// una persona lo propone (sin efecto), y otra persona autenticada, distinta,
// lo confirma (efecto completo; ella es quien resuelve) o lo rechaza. Sin
// segunda persona se registra en un solo paso, con quien resolvió declarado.
const (
	AccionRegistrarNoIncorporacion ClaveCatalogo = "contratacion_temporal.incorporacion.no_incorporacion"
	// Acciones de las actuaciones de los pasos sin efecto (cuatro ojos).
	AccionProponerNoIncorporacion ClaveCatalogo = "contratacion_temporal.incorporacion.no_incorporacion_propuesta"
	AccionRechazarNoIncorporacion ClaveCatalogo = "contratacion_temporal.incorporacion.no_incorporacion_rechazo"
)

// Pasos de la no incorporación.
const (
	PasoNoIncorporacionRegistrar = "registrar"
	PasoNoIncorporacionProponer  = "proponer"
	PasoNoIncorporacionConfirmar = "confirmar"
	PasoNoIncorporacionRechazar  = "rechazar"
)

var (
	ErrNoIncorporacionInvalida = errors.New("contratacion temporal: no incorporacion invalida")

	patronMotivoNoIncorporacion       = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	patronConsecuenciaNoIncorporacion = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
)

// DatosNoIncorporacion es lo que RRHH registra. La consecuencia es la clave
// que fija el catálogo para el motivo (la aplica Bolsa); ResueltaPor es la
// referencia de quien resuelve: declarada en el paso único, vacía al
// proponer y la del actor autenticado al confirmar o rechazar.
type DatosNoIncorporacion struct {
	Paso              string
	PropuestaRef      string
	MotivoClave       string
	ConsecuenciaClave string
	ResolucionRef     string
	ResolucionSHA256  string
	ResueltaPor       string
	SegundaPersona    bool
	FechaNotificacion time.Time
	Observaciones     string
}

// Validar comprueba forma, tamaño y la coherencia del paso con la
// segregación: cada paso exige o prohíbe la segunda persona.
func (d DatosNoIncorporacion) Validar() error {
	if !patronMotivoNoIncorporacion.MatchString(d.MotivoClave) || !patronConsecuenciaNoIncorporacion.MatchString(d.ConsecuenciaClave) ||
		!referenciaValida(d.ResolucionRef) || !huellaEntradaValida(d.ResolucionSHA256) ||
		!fechaCivilCanonica(d.FechaNotificacion) || !textoValido(d.Observaciones, 2000, true) {
		return ErrNoIncorporacionInvalida
	}
	switch d.Paso {
	case PasoNoIncorporacionRegistrar:
		if d.SegundaPersona || d.PropuestaRef != "" || !referenciaValida(d.ResueltaPor) {
			return ErrNoIncorporacionInvalida
		}
	case PasoNoIncorporacionProponer:
		if !d.SegundaPersona || d.PropuestaRef != "" || d.ResueltaPor != "" {
			return ErrNoIncorporacionInvalida
		}
	case PasoNoIncorporacionConfirmar, PasoNoIncorporacionRechazar:
		if !d.SegundaPersona || !referenciaValida(d.PropuestaRef) || !referenciaValida(d.ResueltaPor) {
			return ErrNoIncorporacionInvalida
		}
	default:
		return ErrNoIncorporacionInvalida
	}
	return nil
}

// ValidarPara añade la segregación: al confirmar o rechazar, quien resuelve
// es el actor autenticado del paso (que SQL coteja con quien propuso).
func (d DatosNoIncorporacion) ValidarPara(actorRef string) error {
	if d.Validar() != nil || ((d.Paso == PasoNoIncorporacionConfirmar || d.Paso == PasoNoIncorporacionRechazar) && d.ResueltaPor != actorRef) {
		return ErrNoIncorporacionInvalida
	}
	return nil
}

// ConEfecto indica si el paso registra la no incorporación (vuelta a
// fiscalización, baja en Bolsa y siguiente candidato).
func (d DatosNoIncorporacion) ConEfecto() bool {
	return d.Paso == PasoNoIncorporacionRegistrar || d.Paso == PasoNoIncorporacionConfirmar
}

// AccionActuacion es la acción que queda en la historia del expediente.
func (d DatosNoIncorporacion) AccionActuacion() ClaveCatalogo {
	switch d.Paso {
	case PasoNoIncorporacionProponer:
		return AccionProponerNoIncorporacion
	case PasoNoIncorporacionRechazar:
		return AccionRechazarNoIncorporacion
	}
	return AccionRegistrarNoIncorporacion
}

// FaseDestino: con efecto vuelve a fiscalización; si no, sigue en
// nombramiento.
func (d DatosNoIncorporacion) FaseDestino() ClaveFase {
	if d.ConEfecto() {
		return FaseFiscalizacion
	}
	return FaseNombramiento
}

// MotivoNoIncorporacionValido comprueba la forma de la clave del motivo.
func MotivoNoIncorporacionValido(clave string) bool {
	return patronMotivoNoIncorporacion.MatchString(clave)
}

// ConsecuenciaNoIncorporacionValida comprueba la forma de la clave de la
// consecuencia (una entrada del catálogo de Bolsa).
func ConsecuenciaNoIncorporacionValida(clave string) bool {
	return patronConsecuenciaNoIncorporacion.MatchString(clave)
}

// RegistrarNoIncorporacion aplica el paso: con efecto devuelve el expediente
// nombrado a la fiscalización en curso con la actuación de la no
// incorporación; al proponer o rechazar solo añade su actuación y sigue en
// nombramiento. SQL comprueba la aceptación vigente, que no conste la
// incorporación y la propuesta pendiente.
func (e Expediente) RegistrarNoIncorporacion(versionEsperada uint64, datos DatosNoIncorporacion, actuacion DatosActuacion) (Expediente, error) {
	if e.Validar() != nil || datos.ValidarPara(actuacion.ActorRef) != nil || actuacion.validar() != nil || !e.enNombramientoVigente() ||
		e.TieneAccion(AccionCesarNombramiento) || actuacion.AccionClave != datos.AccionActuacion() ||
		actuacion.FaseDestino != datos.FaseDestino() || actuacion.EstadoDestino != EstadoEnCurso ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef || actuacion.Observaciones != datos.Observaciones ||
		len(actuacion.DocumentosRef) != 1 || actuacion.DocumentosRef[0] != datos.ResolucionRef ||
		actuacion.RetornoRef != "" {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	return siguiente.confirmarTransicion(actuacion)
}
