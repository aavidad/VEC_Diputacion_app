//go:build ignore && linux && amd64

package main

import (
	"errors"
	"syscall"
	"time"
)

var errHandoffO3bM38 = errors.New("handoff O3b no acreditado")

// agregadoO3cM38 mantiene todos los recursos en una sola capacidad privada.
// P6 no añade consumidor ni efecto O3c.
type agregadoO3cM38 struct {
	estado    estadoCapturaO3bM38
	custodia  *custodiaO3aM38
	identidad muestraStatO3bM38
	primera   error
}

func prevalidarTransferenciaO3bM38(a *autoridadCapturaO3bM38, identidad *identidadProcesoO3bM38) (bool, bool) {
	if a == nil || identidad == nil || identidad.autoridad != a || a.estado != capturaB4IdentidadAcreditadaM38 ||
		a.custodia == nil || a.custodia.ticketEscritor != nil || a.custodia.ticketLector != nil ||
		a.custodia.cmd == nil || a.custodia.cmd.Process == nil || a.custodia.controlFD == nil || a.custodia.terminal == nil ||
		a.custodia.lease == nil || a.custodia.observador == nil || a.custodia.lease.registro != a.custodia.observador.registro {
		return false, false
	}
	c := a.custodia
	if c.lease.auto != c.lease || c.lease.registro == nil || c.lease.registro.auto != c.lease.registro ||
		c.lease.registro.leases[c.lease] != c.lease.generacion || c.lease.tid != c.tid ||
		c.lease.registro.tid != c.tid || c.lease.estado.Load() != 3 {
		return false, true
	}
	if c.observador.auto != c.observador || c.observador.registro == nil || c.observador.registro.auto != c.observador.registro ||
		c.observador.registro.observadores[c.observador] != c.observador.generacion || c.observador.registro.tid != c.tid ||
		c.baselineSenal&mascaraEstadoObservadorO3aM38 != 2 || c.observador.palabra.Load()&mascaraEstadoObservadorO3aM38 != 2 {
		return false, true
	}
	if c.observador.palabra.Load() != c.baselineSenal {
		return false, false
	}
	return identidad.pid == c.cmd.Process.Pid && identidad.ppid > 0 && identidad.pgid == identidad.pid &&
		identidad.sid == a.sidSupervisor && identidad.inicio > 0, false
}

func inventarioPostTicketO3bM38(a *autoridadCapturaO3bM38) error {
	if a == nil || a.custodia == nil || !leaseTransferidaO3bM38(a.custodia.lease, a.custodia.tid) {
		return errLeaseBarreraO3bM38
	}
	actual, err := snapshotBarreraO3bM38(a.custodia)
	if err != nil {
		return err
	}
	if !snapshotsIgualesO3aM38(actual, a.custodia.lease.fisico) ||
		a.custodia.controlFD == nil || a.custodia.terminal == nil || a.custodia.ticketEscritor != nil {
		return errHandoffO3bM38
	}
	controlFD, terminalFD := int(a.custodia.controlFD.Fd()), int(a.custodia.terminal.Fd())
	control, okControl := actual.mapa[controlFD]
	terminal, okTerminal := actual.mapa[terminalFD]
	if !okControl || !okTerminal || controlFD != a.fdsBarrera[0] || terminalFD != a.fdsBarrera[1] ||
		control != a.huellasBarrera[0] || terminal != a.huellasBarrera[1] || controlFD == terminalFD ||
		control.identidad.fdflags&syscall.FD_CLOEXEC == 0 || terminal.identidad.fdflags&syscall.FD_CLOEXEC == 0 ||
		control.identidad.flags&syscall.O_ACCMODE != syscall.O_RDONLY || control.identidad.flags&syscall.O_NONBLOCK == 0 ||
		terminal.identidad.modo&syscall.S_IFMT != syscall.S_IFREG || terminal.identidad.flags&syscall.O_ACCMODE != syscall.O_WRONLY {
		return errHandoffO3bM38
	}
	pidfd, existe := actual.mapa[a.custodia.pidfdPrimario]
	if !existe {
		return errHandoffO3bM38
	}
	referencias := 0
	for _, huella := range actual.mapa {
		if identidadFisicaO3aM38(pidfd.identidad, huella.identidad) {
			referencias++
		}
	}
	if referencias != 3 {
		return errHandoffO3bM38
	}
	return nil
}

