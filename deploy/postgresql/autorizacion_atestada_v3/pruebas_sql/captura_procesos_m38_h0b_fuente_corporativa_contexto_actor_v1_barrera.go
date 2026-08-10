//go:build ignore && linux && amd64

package main

import (
	"errors"
	"io"
	"os"
	"syscall"
	"time"
)

type causaBarreraO3bM38 uint8

const (
	barreraO3bControlM38 causaBarreraO3bM38 = iota + 1
	barreraO3bSenalM38
	barreraO3bBootstrapM38
	barreraO3bPidfdM38
	barreraO3bInventarioM38
)

type falloBarreraO3bM38 struct{ causa causaBarreraO3bM38 }

const operacionBarreraO3bM38 operacionGuardiaO3aM38 = operacionCerrarPidfdO3aM38 + 1

var errLeaseBarreraO3bM38 = errors.New("lease de barrera O3b inválida")

func (f *falloBarreraO3bM38) Error() string { return "barrera O3b no acreditada" }

func causaDelFalloBarreraO3bM38(err error) causaBarreraO3bM38 {
	var fallo *falloBarreraO3bM38
	if errors.As(err, &fallo) {
		return fallo.causa
	}
	return 0
}

func retirarBarreraO3bM38(a *autoridadCapturaO3bM38, causa causaBarreraO3bM38) error {
	if a != nil && a.custodia != nil && transicionCapturaO3bM38(a.estado, capturaB7RetirandoM38) {
		a.estado = capturaB7RetirandoM38
	}
	return &falloBarreraO3bM38{causa: causa}
}

func fatalBarreraO3bM38(a *autoridadCapturaO3bM38) {
	if a != nil && a.custodia != nil && transicionCapturaO3bM38(a.estado, capturaBFFatalM38) {
		a.estado = capturaBFFatalM38
	}
	fatalO3aM38()
	select {}
}

func bootstrapDisponibleO3bM38(fin, ahora time.Time) bool {
	return !fin.IsZero() && !fin.Before(ahora.Add(reservaO3bO3cM38))
}

func dentroPresupuestoControlO3bM38(lecturas, total, interrupciones int) bool {
	return lecturas < 4 && total < 4096 && interrupciones <= 8
}

// operarConLeaseBarreraO3bM38 deja cada grupo lógico de syscalls entre un
// permiso opaco y su consolidación, sin alterar el inventario físico.
func operarConLeaseBarreraO3bM38(c *custodiaO3aM38, operacion func() error) error {
	if c == nil || !leaseTransferidaO3bM38(c.lease, c.tid) {
		return errLeaseBarreraO3bM38
	}
	permiso, valido := c.lease.comenzar(operacionBarreraO3bM38, 0, [2]int{-1, -1})
	if !valido {
		return errLeaseBarreraO3bM38
	}
	err := operacion()
	if !c.lease.consolidarCritico(permiso) {
		return errLeaseBarreraO3bM38
	}
	return err
}

func fcntlBarreraO3bM38(c *custodiaO3aM38, fd int, orden int) (uintptr, error) {
	var valor uintptr
	err := operarConLeaseBarreraO3bM38(c, func() error {
		var errno syscall.Errno
		valor, _, errno = syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(orden), 0)
		if errno != 0 {
			return errno
		}
		return nil
	})
	return valor, err
}

func fstatBarreraO3bM38(c *custodiaO3aM38, fd int) (syscall.Stat_t, error) {
	var estado syscall.Stat_t
	err := operarConLeaseBarreraO3bM38(c, func() error { return syscall.Fstat(fd, &estado) })
	return estado, err
}

func lseekBarreraO3bM38(c *custodiaO3aM38, fd int) (uintptr, bool, error) {
	var offset uintptr
	var errno syscall.Errno
	err := operarConLeaseBarreraO3bM38(c, func() error {
		offset, _, errno = syscall.Syscall(syscall.SYS_LSEEK, uintptr(fd), 0, io.SeekCurrent)
		if errno != 0 && errno != syscall.ESPIPE {
			return errno
		}
		return nil
	})
	return offset, errno == 0, err
}

