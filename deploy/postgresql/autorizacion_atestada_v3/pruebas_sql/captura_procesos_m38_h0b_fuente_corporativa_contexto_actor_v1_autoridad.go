//go:build ignore && linux && amd64

package main

import (
	"errors"
	"os"
	"syscall"
)

type estadoCapturaO3bM38 uint8

const (
	capturaB0RecibidoM38 estadoCapturaO3bM38 = iota
	capturaB1BarreraVerdeM38
	capturaB2TicketCerradoM38
	capturaB3StopObservadoM38
	capturaB4IdentidadAcreditadaM38
	capturaB4TTransfiriendoM38
	capturaB5CapturadoM38
	capturaB7RetirandoM38
	capturaB8RetiradoM38
	capturaBFFatalM38
)

var (
	errEntradaO3bM38   = errors.New("entrada O3b inválida")
	errAutoridadO3bM38 = errors.New("autoridad O3b inválida")
)

// autoridadCapturaO3bM38 mantiene opaca la custodia consumida. P1 crea B0 o
// clasifica el rechazo como B7/B8; las fases posteriores poseen sus efectos.
type autoridadCapturaO3bM38 struct {
	estado         estadoCapturaO3bM38
	custodia       *custodiaO3aM38
	fdsBarrera     [3]int
	huellasBarrera [3]huellaFDO3aM38
}

func (a *autoridadCapturaO3bM38) es(estado estadoCapturaO3bM38) bool {
	return a != nil && a.estado == estado && (estado == capturaB8RetiradoM38 || a.custodia != nil)
}

func transicionCapturaO3bM38(desde, hacia estadoCapturaO3bM38) bool {
	switch desde {
	case capturaB0RecibidoM38:
		return hacia == capturaB1BarreraVerdeM38 || hacia == capturaB7RetirandoM38 || hacia == capturaBFFatalM38
	case capturaB1BarreraVerdeM38:
		return hacia == capturaB2TicketCerradoM38 || hacia == capturaB7RetirandoM38 || hacia == capturaBFFatalM38
	case capturaB2TicketCerradoM38:
		return hacia == capturaB3StopObservadoM38 || hacia == capturaB7RetirandoM38 || hacia == capturaBFFatalM38
	case capturaB3StopObservadoM38:
		return hacia == capturaB4IdentidadAcreditadaM38 || hacia == capturaB7RetirandoM38 || hacia == capturaBFFatalM38
	case capturaB4IdentidadAcreditadaM38:
		return hacia == capturaB4TTransfiriendoM38 || hacia == capturaB7RetirandoM38 || hacia == capturaBFFatalM38
	case capturaB4TTransfiriendoM38:
		return hacia == capturaB5CapturadoM38 || hacia == capturaBFFatalM38
	case capturaB7RetirandoM38:
		return hacia == capturaB8RetiradoM38 || hacia == capturaBFFatalM38
	default:
		return false
	}
}

func leaseTransferidaO3bM38(l *leaseGuardiaO3aM38, tid int) bool {
	return l != nil && l.auto == l && l.registro != nil && l.registro.auto == l.registro &&
		l.registro.leases[l] == l.generacion && l.registro.tid == tid && l.tid == tid &&
		l.estado.Load() == 3
}

func observadorTransferidoO3bM38(o *observadorSenalO3aM38, baseline uint64, tid int) bool {
	if o == nil || o.auto != o || o.registro == nil || o.registro.auto != o.registro ||
		o.registro.observadores[o] != o.generacion || o.registro.tid != tid ||
		baseline&mascaraEstadoObservadorO3aM38 != 2 {
		return false
	}
	actual, senal, valido := o.observar()
	return valido && actual == baseline && senal == 0
}

