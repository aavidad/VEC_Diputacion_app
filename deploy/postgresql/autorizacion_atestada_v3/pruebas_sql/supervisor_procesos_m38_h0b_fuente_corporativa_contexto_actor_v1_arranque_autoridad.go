//go:build ignore && linux && amd64

package main

import (
	"errors"
	"maps"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	maxFDInspeccionM38     = 1_048_576
	minFDDuplicadoM38      = 10
	duracionRetiradaO3aM38 = 3 * time.Second
	reservaO3bO3cM38       = time.Second
)

type estadoArranqueO3aM38 uint8

const (
	arranqueA0ObservandoM38 estadoArranqueO3aM38 = iota
	arranqueA1PreparadoM38
	arranqueA2AplazadoM38
	arranqueA3IniciandoM38
	arranqueA4ProvisionalM38
	arranqueA5PidfdTresM38
	arranqueA6EntregadoM38
	arranqueA7RetiradoSinHijoM38
	arranqueA8RetirandoConHijoM38
	arranqueA9RetiradoConHijoM38
	arranqueAFFatalM38
)

type claseResultadoO3aM38 uint8

const (
	resultadoPreparadoO3aM38 claseResultadoO3aM38 = iota + 1
	resultadoAplazadoO3aM38
	resultadoEntregadoO3aM38
	resultadoRetiradoO3aM38
)

type origenRetiradaO3aM38 uint8

const (
	retiradaSinHijoO3aM38 origenRetiradaO3aM38 = iota + 1
	retiradaConHijoO3aM38
	retiradaUsoConsumidoO3aM38
)

var (
	errAutoridadO3aM38      = errors.New("autoridad O3a inválida")
	errEntradaO3aM38        = errors.New("entrada O3a inválida")
	errFormaFDO3aM38        = errors.New("forma FD O3a inválida")
	errInventarioO3aM38     = errors.New("inventario FD O3a inválido")
	errTestigoO3aM38        = errors.New("testigo de vuelta O3a inválido")
	errPreflightPidfdO3aM38 = errors.New("preflight pidfd Go O3a inválido")
	errPlazoO3aM38          = errors.New("plazo O3a insuficiente")
	errControlO3aM38        = errors.New("control O3a no iniciable")
	errProcesoO3aM38        = errors.New("proceso O3a inconsistente")
	errUsoConsumidoO3aM38   = errors.New("custodia O3a ya consumida")
	errRetiradaO3aM38       = errors.New("retirada O3a no acreditada")
	errInvarianteO3aM38     = errors.New("invariante O3a incumplida")
	errAplazamientoO3aM38   = errors.New("arranque O3a aplazado")
	errSubreaperO3aM38      = errors.New("subreaper O3a no acreditado")
	errPdeathsigO3aM38      = errors.New("PDEATHSIG O3a no acreditada")
	errSenalPendienteO3aM38 = errors.New("señal pendiente antes del arranque O3a")
)

type autoridadEstadoO3aM38 struct {
	estado estadoArranqueO3aM38
}

func nuevaAutoridadO3aM38() *autoridadEstadoO3aM38 {
	return &autoridadEstadoO3aM38{estado: arranqueA0ObservandoM38}
}

func (a *autoridadEstadoO3aM38) mover(desde, hacia estadoArranqueO3aM38) error {
	if a == nil || a.estado != desde || !transicionO3aM38(desde, hacia) {
		return errAutoridadO3aM38
	}
	a.estado = hacia
	return nil
}

func (a *autoridadEstadoO3aM38) es(estado estadoArranqueO3aM38) bool {
	return a != nil && a.estado == estado
}