func identidadPidfdBarreraO3bM38(c *custodiaO3aM38, fd int) (identidadFDO3aM38, error) {
	if fd < 0 {
		return identidadFDO3aM38{}, syscall.EBADF
	}
	estado, err := fstatBarreraO3bM38(c, fd)
	if err != nil {
		return identidadFDO3aM38{}, err
	}
	fdflags, err := fcntlBarreraO3bM38(c, fd, syscall.F_GETFD)
	if err != nil {
		return identidadFDO3aM38{}, err
	}
	flags, err := fcntlBarreraO3bM38(c, fd, syscall.F_GETFL)
	if err != nil {
		return identidadFDO3aM38{}, err
	}
	return identidadFDO3aM38{dev: uint64(estado.Dev), ino: estado.Ino, rdev: uint64(estado.Rdev), modo: estado.Mode,
		uid: estado.Uid, enlaces: estado.Nlink, tamano: estado.Size, offset: -1, flags: int(flags), fdflags: int(fdflags)}, nil
}

func pidfdVivoBarreraO3bM38(c *custodiaO3aM38, fd int) (bool, error) {
	if fd < 0 {
		return false, syscall.EBADF
	}
	sonda := pollfdO3aM38{fd: int32(fd), eventos: 1}
	var n uintptr
	err := operarConLeaseBarreraO3bM38(c, func() error {
		var errno syscall.Errno
		n, _, errno = syscall.Syscall(syscall.SYS_POLL, punteroPollO3aM38(&sonda), 1, 0)
		if errno != 0 {
			return errno
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	if n == 0 && sonda.retorno == 0 {
		return true, nil
	}
	if n == 1 && sonda.retorno == 1 {
		return false, nil
	}
	return false, errProcesoO3aM38
}

func snapshotBarreraO3bM38(c *custodiaO3aM38) (snapshotFDO3aM38, error) {
	var limite syscall.Rlimit
	if err := operarConLeaseBarreraO3bM38(c, func() error { return syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limite) }); err != nil ||
		limite.Cur < minFDDuplicadoM38 || limite.Cur > maxFDInspeccionM38 {
		return snapshotFDO3aM38{}, errInventarioO3aM38
	}
	actual := snapshotFDO3aM38{limite: limite.Cur, mapa: make(map[int]huellaFDO3aM38)}
	for fd := 0; fd < int(limite.Cur); fd++ {
		fdflags, err := fcntlBarreraO3bM38(c, fd, syscall.F_GETFD)
		if errors.Is(err, syscall.EBADF) {
			continue
		}
		if err != nil {
			return snapshotFDO3aM38{}, err
		}
		flags, err := fcntlBarreraO3bM38(c, fd, syscall.F_GETFL)
		if err != nil {
			return snapshotFDO3aM38{}, err
		}
		estado, err := fstatBarreraO3bM38(c, fd)
		if err != nil {
			return snapshotFDO3aM38{}, err
		}
		offset, offsetValido, err := lseekBarreraO3bM38(c, fd)
		if err != nil {
			return snapshotFDO3aM38{}, err
		}
		actual.mapa[fd] = huellaFDO3aM38{abierto: true, identidad: identidadFDO3aM38{
			dev: uint64(estado.Dev), ino: estado.Ino, rdev: uint64(estado.Rdev), modo: estado.Mode,
			uid: estado.Uid, enlaces: estado.Nlink, tamano: estado.Size, offset: int64(offset), offsetValido: offsetValido,
			flags: int(flags), fdflags: int(fdflags),
		}}
	}
	for fd := 0; fd <= 2; fd++ {
		if _, existe := actual.mapa[fd]; !existe {
			return snapshotFDO3aM38{}, errInventarioO3aM38
		}
	}
	return actual, nil
}

func leerControlO3bM38(c *custodiaO3aM38) error {
	var buffer [1024]byte
	lecturas, total, interrupciones := 0, 0, 0
	for dentroPresupuestoControlO3bM38(lecturas, total, interrupciones) {
		var n int
		err := operarConLeaseBarreraO3bM38(c, func() error {
			var errLectura error
			n, errLectura = syscall.Read(int(c.controlFD.Fd()), buffer[:])
			return errLectura
		})
		if errors.Is(err, errLeaseBarreraO3bM38) {
			return err
		}
		if errors.Is(err, syscall.EINTR) {
			interrupciones++
			if !dentroPresupuestoControlO3bM38(lecturas, total, interrupciones) {
				return errControlO3aM38
			}
			continue
		}
		if errors.Is(err, syscall.EAGAIN) {
			if controlIniciableO3aM38(c.control) {
				return nil
			}
			return errControlO3aM38
		}
		lecturas++
		if err != nil || n == 0 {
			if n == 0 {
				_, _, _ = c.control.consumir(nil, true)
			}
			c.enclavarCausaControl()
			return errControlO3aM38
		}
		total += n
		consumidos, resultado, fallo := c.control.consumir(buffer[:n], false)
		if resultado == controlPreinicioCausaEnclavadaM38 {
			c.enclavarCausaControl()
		}
		if fallo != nil || consumidos != n || resultado != controlPreinicioInicioPendienteM38 {
			return errControlO3aM38
		}
	}
	return errControlO3aM38
}

func revalidarAutoridadBarreraO3bM38(c *custodiaO3aM38) (causaBarreraO3bM38, error) {
	if c == nil || c.observador == nil {
		return barreraO3bSenalM38, nil
	}
	actual, senal, valido := c.observador.observar()
	if !valido || actual != c.baselineSenal || senal != 0 {
		return barreraO3bSenalM38, nil
	}
	var tid int
	if err := operarConLeaseBarreraO3bM38(c, func() error { tid = syscall.Gettid(); return nil }); err != nil {
		return 0, err
	}
	var ppid int
	if err := operarConLeaseBarreraO3bM38(c, func() error { ppid = os.Getppid(); return nil }); err != nil {
		return 0, err
	}
	if c.tid != tid || c.ppid != ppid {
		return barreraO3bSenalM38, nil
	}
	var subreaper int32
	if err := operarConLeaseBarreraO3bM38(c, func() error {
		var errPrctl error
		subreaper, errPrctl = prctlO3aM38(37)
		return errPrctl
	}); err != nil {
		return barreraO3bSenalM38, err
	}
	if subreaper != 1 {
		return barreraO3bSenalM38, nil
	}
	var pdeathsig int32
	if err := operarConLeaseBarreraO3bM38(c, func() error {
		var errPrctl error
		pdeathsig, errPrctl = prctlO3aM38(2)
		return errPrctl
	}); err != nil {
		return barreraO3bSenalM38, err
	}
	if pdeathsig != int32(syscall.SIGTERM) {
		return barreraO3bSenalM38, nil
	}
	if !bootstrapDisponibleO3bM38(c.finBootstrap, time.Now()) {
		return barreraO3bBootstrapM38, nil
	}
	return 0, nil
}

func acreditarPidfdBarreraO3bM38(c *custodiaO3aM38) (causaBarreraO3bM38, bool, error) {
	primario, errPrimario := identidadPidfdBarreraO3bM38(c, c.pidfdPrimario)
	reserva, errReserva := identidadPidfdBarreraO3bM38(c, c.pidfdReserva)
	opaco, errOpaco := identidadPidfdBarreraO3bM38(c, c.pidfdOpaco)
	vivoPrimario, errVivoPrimario := pidfdVivoBarreraO3bM38(c, c.pidfdPrimario)
	vivoReserva, errVivoReserva := pidfdVivoBarreraO3bM38(c, c.pidfdReserva)
	for _, err := range []error{errPrimario, errReserva, errOpaco, errVivoPrimario, errVivoReserva} {
		if errors.Is(err, errLeaseBarreraO3bM38) {
			return 0, false, err
		}
	}
	fiablePrimario := errPrimario == nil && errVivoPrimario == nil && vivoPrimario
	fiableReserva := errReserva == nil && errVivoReserva == nil && vivoReserva
	if !fiablePrimario && !fiableReserva {
		return barreraO3bPidfdM38, true, nil
	}
	if !fiablePrimario || !fiableReserva || errOpaco != nil ||
		primario.fdflags&syscall.FD_CLOEXEC == 0 || reserva.fdflags&syscall.FD_CLOEXEC == 0 || opaco.fdflags&syscall.FD_CLOEXEC == 0 ||
		!identidadFisicaO3aM38(primario, reserva) || !identidadFisicaO3aM38(primario, opaco) {
		return barreraO3bPidfdM38, false, nil
	}
	return 0, false, nil
}

func recursosExactosBarreraO3bM38(a *autoridadCapturaO3bM38, actual snapshotFDO3aM38) bool {
	c := a.custodia
	if c.controlFD == nil || c.terminal == nil || c.ticketEscritor == nil || !snapshotsIgualesO3aM38(actual, c.lease.fisico) {
		return false
	}
	controlFD, terminalFD, ticketFD := int(c.controlFD.Fd()), int(c.terminal.Fd()), int(c.ticketEscritor.Fd())
	if [3]int{controlFD, terminalFD, ticketFD} != a.fdsBarrera {
		return false
	}
	control, okControl := actual.mapa[controlFD]
	terminal, okTerminal := actual.mapa[terminalFD]
	ticket, okTicket := actual.mapa[ticketFD]
	if !okControl || !okTerminal || !okTicket || [3]huellaFDO3aM38{control, terminal, ticket} != a.huellasBarrera ||
		controlFD == terminalFD || controlFD == ticketFD || terminalFD == ticketFD {
		return false
	}
	if control.identidad.fdflags&syscall.FD_CLOEXEC == 0 || terminal.identidad.fdflags&syscall.FD_CLOEXEC == 0 || ticket.identidad.fdflags&syscall.FD_CLOEXEC == 0 ||
		control.identidad.flags&syscall.O_ACCMODE != syscall.O_RDONLY || control.identidad.flags&syscall.O_NONBLOCK == 0 ||
		terminal.identidad.modo&syscall.S_IFMT != syscall.S_IFREG || terminal.identidad.flags&syscall.O_ACCMODE != syscall.O_WRONLY ||
		ticket.identidad.modo&syscall.S_IFMT != syscall.S_IFIFO || ticket.identidad.flags&syscall.O_ACCMODE != syscall.O_WRONLY {
		return false
	}
	pidfd, existe := actual.mapa[c.pidfdPrimario]
	if !existe {
		return false
	}
	referencias := 0
	for _, huella := range actual.mapa {
		if identidadFisicaO3aM38(pidfd.identidad, huella.identidad) {
			referencias++
		}
	}
	return referencias == 3
}

func acreditarInventarioBarreraO3bM38(a *autoridadCapturaO3bM38) error {
	actual, err := snapshotBarreraO3bM38(a.custodia)
	if err != nil || !recursosExactosBarreraO3bM38(a, actual) {
		return errInventarioO3aM38
	}
	return nil
}

func resolverFalloOperacionBarreraO3bM38(a *autoridadCapturaO3bM38, err error, causa causaBarreraO3bM38) error {
	if causa == 0 {
		causa = barreraO3bInventarioM38
	}
	if errors.Is(err, errLeaseBarreraO3bM38) {
		fatalBarreraO3bM38(a)
		select {}
	}
	return retirarBarreraO3bM38(a, causa)
}

// ejecutarBarreraO3bM38 no escribe ni cierra el ticket. Tras inventario y
// pidfd ejecuta la relectura final; después del verde no queda ningún syscall P2.
func ejecutarBarreraO3bM38(a *autoridadCapturaO3bM38) error {
	if a == nil || a.estado != capturaB0RecibidoM38 || a.custodia == nil {
		return retirarBarreraO3bM38(a, barreraO3bInventarioM38)
	}
	if err := leerControlO3bM38(a.custodia); err != nil {
		return resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bControlM38)
	}
	if causa, err := revalidarAutoridadBarreraO3bM38(a.custodia); err != nil || causa != 0 {
		return resolverFalloOperacionBarreraO3bM38(a, err, causa)
	}
	if causa, fatal, err := acreditarPidfdBarreraO3bM38(a.custodia); err != nil || causa != 0 {
		if fatal || errors.Is(err, errLeaseBarreraO3bM38) {
			fatalBarreraO3bM38(a)
			select {}
		}
		return retirarBarreraO3bM38(a, causa)
	}
	if err := acreditarInventarioBarreraO3bM38(a); err != nil {
		return resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bInventarioM38)
	}
	if err := leerControlO3bM38(a.custodia); err != nil {
		return resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bControlM38)
	}
	if causa, err := revalidarAutoridadBarreraO3bM38(a.custodia); err != nil || causa != 0 {
		return resolverFalloOperacionBarreraO3bM38(a, err, causa)
	}
	if !transicionCapturaO3bM38(a.estado, capturaB1BarreraVerdeM38) {
		return retirarBarreraO3bM38(a, barreraO3bInventarioM38)
	}
	a.estado = capturaB1BarreraVerdeM38
	return nil
}
