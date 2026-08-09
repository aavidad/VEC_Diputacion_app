//go:build ignore && linux && amd64

package main

import (
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func formaArchivoO3aM38(f *os.File, conOffset bool) (identidadFDO3aM38, error) {
	if f == nil || f.Fd() > uintptr(^uint(0)>>1) {
		return identidadFDO3aM38{}, errFormaFDO3aM38
	}
	fd := int(f.Fd())
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
	offset := int64(-1)
	if conOffset {
		var err error
		offset, err = f.Seek(0, io.SeekCurrent)
		if err != nil {
			return identidadFDO3aM38{}, err
		}
	}
	return identidadFDO3aM38{
		dev: uint64(st.Dev), ino: st.Ino, rdev: uint64(st.Rdev), modo: st.Mode,
		uid: st.Uid, enlaces: st.Nlink, tamano: st.Size, offset: offset,
		flags: int(flags), fdflags: int(fdflags), offsetValido: conOffset,
	}, nil
}

func identidadFisicaO3aM38(a, b identidadFDO3aM38) bool {
	return a.dev == b.dev && a.ino == b.ino && a.modo&syscall.S_IFMT == b.modo&syscall.S_IFMT
}

func formaExactaO3aM38(obtenida, sellada identidadFDO3aM38) bool {
	return obtenida == sellada
}

func hashRunnerO3aM38(f *os.File, tamano int64) ([32]byte, error) {
	var cero [32]byte
	if tamano < 0 || tamano > 1<<20 {
		return cero, errFormaFDO3aM38
	}
	h := sha256.New()
	lector := io.NewSectionReader(f, 0, tamano)
	if n, err := io.CopyBuffer(h, lector, make([]byte, 32<<10)); err != nil || n != tamano {
		return cero, errFormaFDO3aM38
	}
	copy(cero[:], h.Sum(nil))
	return cero, nil
}

func validarRegularO3aM38(f *os.File, acceso int, vacio bool) (identidadFDO3aM38, error) {
	forma, err := formaArchivoO3aM38(f, true)
	flagsEsperados := acceso | 0x8000 | syscall.O_NOFOLLOW
	if err != nil || forma.modo&syscall.S_IFMT != syscall.S_IFREG || forma.uid != uint32(os.Geteuid()) ||
		forma.modo&07777 != 0600 || forma.enlaces != 1 || forma.offset != 0 ||
		forma.flags != flagsEsperados ||
		forma.fdflags&syscall.FD_CLOEXEC == 0 || vacio && forma.tamano != 0 {
		return identidadFDO3aM38{}, errFormaFDO3aM38
	}
	return forma, nil
}

func validarPipeO3aM38(f *os.File, exigeEOF bool) (identidadFDO3aM38, error) {
	forma, err := formaArchivoO3aM38(f, false)
	if err != nil || forma.modo&syscall.S_IFMT != syscall.S_IFIFO || forma.uid != uint32(os.Geteuid()) ||
		forma.modo&07777 != 0600 || forma.enlaces != 1 || forma.flags&syscall.O_ACCMODE != syscall.O_RDONLY ||
		forma.flags != syscall.O_NONBLOCK || forma.fdflags&syscall.FD_CLOEXEC == 0 {
		return identidadFDO3aM38{}, errFormaFDO3aM38
	}
	if exigeEOF {
		var b [1]byte
		n, e := syscall.Read(int(f.Fd()), b[:])
		if e != nil || n != 0 {
			return identidadFDO3aM38{}, errFormaFDO3aM38
		}
	}
	return forma, nil
}

func validarRaizO3aM38(f *os.File, sellada formaRaizM38) (identidadFDO3aM38, error) {
	forma, err := formaArchivoO3aM38(f, false)
	if err != nil || forma.modo&syscall.S_IFMT != syscall.S_IFDIR || forma.uid != uint32(os.Geteuid()) ||
		forma.flags != sellada.identidad.flags || forma.flags&syscall.O_ACCMODE != syscall.O_RDONLY || forma.fdflags&syscall.FD_CLOEXEC == 0 ||
		!identidadFisicaO3aM38(forma, sellada.identidad) || forma.uid != sellada.identidad.uid ||
		forma.modo != sellada.identidad.modo || forma.enlaces != sellada.identidad.enlaces {
		return identidadFDO3aM38{}, errFormaFDO3aM38
	}
	return forma, nil
}

func validarRunnerO3aM38(f *os.File, sellada formaRunnerM38) (identidadFDO3aM38, error) {
	forma, err := validarRegularO3aM38(f, syscall.O_RDONLY, false)
	if err != nil || !formaExactaO3aM38(forma, sellada.identidad) {
		return identidadFDO3aM38{}, errFormaFDO3aM38
	}
	h, err := hashRunnerO3aM38(f, forma.tamano)
	if err != nil || h != sellada.sha256 {
		return identidadFDO3aM38{}, errFormaFDO3aM38
	}
	return forma, nil
}

func snapshotActualO3aM38() (snapshotFDO3aM38, error) {
	var limite syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limite); err != nil || limite.Cur < minFDDuplicadoM38 || limite.Cur > maxFDInspeccionM38 {
		return snapshotFDO3aM38{}, errInventarioO3aM38
	}
	s := snapshotFDO3aM38{limite: limite.Cur, mapa: make(map[int]huellaFDO3aM38)}
	for fd := 0; fd < int(limite.Cur); fd++ {
		flags, _, e := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0)
		if e == syscall.EBADF {
			continue
		}
		if e != 0 {
			return snapshotFDO3aM38{}, e
		}
		getfl, _, e := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFL, 0)
		if e != 0 {
			return snapshotFDO3aM38{}, e
		}
		var st syscall.Stat_t
		if err := syscall.Fstat(fd, &st); err != nil {
			return snapshotFDO3aM38{}, err
		}
		offset, _, eOffset := syscall.Syscall(syscall.SYS_LSEEK, uintptr(fd), 0, io.SeekCurrent)
		offsetValido := eOffset == 0
		if eOffset != 0 && eOffset != syscall.ESPIPE {
			return snapshotFDO3aM38{}, eOffset
		}
		s.mapa[fd] = huellaFDO3aM38{abierto: true, identidad: identidadFDO3aM38{
			dev: uint64(st.Dev), ino: st.Ino, rdev: uint64(st.Rdev), modo: st.Mode,
			uid: st.Uid, enlaces: st.Nlink, tamano: st.Size, offset: int64(offset), offsetValido: offsetValido,
			flags: int(getfl), fdflags: int(flags),
		}}
	}
	for fd := 0; fd <= 2; fd++ {
		if _, existe := s.mapa[fd]; !existe {
			return snapshotFDO3aM38{}, errInventarioO3aM38
		}
	}
	return s, nil
}

