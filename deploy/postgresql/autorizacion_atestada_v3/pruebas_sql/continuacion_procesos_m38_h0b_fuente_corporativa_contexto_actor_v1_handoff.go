//go:build ignore && linux && amd64

package main

import (
	"errors"
	"syscall"
	"time"
)

const duracionRetiradaO3cM38 = 3 * time.Second

var errRetiradaO3cM38 = errors.New("retirada O3c no acreditada")

func autoridadesExactasO3cM38(a *autoridadContinuacionO3cM38) bool {
	if a == nil || a.custodia == nil || a.sellos.registro == nil {
		return false
	}
	c, s := a.custodia, a.sellos
	return c.lease != nil && c.lease.auto == c.lease && c.observador != nil && c.observador.auto == c.observador &&
		c.lease.registro == s.registro && c.observador.registro == s.registro && s.registro.auto == s.registro &&
		c.lease.generacion == s.generacionLease && c.observador.generacion == s.generacionObservador &&
		s.registro.leases[c.lease] == s.generacionLease && s.registro.observadores[c.observador] == s.generacionObservador &&
		c.tid == s.tid && c.lease.tid == s.tid && s.registro.tid == s.tid
}

func autoridadHandoffO3cM38(a *autoridadContinuacionO3cM38) bool {
	if a == nil || !a.es(continuacionC3ObservadoM38) || a.salida == nil || a.salida.auto != a.salida ||
		a.salida.autoridad != a.autoridad || a.salida.custodia != a.custodia || a.salida.identidad != a.identidad ||
		!discriminanteObservacionValidoO3cM38(discriminanteObservacionO3cM38(a.salida.primera.Load())) ||
		!tiempoMonotonoO3cM38(a.salida.ahoraCaso) || !tiempoMonotonoO3cM38(a.salida.finCaso) ||
		a.salida.finCaso.Sub(a.salida.ahoraCaso) != 180*time.Second ||
		a.autoridad == nil || a.autoridad.auto != a.autoridad || !a.autoridad.poseeO3c() {
		return false
	}
	c, s := a.custodia, a.sellos
	return autoridadesExactasO3cM38(a) && sellosMemoriaValidosO3cM38(a) && c.lease.estado.Load() == 3 &&
		c.observador.palabra.Load()&mascaraEstadoObservadorO3aM38 == 2 && s.lease == c.lease
}

func transferirHandoffO3cM38(entrada **autoridadContinuacionO3cM38) *agregadoO4aM38 {
	if entrada == nil || *entrada == nil {
		fatalO3cM38()
	}
	a := *entrada
	*entrada = nil
	if !autoridadHandoffO3cM38(a) || !transicionContinuacionO3cM38(a.estado, continuacionC4TTransfiriendoM38) {
		fatalHandoffO3cM38(a)
	}
	a.estado = continuacionC4TTransfiriendoM38
	return consolidarHandoffO3cM38(a)
}

func consolidarHandoffO3cM38(a *autoridadContinuacionO3cM38) *agregadoO4aM38 {
	if !a.autoridad.ownerObservador.CompareAndSwap(uint32(propietarioO3cM38), uint32(propietarioO4aM38)) ||
		!a.autoridad.ownerLease.CompareAndSwap(uint32(propietarioO3cM38), uint32(propietarioO4aM38)) {
		fatalHandoffO3cM38(a)
	}
	a.estado = continuacionC5EntregadoM38
	salida := a.salida
	a.custodia, a.autoridad, a.salida = nil, nil, nil
	return salida
}

func fatalHandoffO3cM38(a *autoridadContinuacionO3cM38) {
	if a != nil {
		a.estado = continuacionCFFatalM38
	}
	fatalO3cM38()
}

func autoridadRetiradaO3cM38(a *autoridadContinuacionO3cM38) bool {
	if a == nil || !a.es(continuacionC7RetirandoM38) || a.custodia == nil || a.salida == nil ||
		a.autoridad == nil || !a.autoridad.poseeO3c() || !autoridadesExactasO3cM38(a) || !sellosMemoriaValidosO3cM38(a) {
		return false
	}
	c := a.custodia
	return c.lease.estado.Load() == 3 && c.observador.palabra.Load()&mascaraEstadoObservadorO3aM38 == 2 &&
		c.controlFD != nil && c.terminal != nil && c.cmd != nil && c.cmd.Process != nil
}

