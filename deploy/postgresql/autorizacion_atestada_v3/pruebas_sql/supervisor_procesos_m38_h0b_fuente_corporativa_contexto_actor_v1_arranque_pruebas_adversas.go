//go:build ignore && linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	estadoErrorExternoO3aM38             = 66
	estadoRetiradaSinHijoExternaO3aM38   = 72
	estadoRetiradaConHijoExternaO3aM38   = 73
	estadoPdeathCreadorExternoO3aM38     = 74
	estadoPdeathOtroExternoO3aM38        = 75
	estadoPdeathAusenteExternoO3aM38     = 76
	estadoNetpollExternoO3aM38           = 77
	estadoSubreaperExternoO3aM38         = 78
	estadoFixtureExternoO3aM38           = 79
	estadoVentanaFDExternaO3aM38         = 80
	estadoMapaFixtureExternoO3aM38       = 81
	estadoCierreHuecoExternoO3aM38       = 82
	estadoSnapshotExternoO3aM38          = 83
	estadoAvanceExternoO3aM38            = 84
	estadoIdentidadExternaO3aM38         = 85
	estadoLimpiezaExternaO3aM38          = 86
	estadoClaseRetiradaExternaO3aM38     = 87
	estadoOrigenRetiradaExternaO3aM38    = 88
	estadoWaitExternoO3aM38              = 89
	estadoPrecedenciaExternaO3aM38       = 90
	estadoHijosExternosO3aM38            = 91
	estadoCasoBaseExternoO3aM38          = 92
	estadoEntradaFixtureExternoO3aM38    = 93
	estadoSubreaperFixtureExternoO3aM38  = 94
	estadoPdeathFixtureExternoO3aM38     = 95
	estadoInventarioFixtureExternoO3aM38 = 96
	estadoFormaFixtureExternoO3aM38      = 97
	estadoOtroFixtureExternoO3aM38       = 98
)

func cerrarRetiradaPruebaO3aM38(r **retiradaO3aM38) error {
	if r == nil || *r == nil {
		return nil
	}
	retirada := *r
	*r = nil
	fallos := append([]error(nil), retirada.secundarios...)
	fallos = append(fallos, cerrarArchivosO3aM38(retirada.lease, []*os.File{retirada.controlFD, retirada.terminal}))
	if retirada.observador != nil {
		fallos = append(fallos, retirada.observador.liberar())
	}
	if retirada.lease != nil {
		fallos = append(fallos, retirada.lease.liberar())
	}
	return errors.Join(fallos...)
}

func consumirAgregadoPruebaO3aM38(a **agregadoO3aM38) error {
	if a == nil || *a == nil || (*a).custodia == nil {
		return errEntradaO3aM38
	}
	agregado := *a
	*a = nil
	c := agregado.custodia
	if !c.tomarAgregadoPrueba() || c.autoridad == nil || !c.autoridad.es(arranqueA6EntregadoM38) ||
		c.ticketEscritor == nil || c.cmd == nil || c.cmd.Process == nil || c.pidfdPrimario < 0 || c.pidfdReserva < 0 || c.pidfdOpaco < 0 {
		fatalO3aM38()
	}
	var fallos []error
	cerrado, errCierre := cerrarUnoConLeaseO3aM38(c.lease, c.ticketEscritor, operacionCerrarTicketO3aM38)
	fallos = append(fallos, errCierre)
	if !cerrado {
		fatalO3aM38()
	}
	c.ticketEscritor = nil
	if err := terminalidadPidfdPruebaO3aM38(c.pidfdPrimario, time.Now().Add(10*time.Second)); err != nil {
		fatalO3aM38()
	}
	errWait := esperarConLeaseO3aM38(c)
	if errWait != nil {
		var estado *execExitErrorPruebaO3aM38
		if !errors.As(errWait, &estado) || estado.ExitCode() != estadoUso {
			fallos = append(fallos, errWait)
		}
	}
	for _, fd := range []int{c.pidfdPrimario, c.pidfdReserva} {
		if cerrado, err := cerrarPidfdConLeaseO3aM38(c.lease, fd); !cerrado {
			fatalO3aM38()
		} else {
			fallos = append(fallos, err)
		}
	}
	c.pidfdPrimario, c.pidfdReserva = -1, -1
	fallos = append(fallos, cerrarArchivosO3aM38(c.lease, []*os.File{c.controlFD, c.terminal}))
	c.controlFD, c.terminal = nil, nil
	fallos = append(fallos, c.observador.liberar(), c.lease.liberar())
	return errors.Join(fallos...)
}