func sellarDestinadosO3aM38(archivos []*os.File) ([]identidadFDO3aM38, error) {
	if len(archivos) != 9 {
		return nil, errFormaFDO3aM38
	}
	formas := make([]identidadFDO3aM38, len(archivos))
	for i, f := range archivos {
		forma, err := formaArchivoO3aM38(f, i >= 1 && i <= 3)
		if err != nil {
			return nil, err
		}
		formas[i] = forma
	}
	return formas, nil
}

func todosCLOEXECO3aM38(s snapshotFDO3aM38) bool {
	for fd, h := range s.mapa {
		if fd >= 3 && h.identidad.fdflags&syscall.FD_CLOEXEC == 0 {
			return false
		}
	}
	return true
}

func escritoresSinAliasO3aM38(s snapshotFDO3aM38, archivos [3]*os.File, formas [3]identidadFDO3aM38) bool {
	for i, archivo := range archivos {
		if archivo == nil {
			return false
		}
		propio := int(archivo.Fd())
		for fd, huella := range s.mapa {
			if fd != propio && identidadFisicaO3aM38(huella.identidad, formas[i]) {
				return false
			}
		}
	}
	return true
}

func consolidarOperacionFisicaO3aM38(l *leaseGuardiaO3aM38, p permisoGuardiaO3aM38, actual snapshotFDO3aM38) (bool, bool) {
	if l == nil || !l.permisoValido(p) || actual.limite != l.pre.limite {
		return false, false
	}
	altas, bajas := 0, 0
	for fd, huella := range actual.mapa {
		anterior, existe := l.pre.mapa[fd]
		if existe && anterior != huella {
			return false, false
		}
		if !existe {
			if huella.identidad.fdflags&syscall.FD_CLOEXEC == 0 {
				return false, false
			}
			altas++
		}
	}
	for fd := range l.pre.mapa {
		if _, existe := actual.mapa[fd]; !existe {
			if fd != p.objetivos[0] && fd != p.objetivos[1] {
				return false, false
			}
			bajas++
		}
	}
	aplicada := (p.operacion == operacionDuplicarO3aM38 || p.operacion == operacionAbrirO3aM38) && altas == 1 && bajas == 0 ||
		p.operacion == operacionPipeO3aM38 && altas == 2 && bajas == 0 ||
		(p.operacion == operacionCerrarTicketO3aM38 || p.operacion == operacionCerrarDestinosO3aM38 ||
			p.operacion == operacionWaitO3aM38 || p.operacion == operacionCerrarPidfdO3aM38) && altas == 0 && bajas == p.cardinalidad
	if aplicada {
		return true, l.consolidarFisico(p, actual, true)
	}
	if altas == 0 && bajas == 0 && snapshotsIgualesO3aM38(actual, l.pre) {
		return false, l.consolidarFisico(p, actual, false)
	}
	return false, false
}