func transicionO3aM38(desde, hacia estadoArranqueO3aM38) bool {
	switch desde {
	case arranqueA0ObservandoM38:
		return hacia == arranqueA1PreparadoM38 || hacia == arranqueA7RetiradoSinHijoM38
	case arranqueA1PreparadoM38:
		return hacia == arranqueA2AplazadoM38 || hacia == arranqueA3IniciandoM38 || hacia == arranqueA7RetiradoSinHijoM38
	case arranqueA2AplazadoM38:
		return hacia == arranqueA2AplazadoM38 || hacia == arranqueA3IniciandoM38 || hacia == arranqueA7RetiradoSinHijoM38
	case arranqueA3IniciandoM38:
		return hacia == arranqueA4ProvisionalM38 || hacia == arranqueA7RetiradoSinHijoM38
	case arranqueA4ProvisionalM38:
		return hacia == arranqueA5PidfdTresM38 || hacia == arranqueA8RetirandoConHijoM38 || hacia == arranqueAFFatalM38
	case arranqueA5PidfdTresM38:
		return hacia == arranqueA6EntregadoM38 || hacia == arranqueA8RetirandoConHijoM38 || hacia == arranqueAFFatalM38
	case arranqueA8RetirandoConHijoM38:
		return hacia == arranqueA9RetiradoConHijoM38 || hacia == arranqueAFFatalM38
	}
	return false
}

type celdaVueltaM38 struct {
	auto     *celdaVueltaM38
	reloj    *relojVueltaM38
	contador uint64
	tid      int
	consumo  atomic.Uint32
}

type testigoVueltaM38 struct{ celda *celdaVueltaM38 }

type relojVueltaM38 struct {
	contador uint64
	tid      int
	registro map[*testigoVueltaM38]*celdaVueltaM38
}

func nuevoRelojVueltaM38() *relojVueltaM38 {
	return &relojVueltaM38{tid: syscall.Gettid(), registro: make(map[*testigoVueltaM38]*celdaVueltaM38)}
}

func (r *relojVueltaM38) emitir() *testigoVueltaM38 {
	if r == nil || r.tid != syscall.Gettid() || r.contador == ^uint64(0) {
		return nil
	}
	r.contador++
	c := &celdaVueltaM38{reloj: r, contador: r.contador, tid: r.tid}
	c.auto = c
	t := &testigoVueltaM38{celda: c}
	r.registro[t] = c
	return t
}

func (r *relojVueltaM38) consumir(t *testigoVueltaM38, posteriorA uint64) (uint64, error) {
	if r == nil || t == nil {
		return 0, errTestigoO3aM38
	}
	c := r.registro[t]
	if c == nil || t.celda != c || c.auto != c || c.reloj != r ||
		c.tid != r.tid || c.tid != syscall.Gettid() || c.contador <= posteriorA ||
		!c.consumo.CompareAndSwap(0, 1) {
		return 0, errTestigoO3aM38
	}
	return c.contador, nil
}

func consumirControlO3aM38(r *relojVueltaM38, t *testigoVueltaM38, c *controladorPreinicioM38, fragmento []byte, fin bool) (int, resultadoControlPreinicioM38, uint64, error) {
	n, resultado, err := c.consumir(fragmento, fin)
	if err != nil || resultado != controlPreinicioInicioPendienteM38 {
		return n, resultado, 0, err
	}
	vuelta, err := r.consumir(t, 0)
	return n, resultado, vuelta, err
}

type registroAutoridadO3aM38 struct {
	auto         *registroAutoridadO3aM38
	tid          int
	generacion   uint64
	preflight    map[*acreditacionPidfdGoM38]uint64
	leases       map[*leaseGuardiaO3aM38]uint64
	observadores map[*observadorSenalO3aM38]uint64
}

func nuevoRegistroAutoridadO3aM38() *registroAutoridadO3aM38 {
	r := &registroAutoridadO3aM38{tid: syscall.Gettid(), preflight: make(map[*acreditacionPidfdGoM38]uint64), leases: make(map[*leaseGuardiaO3aM38]uint64), observadores: make(map[*observadorSenalO3aM38]uint64)}
	r.auto = r
	return r
}