type execExitErrorPruebaO3aM38 = exec.ExitError

func cerrarPreparadoPruebaO3aM38(p **preparadoO3aM38) error {
	if p == nil || *p == nil || (*p).custodia == nil {
		return nil
	}
	c := (*p).custodia
	*p = nil
	var fallos []error
	fallos = append(fallos, cerrarArchivosO3aM38(c.lease, c.destinados))
	fallos = append(fallos, cerrarArchivosO3aM38(c.lease, []*os.File{c.ticketLector, c.ticketEscritor, c.controlFD, c.terminal}))
	if c.lease != nil {
		fallos = append(fallos, c.lease.liberar())
	}
	if c.observador != nil {
		fallos = append(fallos, c.observador.liberar())
	}
	return errors.Join(fallos...)
}

func cerrarEntradaPruebaO3aM38(e *bundleEntradaO3aM38) error {
	if e == nil {
		return nil
	}
	archivos := []*os.File{e.raiz, e.runner, e.salida, e.errorCaso, e.controlFD, e.terminal, e.sobre}
	var fallos []error
	fallos = append(fallos, cerrarArchivosO3aM38(nil, archivos))
	if e.lease != nil {
		fallos = append(fallos, e.lease.liberar())
	}
	if e.observador != nil {
		fallos = append(fallos, e.observador.liberar())
	}
	*e = bundleEntradaO3aM38{}
	return errors.Join(fallos...)
}

