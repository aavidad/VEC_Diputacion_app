//go:build ignore && linux && amd64

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type fixtureO3aM38 struct {
	directorio        string
	controlEscritor   *os.File
	entrada           bundleEntradaO3aM38
	preparado         *preparadoO3aM38
	agregado          *agregadoO3aM38
	retirada          *retiradaO3aM38
	rlimitAnterior    syscall.Rlimit
	rlimitReducido    bool
	pdeathsigAnterior int32
	registro          *registroAutoridadO3aM38
	canalSenal        chan os.Signal
	pararSenal        chan struct{}
	senalParada       chan struct{}
}

func reducirRlimitPruebaO3aM38(f *fixtureO3aM38) error {
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &f.rlimitAnterior); err != nil {
		return err
	}
	if f.rlimitAnterior.Cur <= 256 {
		return nil
	}
	nuevo := f.rlimitAnterior
	nuevo.Cur = 256
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &nuevo); err != nil {
		return err
	}
	f.rlimitReducido = true
	return nil
}

func restaurarRlimitPruebaO3aM38(f *fixtureO3aM38) error {
	if !f.rlimitReducido {
		return nil
	}
	f.rlimitReducido = false
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &f.rlimitAnterior)
}

func fijarPdeathsigPruebaO3aM38(f *fixtureO3aM38) error {
	anterior, err := prctlO3aM38(2)
	if err != nil {
		return err
	}
	f.pdeathsigAnterior = anterior
	_, _, errno := syscall.Syscall6(syscall.SYS_PRCTL, 1, uintptr(syscall.SIGTERM), 0, 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func restaurarPdeathsigPruebaO3aM38(f *fixtureO3aM38) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_PRCTL, 1, uintptr(f.pdeathsigAnterior), 0, 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func abrirDirectorioO3aM38(ruta string) (*os.File, error) {
	fd, err := syscall.Open(ruta, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "raiz-prueba-o3a"), nil
}

func crearRegularPruebaO3aM38(ruta string, contenido []byte, lectura bool) (*os.File, error) {
	fd, err := syscall.Open(ruta, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), filepath.Base(ruta))
	if len(contenido) > 0 {
		n, fallo := f.Write(contenido)
		if fallo != nil || n != len(contenido) {
			_ = f.Close()
			return nil, errFormaFDO3aM38
		}
	}
	if !lectura {
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			_ = f.Close()
			return nil, err
		}
		return f, nil
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	fd, err = syscall.Open(ruta, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), filepath.Base(ruta)), nil
}

func leerRunnerPruebaO3aM38() ([]byte, error) {
	const ruta = "deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh"
	f, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > 64<<10 {
		return nil, errFormaFDO3aM38
	}
	contenido, err := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	if err != nil || int64(len(contenido)) != info.Size() {
		return nil, errFormaFDO3aM38
	}
	return contenido, nil
}

func nuevoPipePruebaO3aM38(nombre string, escritorAbierto bool) (*os.File, *os.File, error) {
	var fd [2]int
	if err := syscall.Pipe2(fd[:], syscall.O_CLOEXEC|syscall.O_NONBLOCK); err != nil {
		return nil, nil, err
	}
	lector := os.NewFile(uintptr(fd[0]), nombre+"-lector")
	escritor := os.NewFile(uintptr(fd[1]), nombre+"-escritor")
	if !escritorAbierto {
		if err := escritor.Close(); err != nil {
			_ = lector.Close()
			return nil, nil, err
		}
		escritor = nil
	}
	return lector, escritor, nil
}

func controladorSelectorPruebaO3aM38(selector string) (*controladorPreinicioM38, *relojVueltaM38, uint64, error) {
	receptor, err := nuevoReceptorSobreS0M38()
	if err != nil {
		return nil, nil, 0, err
	}
	texto := textoSobreS0M38("37", selector, "ticket|prueba")
	if n, resultado, fallo := receptor.consumir([]byte(texto), true); fallo != nil || n != len(texto) || resultado != recepcionSobreConfirmadaM38 {
		return nil, nil, 0, errControlO3aM38
	}
	control, err := nuevoControladorPreinicioM38(receptor)
	if err != nil {
		return nil, nil, 0, err
	}
	h := strings.Repeat("a", 64)
	armar := []byte("V1|CONTROL|ARMAR|" + h + "|37\n")
	if n, resultado, fallo := control.consumir(armar, false); fallo != nil || n != len(armar) || resultado != controlPreinicioArmadoM38 {
		return nil, nil, 0, errControlO3aM38
	}
	reloj := nuevoRelojVueltaM38()
	testigo := reloj.emitir()
	iniciar := []byte("V1|CONTROL|INICIAR|" + h + "\n")
	n, resultado, vuelta, err := consumirControlO3aM38(reloj, testigo, control, iniciar, false)
	if err != nil || n != len(iniciar) || resultado != controlPreinicioInicioPendienteM38 || vuelta == 0 {
		return nil, nil, 0, errControlO3aM38
	}
	return control, reloj, vuelta, nil
}