func finRetiradaO3cM38(ahora, bootstrap time.Time) (time.Time, bool) {
	if !tiempoMonotonoO3cM38(ahora) || !tiempoMonotonoO3cM38(bootstrap) || !ahora.Before(bootstrap) {
		return time.Time{}, false
	}
	fin := ahora.Add(duracionRetiradaO3cM38)
	if !fin.After(ahora) {
		return time.Time{}, false
	}
	if bootstrap.Before(fin) {
		fin = bootstrap
	}
	return fin, tiempoMonotonoO3cM38(fin) && fin.After(ahora)
}

type referenciaRetiradaO3cM38 struct {
	fd            int
	integra, viva bool
}

func seleccionarPidfdRetiradaO3cM38(primario, reserva referenciaRetiradaO3cM38) (int, bool, bool) {
	if primario.integra {
		return primario.fd, primario.viva, true
	}
	if reserva.integra {
		return reserva.fd, reserva.viva, true
	}
	return -1, false, false
}

func pidfdRetiradaO3cM38(a *autoridadContinuacionO3cM38) (int, bool, bool) {
	c := a.custodia
	primario, e1 := identidadPidfdBarreraO3bM38(c, c.pidfdPrimario)
	reserva, e2 := identidadPidfdBarreraO3bM38(c, c.pidfdReserva)
	vivoPrimario, errVivoPrimario := pidfdVivoBarreraO3bM38(c, c.pidfdPrimario)
	vivoReserva, errVivoReserva := pidfdVivoBarreraO3bM38(c, c.pidfdReserva)
	return seleccionarPidfdRetiradaO3cM38(
		referenciaRetiradaO3cM38{c.pidfdPrimario, e1 == nil && errVivoPrimario == nil && primario == a.sellos.fisico.mapa[c.pidfdPrimario].identidad, vivoPrimario},
		referenciaRetiradaO3cM38{c.pidfdReserva, e2 == nil && errVivoReserva == nil && reserva == a.sellos.fisico.mapa[c.pidfdReserva].identidad, vivoReserva})
}

func enviarKillRetiradaO3cM38(c *custodiaO3aM38, fd int) error {
	return operarConLeaseBarreraO3bM38(c, func() error {
		_, _, errno := syscall.Syscall6(sysPidfdSendSignal, uintptr(fd), uintptr(syscall.SIGKILL), 0, 0, 0, 0)
		if errno != 0 {
			return errno
		}
		return nil
	})
}

func esperarTerminalRetiradaO3cM38(c *custodiaO3aM38, fd int, fin time.Time) bool {
	for time.Now().Before(fin) {
		vivo, err := pidfdVivoBarreraO3bM38(c, fd)
		if err != nil {
			return false
		}
		if !vivo {
			return time.Now().Before(fin)
		}
		time.Sleep(time.Millisecond)
	}
	return false
}

func drenarAdoptadosO3cM38(c *custodiaO3aM38, fin time.Time) bool {
	for time.Now().Before(fin) {
		var pid int
		var err error
		falloLease := operarConLeaseBarreraO3bM38(c, func() error {
			var estado syscall.WaitStatus
			pid, err = syscall.Wait4(-1, &estado, syscall.WNOHANG, nil)
			return nil
		})
		if falloLease != nil {
			return false
		}
		if errors.Is(err, syscall.ECHILD) {
			return true
		}
		if err != nil || pid == 0 {
			return false
		}
	}
	return false
}

func grupoAusenteO3cM38(c *custodiaO3aM38, fd int) bool {
	err := sondaGrupoCeroO3bM38(c, fd)
	return errors.Is(err, syscall.ESRCH)
}