func retirarDirectorioFixtureO3aM38(f *fixtureO3aM38) error {
	if f == nil || f.directorio == "" {
		return nil
	}
	for _, nombre := range []string{"runner", "salida", "error", "terminal"} {
		if err := os.Remove(f.directorio + "/" + nombre); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	err := os.Remove(f.directorio)
	f.directorio = ""
	return err
}

func limpiarFixtureO3aM38(f *fixtureO3aM38) error {
	if f == nil {
		return nil
	}
	var fallos []error
	if f.agregado != nil {
		fallos = append(fallos, consumirAgregadoPruebaO3aM38(&f.agregado))
	}
	if f.preparado != nil {
		fallos = append(fallos, cerrarPreparadoPruebaO3aM38(&f.preparado))
	}
	if f.retirada != nil {
		fallos = append(fallos, cerrarRetiradaPruebaO3aM38(&f.retirada))
	}
	fallos = append(fallos, cerrarEntradaPruebaO3aM38(&f.entrada))
	if f.controlEscritor != nil {
		fallos = append(fallos, f.controlEscritor.Close())
		f.controlEscritor = nil
	}
	fallos = append(fallos, retirarDirectorioFixtureO3aM38(f))
	if f.canalSenal != nil {
		signal.Stop(f.canalSenal)
		close(f.pararSenal)
		<-f.senalParada
		f.canalSenal = nil
	}
	fallos = append(fallos, restaurarPdeathsigPruebaO3aM38(f))
	fallos = append(fallos, restaurarRlimitPruebaO3aM38(f))
	return errors.Join(fallos...)
}

func probarPreparacionSinStartO3aM38() (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil {
		return err
	}
	if f.preparado.custodia.cmd.Process != nil || f.preparado.custodia.pidfdPrimario != -1 ||
		f.preparado.custodia.autoridad.estado != arranqueA1PreparadoM38 {
		return errInvarianteO3aM38
	}
	return verificarLiteralComandoO3aM38(f.preparado, "NOMINAL")
}

func selectoresPruebaO3aM38() []string {
	selectores := []string{"A01", "A02", "A03"}
	for _, prefijo := range []string{"N", "E"} {
		for i := 1; i <= 10; i++ {
			selectores = append(selectores, fmt.Sprintf("%s%02d", prefijo, i))
		}
	}
	for i := 1; i <= 15; i++ {
		selectores = append(selectores, fmt.Sprintf("F%02d", i))
	}
	return append(selectores, "NOMINAL")
}

func probarSelectoresO3aM38() error {
	for _, selector := range selectoresPruebaO3aM38() {
		f, err := crearFixtureO3aM38(selector)
		if err != nil {
			return fmt.Errorf("fixture %s: %w", selector, err)
		}
		if err = prepararFixtureO3aM38(f); err == nil {
			err = verificarLiteralComandoO3aM38(f.preparado, selector)
		}
		if cierre := limpiarFixtureO3aM38(f); err == nil {
			err = cierre
		}
		if err != nil {
			return fmt.Errorf("selector %s: %w", selector, err)
		}
	}
	return nil
}

func probarTestigosO3aM38() error {
	r := nuevoRelojVueltaM38()
	primero := r.emitir()
	vuelta, err := r.consumir(primero, 0)
	if err != nil || vuelta != 1 {
		return errTestigoO3aM38
	}
	if _, err = r.consumir(primero, 0); !errors.Is(err, errTestigoO3aM38) {
		return errTestigoO3aM38
	}
	clon := *r.emitir()
	if _, err = r.consumir(&clon, vuelta); !errors.Is(err, errTestigoO3aM38) {
		return errTestigoO3aM38
	}
	ajeno := nuevoRelojVueltaM38().emitir()
	if _, err = r.consumir(ajeno, vuelta); !errors.Is(err, errTestigoO3aM38) {
		return errTestigoO3aM38
	}
	return nil
}

func probarPreflightO3aM38() error {
	registro := nuevoRegistroAutoridadO3aM38()
	a, err := preflightPidfdGoM38(registro)
	if err != nil || !consumirPreflightPidfdGoM38(a) || consumirPreflightPidfdGoM38(a) {
		return errPreflightPidfdO3aM38
	}
	clon := &acreditacionPidfdGoM38{auto: a}
	if consumirPreflightPidfdGoM38(clon) {
		return errPreflightPidfdO3aM38
	}
	forjada := &acreditacionPidfdGoM38{registro: registro, generacion: registro.generacion}
	forjada.auto = forjada
	if consumirPreflightPidfdGoM38(forjada) {
		return errPreflightPidfdO3aM38
	}
	return nil
}

func probarArranqueRealO3aM38() (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil {
		return err
	}
	if err = avanzarFixtureO3aM38(f); err != nil {
		return fmt.Errorf("avance real: %w (%s)", err, inventarioFixtureO3aM38(f))
	}
	if f.agregado == nil || f.agregado.custodia.autoridad.estado != arranqueA6EntregadoM38 ||
		f.agregado.custodia.pidfdPrimario < 0 || f.agregado.custodia.pidfdReserva < 0 ||
		f.agregado.custodia.pidfdOpaco < 0 || f.agregado.custodia.pidfdPrimario == f.agregado.custodia.pidfdReserva ||
		f.agregado.custodia.pidfdOpaco == f.agregado.custodia.pidfdPrimario ||
		f.agregado.custodia.pidfdOpaco == f.agregado.custodia.pidfdReserva || f.agregado.custodia.ticketEscritor == nil {
		return errInvarianteO3aM38
	}
	return nil
}

func escribirControlPruebaO3aM38(f *fixtureO3aM38, datos string) error {
	if f == nil || f.controlEscritor == nil {
		return errEntradaO3aM38
	}
	n, err := f.controlEscritor.Write([]byte(datos))
	if err != nil || n != len(datos) {
		return errControlO3aM38
	}
	return nil
}

func probarBarreraAdversaO3aM38(caso string) (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil {
		return err
	}
	c := f.preparado.custodia
	switch caso {
	case "parcial":
		err = escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|")
	case "cancelar":
		err = escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|"+string(c.control.nonce[:])+"|CANCELADO|65\n")
	case "eof":
		cerrado, fallo := cerrarUnoConLeaseO3aM38(c.lease, f.controlEscritor, operacionCerrarDestinosO3aM38)
		if !cerrado {
			err = errInvarianteO3aM38
		} else {
			err, f.controlEscritor = fallo, nil
		}
	case "senal":
		if !c.observador.anotar(syscall.SIGUSR1) {
			err = errSenalPendienteO3aM38
		}
	case "plazo":
		c.finBootstrap = time.Now().Add(2 * time.Second)
	default:
		err = errEntradaO3aM38
	}
	if err != nil {
		return err
	}
	resultado := avanzarArranqueO3aM38(f.preparado, c.reloj.emitir())
	f.preparado = nil
	if caso == "parcial" {
		if resultado.clase != resultadoAplazadoO3aM38 || resultado.preparado == nil || resultado.preparado.custodia.cmd.Process != nil {
			return errInvarianteO3aM38
		}
		f.preparado = resultado.preparado
		return nil
	}
	if resultado.clase != resultadoRetiradoO3aM38 || resultado.retirada == nil || resultado.retirada.origen != retiradaSinHijoO3aM38 {
		return errInvarianteO3aM38
	}
	f.retirada = resultado.retirada
	if (caso == "cancelar" || caso == "eof") && (resultado.retirada.causa.causa != "CANCELADO" || resultado.retirada.causa.estado != "65") {
		return errInvarianteO3aM38
	}
	if caso == "senal" && !errors.Is(resultado.retirada.primera, errSenalPendienteO3aM38) ||
		caso == "plazo" && !errors.Is(resultado.retirada.primera, errPlazoO3aM38) {
		return errInvarianteO3aM38
	}
	return nil
}

func probarInventarioAjenoO3aM38() (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil {
		return err
	}
	fd, err := syscall.Open("/dev/null", syscall.O_RDONLY, 0)
	if err != nil {
		return err
	}
	falloInventario := inventarioPreStartO3aM38(f.preparado.custodia)
	errCierre := syscall.Close(fd)
	if falloInventario == nil {
		return errInventarioO3aM38
	}
	return errCierre
}

func probarEntradaNulaO3aM38() error {
	resultado := prepararArranqueO3aM38(nil)
	if resultado.clase != resultadoRetiradoO3aM38 || resultado.retirada == nil ||
		resultado.retirada.origen != retiradaSinHijoO3aM38 || !errors.Is(resultado.retirada.primera, errEntradaO3aM38) {
		return errInvarianteO3aM38
	}
	return nil
}

func probarAliasPreparadoO3aM38() (err error) {
	f, err := crearFixtureO3aM38("A01")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil {
		return err
	}
	alias := f.preparado
	testigo := f.preparado.custodia.reloj.emitir()
	resultado := avanzarArranqueO3aM38(f.preparado, testigo)
	f.preparado = nil
	if resultado.clase == resultadoEntregadoO3aM38 {
		f.agregado = resultado.agregado
	} else {
		f.retirada = resultado.retirada
	}
	repetido := avanzarArranqueO3aM38(alias, alias.custodia.reloj.emitir())
	if repetido.clase != resultadoRetiradoO3aM38 || repetido.retirada == nil || repetido.retirada.origen != retiradaUsoConsumidoO3aM38 {
		return errUsoConsumidoO3aM38
	}
	return nil
}

func probarAliasFisicoO3aM38() (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	var alias *os.File
	defer func() {
		err = errors.Join(err, limpiarFixtureO3aM38(f))
		if alias != nil {
			err = errors.Join(err, alias.Close())
		}
	}()
	fd, _, errno := syscall.Syscall(syscall.SYS_FCNTL, f.entrada.salida.Fd(), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)
	if errno != 0 {
		return errno
	}
	alias = os.NewFile(fd, "alias-salida-o3a")
	f.entrada.baseline, err = snapshotActualO3aM38()
	if err != nil {
		return err
	}
	resultado := prepararArranqueO3aM38(&f.entrada)
	if resultado.clase != resultadoRetiradoO3aM38 || resultado.retirada == nil || !errors.Is(resultado.retirada.primera, errFormaFDO3aM38) {
		return errInvarianteO3aM38
	}
	f.retirada = resultado.retirada
	return nil
}

func probarCierreAdversoO3aM38() (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil {
		return err
	}
	c := f.preparado.custodia
	victima := c.destinados[0]
	if victima == nil || syscall.Close(int(victima.Fd())) != nil {
		return errInvarianteO3aM38
	}
	cerrado, fallo := cerrarUnoConLeaseO3aM38(c.lease, victima, operacionCerrarDestinosO3aM38)
	if !cerrado || fallo == nil {
		return errInvarianteO3aM38
	}
	c.destinados[0] = nil
	return nil
}

func probarObservadorLinealO3aM38() error {
	r := nuevoRegistroAutoridadO3aM38()
	o := nuevoObservadorSenalO3aM38(r)
	baseline, _, ok := o.observar()
	if !ok || !o.anotar(syscall.SIGUSR1) {
		return errSenalPendienteO3aM38
	}
	if _, transferido := o.transferirCritico(baseline); transferido || o.liberar() != nil {
		return errSenalPendienteO3aM38
	}
	o = nuevoObservadorSenalO3aM38(r)
	baseline, _, ok = o.observar()
	nuevo, transferido := o.transferirCritico(baseline)
	if !ok || !transferido || !o.anotar(syscall.SIGUSR2) {
		return errSenalPendienteO3aM38
	}
	actual, signo, ok := o.observar()
	if !ok || actual == nuevo || signo != syscall.SIGUSR2 {
		return errSenalPendienteO3aM38
	}
	return o.liberar()
}

func probarOverflowLeaseO3aM38() error {
	r := nuevoRegistroAutoridadO3aM38()
	l := nuevaLeaseGuardiaO3aM38(r)
	s, err := snapshotActualO3aM38()
	if err != nil || !l.sellarFisico(s) {
		return errAutoridadO3aM38
	}
	l.secuencia = ^uint64(0)
	if _, ok := l.comenzarCritico(operacionStartO3aM38, 1); ok || l.estado.Load() != 1 {
		return errAutoridadO3aM38
	}
	return l.liberar()
}

func probarMutacionDestinoO3aM38() (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil {
		return err
	}
	c := f.preparado.custodia
	if _, err = c.destinados[2].Seek(1, 0); err != nil || revalidarDestinadosO3aM38(c) == nil {
		return errInvarianteO3aM38
	}
	_, err = c.destinados[2].Seek(0, 0)
	return err
}

func autoprobarArranqueO3aM38() error {
	inicial, err := contarFD()
	if err != nil || !sinHijos() {
		return errors.New("estado inicial O3a no aislado")
	}
	if err = probarPreflightO3aM38(); err != nil {
		return err
	}
	if err = probarTestigosO3aM38(); err != nil {
		return err
	}
	if err = probarEntradaNulaO3aM38(); err != nil {
		return err
	}
	if err = probarPreparacionSinStartO3aM38(); err != nil {
		return err
	}
	if err = probarSelectoresO3aM38(); err != nil {
		return err
	}
	for i := 0; i < 100; i++ {
		if err = probarArranqueRealO3aM38(); err != nil {
			return fmt.Errorf("iteración O3a %d: %w", i, err)
		}
	}
	if err = probarAliasPreparadoO3aM38(); err != nil {
		return err
	}
	for _, caso := range []string{"parcial", "cancelar", "eof", "senal", "plazo"} {
		if err = probarBarreraAdversaO3aM38(caso); err != nil {
			return fmt.Errorf("barrera %s: %w", caso, err)
		}
	}
	if err = probarInventarioAjenoO3aM38(); err != nil {
		return err
	}
	for _, prueba := range []func() error{probarAliasFisicoO3aM38, probarCierreAdversoO3aM38, probarObservadorLinealO3aM38, probarOverflowLeaseO3aM38, probarMutacionDestinoO3aM38} {
		if err = prueba(); err != nil {
			return err
		}
	}
	final, err := contarFD()
	if err != nil || final != inicial || !sinHijos() {
		return fmt.Errorf("residuos O3a: fd=%d/%d", final, inicial)
	}
	return nil
}

// ejecutarCasoExternoO3aM38 expone únicamente sondas test-only cerradas. Los
// casos AF no escriben: el conductor exterior acredita exit 65 y EOF.
func ejecutarCasoExternoO3aM38(caso string) int {
	switch caso {
	case "DUPFD_FALLO":
		_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, ^uintptr(0), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)
		if errno == syscall.EBADF {
			return 0
		}
		return estadoErrorExternoO3aM38
	case "AF_DIRECTO":
		fatalO3aM38()
	case "AF_PIDFD_PRIMARIO_CERRADO", "AF_PIDFD_RESERVA_CERRADA", "AF_PIDFD_AMBOS_CERRADOS":
		return ejecutarDegradacionPidfdExternaO3aM38(caso)
	case "C10_DUPFD_POST_START", "TUPLA_A", "TUPLA_C":
		return ejecutarTuplaExternaO3aM38(caso)
	case "TUPLA_B", "C08_CUARTA", "C08_FLAG":
		return ejecutarTuplaBExternaO3aM38()
	case "LOW_HOLE", "C08_BARRIDO":
		return ejecutarLowHoleExternoO3aM38()
	case "C11_TERMINAL_POST_START", "C14_EOF", "C14_KILL", "C12_CANCELAR_TERMINAL", "C12_EOF_TERMINAL",
		"C19_EINTR", "C19_PRESUPUESTO", "C19_POLL", "C02_SENAL_PRESTART", "C02_SENAL_POSTSTART", "C02_SENAL_PREHANDOFF":
		return ejecutarRetiradaPostStartExternaO3aM38(caso)
	case "C20_PDEATH_CREADOR", "C20_PDEATH_OTRO":
		return ejecutarPdeathsigExternoO3aM38(caso)
	case "C15_AF":
		fatalO3aM38()
	case "C16_ALIAS", "C17_TESTIGOS_TID", "C18_BORDES":
		return ejecutarCasosLinealesExternosO3aM38(caso)
	case "C21_CIEN_INVENTARIOS":
		return ejecutarCienInventariosExternosO3aM38()
	case "C01_PREFLIGHT", "C02_SENAL_HANDOFF", "C03_SELECTORES", "C04_MAPA_HERENCIA", "C05_LEASE_COPIAS", "C06_TERMINAL", "C07_TICKET_EOF":
		return ejecutarCasosBaseExternosO3aM38(caso)
	}
	return estadoUso
}

