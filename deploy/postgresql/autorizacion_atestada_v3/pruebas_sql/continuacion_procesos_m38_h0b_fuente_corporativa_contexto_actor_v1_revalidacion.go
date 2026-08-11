//go:build ignore && linux && amd64

package main

import (
	"errors"
	"os"
	"syscall"
	"time"
)

type causaPreContO3cM38 uint32

const (
	preContControlO3cM38 causaPreContO3cM38 = iota + 1
	preContSenalO3cM38
	preContBootstrapO3cM38
	preContPidfdO3cM38
	preContIdentidadO3cM38
)

const operacionContO3cM38 operacionGuardiaO3aM38 = operacionEscribirTicketO3bM38 + 1

var errRevalidacionO3cM38 = errors.New("revalidación O3c no acreditada")

// revalidacionO3cM38 es la única capacidad que P3 podrá consumir. El permiso
// permanece opaco y pendiente; P2 nunca ejecuta ni simula CONT.
type revalidacionO3cM38 struct {
	auto      *revalidacionO3cM38
	autoridad *autoridadContinuacionO3cM38
	permiso   permisoGuardiaO3aM38
}

func retirarPreContO3cM38(a *autoridadContinuacionO3cM38, causa causaPreContO3cM38) (*revalidacionO3cM38, error) {
	if a == nil || !a.es(continuacionC0RecibidoM38) || a.salida.auto != a.salida ||
		!transicionContinuacionO3cM38(a.estado, continuacionC7RetirandoM38) ||
		!a.salida.primera.CompareAndSwap(0, uint32(causa)) {
		fatalRevalidacionO3cM38(a)
	}
	a.estado = continuacionC7RetirandoM38
	return nil, errRevalidacionO3cM38
}

func fatalRevalidacionO3cM38(a *autoridadContinuacionO3cM38) {
	if a != nil {
		a.estado = continuacionCFFatalM38
	}
	fatalO3cM38()
}

func resolverRevalidacionO3cM38(a *autoridadContinuacionO3cM38, err error, causa causaPreContO3cM38) (*revalidacionO3cM38, error) {
	if errors.Is(err, errLeaseBarreraO3bM38) || a == nil || a.custodia == nil || a.autoridad == nil || !a.autoridad.poseeO3c() {
		fatalRevalidacionO3cM38(a)
	}
	return retirarPreContO3cM38(a, causa)
}

// sellosMemoriaValidosO3cM38 precede al primer syscall P2. Una capacidad
// sustituida nunca llega a ser leída, abierta ni sondeada.
func sellosMemoriaValidosO3cM38(a *autoridadContinuacionO3cM38) bool {
	if a == nil || a.custodia == nil || a.salida == nil || a.autoridad == nil {
		return false
	}
	c, s := a.custodia, a.sellos
	return c.cmd == s.cmd && c.cmd != nil && c.cmd.Process == s.proceso && c.cmd.Process != nil &&
		c.control == s.control && c.control != nil && c.controlFD == s.controlFD && c.controlFD != nil &&
		c.terminal == s.terminal && c.terminal != nil && c.lease == s.lease && c.lease != nil &&
		c.observador == s.observador && c.observador != nil && c.lease.registro == s.registro &&
		c.observador.registro == s.registro && s.registro != nil && s.registro.auto == s.registro &&
		c.lease.generacion == s.generacionLease && c.observador.generacion == s.generacionObservador &&
		s.registro.leases[c.lease] == s.generacionLease && s.registro.observadores[c.observador] == s.generacionObservador &&
		c.baselineSenal == s.baselineSenal && c.lease.estado.Load() == s.estadoLease && s.estadoLease == 3 &&
		c.observador.palabra.Load()&mascaraEstadoObservadorO3aM38 == s.estadoObservador && s.estadoObservador == 2 &&
		[3]int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco} == s.pidfd && c.tid == s.tid && c.ppid == s.ppid &&
		c.finBootstrap == s.finBootstrap && a.salida.auto == a.salida && a.salida.autoridad == a.autoridad &&
		a.salida.custodia == c && a.salida.identidad == a.identidad && a.autoridad.poseeO3c()
}

func autoridadSenalO3cM38(c *custodiaO3aM38) (bool, error) {
	if c == nil || c.lease == nil || c.observador == nil || c.observador.auto != c.observador || c.observador.registro == nil ||
		c.observador.registro.auto != c.observador.registro || c.observador.registro != c.lease.registro ||
		c.observador.registro.observadores[c.observador] != c.observador.generacion || c.observador.registro.tid != c.tid {
		return false, errLeaseBarreraO3bM38
	}
	var actual uint64
	var senal syscall.Signal
	var valido bool
	err := syscallLeaseO3cM38(c, func() error {
		actual, senal, valido = c.observador.observar()
		return nil
	})
	return valido && actual == c.baselineSenal && actual&mascaraEstadoObservadorO3aM38 == 2 && senal == 0, err
}

func syscallLeaseO3cM38(c *custodiaO3aM38, operacion func() error) error {
	// Cada invocación contiene exactamente un syscall y un permiso distinto.
	return operarConLeaseBarreraO3bM38(c, operacion)
}

