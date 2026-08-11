//go:build ignore && linux && amd64

package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func ejecutarArbitrajeO4aP3AisladoM38(t *testing.T, caso string, estado int) {
	t.Helper()
	antes := hijosDirectosO3cP5Prueba()
	cmd := exec.Command(os.Args[0], "-test.paniconexit0=false", "-test.run=^TestArbitrajeO4aP3CasosAislados$")
	cmd.Env = append(os.Environ(), "O4A_P3_CASO="+caso)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if !retirarDescendientesO3cP5Prueba(antes) {
		t.Fatalf("caso %s dejó recursos", caso)
	}
	if estado == 0 {
		if err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("caso %s: err=%v stdout=%d stderr=%d", caso, err, stdout.Len(), stderr.Len())
		}
		return
	}
	var salida *exec.ExitError
	if !errors.As(err, &salida) || salida.ExitCode() != estado || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("fatal %s: err=%v stdout=%d stderr=%d", caso, err, stdout.Len(), stderr.Len())
	}
}

func autoridadArbitrajeO4aP3PruebaM38(t *testing.T) (*fixtureO3aM38, *autoridadCausaO4aM38) {
	t.Helper()
	f, continuacion := autoridadRealO3cP4PruebaM38(t)
	continuacion = observarInmediatoO3cM38(&continuacion)
	entrada := transferirHandoffO3cM38(&continuacion)
	// Cada caso P3 parte de la continuidad C5 ya acreditada; el evento nuevo se
	// instala después de P2, nunca se hereda del tiempo de arranque del fixture.
	entrada.primera.Store(uint32(observacionPidfdVacioO3cM38))
	entrada.custodia.observador.palabra.Store(entrada.custodia.baselineSenal)
	a, err := consumirAutoridadO4aM38(&entrada)
	if err != nil || a == nil {
		t.Fatal("autoridad P1")
	}
	if err := sembrarCausaO4aM38(a); err != nil || a.estado.Load() != uint32(causaA2ObservandoM38) {
		t.Fatal("semilla P2")
	}
	return f, a
}

func pidfdListoO4aP3PruebaM38(fd int) bool {
	for intentos := 0; intentos < 100000; intentos++ {
		sonda := pollfdO3aM38{fd: int32(fd), eventos: pollInO3cM38}
		n, _, errno := syscall.Syscall(syscall.SYS_POLL, punteroPollO3aM38(&sonda), 1, 0)
		if errno != 0 {
			return false
		}
		if n == 1 && sonda.retorno == pollInO3cM38 {
			return true
		}
		runtime.Gosched()
	}
	return false
}