func sellarRaizPruebaO3aM38(f *os.File) (formaRaizM38, error) {
	forma, err := formaArchivoO3aM38(f, false)
	return formaRaizM38{identidad: forma}, err
}

func sellarRunnerPruebaO3aM38(f *os.File) (formaRunnerM38, error) {
	forma, err := formaArchivoO3aM38(f, true)
	if err != nil {
		return formaRunnerM38{}, err
	}
	h, err := hashRunnerO3aM38(f, forma.tamano)
	return formaRunnerM38{identidad: forma, sha256: h}, err
}

func crearFixtureO3aM38(selector string) (f *fixtureO3aM38, err error) {
	f = &fixtureO3aM38{}
	defer func() {
		if err != nil {
			_ = limpiarFixtureO3aM38(f)
		}
	}()
	if !selectorRunnerO3aM38(selector) {
		return f, errEntradaO3aM38
	}
	f.registro = nuevoRegistroAutoridadO3aM38()
	preflight, err := preflightPidfdGoM38(f.registro)
	if err != nil {
		return f, err
	}
	if err = reducirRlimitPruebaO3aM38(f); err != nil {
		return f, err
	}
	if err = fijarPdeathsigPruebaO3aM38(f); err != nil {
		return f, err
	}
	f.directorio, err = os.MkdirTemp("", "o3a-prueba-")
	if err != nil {
		return f, err
	}
	if err = os.Chmod(f.directorio, 0700); err != nil {
		return f, err
	}
	contenido, err := leerRunnerPruebaO3aM38()
	if err != nil {
		return f, err
	}
	raiz, err := abrirDirectorioO3aM38(".")
	if err != nil {
		return f, err
	}
	runner, err := crearRegularPruebaO3aM38(filepath.Join(f.directorio, "runner"), contenido, true)
	if err != nil {
		return f, err
	}
	salida, err := crearRegularPruebaO3aM38(filepath.Join(f.directorio, "salida"), nil, false)
	if err != nil {
		return f, err
	}
	errorCaso, err := crearRegularPruebaO3aM38(filepath.Join(f.directorio, "error"), nil, false)
	if err != nil {
		return f, err
	}
	terminal, err := crearRegularPruebaO3aM38(filepath.Join(f.directorio, "terminal"), nil, false)
	if err != nil {
		return f, err
	}
	controlLector, controlEscritor, err := nuevoPipePruebaO3aM38("control", true)
	if err != nil {
		return f, err
	}
	sobreLector, _, err := nuevoPipePruebaO3aM38("sobre", false)
	if err != nil {
		return f, err
	}
	control, reloj, vuelta, err := controladorSelectorPruebaO3aM38(selector)
	if err != nil {
		return f, err
	}
	formaRaiz, err := sellarRaizPruebaO3aM38(raiz)
	if err != nil {
		return f, err
	}
	formaRunner, err := sellarRunnerPruebaO3aM38(runner)
	if err != nil {
		return f, err
	}
	lease := nuevaLeaseGuardiaO3aM38(f.registro)
	observador := nuevoObservadorSenalO3aM38(f.registro)
	if lease == nil || observador == nil {
		return f, errAutoridadO3aM38
	}
	f.canalSenal, f.pararSenal, f.senalParada = make(chan os.Signal, 1), make(chan struct{}), make(chan struct{})
	signal.Notify(f.canalSenal, syscall.SIGUSR1)
	go func() {
		defer close(f.senalParada)
		for {
			select {
			case s := <-f.canalSenal:
				if v, ok := s.(syscall.Signal); ok {
					observador.anotar(v)
				}
			case <-f.pararSenal:
				return
			}
		}
	}()
	baselineSenal, _, valido := observador.observar()
	if !valido {
		return f, errAutoridadO3aM38
	}
	f.controlEscritor = controlEscritor
	f.entrada = bundleEntradaO3aM38{
		control: control, raiz: raiz, runner: runner, salida: salida, errorCaso: errorCaso,
		controlFD: controlLector, terminal: terminal, sobre: sobreLector,
		formaRaiz: formaRaiz, formaRunner: formaRunner, reloj: reloj, vueltaInicio: vuelta,
		preflight: preflight, lease: lease, observador: observador, baselineSenal: baselineSenal,
		finBootstrap: time.Now().Add(6 * time.Second), tid: syscall.Gettid(), ppid: os.Getppid(),
	}
	f.entrada.baseline, err = snapshotActualO3aM38()
	if err != nil {
		return f, err
	}
	return f, nil
}