func ejecutarCasosLinealesExternosO3aM38(caso string) int {
	if prepararNetpoll() != nil {
		return estadoNetpollExternoO3aM38
	}
	runtime.LockOSThread()
	if activarSubreaper() != nil {
		return estadoSubreaperExternoO3aM38
	}
	var err error
	switch caso {
	case "C16_ALIAS":
		err = probarAliasPreparadoO3aM38()
		if err == nil {
			err = probarAliasRetiradaExternaO3aM38()
		}
	case "C17_TESTIGOS_TID":
		if err = probarTestigosO3aM38(); err == nil {
			reloj := nuevoRelojVueltaM38()
			testigo := reloj.emitir()
			runtime.LockOSThread()
			resultado := make(chan error, 1)
			go func() { _, fallo := reloj.consumir(testigo, 0); resultado <- fallo }()
			err = <-resultado
			if !errors.Is(err, errTestigoO3aM38) {
				err = errTestigoO3aM38
			} else {
				err = nil
			}
		}
	case "C18_BORDES":
		for _, borde := range []string{"parcial", "plazo"} {
			if err = probarBarreraAdversaO3aM38(borde); err != nil {
				break
			}
		}
		if err == nil {
			err = probarVueltaTardiaExternaO3aM38()
		}
	default:
		err = errEntradaO3aM38
	}
	if err != nil || !sinHijos() {
		return estadoErrorExternoO3aM38
	}
	return 0
}