func cerrarUnoConLeaseO3aM38(l *leaseGuardiaO3aM38, f *os.File, op operacionGuardiaO3aM38) (bool, error) {
	if f == nil {
		return false, errFormaFDO3aM38
	}
	p, ok := l.comenzar(op, 1, [2]int{int(f.Fd()), -1})
	if !ok {
		return false, errAutoridadO3aM38
	}
	errCierre := f.Close()
	actual, errInventario := snapshotActualO3aM38()
	aplicada, consolidada := consolidarOperacionFisicaO3aM38(l, p, actual)
	if errInventario != nil || !consolidada || errCierre == nil && !aplicada {
		fatalO3aM38()
	}
	return aplicada, errCierre
}

func duplicarArchivoO3aM38(l *leaseGuardiaO3aM38, f *os.File, nombre string) (*os.File, error) {
	p, ok := l.comenzar(operacionDuplicarO3aM38, 1, [2]int{-1, -1})
	if !ok {
		return nil, errAutoridadO3aM38
	}
	fd, _, e := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), syscall.F_DUPFD_CLOEXEC, minFDDuplicadoM38)
	actual, errInventario := snapshotActualO3aM38()
	aplicada, consolidada := consolidarOperacionFisicaO3aM38(l, p, actual)
	if errInventario != nil || !consolidada || e == 0 && !aplicada || e != 0 && aplicada {
		fatalO3aM38()
	}
	if e != 0 || fd > uintptr(^uint(0)>>1) {
		return nil, errFormaFDO3aM38
	}
	return os.NewFile(fd, nombre), nil
}

