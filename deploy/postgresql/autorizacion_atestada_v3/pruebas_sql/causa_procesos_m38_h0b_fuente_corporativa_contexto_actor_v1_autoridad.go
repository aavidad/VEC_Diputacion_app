//go:build ignore && linux && amd64

package main

import (
	"errors"
	"maps"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
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

func entradaExactaO4aM38(a *agregadoO4aM38) bool {
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
	palabra := o.palabra.Load()
	p := [3]int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco}
	if l.auto != l || o.auto != o || r == nil || r.auto != r || o.registro != r ||
		r.leases[l] != l.generacion || r.observadores[o] != o.generacion || l.tid != c.tid ||
		r.tid != c.tid || l.estado.Load() != 3 || palabra != c.baselineSenal ||
		palabra&mascaraEstadoObservadorO3aM38 != 2 || syscall.Signal(uint8(palabra>>2)) != 0 ||
		p[0] < 0 || p[1] < 0 || p[2] < 0 || p[0] == p[1] || p[0] == p[2] || p[1] == p[2] ||
		a.identidad.estado != 'T' || a.identidad.pid != c.cmd.Process.Pid || a.identidad.ppid <= 0 ||
		a.identidad.pgid != c.cmd.Process.Pid || a.identidad.sid <= 0 || a.identidad.inicio == 0 ||
		!discriminanteObservacionValidoO3cM38(discriminanteObservacionO3cM38(a.primera.Load())) || a.retornoCont < 0 {
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

func sellarEntradaO4aM38(a *agregadoO4aM38) sellosO4aM38 {
	c, l, o := a.custodia, a.custodia.lease, a.custodia.observador
	return sellosO4aM38{
		autoridad: a.autoridad, autoridadArranque: c.autoridad, custodia: c, lease: l, observador: o, registro: l.registro,
		control: c.control, controlFD: c.controlFD, terminal: c.terminal, cmd: c.cmd, proceso: c.cmd.Process,
		generacionLease: l.generacion, generacionObservador: o.generacion, tid: c.tid, ppid: c.ppid,
		baselineSenal: c.baselineSenal, pidfd: [3]int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco},
		identidad: a.identidad, primera: a.primera.Load(), retornoCont: a.retornoCont,
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
	if !entradaExactaO4aM38(a) {
		fatalO3cM38()
	}
	r := &autoridadCausaO4aM38{origen: a}
	r.auto, r.sellos = r, sellarEntradaO4aM38(a)
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