func prepararFixtureO3aM38(f *fixtureO3aM38) error {
	resultado := prepararArranqueO3aM38(&f.entrada)
	if resultado.clase != resultadoPreparadoO3aM38 || resultado.preparado == nil || resultado.retirada != nil || resultado.agregado != nil {
		if resultado.retirada != nil {
			return fmt.Errorf("preparación O3a discrepante: %d: %w", resultado.clase, resultado.retirada.primera)
		}
		return fmt.Errorf("preparación O3a discrepante: %d", resultado.clase)
	}
	f.preparado = resultado.preparado
	return nil
}

func avanzarFixtureO3aM38(f *fixtureO3aM38) error {
	if f.preparado == nil {
		return errEntradaO3aM38
	}
	testigo := f.preparado.custodia.reloj.emitir()
	resultado := avanzarArranqueO3aM38(f.preparado, testigo)
	f.preparado = nil
	switch resultado.clase {
	case resultadoEntregadoO3aM38:
		f.agregado = resultado.agregado
		return nil
	case resultadoRetiradoO3aM38:
		f.retirada = resultado.retirada
		return fmt.Errorf("arranque O3a retirado: %v", resultado.retirada.primera)
	default:
		return errors.New("resultado O3a no terminal en prueba positiva")
	}
}

func prepararCasoExternoO3aM38() (*fixtureO3aM38, error) {
	if err := prepararNetpoll(); err != nil {
		return nil, err
	}
	runtime.LockOSThread()
	if err := activarSubreaper(); err != nil {
		return nil, err
	}
	f, err := crearFixtureO3aM38("NOMINAL")
	if err == nil {
		err = prepararFixtureO3aM38(f)
	}
	return f, err
}

func ejecutarDegradacionPidfdExternaO3aM38(caso string) int {
	f, err := prepararCasoExternoO3aM38()
	if err != nil {
		return estadoErrorExternoO3aM38
	}
	c := f.preparado.custodia
	resultado := avanzarArranqueO3aM38(f.preparado, c.reloj.emitir())
	f.preparado = nil
	f.retirada = resultado.retirada
	if caso == "AF_PIDFD_AMBOS_CERRADOS" || resultado.clase != resultadoRetiradoO3aM38 ||
		resultado.retirada == nil || resultado.retirada.origen != retiradaConHijoO3aM38 || limpiarFixtureO3aM38(f) != nil {
		return estadoErrorExternoO3aM38
	}
	return estadoRetiradaConHijoExternaO3aM38
}

func ejecutarTuplaExternaO3aM38(caso string) int {
	if prepararNetpoll() != nil {
		return estadoErrorExternoO3aM38
	}
	inicial, err := contarFD()
	f, errPreparacion := prepararCasoExternoO3aM38()
	if err != nil || errPreparacion != nil {
		return estadoErrorExternoO3aM38
	}
	c := f.preparado.custodia
	resultado := avanzarArranqueO3aM38(f.preparado, c.reloj.emitir())
	f.preparado = nil
	valido := caso == "TUPLA_C" && resultado.clase == resultadoEntregadoO3aM38 ||
		caso == "TUPLA_A" && resultado.clase == resultadoRetiradoO3aM38 && resultado.retirada.origen == retiradaSinHijoO3aM38 ||
		caso == "C10_DUPFD_POST_START" && resultado.clase == resultadoRetiradoO3aM38 && resultado.retirada.origen == retiradaConHijoO3aM38
	f.agregado, f.retirada = resultado.agregado, resultado.retirada
	if !valido || limpiarFixtureO3aM38(f) != nil {
		return estadoErrorExternoO3aM38
	}
	final, err := contarFD()
	if err != nil || final != inicial || !sinHijos() {
		return estadoErrorExternoO3aM38
	}
	if caso == "TUPLA_A" {
		return estadoRetiradaSinHijoExternaO3aM38
	}
	if caso == "C10_DUPFD_POST_START" {
		return estadoRetiradaConHijoExternaO3aM38
	}
	return 0
}