func (r *registroAutoridadO3aM38) siguiente() (uint64, bool) {
	if r == nil || r.auto != r || r.tid != syscall.Gettid() || r.generacion == ^uint64(0) {
		return 0, false
	}
	r.generacion++
	return r.generacion, true
}

type acreditacionPidfdGoM38 struct {
	auto       *acreditacionPidfdGoM38
	registro   *registroAutoridadO3aM38
	generacion uint64
	consumo    atomic.Uint32
}

func preflightPidfdGoM38(r *registroAutoridadO3aM38) (*acreditacionPidfdGoM38, error) {
	generacion, ok := r.siguiente()
	if !ok {
		return nil, errPreflightPidfdO3aM38
	}
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return nil, errPreflightPidfdO3aM38
	}
	valido := false
	errHandle := p.WithHandle(func(handle uintptr) {
		flags, _, e := syscall.Syscall(syscall.SYS_FCNTL, handle, syscall.F_GETFD, 0)
		_, _, s := syscall.Syscall6(sysPidfdSendSignal, handle, 0, 0, 0, 0, 0)
		valido = e == 0 && flags&syscall.FD_CLOEXEC != 0 && s == 0
	})
	errRelease := p.Release()
	if errHandle != nil || errRelease != nil || !valido {
		return nil, errPreflightPidfdO3aM38
	}
	a := &acreditacionPidfdGoM38{registro: r, generacion: generacion}
	a.auto, r.preflight[a] = a, generacion
	return a, nil
}

func consumirPreflightPidfdGoM38(a *acreditacionPidfdGoM38) bool {
	if a == nil || a.auto != a || a.registro == nil || a.registro.preflight[a] != a.generacion || a.registro.tid != syscall.Gettid() || !a.consumo.CompareAndSwap(0, 1) {
		return false
	}
	delete(a.registro.preflight, a)
	return true
}

type operacionGuardiaO3aM38 uint8

const (
	operacionDuplicarO3aM38 operacionGuardiaO3aM38 = iota + 1
	operacionAbrirO3aM38
	operacionPipeO3aM38
	operacionStartO3aM38
	operacionReservaPidfdO3aM38
	operacionCerrarTicketO3aM38
	operacionCerrarDestinosO3aM38
	operacionWaitO3aM38
	operacionCerrarPidfdO3aM38
)

type leaseGuardiaO3aM38 struct {
	auto       *leaseGuardiaO3aM38
	registro   *registroAutoridadO3aM38
	generacion uint64
	tid        int
	estado     atomic.Uint32
	secuencia  uint64
	operacion  operacionGuardiaO3aM38
	cardinal   int
	objetivos  [2]int
	pre        snapshotFDO3aM38
	fisico     snapshotFDO3aM38
}

type permisoGuardiaO3aM38 struct {
	lease        *leaseGuardiaO3aM38
	generacion   uint64
	operacion    operacionGuardiaO3aM38
	cardinalidad int
	objetivos    [2]int
	estadoPrevio uint32
}

func nuevaLeaseGuardiaO3aM38(r *registroAutoridadO3aM38) *leaseGuardiaO3aM38 {
	g, ok := r.siguiente()
	if !ok {
		return nil
	}
	l := &leaseGuardiaO3aM38{registro: r, generacion: g, tid: r.tid, objetivos: [2]int{-1, -1}}
	l.auto, r.leases[l] = l, g
	l.estado.Store(1)
	return l
}

func (l *leaseGuardiaO3aM38) valido() bool {
	return l != nil && l.auto == l && l.registro != nil && l.registro.auto == l.registro &&
		l.registro.leases[l] == l.generacion && l.tid == l.registro.tid && l.tid == syscall.Gettid() && l.estado.Load() == 1
}

