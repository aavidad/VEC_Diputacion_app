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
	"syscall"
	"testing"
	"time"
)

func autoridadRealO3cP4PruebaM38(t *testing.T) (*fixtureO3aM38, *autoridadContinuacionO3cM38) {
	t.Helper()
	f, captura := autoridadRealBarreraO3bPruebaM38(t)
	captura.custodia.control.recepcion.sobre.ticket = ticketRealStopO3bPruebaM38(captura)
	if ejecutarBarreraO3bM38(captura) != nil || emitirYCerrarTicketO3bM38(captura) != nil || observarStopO3bM38(captura) != nil {
		t.Fatal("preparar B3")
	}
	identidad, err := acreditarIdentidadO3bM38(captura)
	if err != nil {
		t.Fatalf("identidad B4: %v", err)
	}
	agregado, err := transferirCapturadoO3bM38(&identidad)
	if err != nil {
		t.Fatalf("handoff B5: %v", err)
	}
	a, err := consumirAutoridadO3cM38(&agregado)
	if err != nil {
		t.Fatalf("autoridad C0: %v", err)
	}
	r, err := revalidarAntesContO3cM38(a)
	if err != nil {
		t.Fatalf("revalidar C1: %v", err)
	}
	a = intentarContO3cM38(&r)
	if a == nil || !a.es(continuacionC2ContIntentadoM38) {
		t.Fatal("CONT no llegó a C2")
	}
	return f, a
}

func ejecutarObservacionAisladaO3cPruebaM38(t *testing.T, caso string, estado int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestObservacionO3cCasosAislados$")
	cmd.Env = append(os.Environ(), "O3C_P4_CASO="+caso)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if estado == 0 {
		if err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("caso %s: err=%v stdout=%d stderr=%d", caso, err, stdout.Len(), stderr.Len())
		}
		return
	}
	var salida *exec.ExitError
	if !errors.As(err, &salida) || salida.ExitCode() != estado || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("caso fatal %s: err=%v stdout=%d stderr=%d", caso, err, stdout.Len(), stderr.Len())
	}
}

func TestObservacionO3cCasosAislados(t *testing.T) {
	caso := os.Getenv("O3C_P4_CASO")
	if caso == "" {
		for _, nombre := range []string{"vacio", "natural", "control", "senal", "infra", "una_fiable", "fds_reutilizados"} {
			ejecutarObservacionAisladaO3cPruebaM38(t, nombre, 0)
		}
		ejecutarObservacionAisladaO3cPruebaM38(t, "reuso", estadoFallo)
		ejecutarObservacionAisladaO3cPruebaM38(t, "c2_corrupto", estadoFallo)
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f, a := autoridadRealO3cP4PruebaM38(t)
	rawCont := a.salida.retornoCont
	switch caso {
	case "c2_corrupto":
		a.salida.finCaso = a.salida.ahoraCaso
	case "control":
		if err := escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|"+string(a.custodia.control.nonce[:])+"|CANCELADO|65\n"); err != nil {
			os.Exit(10)
		}
		a.custodia.observador.palabra.Add(1 << 2)
		_ = syscall.Close(a.custodia.pidfdPrimario)
	case "senal":
		a.custodia.observador.palabra.Add(1 << 2)
		_ = syscall.Close(a.custodia.pidfdPrimario)
	case "infra":
		_ = syscall.Close(a.custodia.pidfdPrimario)
		_ = syscall.Close(a.custodia.pidfdReserva)
	case "fds_reutilizados":
		if !reutilizarFDObservacionO3cPruebaM38(a.custodia.pidfdPrimario) || !reutilizarFDObservacionO3cPruebaM38(a.custodia.pidfdReserva) {
			os.Exit(16)
		}
	case "una_fiable":
		_ = syscall.Close(a.custodia.pidfdPrimario)
	case "natural":
		if !esperarTerminalPidfdO3cPruebaM38(a.custodia.pidfdPrimario) {
			os.Exit(11)
		}
	case "vacio", "reuso":
	default:
		os.Exit(12)
	}
	alias := a
	resultado := observarInmediatoO3cM38(&a)
	if a != nil || resultado != alias || alias == nil || !alias.es(continuacionC3ObservadoM38) || alias.salida.retornoCont != rawCont ||
		!discriminanteObservacionValidoO3cM38(discriminanteObservacionO3cM38(alias.salida.primera.Load())) {
		os.Exit(13)
	}
	d := discriminanteObservacionO3cM38(alias.salida.primera.Load())
	if caso == "control" && d != observacionControlRawO3cM38 || caso == "senal" && d != observacionSenalRawO3cM38 ||
		(caso == "infra" || caso == "fds_reutilizados") && d != observacionPidfdInfraestructuraO3cM38 ||
		caso == "natural" && d != observacionPidfdTerminalNaturalO3cM38 ||
		caso == "vacio" && d != observacionPidfdVacioO3cM38 {
		os.Exit(14)
	}
	if caso == "una_fiable" && d == observacionPidfdInfraestructuraO3cM38 {
		os.Exit(15)
	}
	if caso == "reuso" {
		_ = observarInmediatoO3cM38(&alias)
		os.Exit(99)
	}
	os.Exit(0)
}