func ejecutarTuplaBExternaO3aM38() int {
	f, err := prepararCasoExternoO3aM38()
	if err != nil {
		return estadoErrorExternoO3aM38
	}
	c := f.preparado.custodia
	avanzarArranqueO3aM38(f.preparado, c.reloj.emitir())
	return estadoErrorExternoO3aM38
}

func ejecutarRetiradaPostStartExternaO3aM38(caso string) int {
	f, err := prepararCasoExternoO3aM38()
	if err != nil {
		return estadoErrorExternoO3aM38
	}
	c := f.preparado.custodia
	resultado := avanzarArranqueO3aM38(f.preparado, c.reloj.emitir())
	f.preparado = nil
	f.retirada = resultado.retirada
	if caso == "C12_EOF_TERMINAL" {
		f.controlEscritor = nil
	}
	precedenciaValida := caso != "C12_CANCELAR_TERMINAL" && caso != "C12_EOF_TERMINAL" ||
		resultado.retirada != nil && resultado.retirada.causa.causa == "CANCELADO" && resultado.retirada.causa.estado == "65"
	if resultado.clase != resultadoRetiradoO3aM38 || resultado.retirada == nil ||
		resultado.retirada.origen != retiradaConHijoO3aM38 || c.cmd == nil || c.cmd.ProcessState == nil ||
		!precedenciaValida || limpiarFixtureO3aM38(f) != nil || !sinHijos() {
		return estadoErrorExternoO3aM38
	}
	return estadoRetiradaConHijoExternaO3aM38
}

func ejecutarLowHoleExternoO3aM38() int {
	if prepararNetpoll() != nil {
		return estadoErrorExternoO3aM38
	}
	runtime.LockOSThread()
	if activarSubreaper() != nil {
		return estadoErrorExternoO3aM38
	}
	for fd := 3; fd <= 9; fd++ {
		if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0); errno != 0 {
			return estadoErrorExternoO3aM38
		}
	}
	f, err := crearFixtureO3aM38("NOMINAL")
	if err != nil {
		return estadoErrorExternoO3aM38
	}
	for _, archivo := range []*os.File{f.entrada.raiz, f.entrada.runner, f.entrada.salida, f.entrada.errorCaso, f.entrada.controlFD, f.entrada.terminal, f.entrada.sobre} {
		if archivo == nil || archivo.Fd() < 10 {
			return estadoErrorExternoO3aM38
		}
	}
	for fd := 3; fd <= 9; fd++ {
		if syscall.Close(fd) != nil {
			return estadoErrorExternoO3aM38
		}
	}
	f.entrada.baseline, err = snapshotActualO3aM38()
	if err != nil || prepararFixtureO3aM38(f) != nil || avanzarFixtureO3aM38(f) != nil || f.agregado == nil {
		return estadoErrorExternoO3aM38
	}
	c := f.agregado.custodia
	formas := make([]identidadFDO3aM38, 0, 3)
	for _, fd := range []int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco} {
		forma, fallo := identidadPidfdO3aM38(fd)
		if fallo != nil || forma.fdflags&syscall.FD_CLOEXEC == 0 {
			return estadoErrorExternoO3aM38
		}
		formas = append(formas, forma)
	}
	if !identidadFisicaO3aM38(formas[0], formas[1]) || !identidadFisicaO3aM38(formas[0], formas[2]) || limpiarFixtureO3aM38(f) != nil || !sinHijos() {
		return estadoErrorExternoO3aM38
	}
	return 0
}

func verificarLiteralComandoO3aM38(p *preparadoO3aM38, selector string) error {
	if p == nil || p.custodia == nil || p.custodia.cmd == nil {
		return errEntradaO3aM38
	}
	cmd := p.custodia.cmd
	esperados := []string{"/usr/bin/bash", "-p", "/proc/self/fd/8", "--caso-inyeccion-h0b", selector}
	if cmd.Path != "/usr/bin/bash" || cmd.Dir != "/" || len(cmd.Args) != len(esperados) || len(cmd.Env) != 2 ||
		cmd.Env[0] != "LC_ALL=C" || cmd.Env[1] != "PATH=/usr/local/go/bin:/usr/bin:/bin" || len(cmd.ExtraFiles) != 7 {
		return errEntradaO3aM38
	}
	for i := range esperados {
		if cmd.Args[i] != esperados[i] {
			return errEntradaO3aM38
		}
	}
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid || cmd.SysProcAttr.Pgid != 0 ||
		cmd.SysProcAttr.Pdeathsig != syscall.SIGKILL || cmd.SysProcAttr.PidFD == nil {
		return errEntradaO3aM38
	}
	return nil
}

