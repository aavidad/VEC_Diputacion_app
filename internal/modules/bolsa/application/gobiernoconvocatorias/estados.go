package gobiernoconvocatorias

import (
	"errors"
	"math"
	"time"
)

const DuracionMaximaArrendamientoDiario = 5 * time.Minute

var (
	ErrEstadoDiarioInvalido        = errors.New("gobierno convocatorias: estado del diario invalido")
	ErrRevisionDiarioInvalida      = errors.New("gobierno convocatorias: revision del diario invalida")
	ErrCercadoDiarioInvalido       = errors.New("gobierno convocatorias: cercado del diario invalido")
	ErrArrendamientoDiarioInvalido = errors.New("gobierno convocatorias: arrendamiento del diario invalido")
	ErrArrendamientoDiarioExpirado = errors.New("gobierno convocatorias: arrendamiento del diario expirado")
	ErrConflictoRevisionDiario     = errors.New("gobierno convocatorias: conflicto de revision del diario")
	ErrCercadoDiarioObsoleto       = errors.New("gobierno convocatorias: cercado del diario obsoleto")
	ErrTransicionDiarioProhibida   = errors.New("gobierno convocatorias: transicion del diario prohibida")
)

type EstadoDiario string

const (
	EstadoDecisionVinculada    EstadoDiario = "decision_vinculada"
	EstadoConfirmacionIniciada EstadoDiario = "confirmacion_iniciada"
	EstadoIndeterminada        EstadoDiario = "indeterminada"
	EstadoConfirmada           EstadoDiario = "confirmada"
	EstadoNoAplicada           EstadoDiario = "no_aplicada"
)

func (e EstadoDiario) Valido() bool {
	switch e {
	case EstadoDecisionVinculada, EstadoConfirmacionIniciada,
		EstadoIndeterminada, EstadoConfirmada, EstadoNoAplicada:
		return true
	default:
		return false
	}
}

func (e EstadoDiario) Terminal() bool {
	return e == EstadoConfirmada || e == EstadoNoAplicada
}

type RevisionDiario struct {
	bloqueoSerializacionDiario
	valor uint64
}

func NuevaRevisionDiario(valor uint64) (RevisionDiario, error) {
	if valor == 0 {
		return RevisionDiario{}, ErrRevisionDiarioInvalida
	}
	return RevisionDiario{valor: valor}, nil
}

func (r RevisionDiario) Valida() bool { return r.valor > 0 }

func (r RevisionDiario) Siguiente() (RevisionDiario, error) {
	if !r.Valida() || r.valor == math.MaxUint64 {
		return RevisionDiario{}, ErrRevisionDiarioInvalida
	}
	return RevisionDiario{valor: r.valor + 1}, nil
}

func (r RevisionDiario) coincide(otra RevisionDiario) bool {
	return r.Valida() && otra.Valida() && r.valor == otra.valor
}

type CercadoDiario struct {
	bloqueoSerializacionDiario
	valor uint64
}

func NuevoCercadoDiario(valor uint64) (CercadoDiario, error) {
	if valor == 0 {
		return CercadoDiario{}, ErrCercadoDiarioInvalido
	}
	return CercadoDiario{valor: valor}, nil
}

func (c CercadoDiario) Valido() bool { return c.valor > 0 }

func (c CercadoDiario) Siguiente() (CercadoDiario, error) {
	if !c.Valido() || c.valor == math.MaxUint64 {
		return CercadoDiario{}, ErrCercadoDiarioInvalido
	}
	return CercadoDiario{valor: c.valor + 1}, nil
}

func (c CercadoDiario) coincide(otro CercadoDiario) bool {
	return c.Valido() && otro.Valido() && c.valor == otro.valor
}

type ArrendamientoDiario struct {
	bloqueoSerializacionDiario
	iniciaEn time.Time
	venceEn  time.Time
}

func NuevoArrendamientoDiario(iniciaEn, venceEn time.Time) (ArrendamientoDiario, error) {
	arrendamiento := ArrendamientoDiario{iniciaEn: iniciaEn, venceEn: venceEn}
	if !arrendamiento.valido() {
		return ArrendamientoDiario{}, ErrArrendamientoDiarioInvalido
	}
	return arrendamiento, nil
}

func (a ArrendamientoDiario) valido() bool {
	return instanteDiarioCanonico(a.iniciaEn) && instanteDiarioCanonico(a.venceEn) &&
		a.venceEn.After(a.iniciaEn) &&
		a.venceEn.Sub(a.iniciaEn) <= DuracionMaximaArrendamientoDiario
}

// VigenteEn usa el intervalo semicerrado [inicio, vencimiento). En la igualdad
// exacta con el vencimiento el arrendamiento ya esta expirado.
func (a ArrendamientoDiario) VigenteEn(instante time.Time) bool {
	return a.valido() && instanteDiarioCanonico(instante) &&
		!instante.Before(a.iniciaEn) && instante.Before(a.venceEn)
}