func abrirDevNullO3aM38(l *leaseGuardiaO3aM38, nombre string) (*os.File, error) {
	p, ok := l.comenzar(operacionAbrirO3aM38, 1, [2]int{-1, -1})
	if !ok {
		return nil, errAutoridadO3aM38
	}
	fd, err := syscall.Open("/dev/null", syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	actual, errInventario := snapshotActualO3aM38()
	aplicada, consolidada := consolidarOperacionFisicaO3aM38(l, p, actual)
	if errInventario != nil || !consolidada || err == nil && !aplicada || err != nil && aplicada {
		fatalO3aM38()
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), nombre)
	forma, err := formaArchivoO3aM38(f, false)
	if err != nil || forma.modo&syscall.S_IFMT != syscall.S_IFCHR || forma.rdev != 259 ||
		forma.flags&syscall.O_ACCMODE != syscall.O_RDONLY || forma.fdflags&syscall.FD_CLOEXEC == 0 {
		_ = f.Close()
		return nil, errFormaFDO3aM38
	}
	return f, nil
}

func nuevoTicketO3aM38(l *leaseGuardiaO3aM38) (*os.File, *os.File, error) {
	p, ok := l.comenzar(operacionPipeO3aM38, 2, [2]int{-1, -1})
	if !ok {
		return nil, nil, errAutoridadO3aM38
	}
	var fds [2]int
	err := syscall.Pipe2(fds[:], syscall.O_CLOEXEC)
	actual, errInventario := snapshotActualO3aM38()
	aplicada, consolidada := consolidarOperacionFisicaO3aM38(l, p, actual)
	if errInventario != nil || !consolidada || err == nil && !aplicada || err != nil && aplicada {
		fatalO3aM38()
	}
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[0]), "ticket-lector-o3a"), os.NewFile(uintptr(fds[1]), "ticket-escritor-o3a"), nil
}

func selectorRunnerO3aM38(s string) bool {
	if s == "NOMINAL" {
		return true
	}
	if len(s) != 3 {
		return false
	}
	switch s[0] {
	case 'A':
		return s >= "A01" && s <= "A03"
	case 'N', 'E':
		return s >= string([]byte{s[0], '0', '1'}) && s <= string([]byte{s[0], '1', '0'})
	case 'F':
		return s >= "F01" && s <= "F15"
	}
	return false
}

func cerrarArchivosO3aM38(l *leaseGuardiaO3aM38, archivos []*os.File) error {
	var fallos []error
	for i, f := range archivos {
		if f != nil {
			var err error
			if l != nil && l.fisico.mapa != nil {
				cerrado, fallo := cerrarUnoConLeaseO3aM38(l, f, operacionCerrarDestinosO3aM38)
				if !cerrado {
					fatalO3aM38()
				}
				err = fallo
			} else {
				err = f.Close()
			}
			fallos = append(fallos, err)
			archivos[i] = nil
		}
	}
	return errors.Join(fallos...)
}

func cerrarPidfdsPoseidosO3aM38(l *leaseGuardiaO3aM38, fds ...int) error {
	for _, fd := range fds {
		if _, err := identidadPidfdO3aM38(fd); errors.Is(err, syscall.EBADF) {
			continue
		} else if err != nil {
			return err
		}
		if cerrado, err := cerrarPidfdConLeaseO3aM38(l, fd); !cerrado || err != nil {
			return errors.Join(errInvarianteO3aM38, err)
		}
	}
	return nil
}

func retiradaSinHijoDesdeEntradaO3aM38(e bundleEntradaO3aM38, primera error) resultadoArranqueO3aM38 {
	falloCierre := cerrarArchivosO3aM38(e.lease, []*os.File{e.raiz, e.runner, e.salida, e.errorCaso, e.sobre})
	return resultadoArranqueO3aM38{clase: resultadoRetiradoO3aM38, retirada: &retiradaO3aM38{
		origen: retiradaSinHijoO3aM38, primera: primera, controlFD: e.controlFD,
		terminal: e.terminal, lease: e.lease, observador: e.observador, secundarios: []error{falloCierre},
	}}
}