func terminalidadPidfdPruebaO3aM38(fd int, fin time.Time) error {
	for time.Now().Before(fin) {
		vivo, err := pidfdVivoO3aM38(fd)
		if err != nil {
			return err
		}
		if !vivo {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	return errRetiradaO3aM38
}

func inventarioFixtureO3aM38(f *fixtureO3aM38) string {
	return fmt.Sprintf("directorio=%t control=%t preparado=%t agregado=%t retirada=%t", f.directorio != "", f.controlEscritor != nil, f.preparado != nil, f.agregado != nil, f.retirada != nil)
}

func punteroInt32PruebaO3aM38(v *int32) uintptr { return uintptr(unsafe.Pointer(v)) }

func ejecutarPdeathsigExternoO3aM38(caso string) int {
	if caso == "C20_PDEATH_CREADOR" {
		type respuesta struct {
			fixture *fixtureO3aM38
			fallo   error
		}
		canal := make(chan respuesta, 1)
		go func() {
			f, err := prepararCasoExternoO3aM38()
			if err == nil {
				err = avanzarFixtureO3aM38(f)
			}
			canal <- respuesta{fixture: f, fallo: err}
			syscall.RawSyscall(syscall.SYS_EXIT, 0, 0, 0)
			fatalO3aM38()
		}()
		r := <-canal
		if r.fallo != nil || r.fixture == nil || r.fixture.agregado == nil {
			return estadoErrorExternoO3aM38
		}
		pidfd := r.fixture.agregado.custodia.pidfdPrimario
		fin := time.Now().Add(time.Second)
		for time.Now().Before(fin) {
			if vivo, err := pidfdVivoO3aM38(pidfd); err == nil && !vivo {
				return estadoPdeathCreadorExternoO3aM38
			}
			time.Sleep(time.Millisecond)
		}
		return estadoPdeathAusenteExternoO3aM38
	}
	f, err := prepararCasoExternoO3aM38()
	if err != nil || avanzarFixtureO3aM38(f) != nil || f.agregado == nil {
		return estadoErrorExternoO3aM38
	}
	terminado := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		close(terminado)
	}()
	<-terminado
	time.Sleep(20 * time.Millisecond)
	vivo, err := pidfdVivoO3aM38(f.agregado.custodia.pidfdPrimario)
	if err != nil || !vivo || limpiarFixtureO3aM38(f) != nil || !sinHijos() {
		return estadoErrorExternoO3aM38
	}
	return estadoPdeathOtroExternoO3aM38
}

func ejecutarCienInventariosExternosO3aM38() int {
	for i := 0; i < 100; i++ {
		if estado := ejecutarTuplaExternaO3aM38("TUPLA_C"); estado != 0 {
			return estadoErrorExternoO3aM38
		}
	}
	return 0
}

func ejecutarCasosBaseExternosO3aM38(caso string) int {
	if caso == "C04_MAPA_HERENCIA" {
		return ejecutarLowHoleExternoO3aM38()
	}
	if prepararNetpoll() != nil {
		return estadoErrorExternoO3aM38
	}
	runtime.LockOSThread()
	if activarSubreaper() != nil {
		return estadoErrorExternoO3aM38
	}
	var err error
	switch caso {
	case "C01_PREFLIGHT":
		err = probarPreflightO3aM38()
		if err == nil && consumirPreflightPidfdGoM38(nil) {
			err = errPreflightPidfdO3aM38
		}
		if err == nil {
			registro := nuevoRegistroAutoridadO3aM38()
			token, fallo := preflightPidfdGoM38(registro)
			resultado := make(chan bool, 1)
			go func() { resultado <- consumirPreflightPidfdGoM38(token) }()
			if fallo != nil || <-resultado || !consumirPreflightPidfdGoM38(token) {
				err = errPreflightPidfdO3aM38
			}
		}
	case "C02_SENAL_HANDOFF":
		err = probarObservadorLinealO3aM38()
		if err == nil {
			err = probarBarreraAdversaO3aM38("senal")
		}
	case "C03_SELECTORES":
		err = probarSelectoresO3aM38()
	case "C05_LEASE_COPIAS":
		for _, prueba := range []func() error{probarInventarioAjenoO3aM38, probarCierreAdversoO3aM38, probarOverflowLeaseO3aM38, probarMutacionDestinoO3aM38} {
			if err = prueba(); err != nil {
				break
			}
		}
	case "C06_TERMINAL", "C07_TICKET_EOF":
		err = probarArranqueRealO3aM38()
	default:
		err = errEntradaO3aM38
	}
	if err != nil || !sinHijos() {
		return estadoErrorExternoO3aM38
	}
	return 0
}
