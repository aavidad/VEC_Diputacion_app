package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrOperacionSituacionParticipacionInvalida = errors.New("bolsa: operacion de situacion de participacion invalida")

// B8 nombra las tres operaciones que RRHH ejecuta sobre la permanencia de una
// persona en la bolsa. No son situaciones nuevas: cada una produce exactamente
// una de las de B2, de modo que el historial sigue siendo el mismo.
const (
	OperacionPausar    = "pausar"
	OperacionReactivar = "reactivar"
	OperacionExcluir   = "excluir"
)

var destinoOperacionSituacion = map[string]string{
	OperacionPausar:    SituacionNoDisponible,
	OperacionReactivar: SituacionDisponible,
	OperacionExcluir:   SituacionExcluido,
}

// Tipos de justificante admitidos. El documento vive en su custodia; VEC
// conserva solo su referencia y su huella, nunca el contenido.
const (
	JustificanteSolicitudCandidato = "solicitud_candidato"
	JustificanteInformeMedico      = "informe_medico"
	JustificanteResolucion         = "resolucion"
	JustificanteCorreo             = "correo"
	JustificanteActaBolsa          = "acta_bolsa"
	JustificanteOtro               = "otro"
)

var catalogoTiposJustificante = map[string]struct{}{
	JustificanteSolicitudCandidato: {}, JustificanteInformeMedico: {},
	JustificanteResolucion: {}, JustificanteCorreo: {},
	JustificanteActaBolsa: {}, JustificanteOtro: {},
}

var patronHuellaJustificante = regexp.MustCompile(`^[a-f0-9]{64}$`)

// JustificanteOperacionSituacion acredita la operación sin incorporar el
// documento: referencia opaca en su custodia y huella SHA-256 en minúsculas.
type JustificanteOperacionSituacion struct {
	Tipo       string
	Referencia string
	SHA256     string
}

func (j JustificanteOperacionSituacion) Validar() error {
	if _, ok := catalogoTiposJustificante[j.Tipo]; !ok {
		return ErrOperacionSituacionParticipacionInvalida
	}
	if !referenciaLlamamientoOpacaValida(j.Referencia) || len(j.Referencia) > 256 ||
		!patronHuellaJustificante.MatchString(j.SHA256) {
		return ErrOperacionSituacionParticipacionInvalida
	}
	return nil
}

// OperacionSituacionParticipacion añade a un cambio de B2 quién lo registra,
// con qué justificante y quién lo valida.
type OperacionSituacionParticipacion struct {
	Cambio       CambioSituacionParticipacion
	Operacion    string
	Justificante JustificanteOperacionSituacion
	Actor        string
	Validador    string
	ValidadaEn   time.Time
}

// OperacionesSituacionParticipacion devuelve el catálogo en orden estable.
func OperacionesSituacionParticipacion() []string {
	return []string{OperacionPausar, OperacionReactivar, OperacionExcluir}
}

// DestinoOperacionSituacion traduce la operación a la situación de B2 que
// produce. Una operación desconocida no tiene destino.
func DestinoOperacionSituacion(operacion string) (string, bool) {
	destino, ok := destinoOperacionSituacion[operacion]
	return destino, ok
}

// OperacionesDisponiblesDesde enumera las operaciones de B8 que la situación
// vigente admite, según las transiciones ya fijadas por B2.
func OperacionesDisponiblesDesde(origen string) []string {
	disponibles := make([]string, 0, len(destinoOperacionSituacion))
	for _, operacion := range OperacionesSituacionParticipacion() {
		destino := destinoOperacionSituacion[operacion]
		if _, ok := transicionesSituacionParticipacion[origen][destino]; ok {
			disponibles = append(disponibles, operacion)
		}
	}
	return disponibles
}

// ExigeValidadorDistinto marca las operaciones en las que quien valida no
// puede ser quien registra. Regla provisional de dirección mientras RRHH no
// responda la duda 6: solo se exige en la exclusión, que no tiene vuelta
// atrás dentro de B2.
func ExigeValidadorDistinto(operacion string) bool { return operacion == OperacionExcluir }

func (o OperacionSituacionParticipacion) Validar() error {
	destino, conocida := destinoOperacionSituacion[o.Operacion]
	if !conocida || o.Cambio.Destino != destino {
		return ErrOperacionSituacionParticipacionInvalida
	}
	if err := o.Cambio.Validar(); err != nil {
		return errors.Join(ErrOperacionSituacionParticipacionInvalida, err)
	}
	if err := o.Justificante.Validar(); err != nil {
		return err
	}
	if !identidadOperacionValida(o.Actor) || !identidadOperacionValida(o.Validador) {
		return ErrOperacionSituacionParticipacionInvalida
	}
	if ExigeValidadorDistinto(o.Operacion) && o.Validador == o.Actor {
		return ErrOperacionSituacionParticipacionInvalida
	}
	if !instanteLlamamientoCanonico(o.ValidadaEn) || o.ValidadaEn.After(o.Cambio.RegistradaEn) {
		return ErrOperacionSituacionParticipacionInvalida
	}
	return nil
}

func identidadOperacionValida(valor string) bool {
	return strings.TrimSpace(valor) == valor && len(valor) > 0 && len(valor) <= 256
}