func identidadEjecucionO3cM38(c *custodiaO3aM38) (bool, error) {
	var tid int
	if err := syscallLeaseO3cM38(c, func() error { tid = syscall.Gettid(); return nil }); err != nil {
		return false, err
	}
	var ppid int
	if err := syscallLeaseO3cM38(c, func() error { ppid = syscall.Getppid(); return nil }); err != nil {
		return false, err
	}
	var pdeathsig int32
	if err := syscallLeaseO3cM38(c, func() error {
		var errPrctl error
		pdeathsig, errPrctl = prctlO3aM38(2)
		return errPrctl
	}); err != nil {
		return false, err
	}
	return tid == c.tid && ppid == c.ppid && pdeathsig == int32(syscall.SIGTERM), nil
}

func pidfdEInventarioO3cM38(a *autoridadContinuacionO3cM38) (bool, bool, error) {
	c := a.custodia
	primario, errPrimario := identidadPidfdBarreraO3bM38(c, c.pidfdPrimario)
	reserva, errReserva := identidadPidfdBarreraO3bM38(c, c.pidfdReserva)
	opaco, errOpaco := identidadPidfdBarreraO3bM38(c, c.pidfdOpaco)
	vivoPrimario, errVivoPrimario := pidfdVivoBarreraO3bM38(c, c.pidfdPrimario)
	vivoReserva, errVivoReserva := pidfdVivoBarreraO3bM38(c, c.pidfdReserva)
	for _, err := range []error{errPrimario, errReserva, errOpaco, errVivoPrimario, errVivoReserva} {
		if errors.Is(err, errLeaseBarreraO3bM38) {
			return false, true, err
		}
	}
	fiablePrimario := errPrimario == nil && errVivoPrimario == nil && vivoPrimario
	fiableReserva := errReserva == nil && errVivoReserva == nil && vivoReserva
	if !fiablePrimario && !fiableReserva {
		return false, true, errRevalidacionO3cM38
	}
	if !fiablePrimario || !fiableReserva || errOpaco != nil ||
		primario.fdflags&syscall.FD_CLOEXEC == 0 || reserva.fdflags&syscall.FD_CLOEXEC == 0 || opaco.fdflags&syscall.FD_CLOEXEC == 0 ||
		!identidadFisicaO3aM38(primario, reserva) || !identidadFisicaO3aM38(primario, opaco) {
		return false, false, errRevalidacionO3cM38
	}
	actual, err := snapshotBarreraO3bM38(c)
	if err != nil {
		return false, true, err
	}
	if !snapshotsIgualesO3aM38(actual, c.lease.fisico) || c.controlFD == nil || c.terminal == nil || c.ticketEscritor != nil || c.ticketLector != nil {
		return false, true, errRevalidacionO3cM38
	}
	control, okControl := actual.mapa[int(c.controlFD.Fd())]
	terminal, okTerminal := actual.mapa[int(c.terminal.Fd())]
	if !okControl || !okTerminal || int(c.controlFD.Fd()) == int(c.terminal.Fd()) ||
		control.identidad.fdflags&syscall.FD_CLOEXEC == 0 || control.identidad.flags&syscall.O_ACCMODE != syscall.O_RDONLY ||
		control.identidad.flags&syscall.O_NONBLOCK == 0 || terminal.identidad.fdflags&syscall.FD_CLOEXEC == 0 ||
		terminal.identidad.modo&syscall.S_IFMT != syscall.S_IFREG || terminal.identidad.flags&syscall.O_ACCMODE != syscall.O_WRONLY {
		return false, true, errRevalidacionO3cM38
	}
	referencias := 0
	for _, huella := range actual.mapa {
		if identidadFisicaO3aM38(primario, huella.identidad) {
			referencias++
		}
	}
	if referencias != 3 || !sellosCustodiaValidosO3cM38(a, actual) {
		return false, true, errRevalidacionO3cM38
	}
	return true, false, nil
}

func sellosCustodiaValidosO3cM38(a *autoridadContinuacionO3cM38, actual snapshotFDO3aM38) bool {
	c, s := a.custodia, a.sellos
	if c.cmd != s.cmd || c.cmd.Process != s.proceso || c.control != s.control || c.controlFD != s.controlFD ||
		c.terminal != s.terminal || c.lease != s.lease || c.observador != s.observador ||
		[3]int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco} != s.pidfd || c.tid != s.tid || c.ppid != s.ppid ||
		c.finBootstrap != s.finBootstrap || !snapshotsIgualesO3aM38(actual, s.fisico) {
		return false
	}
	control, okControl := actual.mapa[int(c.controlFD.Fd())]
	terminal, okTerminal := actual.mapa[int(c.terminal.Fd())]
	if !okControl || !okTerminal || control != s.huellaControl || terminal != s.terminalH {
		return false
	}
	handleValido := false
	err := c.cmd.Process.WithHandle(func(handle uintptr) { handleValido = handle == uintptr(c.pidfdOpaco) })
	return err == nil && handleValido
}

