//go:build ignore && linux && amd64

package main

import (
	"errors"
	"sync/atomic"
	"time"
)

type estadoContinuacionO3cM38 uint8

const (
	continuacionC0RecibidoM38 estadoContinuacionO3cM38 = iota
	continuacionC1RevalidadoM38
	continuacionC2ContIntentadoM38
	continuacionC3ObservadoM38
	continuacionC4TTransfiriendoM38
	continuacionC5EntregadoM38
	continuacionC7RetirandoM38
	continuacionC8RetiradoM38
	continuacionCFFatalM38
)

type propietarioCustodiaO3cM38 uint32

const (
	propietarioInactivoO3cM38 propietarioCustodiaO3cM38 = iota
	propietarioO3cM38
	propietarioO4aM38
	propietarioLiberadoO3cM38
)

var (
	errEntradaO3cM38      = errors.New("entrada O3c inválida")
	errUsoConsumidoO3cM38 = errors.New("entrada O3c consumida")
	errAutoridadO3cM38    = errors.New("autoridad O3c inválida")
)

// autoridadCustodiaO3cM38 es la única autoridad que O3c podrá entregar a O4a.
// Sus owners nacen juntos después de consumir las autoridades O3b; P1 no los
// transfiere ni libera.
type autoridadCustodiaO3cM38 struct {
	auto            *autoridadCustodiaO3cM38
	ownerObservador atomic.Uint32
	ownerLease      atomic.Uint32
}

func nuevaAutoridadCustodiaO3cM38() *autoridadCustodiaO3cM38 {
	a := &autoridadCustodiaO3cM38{}
	a.auto = a
	return a
}

func (a *autoridadCustodiaO3cM38) activarO3c() bool {
	if a == nil || a.auto != a ||
		!a.ownerObservador.CompareAndSwap(uint32(propietarioInactivoO3cM38), uint32(propietarioO3cM38)) {
		return false
	}
	if !a.ownerLease.CompareAndSwap(uint32(propietarioInactivoO3cM38), uint32(propietarioO3cM38)) {
		fatalO3cM38()
	}
	return true
}

func (a *autoridadCustodiaO3cM38) poseeO3c() bool {
	return a != nil && a.auto == a &&
		a.ownerObservador.Load() == uint32(propietarioO3cM38) &&
		a.ownerLease.Load() == uint32(propietarioO3cM38)
}

// agregadoO4aM38 permanece privado y sin métodos de efecto. P1 solo reserva
// su identidad y custodia; las fases posteriores llenarán los sellos ya
// preasignados sin exponer PID, pidfd ni recursos por separado.
type agregadoO4aM38 struct {
	auto        *agregadoO4aM38
	autoridad   *autoridadCustodiaO3cM38
	custodia    *custodiaO3aM38
	identidad   muestraStatO3bM38
	primera     atomic.Uint32
	ahoraCaso   time.Time
	finCaso     time.Time
	retornoCont int
}

type autoridadContinuacionO3cM38 struct {
	estado    estadoContinuacionO3cM38
	custodia  *custodiaO3aM38
	identidad muestraStatO3bM38
	autoridad *autoridadCustodiaO3cM38
	salida    *agregadoO4aM38
}

func (a *autoridadContinuacionO3cM38) es(estado estadoContinuacionO3cM38) bool {
	if a == nil || a.estado != estado {
		return false
	}
	return estado == continuacionC8RetiradoM38 || estado == continuacionCFFatalM38 ||
		a.custodia != nil && a.autoridad != nil && a.salida != nil
}

func transicionContinuacionO3cM38(desde, hacia estadoContinuacionO3cM38) bool {
	switch desde {
	case continuacionC0RecibidoM38:
		return hacia == continuacionC1RevalidadoM38 || hacia == continuacionC7RetirandoM38 || hacia == continuacionCFFatalM38
	case continuacionC1RevalidadoM38:
		return hacia == continuacionC2ContIntentadoM38 || hacia == continuacionCFFatalM38
	case continuacionC2ContIntentadoM38:
		return hacia == continuacionC3ObservadoM38 || hacia == continuacionCFFatalM38
	case continuacionC3ObservadoM38:
		return hacia == continuacionC4TTransfiriendoM38 || hacia == continuacionCFFatalM38
	case continuacionC4TTransfiriendoM38:
		return hacia == continuacionC5EntregadoM38 || hacia == continuacionCFFatalM38
	case continuacionC7RetirandoM38:
		return hacia == continuacionC8RetiradoM38 || hacia == continuacionCFFatalM38
	default:
		return false
	}
}

