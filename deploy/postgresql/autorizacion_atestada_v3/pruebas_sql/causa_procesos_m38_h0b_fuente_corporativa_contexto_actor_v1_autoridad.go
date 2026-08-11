//go:build ignore && linux && amd64

package main

import (
	"errors"
	"maps"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

type estadoCausaO4aM38 uint32

const (
	causaA0RecibidoM38 estadoCausaO4aM38 = iota
	causaA1ValidadoM38
	causaA2ObservandoM38
	causaA3CausaFijadaM38
	causaA4PermisoPreparadoM38
	causaA5EsperandoResultadoM38
	causaA7EntregaO4cPreparadaM38
	causaA8EntregadoO4cM38
	causaAFFatalM38
)

type causaPrimariaO4aM38 uint32

const causaVaciaO4aM38 causaPrimariaO4aM38 = 0

const (
	custodiaEntregadaO3cM38 uint32 = 2
	custodiaRecibidaO4aM38  uint32 = 3
)

var errConsumoO4aM38 = errors.New("autoridad O4a ya consumida")

type canonControlRawO4aM38 uint8

const (
	controlRawVacioO4aM38 canonControlRawO4aM38 = iota
	controlRawCancelado65O4aM38
	controlRawProtocolo65O4aM38
	controlRawSenalInt130O4aM38
	controlRawSenalTerm143O4aM38
)

type sellosO4aM38 struct {
	autoridad                     *autoridadCustodiaO3cM38
	autoridadArranque             *autoridadEstadoO3aM38
	custodia                      *custodiaO3aM38
	lease                         *leaseGuardiaO3aM38
	observador                    *observadorSenalO3aM38
	registro                      *registroAutoridadO3aM38
	control                       *controladorPreinicioM38
	controlFD                     *os.File
	terminal                      *os.File
	cmd                           *exec.Cmd
	proceso                       *os.Process
	generacionLease               uint64
	generacionObservador          uint64
	tid, ppid                     int
	baselineSenal                 uint64
	pidfd                         [3]int
	identidad                     muestraStatO3bM38
	primera                       uint32
	retornoCont                   int
	palabraObservada              uint64
	canonControlRaw               canonControlRawO4aM38
	ahoraCaso, finCaso            time.Time
	fisico                        snapshotFDO3aM38
	huellaControl, huellaTerminal huellaFDO3aM38
}

type autoridadCausaO4aM38 struct {
	auto      *autoridadCausaO4aM38
	origen    *agregadoO4aM38
	estado    atomic.Uint32
	causa     atomic.Uint32
	incidente atomic.Uint32
	sellos    sellosO4aM38
}

func transicionCausaO4aM38(desde, hacia estadoCausaO4aM38) bool {
	switch desde {
	case causaA0RecibidoM38:
		return hacia == causaA1ValidadoM38 || hacia == causaAFFatalM38
	case causaA1ValidadoM38:
		return hacia == causaA2ObservandoM38 || hacia == causaAFFatalM38
	case causaA2ObservandoM38:
		return hacia == causaA2ObservandoM38 || hacia == causaA3CausaFijadaM38 || hacia == causaAFFatalM38
	case causaA3CausaFijadaM38:
		return hacia == causaA4PermisoPreparadoM38 || hacia == causaA7EntregaO4cPreparadaM38 || hacia == causaAFFatalM38
	case causaA4PermisoPreparadoM38:
		return hacia == causaA5EsperandoResultadoM38 || hacia == causaAFFatalM38
	case causaA5EsperandoResultadoM38:
		return hacia == causaA3CausaFijadaM38 || hacia == causaA4PermisoPreparadoM38 ||
			hacia == causaA7EntregaO4cPreparadaM38 || hacia == causaAFFatalM38
	case causaA7EntregaO4cPreparadaM38:
		return hacia == causaA8EntregadoO4cM38 || hacia == causaAFFatalM38
	}
	return false
}

func copiaSnapshotO4aM38(s snapshotFDO3aM38) snapshotFDO3aM38 {
	return snapshotFDO3aM38{limite: s.limite, mapa: maps.Clone(s.mapa)}
}

func normalizarControlRawO4aM38(a *agregadoO4aM38, primera uint32) (canonControlRawO4aM38, bool) {
	if discriminanteObservacionO3cM38(primera) != observacionControlRawO3cM38 {
		return controlRawVacioO4aM38, true
	}
	if a == nil || a.custodia == nil || a.custodia.control == nil {
		return 0, false
	}
	c, primeraCausa := a.custodia.control, a.custodia.primeraCausa
	if terminalFuncionalValidoPreinicioM38(c) {
		if primeraCausa != c.causa {
			return 0, false
		}
		causa, estado, valida := causaTransportadaPreinicioM38(c.causa.causa, c.causa.estado)
		if !valida {
			return 0, false
		}
		switch causa + "/" + estado {
		case "CANCELADO/65":
			return controlRawCancelado65O4aM38, true
		case "PROTOCOLO/65":
			return controlRawProtocolo65O4aM38, true
		case "SENAL_INT/130":
			return controlRawSenalInt130O4aM38, true
		case "SENAL_TERM/143":
			return controlRawSenalTerm143O4aM38, true
		}
	}
	if c.fase == controlPreinicioS3M38 && c.causa == (causaPreinicioM38{}) &&
		primeraCausa == (causaPreinicioM38{}) && c.fallo == nil && falloInvarianteActivoPreinicioM38(c) == nil {
		return controlRawProtocolo65O4aM38, true
	}
	return 0, false
}

func relacionPalabraRawO4aM38(primera uint32, baseline, observada uint64) bool {
	if baseline&mascaraEstadoObservadorO3aM38 != 2 || uint8(baseline>>2) != 0 ||
		observada&mascaraEstadoObservadorO3aM38 != 2 {
		return false
	}
	if discriminanteObservacionO3cM38(primera) == observacionSenalRawO3cM38 {
		return observada>>10 >= baseline>>10
	}
	return observada == baseline
}

func tiemposEntradaExactosO4aM38(ahora, fin, finBootstrap time.Time) bool {
	finCalculado, valido := finCasoExactoO3cM38(ahora)
	return valido && tiempoMonotonoO3cM38(fin) && tiempoMonotonoO3cM38(finBootstrap) &&
		fin == finCalculado && fin.After(ahora) && fin.Sub(ahora) == duracionCasoO3cM38 &&
		ahora.Before(finBootstrap)
}

func entradaBaseExactaO4aM38(a *agregadoO4aM38) bool {
	if a == nil || a.auto != a || a.autoridad == nil || a.autoridad.auto != a.autoridad || a.custodia == nil ||
		a.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
		a.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) || a.custodia.lease == nil ||
		a.custodia.observador == nil || a.custodia.control == nil || a.custodia.controlFD == nil ||
		a.custodia.terminal == nil || a.custodia.autoridad == nil || !a.custodia.autoridad.es(arranqueA6EntregadoM38) ||
		a.custodia.cmd == nil || a.custodia.cmd.Process == nil ||
		a.custodia.ticketEscritor != nil || a.custodia.ticketLector != nil {
		return false
	}
	c := a.custodia
	l, o := c.lease, c.observador
	r := l.registro
	p := [3]int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco}
	if l.auto != l || o.auto != o || r == nil || r.auto != r || o.registro != r ||
		r.leases[l] != l.generacion || r.observadores[o] != o.generacion || l.tid != c.tid ||
		r.tid != c.tid || l.estado.Load() != 3 || c.baselineSenal&mascaraEstadoObservadorO3aM38 != 2 ||
		uint8(c.baselineSenal>>2) != 0 ||
		p[0] < 0 || p[1] < 0 || p[2] < 0 || p[0] == p[1] || p[0] == p[2] || p[1] == p[2] ||
		a.identidad.estado != 'T' || a.identidad.pid != c.cmd.Process.Pid || a.identidad.ppid <= 0 ||
		a.identidad.pgid != c.cmd.Process.Pid || a.identidad.sid <= 0 || a.identidad.inicio == 0 {
		return false
	}
	f := l.fisico
	fdControl, fdTerminal := int(c.controlFD.Fd()), int(c.terminal.Fd())
	if fdControl == fdTerminal || fdControl == p[0] || fdControl == p[1] || fdControl == p[2] ||
		fdTerminal == p[0] || fdTerminal == p[1] || fdTerminal == p[2] {
		return false
	}
	h0, ok0 := f.mapa[p[0]]
	h1, ok1 := f.mapa[p[1]]
	h2, ok2 := f.mapa[p[2]]
	hc, okc := f.mapa[fdControl]
	ht, okt := f.mapa[fdTerminal]
	return f.mapa != nil && f.limite > 0 && ok0 && ok1 && ok2 && h0.abierto && h1.abierto && h2.abierto &&
		h0.identidad == h1.identidad && h1.identidad == h2.identidad && okc && okt && hc.abierto && ht.abierto
}

