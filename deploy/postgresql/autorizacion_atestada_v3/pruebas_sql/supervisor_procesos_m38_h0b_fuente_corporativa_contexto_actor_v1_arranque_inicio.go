//go:build ignore && linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

type resultadoBarreraO3aM38 uint8

const (
	barreraVerdeO3aM38 resultadoBarreraO3aM38 = iota + 1
	barreraAplazadaO3aM38
	barreraRetiradaO3aM38
)

func punteroPollO3aM38(p *pollfdO3aM38) uintptr { return uintptr(unsafe.Pointer(p)) }

type pollfdO3aM38 struct {
	fd               int32
	eventos, retorno int16
}

func controlIniciableO3aM38(c *controladorPreinicioM38) bool {
	return c != nil && c.fase == controlPreinicioS3M38 && c.causa == (causaPreinicioM38{}) &&
		c.fallo == nil && c.lector != nil && c.lector.clase == "CONTROL" && c.lector.limite == 1024 &&
		c.lector.estado == lectorAbiertoVacioM38 && c.lector.err == nil && lectorLimpioM38(c.lector)
}

func entornoEstableO3aM38(c *custodiaO3aM38, reserva time.Duration) error {
	if c == nil || !c.lease.valido() || c.tid != syscall.Gettid() || c.ppid != os.Getppid() {
		return errEntradaO3aM38
	}
	contador, _, valido := c.observador.observar()
	if !valido || contador != c.baselineSenal {
		return errSenalPendienteO3aM38
	}
	if sub, err := prctlO3aM38(37); err != nil || sub != 1 {
		return errSubreaperO3aM38
	}
	if muerte, err := prctlO3aM38(2); err != nil || muerte != int32(syscall.SIGTERM) {
		return errPdeathsigO3aM38
	}
	if time.Until(c.finBootstrap) < reserva {
		return errPlazoO3aM38
	}
	return nil
}

func leerControlBarreraO3aM38(c *custodiaO3aM38, permiteAplazar bool) (resultadoBarreraO3aM38, error) {
	var buffer [1024]byte
	lecturas, total, interrupciones := 0, 0, 0
	for lecturas < 4 && total < 4096 {
		n, err := syscall.Read(int(c.controlFD.Fd()), buffer[:])
		if errors.Is(err, syscall.EINTR) {
			interrupciones++
			if interrupciones > 8 {
				return barreraRetiradaO3aM38, errControlO3aM38
			}
			if fallo := entornoEstableO3aM38(c, reservaO3bO3cM38); fallo != nil {
				return barreraRetiradaO3aM38, fallo
			}
			continue
		}
		if errors.Is(err, syscall.EAGAIN) {
			if controlIniciableO3aM38(c.control) {
				return barreraVerdeO3aM38, nil
			}
			if permiteAplazar && c.control != nil && c.control.lector != nil && c.control.lector.estado == lectorAbiertoParcialM38 {
				return barreraAplazadaO3aM38, errAplazamientoO3aM38
			}
			return barreraRetiradaO3aM38, errControlO3aM38
		}
		if err != nil {
			return barreraRetiradaO3aM38, err
		}
		lecturas++
		if n == 0 {
			_, resultado, fallo := c.control.consumir(nil, true)
			c.enclavarCausaControl()
			if fallo != nil || resultado != controlPreinicioCausaEnclavadaM38 {
				return barreraRetiradaO3aM38, errControlO3aM38
			}
			return barreraRetiradaO3aM38, errControlO3aM38
		}
		total += n
		consumidos, resultado, fallo := c.control.consumir(buffer[:n], false)
		if resultado == controlPreinicioCausaEnclavadaM38 {
			c.enclavarCausaControl()
		}
		if fallo != nil || consumidos != n || resultado == controlPreinicioCausaEnclavadaM38 {
			return barreraRetiradaO3aM38, errControlO3aM38
		}
		if resultado == controlPreinicioNecesitaDatosM38 || c.control.lector.estado == lectorAbiertoParcialM38 {
			if permiteAplazar {
				return barreraAplazadaO3aM38, errAplazamientoO3aM38
			}
			return barreraRetiradaO3aM38, errControlO3aM38
		}
	}
	if permiteAplazar {
		return barreraAplazadaO3aM38, errAplazamientoO3aM38
	}
	return barreraRetiradaO3aM38, errControlO3aM38
}

func inventarioPreStartO3aM38(c *custodiaO3aM38) error {
	actual, err := snapshotActualO3aM38()
	if err != nil || !todosCLOEXECO3aM38(actual) || !snapshotsIgualesO3aM38(actual, c.snapshot) {
		return errInventarioO3aM38
	}
	return nil
}

