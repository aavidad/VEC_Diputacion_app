//go:build ignore && linux && amd64

package main

import (
	"errors"
	"syscall"
)

type discriminanteObservacionO3cM38 uint32

const (
	observacionControlRawO3cM38 discriminanteObservacionO3cM38 = 0x101 + iota
	observacionSenalRawO3cM38
	observacionPidfdVacioO3cM38
	observacionPidfdTerminalNaturalO3cM38
	observacionPidfdInfraestructuraO3cM38
)

const (
	pollInO3cM38   int16 = 0x001
	pollErrO3cM38  int16 = 0x008
	pollHupO3cM38  int16 = 0x010
	pollNvalO3cM38 int16 = 0x020
)

func discriminanteObservacionValidoO3cM38(d discriminanteObservacionO3cM38) bool {
	return d >= observacionControlRawO3cM38 && d <= observacionPidfdInfraestructuraO3cM38
}

func instalarObservacionO3cM38(a *autoridadContinuacionO3cM38, d discriminanteObservacionO3cM38) *autoridadContinuacionO3cM38 {
	if a == nil || !discriminanteObservacionValidoO3cM38(d) || a.salida == nil || a.salida.auto != a.salida ||
		!a.salida.primera.CompareAndSwap(0, uint32(d)) ||
		!transicionContinuacionO3cM38(a.estado, continuacionC3ObservadoM38) {
		fatalObservacionO3cM38(a)
	}
	a.estado = continuacionC3ObservadoM38
	return a
}

func fatalObservacionO3cM38(a *autoridadContinuacionO3cM38) {
	if a != nil {
		a.estado = continuacionCFFatalM38
	}
	fatalO3cM38()
}

func autoridadObservacionValidaO3cM38(a *autoridadContinuacionO3cM38) bool {
	return a != nil && a.es(continuacionC2ContIntentadoM38) && a.custodia != nil && a.salida != nil &&
		a.salida.auto == a.salida && a.salida.autoridad == a.autoridad && a.salida.custodia == a.custodia &&
		a.salida.identidad == a.identidad && a.salida.primera.Load() == 0 && a.salida.retornoCont >= 0 &&
		tiempoMonotonoO3cM38(a.salida.ahoraCaso) && tiempoMonotonoO3cM38(a.salida.finCaso) &&
		a.salida.finCaso.After(a.salida.ahoraCaso) && a.salida.finCaso.Sub(a.salida.ahoraCaso) == duracionCasoO3cM38 &&
		a.autoridad != nil && a.autoridad.poseeO3c() && sellosMemoriaValidosO3cM38(a) &&
		a.custodia.lease != nil && a.custodia.lease.estado.Load() == 3
}

func pidfdExplicitosFiablesO3cM38(a *autoridadContinuacionO3cM38) (bool, bool) {
	c := a.custodia
	primario, errPrimario := identidadPidfdBarreraO3bM38(c, c.pidfdPrimario)
	reserva, errReserva := identidadPidfdBarreraO3bM38(c, c.pidfdReserva)
	if errors.Is(errPrimario, errLeaseBarreraO3bM38) || errors.Is(errReserva, errLeaseBarreraO3bM38) {
		fatalObservacionO3cM38(a)
	}
	selloPrimario, existePrimario := a.sellos.fisico.mapa[c.pidfdPrimario]
	selloReserva, existeReserva := a.sellos.fisico.mapa[c.pidfdReserva]
	fiablePrimario := errPrimario == nil && existePrimario && primario == selloPrimario.identidad && primario.fdflags&syscall.FD_CLOEXEC != 0
	fiableReserva := errReserva == nil && existeReserva && reserva == selloReserva.identidad && reserva.fdflags&syscall.FD_CLOEXEC != 0
	return fiablePrimario, fiableReserva
}

func clasificarPollO3cM38(n uintptr, primero, segundo int16, err error) discriminanteObservacionO3cM38 {
	if err != nil {
		return observacionPidfdInfraestructuraO3cM38
	}
	if n == 0 && primero == 0 && segundo == 0 {
		return observacionPidfdVacioO3cM38
	}
	eventos := uintptr(0)
	if primero == pollInO3cM38 {
		eventos++
	}
	if segundo == pollInO3cM38 {
		eventos++
	}
	if eventos > 0 && n == eventos && (primero == 0 || primero == pollInO3cM38) && (segundo == 0 || segundo == pollInO3cM38) {
		return observacionPidfdTerminalNaturalO3cM38
	}
	return observacionPidfdInfraestructuraO3cM38
}

func observarPidfdO3cM38(a *autoridadContinuacionO3cM38) discriminanteObservacionO3cM38 {
	fiablePrimario, fiableReserva := pidfdExplicitosFiablesO3cM38(a)
	if !fiablePrimario && !fiableReserva {
		return observacionPidfdInfraestructuraO3cM38
	}
	sondas := [2]pollfdO3aM38{{fd: -1}, {fd: -1}}
	cardinalidad := 0
	if fiablePrimario {
		sondas[cardinalidad] = pollfdO3aM38{fd: int32(a.custodia.pidfdPrimario), eventos: pollInO3cM38}
		cardinalidad++
	}
	if fiableReserva {
		sondas[cardinalidad] = pollfdO3aM38{fd: int32(a.custodia.pidfdReserva), eventos: pollInO3cM38}
		cardinalidad++
	}
	var n uintptr
	err := syscallLeaseO3cM38(a.custodia, func() error {
		var errno syscall.Errno
		n, _, errno = syscall.Syscall(syscall.SYS_POLL, punteroPollO3aM38(&sondas[0]), uintptr(cardinalidad), 0)
		if errno != 0 {
			return errno
		}
		return nil
	})
	if errors.Is(err, errLeaseBarreraO3bM38) {
		fatalObservacionO3cM38(a)
	}
	return clasificarPollO3cM38(n, sondas[0].retorno, sondas[1].retorno, err)
}

// observarInmediatoO3cM38 consume C2 y ejecuta una única ronda sin espera.
func observarInmediatoO3cM38(entrada **autoridadContinuacionO3cM38) *autoridadContinuacionO3cM38 {
	if entrada == nil || *entrada == nil {
		fatalObservacionO3cM38(nil)
	}
	a := *entrada
	*entrada = nil
	if !autoridadObservacionValidaO3cM38(a) {
		fatalObservacionO3cM38(a)
	}
	if err := leerControlO3bM38(a.custodia); err != nil {
		if errors.Is(err, errLeaseBarreraO3bM38) {
			fatalObservacionO3cM38(a)
		}
		return instalarObservacionO3cM38(a, observacionControlRawO3cM38)
	}
	senalVerde, err := autoridadSenalO3cM38(a.custodia)
	if errors.Is(err, errLeaseBarreraO3bM38) {
		fatalObservacionO3cM38(a)
	}
	if err != nil || !senalVerde {
		return instalarObservacionO3cM38(a, observacionSenalRawO3cM38)
	}
	return instalarObservacionO3cM38(a, observarPidfdO3cM38(a))
}