func (a ArrendamientoDiario) ExpiradoEn(instante time.Time) bool {
	return a.valido() && instanteDiarioCanonico(instante) && !instante.Before(a.venceEn)
}

type DesenlaceTransicionDiario string

const (
	DesenlaceTransicionAplicada    DesenlaceTransicionDiario = "aplicada"
	DesenlaceTransicionIdempotente DesenlaceTransicionDiario = "idempotente"
)

type ControlEstadoDiario struct {
	bloqueoSerializacionDiario
	estado        EstadoDiario
	revision      RevisionDiario
	cercado       CercadoDiario
	arrendamiento ArrendamientoDiario
}

// nuevoControlEstadoDiario no es una fabrica publica de autoridad. En el
// siguiente corte solo podra invocarlo la reserva CAS posterior a una
// concesion PDP exacta, que ligara conjuntamente L, F, intencion y decision.
func nuevoControlEstadoDiario(
	arrendamiento ArrendamientoDiario,
) (ControlEstadoDiario, error) {
	revision, _ := NuevaRevisionDiario(1)
	cercado, _ := NuevoCercadoDiario(1)
	control := ControlEstadoDiario{
		estado:   EstadoDecisionVinculada,
		revision: revision, cercado: cercado, arrendamiento: arrendamiento,
	}
	if !control.valido() {
		return ControlEstadoDiario{}, ErrEstadoDiarioInvalido
	}
	return control, nil
}

func (c ControlEstadoDiario) valido() bool {
	return c.estado.Valido() && c.revision.Valida() && c.cercado.Valido() && c.arrendamiento.valido()
}

func (c ControlEstadoDiario) Estado() EstadoDiario { return c.estado }

func (c ControlEstadoDiario) Transicionar(
	destino EstadoDiario,
	revisionEsperada RevisionDiario,
	cercadoPresentado CercadoDiario,
	instante time.Time,
) (ControlEstadoDiario, DesenlaceTransicionDiario, error) {
	if !c.valido() || !destino.Valido() || !instanteDiarioCanonico(instante) {
		return ControlEstadoDiario{}, "", ErrEstadoDiarioInvalido
	}
	if !c.revision.coincide(revisionEsperada) {
		return ControlEstadoDiario{}, "", ErrConflictoRevisionDiario
	}
	if !c.cercado.coincide(cercadoPresentado) {
		return ControlEstadoDiario{}, "", ErrCercadoDiarioObsoleto
	}
	if c.estado.Terminal() && destino == c.estado {
		return c, DesenlaceTransicionIdempotente, nil
	}
	if !c.arrendamiento.VigenteEn(instante) {
		return ControlEstadoDiario{}, "", ErrArrendamientoDiarioExpirado
	}
	if !transicionEstadoDiarioPermitida(c.estado, destino) {
		return ControlEstadoDiario{}, "", ErrTransicionDiarioProhibida
	}
	siguiente, err := c.revision.Siguiente()
	if err != nil {
		return ControlEstadoDiario{}, "", err
	}
	c.estado = destino
	c.revision = siguiente
	return c, DesenlaceTransicionAplicada, nil
}

func (c ControlEstadoDiario) ReclamarTrasExpiracion(
	revisionEsperada RevisionDiario,
	cercadoEsperado CercadoDiario,
	instante time.Time,
	nuevoArrendamiento ArrendamientoDiario,
) (ControlEstadoDiario, error) {
	if !c.valido() || c.estado.Terminal() || !revisionEsperada.Valida() ||
		!c.revision.coincide(revisionEsperada) {
		return ControlEstadoDiario{}, ErrConflictoRevisionDiario
	}
	if !c.cercado.coincide(cercadoEsperado) {
		return ControlEstadoDiario{}, ErrCercadoDiarioObsoleto
	}
	if !c.arrendamiento.ExpiradoEn(instante) || !nuevoArrendamiento.valido() ||
		!nuevoArrendamiento.iniciaEn.Equal(instante) {
		return ControlEstadoDiario{}, ErrArrendamientoDiarioInvalido
	}
	siguienteRevision, errRevision := c.revision.Siguiente()
	siguienteCercado, errCercado := c.cercado.Siguiente()
	if errRevision != nil || errCercado != nil {
		return ControlEstadoDiario{}, ErrCercadoDiarioInvalido
	}
	c.revision = siguienteRevision
	c.cercado = siguienteCercado
	c.arrendamiento = nuevoArrendamiento
	return c, nil
}

func transicionEstadoDiarioPermitida(origen, destino EstadoDiario) bool {
	switch origen {
	case EstadoDecisionVinculada:
		return destino == EstadoConfirmacionIniciada || destino == EstadoNoAplicada
	case EstadoConfirmacionIniciada:
		return destino == EstadoIndeterminada || destino == EstadoConfirmada || destino == EstadoNoAplicada
	case EstadoIndeterminada:
		return destino == EstadoIndeterminada || destino == EstadoConfirmada || destino == EstadoNoAplicada
	default:
		return false
	}
}

func instanteDiarioCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Nanosecond()%1_000 == 0
}