func TestArbitrajeO4aP3CasosAislados(t *testing.T) {
	caso := os.Getenv("O4A_P3_CASO")
	if caso == "" {
		for _, n := range []string{"vacio", "control_cancelado", "control_protocolo", "control_int", "control_term", "control_framing", "control_eof", "control_senal_pidfd", "senal_int", "senal_term", "senal_ambigua", "senal_pidfd", "natural", "infra", "plazo", "borde", "previa", "replay"} {
			ejecutarArbitrajeO4aP3AisladoM38(t, n, 0)
		}
		for _, n := range []string{"auto", "estado", "causa", "ahora_sustituido", "fin_sustituido", "par_sustituido", "lease", "observador", "raw", "primera", "palabra", "canon", "sello_pidfd", "carrera", "carrera_causa"} {
			ejecutarArbitrajeO4aP3AisladoM38(t, n, estadoFallo)
		}
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f, a := autoridadArbitrajeO4aP3PruebaM38(t)
	base := a.sellos.palabraObservada
	palabra := func(delta uint64, signo syscall.Signal) uint64 {
		return ((base>>10)+delta)<<10 | uint64(uint8(signo))<<2 | 2
	}
	switch caso {
	case "control_framing":
		if err := escribirControlPruebaO3aM38(f, "trama-invalida\n"); err != nil {
			os.Exit(29)
		}
	case "control_eof":
		if f.controlEscritor == nil || f.controlEscritor.Close() != nil {
			os.Exit(29)
		}
		f.controlEscritor = nil
	case "control_cancelado", "control_protocolo", "control_int", "control_term", "control_senal_pidfd":
		causa, estado := "CANCELADO", "65"
		switch caso {
		case "control_protocolo":
			causa = "PROTOCOLO"
		case "control_int":
			causa, estado = "SENAL_INT", "130"
		case "control_term":
			causa, estado = "SENAL_TERM", "143"
		}
		a.origen.custodia.control.enclavarCausa(causa, estado)
		a.origen.custodia.primeraCausa = a.origen.custodia.control.causa
		if caso == "control_senal_pidfd" {
			a.origen.custodia.observador.palabra.Store(palabra(1, syscall.SIGTERM))
			if err := a.origen.custodia.cmd.Process.Kill(); err != nil || !pidfdListoO4aP3PruebaM38(a.sellos.pidfd[0]) {
				os.Exit(30)
			}
		}
	case "senal_int", "senal_pidfd":
		a.origen.custodia.observador.palabra.Store(palabra(1, syscall.SIGINT))
		if caso == "senal_pidfd" {
			if err := a.origen.custodia.cmd.Process.Kill(); err != nil || !pidfdListoO4aP3PruebaM38(a.sellos.pidfd[0]) {
				os.Exit(31)
			}
		}
	case "senal_term":
		a.origen.custodia.observador.palabra.Store(palabra(1, syscall.SIGTERM))
	case "senal_ambigua":
		a.origen.custodia.observador.palabra.Store(palabra(2, syscall.SIGINT))
	case "natural":
		if err := a.origen.custodia.cmd.Process.Kill(); err != nil || !pidfdListoO4aP3PruebaM38(a.sellos.pidfd[0]) {
			os.Exit(32)
		}
	case "infra":
		delete(a.sellos.fisico.mapa, a.sellos.pidfd[0])
	case "plazo", "borde":
		ahora := time.Now().Add(-181 * time.Second)
		if caso == "borde" {
			ahora = time.Now().Add(-duracionCasoO3cM38)
		}
		fin := ahora.Add(duracionCasoO3cM38)
		a.sellos.ahoraCaso, a.sellos.finCaso = ahora, fin
		a.origen.ahoraCaso, a.origen.finCaso = ahora, fin
	case "previa":
		a.causa.Store(uint32(causaSenalTerm143O4aM38))
	case "auto":
		a.auto = nil
	case "estado":
		a.estado.Store(uint32(causaA1ValidadoM38))
	case "causa":
		a.causa.Store(99)
	case "ahora_sustituido":
		a.origen.ahoraCaso = a.origen.ahoraCaso.Add(time.Second)
	case "fin_sustituido":
		a.origen.finCaso = a.origen.finCaso.Add(time.Second)
	case "par_sustituido":
		a.origen.ahoraCaso = a.origen.ahoraCaso.Add(time.Second)
		a.origen.finCaso = a.origen.finCaso.Add(time.Second)
	case "lease":
		a.sellos.lease.estado.Store(2)
	case "observador":
		a.sellos.observador.auto = nil
	case "raw":
		a.sellos.retornoCont = 1
	case "primera":
		a.sellos.primera = uint32(observacionControlRawO3cM38)
	case "palabra":
		a.sellos.palabraObservada++
	case "canon":
		a.sellos.canonControlRaw = controlRawCancelado65O4aM38
	case "sello_pidfd":
		a.sellos.pidfd[0] = a.sellos.pidfd[1]
	case "carrera", "carrera_causa":
		if caso == "carrera_causa" {
			a.origen.custodia.observador.palabra.Store(palabra(1, syscall.SIGINT))
		}
		fisico := copiaSnapshotO4aM38(a.origen.custodia.lease.fisico)
		resultados := make(chan error, 2)
		go func() { resultados <- arbitrarInmediatoO4aM38(a) }()
		go func() { resultados <- arbitrarInmediatoO4aM38(a) }()
		e1, e2 := <-resultados, <-resultados
		if caso == "carrera" {
			if e1 != nil || e2 != nil || a.causa.Load() != 0 || a.estado.Load() != uint32(causaA2ObservandoM38) {
				os.Exit(33)
			}
		} else if (e1 == nil) == (e2 == nil) || a.causa.Load() != uint32(causaSenalInt130O4aM38) || a.estado.Load() != uint32(causaA3CausaFijadaM38) {
			os.Exit(34)
		}
		if a.origen.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) ||
			a.origen.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
			!snapshotsIgualesO3aM38(a.origen.custodia.lease.fisico, fisico) {
			os.Exit(35)
		}
		os.Exit(0)
	}
	primera, raw, historica, canon := a.sellos.primera, a.sellos.retornoCont, a.sellos.palabraObservada, a.sellos.canonControlRaw
	ahoraSellada, finSellado := a.sellos.ahoraCaso, a.sellos.finCaso
	fisico := copiaSnapshotO4aM38(a.origen.custodia.lease.fisico)
	err := arbitrarInmediatoO4aM38(a)
	if caso == "previa" {
		if !errors.Is(err, errArbitrajeConsumidoO4aM38) || a.causa.Load() != uint32(causaSenalTerm143O4aM38) {
			os.Exit(36)
		}
		os.Exit(0)
	}
	if err != nil {
		os.Exit(37)
	}
	esperada, estado := causaVaciaO4aM38, causaA2ObservandoM38
	switch caso {
	case "control_cancelado", "control_senal_pidfd":
		esperada, estado = causaCancelado65O4aM38, causaA3CausaFijadaM38
	case "control_protocolo", "control_framing":
		esperada, estado = causaProtocolo65O4aM38, causaA3CausaFijadaM38
	case "control_eof":
		esperada, estado = causaCancelado65O4aM38, causaA3CausaFijadaM38
	case "control_int", "senal_int", "senal_pidfd":
		esperada, estado = causaSenalInt130O4aM38, causaA3CausaFijadaM38
	case "control_term", "senal_term":
		esperada, estado = causaSenalTerm143O4aM38, causaA3CausaFijadaM38
	case "senal_ambigua", "infra":
		esperada, estado = causaIncidente65O4aM38, causaA3CausaFijadaM38
	case "natural":
		esperada, estado = causaSalidaO4aM38, causaA3CausaFijadaM38
	case "plazo", "borde":
		esperada, estado = causaPlazo65O4aM38, causaA3CausaFijadaM38
	}
	if a.causa.Load() != uint32(esperada) || a.estado.Load() != uint32(estado) || a.incidente.Load() != 0 ||
		a.sellos.primera != primera || a.sellos.retornoCont != raw || a.sellos.palabraObservada != historica ||
		a.sellos.canonControlRaw != canon || a.sellos.ahoraCaso != ahoraSellada || a.sellos.finCaso != finSellado ||
		a.origen.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) ||
		a.origen.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
		(caso != "infra" && !snapshotsIgualesO3aM38(a.origen.custodia.lease.fisico, fisico)) {
		os.Exit(38)
	}
	if caso == "replay" {
		if err := arbitrarInmediatoO4aM38(a); err != nil {
			os.Exit(39)
		}
		if a.causa.Load() != 0 {
			os.Exit(40)
		}
		if a.estado.Load() != uint32(causaA2ObservandoM38) {
			os.Exit(41)
		}
	}
	os.Exit(0)
}