func barreraAntesStartO3aM38(c *custodiaO3aM38) (resultadoBarreraO3aM38, error) {
	resultado, err := leerControlBarreraO3aM38(c, true)
	if resultado != barreraVerdeO3aM38 {
		return resultado, err
	}
	if err = entornoEstableO3aM38(c, 4*time.Second); err != nil {
		return barreraRetiradaO3aM38, err
	}
	if err = inventarioPreStartO3aM38(c); err != nil {
		return barreraRetiradaO3aM38, err
	}
	resultado, err = leerControlBarreraO3aM38(c, true)
	if resultado != barreraVerdeO3aM38 {
		return resultado, err
	}
	if err = entornoEstableO3aM38(c, 4*time.Second); err != nil {
		return barreraRetiradaO3aM38, err
	}
	return barreraVerdeO3aM38, nil
}

func pidfdVivoO3aM38(fd int) (bool, error) {
	if fd < 0 {
		return false, errProcesoO3aM38
	}
	p := pollfdO3aM38{fd: int32(fd), eventos: 1}
	n, _, errno := syscall.Syscall(syscall.SYS_POLL, punteroPollO3aM38(&p), 1, 0)
	if errno != 0 {
		return false, errno
	}
	if n == 0 && p.retorno == 0 {
		return true, nil
	}
	if n == 1 && p.retorno == 1 {
		return false, nil
	}
	return false, errProcesoO3aM38
}

func identidadPidfdO3aM38(fd int) (identidadFDO3aM38, error) {
	if fd < 0 {
		return identidadFDO3aM38{}, syscall.EBADF
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return identidadFDO3aM38{}, err
	}
	fdflags, _, e := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0)
	if e != 0 {
		return identidadFDO3aM38{}, e
	}
	flags, _, e := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFL, 0)
	if e != 0 {
		return identidadFDO3aM38{}, e
	}
	return identidadFDO3aM38{dev: uint64(st.Dev), ino: st.Ino, rdev: uint64(st.Rdev), modo: st.Mode,
		uid: st.Uid, enlaces: st.Nlink, tamano: st.Size, offset: -1, flags: int(flags), fdflags: int(fdflags)}, nil
}

func deltaPidfdO3aM38(antes, despues snapshotFDO3aM38, primario, reserva int) (int, bool) {
	esperados := 2
	if reserva >= 0 {
		esperados = 3
	}
	if antes.limite != despues.limite || len(despues.mapa) != len(antes.mapa)+esperados || reserva >= 0 && primario == reserva {
		return -1, false
	}
	identidadPrimaria, err := identidadPidfdO3aM38(primario)
	if err != nil || identidadPrimaria.fdflags&syscall.FD_CLOEXEC == 0 {
		return -1, false
	}
	nuevos, opaco := 0, -1
	for fd, huella := range despues.mapa {
		if anterior, existe := antes.mapa[fd]; existe {
			if huella != anterior {
				return -1, false
			}
			continue
		}
		nuevos++
		if fd != primario && fd != reserva {
			if opaco >= 0 {
				return -1, false
			}
			opaco = fd
		}
		if huella.identidad.fdflags&syscall.FD_CLOEXEC == 0 ||
			!identidadFisicaO3aM38(huella.identidad, identidadPrimaria) {
			return -1, false
		}
	}
	return opaco, nuevos == esperados && opaco >= 0
}

func esperarConLeaseO3aM38(c *custodiaO3aM38) error {
	if c == nil || c.cmd == nil || c.pidfdOpaco < 0 {
		fatalO3aM38()
	}
	p, ok := c.lease.comenzar(operacionWaitO3aM38, 1, [2]int{c.pidfdOpaco, -1})
	if !ok {
		fatalO3aM38()
	}
	errWait := c.cmd.Wait()
	actual, err := snapshotActualO3aM38()
	aplicada, consolidada := consolidarOperacionFisicaO3aM38(c.lease, p, actual)
	if err != nil || !aplicada || !consolidada {
		fatalO3aM38()
	}
	c.pidfdOpaco = -1
	return errWait
}

func cerrarPidfdConLeaseO3aM38(l *leaseGuardiaO3aM38, fd int) (bool, error) {
	if l == nil || fd < 0 {
		return false, errFormaFDO3aM38
	}
	p, ok := l.comenzar(operacionCerrarPidfdO3aM38, 1, [2]int{fd, -1})
	if !ok {
		return false, errAutoridadO3aM38
	}
	errCierre := syscall.Close(fd)
	actual, err := snapshotActualO3aM38()
	aplicada, consolidada := consolidarOperacionFisicaO3aM38(l, p, actual)
	if err != nil || !consolidada || errCierre == nil && !aplicada {
		fatalO3aM38()
	}
	return aplicada, errCierre
}

