//go:build ignore && linux && amd64

package main

import (
	"errors"
	"syscall"
	"time"
)

var errArbitrajeConsumidoO4aM38 = errors.New("arbitraje O4a no disponible")

func fatalArbitrajeO4aM38(a *autoridadCausaO4aM38) {
	if a != nil {
		a.estado.Store(uint32(causaAFFatalM38))
	}
	fatalO3cM38()
}

func enclavarArbitrajeO4aM38(a *autoridadCausaO4aM38, causa causaPrimariaO4aM38) error {
	if !causaPrimariaValidaO4aM38(causa) ||
		!a.causa.CompareAndSwap(uint32(causaVaciaO4aM38), uint32(causa)) ||
		!a.estado.CompareAndSwap(uint32(causaA2ObservandoM38), uint32(causaA3CausaFijadaM38)) {
		fatalArbitrajeO4aM38(a)
	}
	return nil
}

func tiempoSelladoArbitrajeO4aM38(a *autoridadCausaO4aM38) bool {
	if a == nil || a.origen == nil || a.origen.ahoraCaso != a.sellos.ahoraCaso ||
		a.origen.finCaso != a.sellos.finCaso {
		return false
	}
	inicio, fin := a.sellos.ahoraCaso, a.sellos.finCaso
	finExacto, ok := finCasoExactoO3cM38(inicio)
	return ok && tiempoMonotonoO3cM38(fin) && fin == finExacto && fin.After(inicio) &&
		fin.Sub(inicio) == duracionCasoO3cM38
}

func autoridadArbitrajeExactaO4aM38(a *autoridadCausaO4aM38) bool {
	if a == nil || a.auto != a || a.origen == nil || a.origen.auto != a.origen ||
		a.sellos.custodia != a.origen.custodia || a.sellos.lease == nil || a.sellos.observador == nil ||
		a.sellos.registro == nil || a.sellos.lease != a.origen.custodia.lease ||
		a.sellos.observador != a.origen.custodia.observador || a.origen.autoridad != a.sellos.autoridad ||
		a.sellos.autoridadArranque != a.sellos.custodia.autoridad || a.sellos.control != a.sellos.custodia.control ||
		a.sellos.controlFD != a.sellos.custodia.controlFD || a.sellos.terminal != a.sellos.custodia.terminal ||
		a.sellos.cmd != a.sellos.custodia.cmd || a.sellos.cmd == nil || a.sellos.proceso != a.sellos.cmd.Process ||
		a.sellos.identidad != a.origen.identidad || a.sellos.tid != a.sellos.custodia.tid ||
		a.sellos.ppid != a.sellos.custodia.ppid || a.sellos.baselineSenal != a.sellos.custodia.baselineSenal ||
		a.origen.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) ||
		a.origen.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
		a.sellos.lease.estado.Load() != 3 || a.sellos.lease.registro != a.sellos.registro ||
		a.sellos.observador.registro != a.sellos.registro ||
		a.sellos.registro.leases[a.sellos.lease] != a.sellos.generacionLease ||
		a.sellos.registro.observadores[a.sellos.observador] != a.sellos.generacionObservador ||
		a.sellos.retornoCont != 0 || discriminanteObservacionO3cM38(a.sellos.primera) != observacionPidfdVacioO3cM38 ||
		a.sellos.canonControlRaw != controlRawVacioO4aM38 || a.sellos.palabraObservada != a.sellos.baselineSenal {
		return false
	}
	p := a.sellos.pidfd
	if p != [3]int{a.sellos.custodia.pidfdPrimario, a.sellos.custodia.pidfdReserva, a.sellos.custodia.pidfdOpaco} ||
		p[0] < 0 || p[1] < 0 || p[2] < 0 || p[0] == p[1] || p[0] == p[2] || p[1] == p[2] {
		return false
	}
	return tiempoSelladoArbitrajeO4aM38(a)
}

func causaControlActualO4aM38(a *autoridadCausaO4aM38) (causaPrimariaO4aM38, bool) {
	canon, valida := normalizarControlRawO4aM38(a.origen, uint32(observacionControlRawO3cM38))
	if !valida {
		fatalArbitrajeO4aM38(a)
	}
	return causaControlSelladaO4aM38(canon)
}

func observarSenalArbitrajeO4aM38(a *autoridadCausaO4aM38) (causaPrimariaO4aM38, bool) {
	var palabra uint64
	var senal syscall.Signal
	var valida bool
	err := operarConLeaseBarreraO3bM38(a.sellos.custodia, func() error {
		palabra, senal, valida = a.sellos.observador.observar()
		return nil
	})
	if err != nil {
		fatalArbitrajeO4aM38(a)
	}
	base := a.sellos.palabraObservada
	if !valida || palabra&mascaraEstadoObservadorO3aM38 != 2 || palabra>>10 < base>>10 {
		fatalArbitrajeO4aM38(a)
	}
	if palabra == base {
		return causaVaciaO4aM38, false
	}
	if palabra>>10 != base>>10+1 {
		return causaIncidente65O4aM38, true
	}
	switch uint8(senal) {
	case signoIntRawO4aM38:
		return causaSenalInt130O4aM38, true
	case signoTermRawO4aM38:
		return causaSenalTerm143O4aM38, true
	}
	return causaIncidente65O4aM38, true
}