func prepararArranqueO3aM38(entrada *bundleEntradaO3aM38) resultadoArranqueO3aM38 {
	if entrada == nil {
		return resultadoArranqueO3aM38{clase: resultadoRetiradoO3aM38, retirada: &retiradaO3aM38{origen: retiradaSinHijoO3aM38, primera: errEntradaO3aM38}}
	}
	e := *entrada
	*entrada = bundleEntradaO3aM38{}
	autoridad := nuevaAutoridadO3aM38()
	fallar := func(err error) resultadoArranqueO3aM38 {
		_ = autoridad.mover(arranqueA0ObservandoM38, arranqueA7RetiradoSinHijoM38)
		return retiradaSinHijoDesdeEntradaO3aM38(e, err)
	}
	if !consumirPreflightPidfdGoM38(e.preflight) || e.control == nil ||
		e.control.fase != controlPreinicioS3M38 || e.control.causa != (causaPreinicioM38{}) || e.control.fallo != nil ||
		e.control.lector == nil || e.control.lector.estado != lectorAbiertoVacioM38 || !lectorLimpioM38(e.control.lector) ||
		e.reloj == nil || e.vueltaInicio == 0 || !e.lease.valido() {
		return fallar(errEntradaO3aM38)
	}
	contadorSenal, _, observadorValido := e.observador.observar()
	if e.tid != syscall.Gettid() || e.ppid != os.Getppid() || !observadorValido || contadorSenal != e.baselineSenal || time.Until(e.finBootstrap) < 4*time.Second {
		return fallar(errEntradaO3aM38)
	}
	if sub, err := prctlO3aM38(37); err != nil || sub != 1 {
		return fallar(errSubreaperO3aM38)
	}
	if muerte, err := prctlO3aM38(2); err != nil || muerte != int32(syscall.SIGTERM) {
		return fallar(errPdeathsigO3aM38)
	}
	actual, err := snapshotActualO3aM38()
	if err != nil || !snapshotsIgualesO3aM38(actual, e.baseline) || !todosCLOEXECO3aM38(actual) {
		return fallar(errInventarioO3aM38)
	}
	formas := make([]identidadFDO3aM38, 7)
	formas[0], err = validarRaizO3aM38(e.raiz, e.formaRaiz)
	if err == nil {
		formas[1], err = validarRunnerO3aM38(e.runner, e.formaRunner)
	}
	if err == nil {
		formas[2], err = validarRegularO3aM38(e.salida, syscall.O_WRONLY, true)
	}
	if err == nil {
		formas[3], err = validarRegularO3aM38(e.errorCaso, syscall.O_WRONLY, true)
	}
	if err == nil {
		formas[4], err = validarPipeO3aM38(e.controlFD, false)
	}
	if err == nil {
		formas[5], err = validarRegularO3aM38(e.terminal, syscall.O_WRONLY, true)
	}
	if err == nil {
		formas[6], err = validarPipeO3aM38(e.sobre, true)
	}
	for i := range formas {
		for j := 0; j < i; j++ {
			if identidadFisicaO3aM38(formas[i], formas[j]) {
				err = errFormaFDO3aM38
			}
		}
	}
	if err != nil || !selectorRunnerO3aM38(e.control.recepcion.sobre.selector) {
		return fallar(errFormaFDO3aM38)
	}
	if !escritoresSinAliasO3aM38(actual, [3]*os.File{e.salida, e.errorCaso, e.terminal}, [3]identidadFDO3aM38{formas[2], formas[3], formas[5]}) || !e.lease.sellarFisico(actual) {
		return fallar(errFormaFDO3aM38)
	}
	raiz, er := duplicarArchivoO3aM38(e.lease, e.raiz, "raiz-hijo-o3a")
	runner, eu := duplicarArchivoO3aM38(e.lease, e.runner, "runner-hijo-o3a")
	salida, es := duplicarArchivoO3aM38(e.lease, e.salida, "salida-hijo-o3a")
	errorCaso, ee := duplicarArchivoO3aM38(e.lease, e.errorCaso, "error-hijo-o3a")
	if er != nil || eu != nil || es != nil || ee != nil {
		_ = cerrarArchivosO3aM38(e.lease, []*os.File{raiz, runner, salida, errorCaso})
		return fallar(errFormaFDO3aM38)
	}
	destinados := []*os.File{raiz, runner, salida, errorCaso}
	for i := 0; i < 5; i++ {
		nulo, fallo := abrirDevNullO3aM38(e.lease, "nulo-o3a")
		if fallo != nil {
			_ = cerrarArchivosO3aM38(e.lease, destinados)
			return fallar(fallo)
		}
		destinados = append(destinados, nulo)
	}
	ticketLector, ticketEscritor, err := nuevoTicketO3aM38(e.lease)
	if err != nil {
		_ = cerrarArchivosO3aM38(e.lease, destinados)
		return fallar(err)
	}
	for _, original := range []*os.File{e.raiz, e.runner, e.salida, e.errorCaso, e.sobre} {
		if cerrado, fallo := cerrarUnoConLeaseO3aM38(e.lease, original, operacionCerrarDestinosO3aM38); !cerrado || fallo != nil {
			fatalO3aM38()
		}
	}
	e.raiz, e.runner, e.salida, e.errorCaso, e.sobre = nil, nil, nil, nil, nil
	pidfdPrimario := -1
	cmd := &exec.Cmd{
		Path: "/usr/bin/bash", Args: []string{"/usr/bin/bash", "-p", "/proc/self/fd/8", "--caso-inyeccion-h0b", e.control.recepcion.sobre.selector},
		Env: []string{"LC_ALL=C", "PATH=/usr/local/go/bin:/usr/bin:/bin"}, Dir: "/",
		Stdin: destinados[4], Stdout: salida, Stderr: errorCaso,
		ExtraFiles:  []*os.File{destinados[5], destinados[6], destinados[7], destinados[8], raiz, runner, ticketLector},
		SysProcAttr: &syscall.SysProcAttr{Setpgid: true, Pgid: 0, Pdeathsig: syscall.SIGKILL, PidFD: &pidfdPrimario},
	}
	preparado := &custodiaO3aM38{
		autoridad: autoridad, control: e.control, controlFD: e.controlFD, terminal: e.terminal,
		lease: e.lease, observador: e.observador, baselineSenal: e.baselineSenal,
		reloj: e.reloj, vueltaInicio: e.vueltaInicio, finBootstrap: e.finBootstrap,
		tid: e.tid, ppid: e.ppid, cmd: cmd,
		pidfdPrimario: pidfdPrimario, pidfdReserva: -1, ticketLector: ticketLector,
		ticketEscritor: ticketEscritor, destinados: destinados, baseline: e.baseline,
		formaRaiz: e.formaRaiz, formaRunner: e.formaRunner,
	}
	preparado.huellasDestinadas, err = sellarDestinadosO3aM38(destinados)
	if err != nil {
		_ = cerrarArchivosO3aM38(e.lease, append(destinados, ticketLector, ticketEscritor))
		return retiradaSinHijoDesdeEntradaO3aM38(e, err)
	}
	preparado.snapshot, err = snapshotActualO3aM38()
	if err != nil || !todosCLOEXECO3aM38(preparado.snapshot) {
		_ = cerrarArchivosO3aM38(e.lease, append(destinados, ticketLector, ticketEscritor))
		return retiradaSinHijoDesdeEntradaO3aM38(e, errInventarioO3aM38)
	}
	if err = autoridad.mover(arranqueA0ObservandoM38, arranqueA1PreparadoM38); err != nil {
		_ = cerrarArchivosO3aM38(e.lease, append(destinados, ticketLector, ticketEscritor))
		return retiradaSinHijoDesdeEntradaO3aM38(e, err)
	}
	return resultadoArranqueO3aM38{clase: resultadoPreparadoO3aM38, preparado: &preparadoO3aM38{custodia: preparado}}
}