func revalidarDestinadosO3aM38(c *custodiaO3aM38) error {
	if len(c.destinados) != 9 || len(c.huellasDestinadas) != 9 {
		return errFormaFDO3aM38
	}
	for i, f := range c.destinados {
		forma, err := formaArchivoO3aM38(f, i >= 1 && i <= 3)
		if err != nil || forma != c.huellasDestinadas[i] || forma.fdflags&syscall.FD_CLOEXEC == 0 {
			return errFormaFDO3aM38
		}
	}
	return nil
}

func cerrarDestinadosConLeaseO3aM38(c *custodiaO3aM38) error {
	var fallos []error
	for i, f := range c.destinados {
		cerrado, err := cerrarUnoConLeaseO3aM38(c.lease, f, operacionCerrarDestinosO3aM38)
		fallos = append(fallos, err)
		if cerrado {
			c.destinados[i] = nil
		}
	}
	return errors.Join(fallos...)
}

func barreraDespuesStartO3aM38(c *custodiaO3aM38) error {
	resultado, err := leerControlBarreraO3aM38(c, false)
	if resultado != barreraVerdeO3aM38 || err != nil {
		return errControlO3aM38
	}
	if err = entornoEstableO3aM38(c, reservaO3bO3cM38); err != nil {
		return err
	}
	vivoPrimario, errPrimario := pidfdVivoO3aM38(c.pidfdPrimario)
	vivoReserva, errReserva := pidfdVivoO3aM38(c.pidfdReserva)
	if errPrimario != nil && errReserva != nil {
		fatalO3aM38()
	}
	if errPrimario != nil || errReserva != nil || !vivoPrimario || !vivoReserva {
		return errProcesoO3aM38
	}
	actual, err := snapshotActualO3aM38()
	if err != nil || !todosCLOEXECO3aM38(actual) || !snapshotsIgualesO3aM38(c.snapshot, actual) {
		return errInventarioO3aM38
	}
	if err = revalidarDestinadosO3aM38(c); err != nil {
		return err
	}
	if err = cerrarDestinadosConLeaseO3aM38(c); err != nil {
		return err
	}
	c.destinados = nil
	if err = entornoEstableO3aM38(c, reservaO3bO3cM38); err != nil {
		return err
	}
	return nil
}

func terminalAntesO3aM38(pidfd int, fin time.Time) bool {
	for time.Now().Before(fin) {
		vivo, err := pidfdVivoO3aM38(pidfd)
		if err == nil && !vivo {
			return true
		}
		if err != nil {
			return false
		}
		time.Sleep(time.Millisecond)
	}
	return false
}

func retirarConHijoO3aM38(c *custodiaO3aM38, primera error, fin time.Time) resultadoArranqueO3aM38 {
	desde := arranqueA4ProvisionalM38
	if c.autoridad.es(arranqueA5PidfdTresM38) {
		desde = arranqueA5PidfdTresM38
	} else if !c.autoridad.es(desde) {
		fatalO3aM38()
	}
	if c.autoridad.mover(desde, arranqueA8RetirandoConHijoM38) != nil {
		fatalO3aM38()
	}
	c.primera = primera
	cerrado, errCierre := cerrarUnoConLeaseO3aM38(c.lease, c.ticketEscritor, operacionCerrarTicketO3aM38)
	if c.ticketEscritor == nil || !cerrado {
		fatalO3aM38()
	}
	if errCierre != nil {
		c.secundarios = append(c.secundarios, errCierre)
	}
	c.ticketEscritor = nil
	if err := cerrarDestinadosConLeaseO3aM38(c); err != nil {
		c.secundarios = append(c.secundarios, err)
	}
	c.destinados = nil
	if c.ticketLector != nil {
		cerrado, err := cerrarUnoConLeaseO3aM38(c.lease, c.ticketLector, operacionCerrarTicketO3aM38)
		if !cerrado {
			fatalO3aM38()
		}
		c.secundarios = append(c.secundarios, err)
		c.ticketLector = nil
	}
	fiable := c.pidfdPrimario
	if vivo, err := pidfdVivoO3aM38(fiable); err != nil {
		fiable = c.pidfdReserva
	} else if !vivo {
		fiable = c.pidfdPrimario
	}
	if !terminalAntesO3aM38(fiable, fin) {
		if err := enviarPidfdIndividualO3aM38(fiable, syscall.SIGKILL); err != nil || !terminalAntesO3aM38(fiable, fin) {
			fatalO3aM38()
		}
	}
	if c.cmd == nil || c.cmd.Process == nil {
		fatalO3aM38()
	}
	if err := esperarConLeaseO3aM38(c); err != nil {
		var salida *execExitErrorO3aM38
		if !errors.As(err, &salida) {
			fatalO3aM38()
		}
	}
	if err := cerrarPidfdsPoseidosO3aM38(c.lease, c.pidfdPrimario, c.pidfdReserva); err != nil {
		fatalO3aM38()
	}
	c.pidfdPrimario, c.pidfdReserva = -1, -1
	for {
		var estado syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &estado, syscall.WNOHANG, nil)
		if errors.Is(err, syscall.ECHILD) {
			break
		}
		if err != nil || pid == 0 || time.Now().After(fin) {
			fatalO3aM38()
		}
	}
	if err := c.autoridad.mover(arranqueA8RetirandoConHijoM38, arranqueA9RetiradoConHijoM38); err != nil {
		fatalO3aM38()
	}
	return resultadoArranqueO3aM38{clase: resultadoRetiradoO3aM38, retirada: &retiradaO3aM38{
		origen: retiradaConHijoO3aM38, primera: primera, controlFD: c.controlFD,
		terminal: c.terminal, lease: c.lease, observador: c.observador, causa: c.primeraCausa,
		secundarios: c.secundarios,
	}}
}