func pidfdFiableArbitrajeO4aM38(a *autoridadCausaO4aM38, fd int) bool {
	identidad, err := identidadPidfdBarreraO3bM38(a.sellos.custodia, fd)
	if errors.Is(err, errLeaseBarreraO3bM38) {
		fatalArbitrajeO4aM38(a)
	}
	huella, existe := a.sellos.fisico.mapa[fd]
	return err == nil && existe && huella.abierto && identidad == huella.identidad &&
		identidad.fdflags&syscall.FD_CLOEXEC != 0
}

func pollPidfdArbitrajeO4aM38(a *autoridadCausaO4aM38, fd int) (uintptr, int16, error) {
	sonda := pollfdO3aM38{fd: int32(fd), eventos: pollInO3cM38}
	var n uintptr
	err := operarConLeaseBarreraO3bM38(a.sellos.custodia, func() error {
		var errno syscall.Errno
		n, _, errno = syscall.Syscall(syscall.SYS_POLL, punteroPollO3aM38(&sonda), 1, 0)
		if errno != 0 {
			return errno
		}
		return nil
	})
	if errors.Is(err, errLeaseBarreraO3bM38) {
		fatalArbitrajeO4aM38(a)
	}
	return n, sonda.retorno, err
}

func observarPidfdArbitrajeO4aM38(a *autoridadCausaO4aM38) discriminanteObservacionO3cM38 {
	primario, reserva := a.sellos.pidfd[0], a.sellos.pidfd[1]
	if !pidfdFiableArbitrajeO4aM38(a, primario) || !pidfdFiableArbitrajeO4aM38(a, reserva) {
		return observacionPidfdInfraestructuraO3cM38
	}
	n1, r1, err1 := pollPidfdArbitrajeO4aM38(a, primario)
	n2, r2, err2 := pollPidfdArbitrajeO4aM38(a, reserva)
	if err1 != nil || err2 != nil {
		return observacionPidfdInfraestructuraO3cM38
	}
	return clasificarPollO3cM38(n1+n2, r1, r2, nil)
}

// arbitrarInmediatoO4aM38 ejecuta una única ronda no recolectora desde A2.
func arbitrarInmediatoO4aM38(a *autoridadCausaO4aM38) error {
	if a == nil {
		return errArbitrajeConsumidoO4aM38
	}
	previa := causaPrimariaO4aM38(a.causa.Load())
	if previa != causaVaciaO4aM38 {
		if !causaPrimariaValidaO4aM38(previa) {
			fatalArbitrajeO4aM38(a)
		}
		return errArbitrajeConsumidoO4aM38
	}
	// Este cotejo estructural precede al primer evento, permiso, syscall o reloj.
	if a.estado.Load() != uint32(causaA2ObservandoM38) || !autoridadArbitrajeExactaO4aM38(a) {
		fatalArbitrajeO4aM38(a)
	}
	if err := leerControlO3bM38(a.sellos.custodia); err != nil {
		if errors.Is(err, errLeaseBarreraO3bM38) {
			fatalArbitrajeO4aM38(a)
		}
		causa, valida := causaControlActualO4aM38(a)
		if !valida {
			fatalArbitrajeO4aM38(a)
		}
		return enclavarArbitrajeO4aM38(a, causa)
	}
	if causa, existe := observarSenalArbitrajeO4aM38(a); existe {
		return enclavarArbitrajeO4aM38(a, causa)
	}
	switch observarPidfdArbitrajeO4aM38(a) {
	case observacionPidfdTerminalNaturalO3cM38:
		return enclavarArbitrajeO4aM38(a, causaSalidaO4aM38)
	case observacionPidfdInfraestructuraO3cM38:
		return enclavarArbitrajeO4aM38(a, causaIncidente65O4aM38)
	case observacionPidfdVacioO3cM38:
	default:
		fatalArbitrajeO4aM38(a)
	}
	ahora := time.Now()
	if !tiempoMonotonoO3cM38(ahora) || ahora.Before(a.sellos.ahoraCaso) {
		fatalArbitrajeO4aM38(a)
	}
	if !ahora.Before(a.sellos.finCaso) {
		return enclavarArbitrajeO4aM38(a, causaPlazo65O4aM38)
	}
	return nil
}