func (l *leaseGuardiaO3aM38) sellarFisico(s snapshotFDO3aM38) bool {
	if !l.valido() || s.mapa == nil || s.limite == 0 || l.fisico.mapa != nil {
		return false
	}
	l.fisico = s
	return true
}

func (l *leaseGuardiaO3aM38) comenzar(op operacionGuardiaO3aM38, cardinalidad int, objetivos [2]int) (permisoGuardiaO3aM38, bool) {
	if l == nil || l.auto != l || l.registro == nil || l.registro.auto != l.registro ||
		l.registro.leases[l] != l.generacion || l.tid != l.registro.tid || l.tid != syscall.Gettid() || l.fisico.mapa == nil {
		return permisoGuardiaO3aM38{}, false
	}
	estadoPrevio := l.estado.Load()
	if (estadoPrevio != 1 && estadoPrevio != 3) || !l.estado.CompareAndSwap(estadoPrevio, 2) {
		return permisoGuardiaO3aM38{}, false
	}
	if l.secuencia == ^uint64(0) {
		l.estado.Store(estadoPrevio)
		return permisoGuardiaO3aM38{}, false
	}
	l.secuencia++
	l.operacion, l.cardinal, l.objetivos, l.pre = op, cardinalidad, objetivos, l.fisico
	return permisoGuardiaO3aM38{lease: l, generacion: l.secuencia, operacion: op, cardinalidad: cardinalidad, objetivos: objetivos, estadoPrevio: estadoPrevio}, true
}

func (l *leaseGuardiaO3aM38) comenzarCritico(op operacionGuardiaO3aM38, cardinalidad int) (permisoGuardiaO3aM38, bool) {
	if l == nil || l.auto != l || l.registro == nil || l.registro.auto != l.registro ||
		l.registro.leases[l] != l.generacion || l.tid != l.registro.tid || l.tid != syscall.Gettid() ||
		l.secuencia == ^uint64(0) || !l.estado.CompareAndSwap(1, 2) {
		return permisoGuardiaO3aM38{}, false
	}
	l.secuencia++
	l.operacion, l.cardinal, l.objetivos, l.pre = op, cardinalidad, [2]int{-1, -1}, l.fisico
	return permisoGuardiaO3aM38{lease: l, generacion: l.secuencia, operacion: op, cardinalidad: cardinalidad, objetivos: [2]int{-1, -1}, estadoPrevio: 1}, true
}

func (l *leaseGuardiaO3aM38) permisoValido(p permisoGuardiaO3aM38) bool {
	return l != nil && l.auto == l && l.registro != nil && l.registro.auto == l.registro &&
		l.registro.leases[l] == l.generacion && l.tid == l.registro.tid && l.tid == syscall.Gettid() &&
		p.lease == l && p.generacion == l.secuencia && p.operacion == l.operacion &&
		p.cardinalidad == l.cardinal && p.objetivos == l.objetivos && l.estado.Load() == 2
}

func (l *leaseGuardiaO3aM38) consolidarFisico(p permisoGuardiaO3aM38, actual snapshotFDO3aM38, exito bool) bool {
	if !l.permisoValido(p) || actual.mapa == nil {
		return false
	}
	if exito {
		l.fisico = actual
	} else if !snapshotsIgualesO3aM38(actual, l.pre) {
		l.estado.Store(5)
		return false
	}
	return l.estado.CompareAndSwap(2, p.estadoPrevio)
}

func (l *leaseGuardiaO3aM38) consolidarCritico(p permisoGuardiaO3aM38) bool {
	return l.permisoValido(p) && l.estado.CompareAndSwap(2, p.estadoPrevio)
}

func (l *leaseGuardiaO3aM38) fatalPendiente(p permisoGuardiaO3aM38) bool {
	return l.permisoValido(p) && l.estado.CompareAndSwap(2, 5)
}