func identidadProcesoFinalO3cM38(a *autoridadContinuacionO3cM38) (bool, error) {
	if a == nil || a.custodia == nil || a.custodia.cmd == nil || a.custodia.cmd.Process == nil {
		return false, errLeaseBarreraO3bM38
	}
	raw, err := leerStatStopO3bM38(a.custodia)
	if err != nil {
		return false, err
	}
	muestra, err := parsearStatO3bM38(raw)
	if err != nil {
		return false, err
	}
	c := a.custodia
	return c.cmd != nil && c.cmd.Process != nil && muestra == a.identidad && a.identidad.estado == 'T' &&
		a.identidad.pid == c.cmd.Process.Pid && a.identidad.ppid > 0 && a.identidad.pgid == c.cmd.Process.Pid &&
		a.identidad.sid > 0 && a.identidad.inicio > 0, nil
}

func segundaRondaO3cM38(a *autoridadContinuacionO3cM38) (bool, error) {
	if err := leerControlO3bM38(a.custodia); err != nil {
		return false, err
	}
	senalValida, err := autoridadSenalO3cM38(a.custodia)
	if err != nil {
		return false, err
	}
	if !senalValida {
		return false, nil
	}
	var ppid, tid int
	if err := syscallLeaseO3cM38(a.custodia, func() error { ppid = os.Getppid(); return nil }); err != nil {
		return false, err
	}
	if err := syscallLeaseO3cM38(a.custodia, func() error { tid = syscall.Gettid(); return nil }); err != nil {
		return false, err
	}
	return ppid == a.custodia.ppid && tid == a.custodia.tid, nil
}

func preasignacionO4aValidaO3cM38(a *autoridadContinuacionO3cM38) bool {
	return a.salida != nil && a.salida.auto == a.salida && a.salida.autoridad == a.autoridad &&
		a.salida.custodia == a.custodia && a.salida.identidad == a.identidad && a.salida.primera.Load() == 0 &&
		a.salida.ahoraCaso.IsZero() && a.salida.finCaso.IsZero() && a.salida.retornoCont == 0
}

// revalidarAntesContO3cM38 termina en C1 únicamente con la ronda completa y
// devuelve el permiso CONT pendiente dentro de una capacidad no copiable por API.
func revalidarAntesContO3cM38(a *autoridadContinuacionO3cM38) (*revalidacionO3cM38, error) {
	if a == nil || !a.es(continuacionC0RecibidoM38) || !sellosMemoriaValidosO3cM38(a) {
		fatalRevalidacionO3cM38(a)
	}
	c := a.custodia
	if err := leerControlO3bM38(c); err != nil {
		return resolverRevalidacionO3cM38(a, err, preContControlO3cM38)
	}
	senalValida, err := autoridadSenalO3cM38(c)
	if err != nil {
		return resolverRevalidacionO3cM38(a, err, preContSenalO3cM38)
	}
	if !senalValida {
		return retirarPreContO3cM38(a, preContSenalO3cM38)
	}
	valida, err := identidadEjecucionO3cM38(c)
	if err != nil || !valida {
		return resolverRevalidacionO3cM38(a, err, preContSenalO3cM38)
	}
	if c.finBootstrap.IsZero() || !time.Now().Before(c.finBootstrap) {
		return retirarPreContO3cM38(a, preContBootstrapO3cM38)
	}
	pidfdValido, fatal, err := pidfdEInventarioO3cM38(a)
	if fatal {
		fatalRevalidacionO3cM38(a)
	}
	if err != nil || !pidfdValido {
		return resolverRevalidacionO3cM38(a, err, preContPidfdO3cM38)
	}
	identidadValida, err := identidadProcesoFinalO3cM38(a)
	if err != nil || !identidadValida {
		return resolverRevalidacionO3cM38(a, err, preContIdentidadO3cM38)
	}
	segundaValida, err := segundaRondaO3cM38(a)
	if err != nil || !segundaValida {
		return resolverRevalidacionO3cM38(a, err, preContSenalO3cM38)
	}
	permiso, valido := c.lease.comenzar(operacionContO3cM38, 1, [2]int{c.pidfdPrimario, -1})
	if !valido {
		fatalRevalidacionO3cM38(a)
	}
	if !preasignacionO4aValidaO3cM38(a) {
		_ = c.lease.fatalPendiente(permiso)
		fatalRevalidacionO3cM38(a)
	}
	if !time.Now().Before(c.finBootstrap) {
		if !c.lease.consolidarCritico(permiso) {
			fatalRevalidacionO3cM38(a)
		}
		return retirarPreContO3cM38(a, preContBootstrapO3cM38)
	}
	if !transicionContinuacionO3cM38(a.estado, continuacionC1RevalidadoM38) {
		_ = c.lease.fatalPendiente(permiso)
		fatalRevalidacionO3cM38(a)
	}
	a.estado = continuacionC1RevalidadoM38
	r := &revalidacionO3cM38{autoridad: a, permiso: permiso}
	r.auto = r
	return r, nil
}