func reutilizarFDObservacionO3cPruebaM38(fd int) bool {
	if syscall.Close(fd) != nil {
		return false
	}
	nuevo, err := syscall.Open("/dev/null", syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return false
	}
	if nuevo != fd {
		if err := syscall.Dup3(nuevo, fd, syscall.O_CLOEXEC); err != nil {
			return false
		}
		_ = syscall.Close(nuevo)
	}
	return true
}

func esperarTerminalPidfdO3cPruebaM38(fd int) bool {
	fin := time.Now().Add(2 * time.Second)
	for time.Now().Before(fin) {
		p := pollfdO3aM38{fd: int32(fd), eventos: pollInO3cM38}
		n, _, errno := syscall.Syscall(syscall.SYS_POLL, punteroPollO3aM38(&p), 1, 0)
		if errno == 0 && n == 1 && p.retorno == pollInO3cM38 {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func TestObservacionO3cClasificacionCerrada(t *testing.T) {
	casos := []struct {
		n                uintptr
		primero, segundo int16
		err              error
		esperado         discriminanteObservacionO3cM38
	}{
		{0, 0, 0, nil, observacionPidfdVacioO3cM38},
		{1, pollInO3cM38, 0, nil, observacionPidfdTerminalNaturalO3cM38},
		{2, pollInO3cM38, pollInO3cM38, nil, observacionPidfdTerminalNaturalO3cM38},
		{0, pollInO3cM38, 0, nil, observacionPidfdInfraestructuraO3cM38},
		{1, pollInO3cM38 | pollHupO3cM38, 0, nil, observacionPidfdInfraestructuraO3cM38},
		{1, pollErrO3cM38, 0, nil, observacionPidfdInfraestructuraO3cM38},
		{1, pollNvalO3cM38, 0, nil, observacionPidfdInfraestructuraO3cM38},
		{0, 0, 0, syscall.EBADF, observacionPidfdInfraestructuraO3cM38},
	}
	for i, caso := range casos {
		if actual := clasificarPollO3cM38(caso.n, caso.primero, caso.segundo, caso.err); actual != caso.esperado {
			t.Fatalf("clasificación %d: %d", i, actual)
		}
	}
}

func TestObservacionO3cPrecedenciaCASYAusenciaP5(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta ausente")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_observacion.go")
	f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	orden := make([]string, 0, 3)
	polls, cas := 0, 0
	prohibidas := map[string]bool{"Wait": true, "Wait4": true, "Signal": true, "Kill": true, "Close": true, "Sleep": true, "Write": true}
	for _, declaracion := range f.Decls {
		fn, ok := declaracion.(*ast.FuncDecl)
		if !ok {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (id.Name == "SIGCONT" || id.Name == "SIGSTOP" || id.Name == "SIGTERM" || id.Name == "SIGKILL") {
				t.Fatalf("señal P5+ prohibida: %s", id.Name)
			}
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch llamada := c.Fun.(type) {
			case *ast.Ident:
				if fn.Name.Name == "observarInmediatoO3cM38" && (llamada.Name == "leerControlO3bM38" || llamada.Name == "autoridadSenalO3cM38" || llamada.Name == "observarPidfdO3cM38") {
					orden = append(orden, llamada.Name)
				}
			case *ast.SelectorExpr:
				if prohibidas[llamada.Sel.Name] {
					t.Fatalf("API P5+ prohibida: %s", llamada.Sel.Name)
				}
				if llamada.Sel.Name == "CompareAndSwap" {
					cas++
				}
				if llamada.Sel.Name == "Syscall" {
					polls++
					if len(c.Args) != 4 {
						t.Fatal("poll no tiene cuatro argumentos")
					}
				}
			}
			return true
		})
	}
	esperado := []string{"leerControlO3bM38", "autoridadSenalO3cM38", "observarPidfdO3cM38"}
	if len(orden) != len(esperado) {
		t.Fatalf("precedencia incompleta: %v", orden)
	}
	for i := range esperado {
		if orden[i] != esperado[i] {
			t.Fatalf("precedencia inválida: %v", orden)
		}
	}
	if polls != 1 || cas != 1 {
		t.Fatalf("poll/CAS cardinalidad inválida: poll=%d cas=%d", polls, cas)
	}
}