func transferirObservadorCapturadoO3bM38(o *observadorSenalO3aM38, baseline uint64) (uint64, bool) {
	if o == nil || baseline&mascaraEstadoObservadorO3aM38 != 2 {
		return 0, false
	}
	nuevo := baseline - 2 + 1
	return nuevo, o.palabra.CompareAndSwap(baseline, nuevo)
}

func transferirLeaseCapturadaO3bM38(l *leaseGuardiaO3aM38) bool {
	return l != nil && l.estado.CompareAndSwap(3, 1)
}

func consolidarHandoffO3bM38(a *autoridadCapturaO3bM38, identidad *identidadProcesoO3bM38, salida *agregadoO3cM38) *agregadoO3cM38 {
	if a == nil || identidad == nil || salida == nil || !transicionCapturaO3bM38(a.estado, capturaB4TTransfiriendoM38) {
		fatalBarreraO3bM38(a)
		select {}
	}
	a.estado = capturaB4TTransfiriendoM38
	nuevoBaseline, ok := transferirObservadorCapturadoO3bM38(a.custodia.observador, a.custodia.baselineSenal)
	if !ok || !transferirLeaseCapturadaO3bM38(a.custodia.lease) {
		fatalBarreraO3bM38(a)
		select {}
	}
	a.custodia.baselineSenal = nuevoBaseline
	salida.estado = capturaB5CapturadoM38
	salida.custodia = a.custodia
	salida.identidad = muestraStatO3bM38{pid: identidad.pid, estado: 'T', ppid: identidad.ppid,
		pgid: identidad.pgid, sid: identidad.sid, inicio: identidad.inicio}
	salida.primera = nil
	a.estado = capturaB5CapturadoM38
	a.custodia = nil
	*identidad = identidadProcesoO3bM38{}
	return salida
}

func revalidarHandoffO3bM38(a *autoridadCapturaO3bM38) error {
	if err := leerControlO3bM38(a.custodia); err != nil {
		return err
	}
	actual, senal, valido := a.custodia.observador.observar()
	if !valido || actual != a.custodia.baselineSenal || senal != 0 {
		return errHandoffO3bM38
	}
	var tid int
	if err := operarConLeaseBarreraO3bM38(a.custodia, func() error { tid = syscall.Gettid(); return nil }); err != nil {
		return err
	}
	var ppid int
	if err := operarConLeaseBarreraO3bM38(a.custodia, func() error { ppid = syscall.Getppid(); return nil }); err != nil {
		return err
	}
	if tid != a.custodia.tid || ppid != a.custodia.ppid {
		return errHandoffO3bM38
	}
	causa, fatal, err := acreditarPidfdBarreraO3bM38(a.custodia)
	if fatal || errors.Is(err, errLeaseBarreraO3bM38) {
		fatalBarreraO3bM38(a)
		select {}
	}
	if err != nil || causa != 0 {
		return errHandoffO3bM38
	}
	if err := inventarioPostTicketO3bM38(a); err != nil {
		return err
	}
	if !time.Now().Before(a.custodia.finBootstrap) {
		return errHandoffO3bM38
	}
	return nil
}

// transferirCapturadoO3bM38 ejecuta todos los syscalls antes del último verde.
// Desde la prevalidación conjunta hasta B5 sólo hay transiciones en memoria.
func transferirCapturadoO3bM38(origen **identidadProcesoO3bM38) (*agregadoO3cM38, error) {
	if origen == nil || *origen == nil {
		return nil, errHandoffO3bM38
	}
	identidad := *origen
	*origen = nil
	a := identidad.autoridad
	if a == nil || a.estado != capturaB4IdentidadAcreditadaM38 || a.custodia == nil {
		return nil, errHandoffO3bM38
	}
	if a.custodia.lease == nil || a.custodia.observador == nil || a.custodia.lease.registro == nil ||
		a.custodia.lease.registro != a.custodia.observador.registro {
		fatalBarreraO3bM38(a)
		select {}
	}
	// La reserva ocurre antes de la última ronda; no se asigna entre verde y B5.
	salida := &agregadoO3cM38{}
	if err := revalidarHandoffO3bM38(a); err != nil {
		if errors.Is(err, errLeaseBarreraO3bM38) {
			fatalBarreraO3bM38(a)
			select {}
		}
		return nil, retirarBarreraO3bM38(a, barreraO3bInventarioM38)
	}
	valida, fatal := prevalidarTransferenciaO3bM38(a, identidad)
	if fatal {
		fatalBarreraO3bM38(a)
		select {}
	}
	if !valida {
		return nil, retirarBarreraO3bM38(a, barreraO3bInventarioM38)
	}
	return consolidarHandoffO3bM38(a, identidad, salida), nil
}