func sellarEntradaO4aM38(a *agregadoO4aM38, primera uint32, palabra uint64, canon canonControlRawO4aM38, ahora, fin time.Time) sellosO4aM38 {
	c, l, o := a.custodia, a.custodia.lease, a.custodia.observador
	return sellosO4aM38{
		autoridad: a.autoridad, autoridadArranque: c.autoridad, custodia: c, lease: l, observador: o, registro: l.registro,
		control: c.control, controlFD: c.controlFD, terminal: c.terminal, cmd: c.cmd, proceso: c.cmd.Process,
		generacionLease: l.generacion, generacionObservador: o.generacion, tid: c.tid, ppid: c.ppid,
		baselineSenal: c.baselineSenal, pidfd: [3]int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco},
		identidad: a.identidad, primera: primera, retornoCont: a.retornoCont,
		palabraObservada: palabra, canonControlRaw: canon, ahoraCaso: ahora, finCaso: fin,
		fisico: copiaSnapshotO4aM38(l.fisico), huellaControl: l.fisico.mapa[int(c.controlFD.Fd())],
		huellaTerminal: l.fisico.mapa[int(c.terminal.Fd())],
	}
}

func consumirAutoridadO4aM38(entrada **agregadoO4aM38) (*autoridadCausaO4aM38, error) {
	if entrada == nil || *entrada == nil {
		return nil, errConsumoO4aM38
	}
	a := *entrada
	*entrada = nil
	if a.auto != a {
		return nil, errConsumoO4aM38
	}
	if a.custodia == nil {
		fatalO3cM38()
	}
	if a.custodia.consumida.Load() != custodiaEntregadaO3cM38 {
		return nil, errConsumoO4aM38
	}
	if !entradaBaseExactaO4aM38(a) {
		fatalO3cM38()
	}
	primera, palabra := a.primera.Load(), a.custodia.observador.palabra.Load()
	ahora, fin, finBootstrap := a.ahoraCaso, a.finCaso, a.custodia.finBootstrap
	canon, capturaValida := normalizarControlRawO4aM38(a, primera)
	if !capturaValida || a.retornoCont < 0 ||
		!tiemposEntradaExactosO4aM38(ahora, fin, finBootstrap) ||
		!discriminanteObservacionValidoO3cM38(discriminanteObservacionO3cM38(primera)) ||
		!relacionPalabraRawO4aM38(primera, a.custodia.baselineSenal, palabra) ||
		(discriminanteObservacionO3cM38(primera) == observacionControlRawO3cM38) != (canon != controlRawVacioO4aM38) {
		fatalO3cM38()
	}
	r := &autoridadCausaO4aM38{origen: a}
	r.auto, r.sellos = r, sellarEntradaO4aM38(a, primera, palabra, canon, ahora, fin)
	r.estado.Store(uint32(causaA0RecibidoM38))
	if !a.custodia.consumida.CompareAndSwap(custodiaEntregadaO3cM38, custodiaRecibidaO4aM38) {
		return nil, errConsumoO4aM38
	}
	if !r.estado.CompareAndSwap(uint32(causaA0RecibidoM38), uint32(causaA1ValidadoM38)) {
		r.estado.Store(uint32(causaAFFatalM38))
		fatalO3cM38()
	}
	return r, nil
}