// execExitErrorO3aM38 evita que G6c acepte errores ajenos a la salida real.
type execExitErrorO3aM38 = exec.ExitError

func retirarPreparadoO3aM38(c *custodiaO3aM38, primera error) resultadoArranqueO3aM38 {
	if err := cerrarDestinadosConLeaseO3aM38(c); err != nil {
		c.secundarios = append(c.secundarios, err)
	}
	if c.ticketLector != nil {
		cerrado, err := cerrarUnoConLeaseO3aM38(c.lease, c.ticketLector, operacionCerrarTicketO3aM38)
		if !cerrado {
			fatalO3aM38()
		}
		c.secundarios = append(c.secundarios, err)
	}
	if c.ticketEscritor != nil {
		cerrado, err := cerrarUnoConLeaseO3aM38(c.lease, c.ticketEscritor, operacionCerrarTicketO3aM38)
		if !cerrado {
			fatalO3aM38()
		}
		c.secundarios = append(c.secundarios, err)
	}
	desde := arranqueA1PreparadoM38
	if c.autoridad.es(arranqueA2AplazadoM38) {
		desde = arranqueA2AplazadoM38
	} else if c.autoridad.es(arranqueA3IniciandoM38) {
		desde = arranqueA3IniciandoM38
	} else if !c.autoridad.es(desde) {
		fatalO3aM38()
	}
	if c.autoridad.mover(desde, arranqueA7RetiradoSinHijoM38) != nil {
		fatalO3aM38()
	}
	return resultadoArranqueO3aM38{clase: resultadoRetiradoO3aM38, retirada: &retiradaO3aM38{
		origen: retiradaSinHijoO3aM38, primera: primera, controlFD: c.controlFD,
		terminal: c.terminal, lease: c.lease, observador: c.observador, causa: c.primeraCausa,
		secundarios: c.secundarios,
	}}
}