func entradaIntegraO3bM38(c *custodiaO3aM38) bool {
	if c == nil || c.autoridad == nil || !c.autoridad.es(arranqueA6EntregadoM38) ||
		c.tid != syscall.Gettid() || c.ppid != os.Getppid() ||
		!leaseTransferidaO3bM38(c.lease, c.tid) ||
		!observadorTransferidoO3bM38(c.observador, c.baselineSenal, c.tid) {
		return false
	}
	if c.control == nil || c.control.fase != controlPreinicioS3M38 ||
		c.control.lector == nil || c.control.lector.clase != "CONTROL" || c.control.lector.limite != 1024 ||
		c.control.lector.estado != lectorAbiertoVacioM38 || c.control.lector.err != nil || !lectorLimpioM38(c.control.lector) ||
		c.control.causa != (causaPreinicioM38{}) || c.control.fallo != nil {
		return false
	}
	if c.lease.registro != c.observador.registro {
		return false
	}
	if c.cmd == nil || c.cmd.Process == nil || c.cmd.Process.Pid <= 0 ||
		c.pidfdPrimario < 0 || c.pidfdReserva < 0 || c.pidfdOpaco < 0 ||
		c.pidfdPrimario == c.pidfdReserva || c.pidfdPrimario == c.pidfdOpaco || c.pidfdReserva == c.pidfdOpaco ||
		c.controlFD == nil || c.terminal == nil || c.ticketEscritor == nil || c.ticketLector != nil {
		return false
	}
	controlFD, terminalFD, ticketFD := c.controlFD.Fd(), c.terminal.Fd(), c.ticketEscritor.Fd()
	if controlFD == ^uintptr(0) || terminalFD == ^uintptr(0) || ticketFD == ^uintptr(0) ||
		controlFD == terminalFD || controlFD == ticketFD || terminalFD == ticketFD ||
		fdAliasPidfdO3bM38(controlFD, c) || fdAliasPidfdO3bM38(terminalFD, c) || fdAliasPidfdO3bM38(ticketFD, c) {
		return false
	}
	return !c.finBootstrap.IsZero() && c.snapshot.mapa != nil && c.baseline.mapa != nil &&
		c.formaRaiz.identidad != (identidadFDO3aM38{}) &&
		c.formaRunner.identidad != (identidadFDO3aM38{}) && c.formaRunner.sha256 != ([32]byte{})
}

func fdAliasPidfdO3bM38(fd uintptr, c *custodiaO3aM38) bool {
	return fd == uintptr(c.pidfdPrimario) || fd == uintptr(c.pidfdReserva) || fd == uintptr(c.pidfdOpaco)
}

// consumirAutoridadO3bM38 consume por CAS antes de observar la custodia y
// anula el alias del llamador. No realiza syscalls ni transfiere autoridades.
func consumirAutoridadO3bM38(entrada **agregadoO3aM38) (*autoridadCapturaO3bM38, error) {
	if entrada == nil || *entrada == nil {
		return nil, errEntradaO3bM38
	}
	agregado := *entrada
	*entrada = nil
	if agregado.custodia == nil {
		return &autoridadCapturaO3bM38{estado: capturaB8RetiradoM38}, errEntradaO3bM38
	}
	if !agregado.custodia.consumida.CompareAndSwap(1, 2) {
		return &autoridadCapturaO3bM38{estado: capturaB8RetiradoM38}, errEntradaO3bM38
	}
	if !entradaIntegraO3bM38(agregado.custodia) {
		return &autoridadCapturaO3bM38{estado: capturaB7RetirandoM38, custodia: agregado.custodia}, errAutoridadO3bM38
	}
	fds := [3]int{int(agregado.custodia.controlFD.Fd()), int(agregado.custodia.terminal.Fd()), int(agregado.custodia.ticketEscritor.Fd())}
	var huellas [3]huellaFDO3aM38
	for i, fd := range fds {
		huella, existe := agregado.custodia.lease.fisico.mapa[fd]
		if existe && huella.abierto {
			huellas[i] = huella
		}
	}
	return &autoridadCapturaO3bM38{estado: capturaB0RecibidoM38, custodia: agregado.custodia, fdsBarrera: fds, huellasBarrera: huellas}, nil
}