type claseEntradaO3cM38 uint8

const (
	entradaO3cValidaM38 claseEntradaO3cM38 = iota
	entradaO3cConsumidaM38
	entradaO3cFatalM38
)

func clasificarEntradaO3cM38(a *agregadoO3cM38) claseEntradaO3cM38 {
	if a == nil || a.estado != capturaB5CapturadoM38 || a.custodia == nil {
		return entradaO3cConsumidaM38
	}
	c := a.custodia
	if c.lease == nil || c.observador == nil {
		return entradaO3cFatalM38
	}
	estadoLease := c.lease.estado.Load()
	estadoObservador := c.observador.palabra.Load() & mascaraEstadoObservadorO3aM38
	if c.lease.auto != c.lease || c.observador.auto != c.observador ||
		c.lease.registro == nil || c.lease.registro.auto != c.lease.registro ||
		c.lease.registro != c.observador.registro || c.lease.registro.tid != c.tid || c.lease.tid != c.tid ||
		c.lease.registro.leases[c.lease] != c.lease.generacion ||
		c.observador.registro.observadores[c.observador] != c.observador.generacion ||
		c.baselineSenal != c.observador.palabra.Load() || uint8(c.baselineSenal>>2) != 0 {
		return entradaO3cFatalM38
	}
	if estadoLease == 3 && estadoObservador == 2 {
		return entradaO3cConsumidaM38
	}
	if estadoLease != 1 || estadoObservador != 1 {
		return entradaO3cFatalM38
	}
	return entradaO3cValidaM38
}

// custodiaConsumidaValidaO3cM38 solo se invoca cuando ambos CAS ya
// consolidaron. Así ningún recurso ajeno a las dos autoridades se observa
// antes de que O3c sea su propietario irrevocable.
func custodiaConsumidaValidaO3cM38(a *agregadoO3cM38) bool {
	c := a.custodia
	return c.cmd != nil && c.cmd.Process != nil && c.controlFD != nil && c.terminal != nil &&
		c.ticketEscritor == nil && c.ticketLector == nil && a.primera == nil &&
		c.pidfdPrimario >= 0 && c.pidfdReserva >= 0 && c.pidfdOpaco >= 0 &&
		c.pidfdPrimario != c.pidfdReserva && c.pidfdPrimario != c.pidfdOpaco && c.pidfdReserva != c.pidfdOpaco &&
		a.identidad.pid == c.cmd.Process.Pid && a.identidad.estado == 'T' && a.identidad.inicio > 0
}

// consolidarEntradaO3cM38 materializa el orden contractual observador→lease.
// Si el segundo CAS falla después del primero, la partición es irreversible.
func consolidarEntradaO3cM38(c *custodiaO3aM38, autoridad *autoridadCustodiaO3cM38) uint64 {
	nuevoBaseline, ok := c.observador.transferirCritico(c.baselineSenal)
	if !ok || !c.lease.transferirCritico() || !autoridad.activarO3c() {
		fatalO3cM38()
	}
	return nuevoBaseline
}

// consumirAutoridadO3cM38 anula el puntero del llamador antes de observar B5.
// No ejecuta revalidación operativa, reloj, señal, poll, Wait ni efecto O4.
func consumirAutoridadO3cM38(entrada **agregadoO3cM38) (*autoridadContinuacionO3cM38, error) {
	if entrada == nil || *entrada == nil {
		return nil, errEntradaO3cM38
	}
	agregado := *entrada
	*entrada = nil
	autoridad := nuevaAutoridadCustodiaO3cM38()
	salida := &agregadoO4aM38{autoridad: autoridad}
	salida.auto = salida

	switch clasificarEntradaO3cM38(agregado) {
	case entradaO3cConsumidaM38:
		return &autoridadContinuacionO3cM38{estado: continuacionC8RetiradoM38}, errUsoConsumidoO3cM38
	case entradaO3cFatalM38:
		fatalO3cM38()
	}

	c := agregado.custodia
	nuevoBaseline := consolidarEntradaO3cM38(c, autoridad)
	c.baselineSenal = nuevoBaseline
	if !custodiaConsumidaValidaO3cM38(agregado) {
		fatalO3cM38()
	}
	salida.custodia, salida.identidad = c, agregado.identidad
	return &autoridadContinuacionO3cM38{
		estado: continuacionC0RecibidoM38, custodia: c, identidad: agregado.identidad,
		autoridad: autoridad, salida: salida,
	}, nil
}

func fatalO3cM38() {
	fatalO3aM38()
}