func avanzarArranqueO3aM38(p *preparadoO3aM38, testigo *testigoVueltaM38) resultadoArranqueO3aM38 {
	if p == nil || p.custodia == nil || !p.custodia.tomarPreparado() {
		return resultadoArranqueO3aM38{clase: resultadoRetiradoO3aM38, retirada: &retiradaO3aM38{origen: retiradaUsoConsumidoO3aM38, primera: errUsoConsumidoO3aM38}}
	}
	c := p.custodia
	vuelta, err := c.reloj.consumir(testigo, c.vueltaInicio)
	if err != nil || vuelta <= c.vueltaInicio {
		return retirarPreparadoO3aM38(c, errTestigoO3aM38)
	}
	resultado, err := barreraAntesStartO3aM38(c)
	if resultado == barreraAplazadaO3aM38 {
		if c.autoridad.es(arranqueA1PreparadoM38) {
			if c.autoridad.mover(arranqueA1PreparadoM38, arranqueA2AplazadoM38) != nil {
				fatalO3aM38()
			}
		}
		if !c.reabrirAplazado() {
			fatalO3aM38()
		}
		return resultadoArranqueO3aM38{clase: resultadoAplazadoO3aM38, preparado: &preparadoO3aM38{custodia: c}}
	}
	if resultado != barreraVerdeO3aM38 || err != nil {
		return retirarPreparadoO3aM38(c, err)
	}
	desde := arranqueA1PreparadoM38
	if c.autoridad.es(arranqueA2AplazadoM38) {
		desde = arranqueA2AplazadoM38
	}
	if err = c.autoridad.mover(desde, arranqueA3IniciandoM38); err != nil {
		return retirarPreparadoO3aM38(c, err)
	}
	permisoStart, permisoValido := c.lease.comenzarCritico(operacionStartO3aM38, 1)
	if !permisoValido {
		return retirarPreparadoO3aM38(c, errAutoridadO3aM38)
	}
	errStart := c.cmd.Start()
	pidfdPrimario := *c.cmd.SysProcAttr.PidFD
	c.pidfdPrimario = pidfdPrimario
	if errStart != nil && c.cmd.Process == nil && pidfdPrimario == -1 {
		if !c.lease.consolidarCritico(permisoStart) {
			fatalO3aM38()
		}
		return retirarPreparadoO3aM38(c, errStart)
	}
	if c.cmd.Process == nil || c.cmd.Process.Pid <= 0 || pidfdPrimario < 0 {
		if !c.lease.fatalPendiente(permisoStart) {
			fatalO3aM38()
		}
		fatalO3aM38()
	}
	if !c.lease.consolidarCritico(permisoStart) {
		fatalO3aM38()
	}
	permisoReserva, permisoValido := c.lease.comenzarCritico(operacionReservaPidfdO3aM38, 1)
	if !permisoValido {
		fatalO3aM38()
	}
	reserva, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfdPrimario), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)
	ahora := time.Now()
	finRetirada := ahora.Add(duracionRetiradaO3aM38)
	if tope := c.finBootstrap.Add(-reservaO3bO3cM38); finRetirada.After(tope) {
		finRetirada = tope
	}
	conPidfd, errInventario := snapshotActualO3aM38()
	reservaEntera := errno == 0 && reserva <= uintptr(^uint(0)>>1) && int(reserva) != pidfdPrimario
	reservaFD := -1
	if reservaEntera {
		reservaFD = int(reserva)
	}
	pidfdOpaco, deltaValido := deltaPidfdO3aM38(c.lease.pre, conPidfd, pidfdPrimario, reservaFD)
	if errInventario != nil || !deltaValido ||
		!c.lease.consolidarFisico(permisoReserva, conPidfd, true) {
		fatalO3aM38()
	}
	c.pidfdOpaco = pidfdOpaco
	if err = c.autoridad.mover(arranqueA3IniciandoM38, arranqueA4ProvisionalM38); err != nil {
		fatalO3aM38()
	}
	if !reservaEntera {
		return retirarConHijoO3aM38(c, fmt.Errorf("%w: %v", errProcesoO3aM38, errno), finRetirada)
	}
	c.pidfdReserva = int(reserva)
	if errStart != nil {
		return retirarConHijoO3aM38(c, errStart, finRetirada)
	}
	if err = c.autoridad.mover(arranqueA4ProvisionalM38, arranqueA5PidfdTresM38); err != nil {
		fatalO3aM38()
	}
	conTres := conPidfd
	if c.ticketLector == nil {
		return retirarConHijoO3aM38(c, errProcesoO3aM38, finRetirada)
	}
	fdTicket := int(c.ticketLector.Fd())
	cerradoTicket, errTicket := cerrarUnoConLeaseO3aM38(c.lease, c.ticketLector, operacionCerrarTicketO3aM38)
	if !cerradoTicket || errTicket != nil {
		return retirarConHijoO3aM38(c, errProcesoO3aM38, finRetirada)
	}
	c.ticketLector = nil
	delete(conTres.mapa, fdTicket)
	c.snapshot = conTres
	if err = barreraDespuesStartO3aM38(c); err != nil {
		return retirarConHijoO3aM38(c, err, finRetirada)
	}
	nuevoBaseline, observadorTransferido := c.observador.transferirCritico(c.baselineSenal)
	if !observadorTransferido {
		return retirarConHijoO3aM38(c, errSenalPendienteO3aM38, finRetirada)
	}
	c.baselineSenal = nuevoBaseline
	if !c.lease.transferirCritico() || c.autoridad.mover(arranqueA5PidfdTresM38, arranqueA6EntregadoM38) != nil {
		fatalO3aM38()
	}
	return resultadoArranqueO3aM38{clase: resultadoEntregadoO3aM38, agregado: &agregadoO3aM38{custodia: c}}
}