func cerrarRecursosRetiradaO3cM38(c *custodiaO3aM38) bool {
	if c.pidfdPrimario >= 0 {
		if cerrado, err := cerrarPidfdConLeaseO3aM38(c.lease, c.pidfdPrimario); !cerrado || err != nil {
			return false
		}
	}
	c.pidfdPrimario = -1
	if c.pidfdReserva >= 0 {
		if cerrado, err := cerrarPidfdConLeaseO3aM38(c.lease, c.pidfdReserva); !cerrado || err != nil {
			return false
		}
	}
	c.pidfdReserva = -1
	if cerrado, err := cerrarUnoConLeaseO3aM38(c.lease, c.controlFD, operacionCerrarDestinosO3aM38); !cerrado || err != nil {
		return false
	}
	c.controlFD = nil
	if cerrado, err := cerrarUnoConLeaseO3aM38(c.lease, c.terminal, operacionCerrarDestinosO3aM38); !cerrado || err != nil {
		return false
	}
	c.terminal = nil
	return true
}

func inventarioLiberadoO3cM38(c *custodiaO3aM38, previo snapshotFDO3aM38, poseidos [5]int) bool {
	actual, err := snapshotBarreraO3bM38(c)
	if err != nil || c.pidfdPrimario != -1 || c.pidfdReserva != -1 || c.pidfdOpaco != -1 || c.controlFD != nil || c.terminal != nil {
		return false
	}
	for fd, huella := range previo.mapa {
		poseido := false
		for _, propio := range poseidos {
			poseido = poseido || fd == propio
		}
		actualHuella, existe := actual.mapa[fd]
		if poseido && existe || !poseido && (!existe || actualHuella != huella) {
			return false
		}
	}
	for fd, huella := range actual.mapa {
		if anterior, existe := previo.mapa[fd]; !existe || anterior != huella {
			return false
		}
	}
	return true
}

func liberarRetiradaO3cM38(a *autoridadContinuacionO3cM38) bool {
	c := a.custodia
	if err := c.observador.liberar(); err != nil {
		return false
	}
	if err := c.lease.liberar(); err != nil ||
		!a.autoridad.ownerObservador.CompareAndSwap(uint32(propietarioO3cM38), uint32(propietarioLiberadoO3cM38)) ||
		!a.autoridad.ownerLease.CompareAndSwap(uint32(propietarioO3cM38), uint32(propietarioLiberadoO3cM38)) {
		return false
	}
	return true
}

func retirarAntesContO3cM38(entrada **autoridadContinuacionO3cM38) error {
	if entrada == nil || *entrada == nil {
		fatalO3cM38()
	}
	a := *entrada
	*entrada = nil
	if !autoridadRetiradaO3cM38(a) {
		fatalHandoffO3cM38(a)
	}
	fin, ok := finRetiradaO3cM38(time.Now(), a.custodia.finBootstrap)
	if !ok {
		fatalHandoffO3cM38(a)
	}
	fd, vivo, fiable := pidfdRetiradaO3cM38(a)
	poseidos := [5]int{a.custodia.pidfdPrimario, a.custodia.pidfdReserva, a.custodia.pidfdOpaco,
		int(a.custodia.controlFD.Fd()), int(a.custodia.terminal.Fd())}
	if !fiable || vivo && enviarKillRetiradaO3cM38(a.custodia, fd) != nil ||
		!esperarTerminalRetiradaO3cM38(a.custodia, fd, fin) {
		fatalHandoffO3cM38(a)
	}
	if err := esperarConLeaseO3aM38(a.custodia); err != nil {
		var salida *execExitErrorO3aM38
		if !errors.As(err, &salida) {
			fatalHandoffO3cM38(a)
		}
	}
	if !drenarAdoptadosO3cM38(a.custodia, fin) || !grupoAusenteO3cM38(a.custodia, fd) ||
		!cerrarRecursosRetiradaO3cM38(a.custodia) || !inventarioLiberadoO3cM38(a.custodia, a.sellos.fisico, poseidos) || !liberarRetiradaO3cM38(a) ||
		!transicionContinuacionO3cM38(a.estado, continuacionC8RetiradoM38) {
		fatalHandoffO3cM38(a)
	}
	a.custodia.cmd.Process = nil
	a.custodia.cmd = nil
	a.estado = continuacionC8RetiradoM38
	a.custodia, a.autoridad, a.salida = nil, nil, nil
	return errRetiradaO3cM38
}