func (l *leaseGuardiaO3aM38) transferirCritico() bool {
	return l != nil && l.auto == l && l.registro != nil && l.registro.auto == l.registro &&
		l.registro.leases[l] == l.generacion && l.tid == l.registro.tid && l.tid == syscall.Gettid() && l.estado.CompareAndSwap(1, 3)
}

func (l *leaseGuardiaO3aM38) liberar() error {
	if l == nil || l.registro == nil || l.registro.leases[l] != l.generacion || !(l.estado.CompareAndSwap(1, 4) || l.estado.CompareAndSwap(3, 4)) {
		return errAutoridadO3aM38
	}
	delete(l.registro.leases, l)
	return nil
}

type observadorSenalO3aM38 struct {
	auto       *observadorSenalO3aM38
	registro   *registroAutoridadO3aM38
	generacion uint64
	palabra    atomic.Uint64
}

const mascaraEstadoObservadorO3aM38 uint64 = 3

func nuevoObservadorSenalO3aM38(r *registroAutoridadO3aM38) *observadorSenalO3aM38 {
	g, ok := r.siguiente()
	if !ok {
		return nil
	}
	o := &observadorSenalO3aM38{registro: r, generacion: g}
	o.auto, r.observadores[o] = o, g
	o.palabra.Store(1)
	return o
}

func (o *observadorSenalO3aM38) anotar(s syscall.Signal) bool {
	if o == nil || o.auto != o || s <= 0 || s > 255 {
		return false
	}
	for {
		vieja := o.palabra.Load()
		estado := vieja & mascaraEstadoObservadorO3aM38
		contador := vieja >> 10
		if (estado != 1 && estado != 2) || contador == (uint64(1)<<54)-1 {
			return false
		}
		nueva := ((contador + 1) << 10) | (uint64(uint8(s)) << 2) | estado
		if o.palabra.CompareAndSwap(vieja, nueva) {
			return true
		}
	}
}

func (o *observadorSenalO3aM38) observar() (uint64, syscall.Signal, bool) {
	if o == nil || o.auto != o || o.registro == nil || o.registro.observadores[o] != o.generacion || o.registro.tid != syscall.Gettid() {
		return 0, 0, false
	}
	palabra := o.palabra.Load()
	estado := palabra & mascaraEstadoObservadorO3aM38
	return palabra, syscall.Signal(uint8(palabra >> 2)), estado == 1 || estado == 2
}

func (o *observadorSenalO3aM38) transferirCritico(baseline uint64) (uint64, bool) {
	if o == nil || o.auto != o || o.registro == nil || o.registro.auto != o.registro || o.registro.observadores[o] != o.generacion || o.registro.tid != syscall.Gettid() || baseline&mascaraEstadoObservadorO3aM38 != 1 {
		return 0, false
	}
	nueva := baseline - 1 + 2
	return nueva, o.palabra.CompareAndSwap(baseline, nueva)
}

func (o *observadorSenalO3aM38) liberar() error {
	if o == nil || o.registro == nil || o.registro.observadores[o] != o.generacion {
		return errAutoridadO3aM38
	}
	for {
		vieja := o.palabra.Load()
		estado := vieja & mascaraEstadoObservadorO3aM38
		if (estado != 1 && estado != 2) || !o.palabra.CompareAndSwap(vieja, vieja-estado+3) {
			if estado == 1 || estado == 2 {
				continue
			}
			return errAutoridadO3aM38
		}
		break
	}
	delete(o.registro.observadores, o)
	return nil
}

type identidadFDO3aM38 struct {
	dev, ino, rdev uint64
	modo           uint32
	uid            uint32
	enlaces        uint64
	tamano, offset int64
	offsetValido   bool
	flags, fdflags int
}

type formaRunnerM38 struct {
	identidad identidadFDO3aM38
	sha256    [32]byte
}

type formaRaizM38 struct{ identidad identidadFDO3aM38 }

type huellaFDO3aM38 struct {
	identidad identidadFDO3aM38
	abierto   bool
}