func probarAliasRetiradaExternaO3aM38() (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil || escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|"+string(f.preparado.custodia.control.nonce[:])+"|CANCELADO|65\n") != nil {
		return err
	}
	alias := f.preparado
	resultado := avanzarArranqueO3aM38(f.preparado, f.preparado.custodia.reloj.emitir())
	f.preparado, f.retirada = nil, resultado.retirada
	repetido := avanzarArranqueO3aM38(alias, alias.custodia.reloj.emitir())
	if resultado.clase != resultadoRetiradoO3aM38 || repetido.clase != resultadoRetiradoO3aM38 ||
		repetido.retirada == nil || repetido.retirada.origen != retiradaUsoConsumidoO3aM38 {
		return errUsoConsumidoO3aM38
	}
	return nil
}

func probarVueltaTardiaExternaO3aM38() (err error) {
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, limpiarFixtureO3aM38(f)) }()
	if err = prepararFixtureO3aM38(f); err != nil || escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|") != nil {
		return err
	}
	c := f.preparado.custodia
	primero := avanzarArranqueO3aM38(f.preparado, c.reloj.emitir())
	if primero.clase != resultadoAplazadoO3aM38 || escribirControlPruebaO3aM38(f, string(c.control.nonce[:])+"|CANCELADO|65\n") != nil {
		return errInvarianteO3aM38
	}
	segundo := avanzarArranqueO3aM38(primero.preparado, c.reloj.emitir())
	f.preparado, f.retirada = nil, segundo.retirada
	if segundo.clase != resultadoRetiradoO3aM38 || segundo.retirada == nil || segundo.retirada.causa.causa != "CANCELADO" {
		return errInvarianteO3aM38
	}
	return nil
}