func TestArbitrajeO4aP3EstructuraCerrada(t *testing.T) {
	_, prueba, _, _ := runtime.Caller(0)
	ruta := filepath.Join(filepath.Dir(prueba), "causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arbitraje.go")
	fset := token.NewFileSet()
	nodo, err := parser.ParseFile(fset, ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(nodo, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.GoStmt, *ast.ChanType:
			t.Fatalf("concurrencia productiva %T", n)
		}
		return true
	})
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)
	for _, p := range []string{"time.Sleep(", "time.NewTimer(", "time.After(", "Ticker", "Kill(", "Signal(", "Wait(", "Wait4(", "Waitid(", "Close(", "Write(", "Print(", "Log(", "pidfd_open", "F_DUPFD", "/proc/", "context."} {
		if bytes.Contains(contenido, []byte(p)) {
			t.Fatalf("efecto prohibido %s", p)
		}
	}
	if strings.Count(texto, "a.causa.CompareAndSwap(") != 1 || strings.Contains(texto, "a.causa.Store(") || strings.Contains(texto, "a.causa.Swap(") {
		t.Fatal("latch causa no único")
	}
	if !strings.Contains(texto, "a.origen.ahoraCaso != a.sellos.ahoraCaso") || !strings.Contains(texto, "a.origen.finCaso != a.sellos.finCaso") {
		t.Fatal("cotejo temporal estructural ausente")
	}
	if strings.Count(texto, "a.origen.ahoraCaso") != 1 || strings.Count(texto, "a.origen.finCaso") != 1 || strings.Count(texto, "time.Now()") != 1 {
		t.Fatal("deadline releído o reconstruido")
	}
	if strings.Count(texto, "if !ahora.Before(a.sellos.finCaso)") != 1 {
		t.Fatal("borde del deadline divergente")
	}
	posTiempo := strings.Index(texto, "!autoridadArbitrajeExactaO4aM38(a)")
	posControl := strings.Index(texto, "leerControlO3bM38(a.sellos.custodia)")
	posSenal := strings.LastIndex(texto, "observarSenalArbitrajeO4aM38(a)")
	posPidfd := strings.LastIndex(texto, "observarPidfdArbitrajeO4aM38(a)")
	posReloj := strings.LastIndex(texto, "time.Now()")
	if posTiempo < 0 || !(posTiempo < posControl && posControl < posSenal && posSenal < posPidfd && posPidfd < posReloj) {
		t.Fatal("precedencia temporal/CONTROL/señal/pidfd/reloj divergente")
	}
	if strings.Count(texto, "operarConLeaseBarreraO3bM38(") < 2 || strings.Count(texto, "identidadPidfdBarreraO3bM38(") != 1 ||
		strings.Count(texto, "pollPidfdArbitrajeO4aM38(a,") != 2 {
		t.Fatal("permisos lease no acreditados")
	}
}