type snapshotFDO3aM38 struct {
	limite uint64
	mapa   map[int]huellaFDO3aM38
}

func snapshotsIgualesO3aM38(a, b snapshotFDO3aM38) bool {
	return a.limite == b.limite && maps.Equal(a.mapa, b.mapa)
}

type bundleEntradaO3aM38 struct {
	control                         *controladorPreinicioM38
	raiz, runner, salida, errorCaso *os.File
	controlFD, terminal, sobre      *os.File
	formaRaiz                       formaRaizM38
	formaRunner                     formaRunnerM38
	baseline                        snapshotFDO3aM38
	reloj                           *relojVueltaM38
	vueltaInicio                    uint64
	preflight                       *acreditacionPidfdGoM38
	lease                           *leaseGuardiaO3aM38
	observador                      *observadorSenalO3aM38
	baselineSenal                   uint64
	finBootstrap                    time.Time
	tid, ppid                       int
}

type custodiaO3aM38 struct {
	autoridad                    *autoridadEstadoO3aM38
	control                      *controladorPreinicioM38
	controlFD, terminal          *os.File
	lease                        *leaseGuardiaO3aM38
	observador                   *observadorSenalO3aM38
	baselineSenal                uint64
	reloj                        *relojVueltaM38
	vueltaInicio                 uint64
	finBootstrap                 time.Time
	tid, ppid                    int
	cmd                          *exec.Cmd
	pidfdPrimario, pidfdReserva  int
	pidfdOpaco                   int
	ticketEscritor, ticketLector *os.File
	destinados                   []*os.File
	huellasDestinadas            []identidadFDO3aM38
	snapshot, baseline           snapshotFDO3aM38
	formaRaiz                    formaRaizM38
	formaRunner                  formaRunnerM38
	consumida                    atomic.Uint32
	primera                      error
	secundarios                  []error
	primeraCausa                 causaPreinicioM38
}

func (c *custodiaO3aM38) tomarPreparado() bool  { return c != nil && c.consumida.CompareAndSwap(0, 1) }
func (c *custodiaO3aM38) reabrirAplazado() bool { return c != nil && c.consumida.CompareAndSwap(1, 0) }
func (c *custodiaO3aM38) tomarAgregadoPrueba() bool {
	return c != nil && c.consumida.CompareAndSwap(1, 2)
}

func (c *custodiaO3aM38) enclavarCausaControl() {
	if c != nil && c.primeraCausa == (causaPreinicioM38{}) && c.control != nil && c.control.causa != (causaPreinicioM38{}) {
		c.primeraCausa = c.control.causa
	}
}

type preparadoO3aM38 struct{ custodia *custodiaO3aM38 }
type agregadoO3aM38 struct{ custodia *custodiaO3aM38 }

type retiradaO3aM38 struct {
	origen              origenRetiradaO3aM38
	primera             error
	secundarios         []error
	causa               causaPreinicioM38
	controlFD, terminal *os.File
	lease               *leaseGuardiaO3aM38
	observador          *observadorSenalO3aM38
}

type resultadoArranqueO3aM38 struct {
	clase     claseResultadoO3aM38
	preparado *preparadoO3aM38
	agregado  *agregadoO3aM38
	retirada  *retiradaO3aM38
}

func fatalO3aM38() { os.Exit(estadoFallo) }

func enviarPidfdIndividualO3aM38(pidfd int, senal syscall.Signal) error {
	_, _, errno := syscall.Syscall6(sysPidfdSendSignal, uintptr(pidfd), uintptr(senal), 0, 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func prctlO3aM38(opcion uintptr) (int32, error) {
	var valor int32
	_, _, errno := syscall.Syscall6(syscall.SYS_PRCTL, opcion, uintptr(unsafe.Pointer(&valor)), 0, 0, 0, 0)
	if errno != 0 {
		return 0, errno
	}
	return valor, nil
}
