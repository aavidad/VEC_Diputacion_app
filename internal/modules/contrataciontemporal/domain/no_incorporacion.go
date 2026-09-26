package domain

import (
	"errors"
	"regexp"
	"time"
)

// No incorporación (duda 12 de RRHH, respuesta de ejemplo: «no incorporación
// = baja y siguiente»): la persona aceptada y nombrada no llega a
// incorporarse. RRHH lo registra con la resolución (referencia y huella) y,
// si el catálogo lo pide, con una segunda persona que la resuelve. El
// expediente vuelve a la fiscalización en curso: la necesidad sigue
// fiscalizada y el llamamiento continúa con el siguiente candidato.
const AccionRegistrarNoIncorporacion ClaveCatalogo = "contratacion_temporal.incorporacion.no_incorporacion"

var (
	ErrNoIncorporacionInvalida = errors.New("contratacion temporal: no incorporacion invalida")

	patronMotivoNoIncorporacion       = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	patronConsecuenciaNoIncorporacion = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
)

// DatosNoIncorporacion es lo que RRHH registra. La consecuencia es la clave
// que fija el catálogo para el motivo (la aplica Bolsa); ResueltaPor es la
// referencia de quien firma la resolución.
type DatosNoIncorporacion struct {
	MotivoClave       string
	ConsecuenciaClave string
	ResolucionRef     string
	ResolucionSHA256  string
	ResueltaPor       string
	SegundaPersona    bool
	FechaNotificacion time.Time
	Observaciones     string
}

// Validar comprueba forma y tamaño. Con segunda persona, quien resuelve no
// puede ser quien registra: lo compara ValidarPara con el actor.
func (d DatosNoIncorporacion) Validar() error {
	if !patronMotivoNoIncorporacion.MatchString(d.MotivoClave) || !patronConsecuenciaNoIncorporacion.MatchString(d.ConsecuenciaClave) ||
		!referenciaValida(d.ResolucionRef) || !huellaEntradaValida(d.ResolucionSHA256) || !referenciaValida(d.ResueltaPor) ||
		!fechaCivilCanonica(d.FechaNotificacion) || !textoValido(d.Observaciones, 2000, true) {
		return ErrNoIncorporacionInvalida
	}
	return nil
}

// ValidarPara añade la segregación: con segunda persona, otra persona.
func (d DatosNoIncorporacion) ValidarPara(actorRef string) error {
	if d.Validar() != nil || (d.SegundaPersona && d.ResueltaPor == actorRef) {
		return ErrNoIncorporacionInvalida
	}
	return nil
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

// RegistrarNoIncorporacion devuelve el expediente nombrado a la
// fiscalización en curso con la actuación de la no incorporación. SQL
// comprueba la aceptación vigente y que no conste la incorporación.
func (e Expediente) RegistrarNoIncorporacion(versionEsperada uint64, datos DatosNoIncorporacion, actuacion DatosActuacion) (Expediente, error) {
	if e.Validar() != nil || datos.ValidarPara(actuacion.ActorRef) != nil || actuacion.validar() != nil || !e.enNombramientoVigente() ||
		e.TieneAccion(AccionCesarNombramiento) || actuacion.AccionClave != AccionRegistrarNoIncorporacion ||
		actuacion.FaseDestino != FaseFiscalizacion || actuacion.EstadoDestino != EstadoEnCurso ||
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
