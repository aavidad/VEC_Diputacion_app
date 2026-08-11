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

func autoridadRealO3cP2PruebaM38(t *testing.T) *autoridadContinuacionO3cM38 {
	t.Helper()
	identidad := identidadRealHandoffO3bPruebaM38(t)
	agregado, err := transferirCapturadoO3bM38(&identidad)
	if err != nil || agregado == nil {
		t.Fatalf("handoff O3b: %v", err)
	}
	a, err := consumirAutoridadO3cM38(&agregado)
	if err != nil || a == nil || !a.es(continuacionC0RecibidoM38) {
		t.Fatalf("autoridad O3c: %v", err)
	}
	return a
}

func ejecutarCasoO3cP2PruebaM38(t *testing.T, caso string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRevalidacionO3cCasosAislados$")
	cmd.Env = append(os.Environ(), "O3C_P2_CASO="+caso)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if caso == "cuarta" || caso == "ambos_pidfd" || caso == "process" || caso == "terminal_sello" || caso == "controlador_sello" || caso == "controlfd_sello" || caso == "pidfd_sello" || caso == "baseline_sello" || caso == "bootstrap_sello" || caso == "generacion_sello" || caso == "registro_sello" || caso == "identidad" || caso == "estado" || caso == "pgid" || caso == "ppid" || caso == "tid" {
		var salida *exec.ExitError
		if !errors.As(err, &salida) || salida.ExitCode() != estadoFallo || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("caso cuarta no fue CF 65/0/0: err=%v stdout=%d stderr=%d", err, stdout.Len(), stderr.Len())
		}
		return
	}
	if err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("caso %s: err=%v stdout=%d stderr=%d", caso, err, stdout.Len(), stderr.Len())
	}
}

func TestRevalidacionO3cCasosAislados(t *testing.T) {
	caso := os.Getenv("O3C_P2_CASO")
	if caso == "" {
		for _, nombre := range []string{"verde", "observador_permisos", "senal", "bootstrap", "bootstrap_sello", "identidad", "estado", "pgid", "ppid", "tid", "control", "primario", "ambos_pidfd", "cuarta", "process", "terminal_sello", "controlador_sello", "controlfd_sello", "pidfd_sello", "baseline_sello", "generacion_sello", "registro_sello"} {
			ejecutarCasoO3cP2PruebaM38(t, nombre)
		}
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	a := autoridadRealO3cP2PruebaM38(t)
	switch caso {
	case "observador_permisos":
		secuencia := a.custodia.lease.secuencia
		primera, errPrimera := autoridadSenalO3cM38(a.custodia)
		segunda, errSegunda := autoridadSenalO3cM38(a.custodia)
		if !primera || !segunda || errPrimera != nil || errSegunda != nil ||
			a.custodia.lease.secuencia != secuencia+2 || a.custodia.lease.estado.Load() != 3 {
			os.Exit(17)
		}
		os.Exit(0)
	case "senal":
		a.custodia.observador.palabra.Add(1 << 2)
	case "bootstrap":
		a.custodia.finBootstrap = time.Now()
		a.sellos.finBootstrap = a.custodia.finBootstrap
	case "bootstrap_sello":
		a.custodia.finBootstrap = time.Now()
	case "identidad":
		a.identidad.inicio++
	case "estado":
		a.identidad.estado = 'S'
	case "pgid":
		a.identidad.pgid++
	case "ppid":
		a.custodia.ppid++
	case "tid":
		a.custodia.tid++
	case "cuarta":
		duplicado, err := syscall.Dup(a.custodia.pidfdPrimario)
		if err != nil {
			os.Exit(10)
		}
		defer syscall.Close(duplicado)
	case "control":
		_ = a.custodia.controlFD.Close()
	case "primario":
		_ = syscall.Close(a.custodia.pidfdPrimario)
	case "ambos_pidfd":
		_ = syscall.Close(a.custodia.pidfdPrimario)
		_ = syscall.Close(a.custodia.pidfdReserva)
	case "process":
		a.custodia.cmd.Process = &os.Process{Pid: a.identidad.pid}
	case "terminal_sello":
		a.custodia.terminal = a.custodia.controlFD
	case "controlador_sello":
		clon := *a.custodia.control
		a.custodia.control = &clon
	case "controlfd_sello":
		a.custodia.controlFD = a.custodia.terminal
	case "pidfd_sello":
		a.custodia.pidfdPrimario, a.custodia.pidfdReserva = a.custodia.pidfdReserva, a.custodia.pidfdPrimario
	case "baseline_sello":
		a.custodia.baselineSenal += 1 << 10
		a.custodia.observador.palabra.Store(a.custodia.baselineSenal)
	case "generacion_sello":
		a.custodia.lease.generacion++
	case "registro_sello":
		r := nuevoRegistroAutoridadO3aM38()
		r.leases[a.custodia.lease] = a.custodia.lease.generacion
		r.observadores[a.custodia.observador] = a.custodia.observador.generacion
		a.custodia.lease.registro, a.custodia.observador.registro = r, r
	case "verde":
	default:
		os.Exit(11)
	}
	r, err := revalidarAntesContO3cM38(a)
	if caso == "verde" {
		if err != nil || r == nil {
			os.Exit(12)
		}
		if r.auto != r || r.autoridad != a || !a.es(continuacionC1RevalidadoM38) {
			os.Exit(14)
		}
		if a.custodia.lease.estado.Load() != 2 || !a.custodia.lease.permisoValido(r.permiso) {
			os.Exit(15)
		}
		if r.permiso.operacion != operacionContO3cM38 || r.permiso.cardinalidad != 1 ||
			r.permiso.objetivos != [2]int{a.custodia.pidfdPrimario, -1} {
			os.Exit(16)
		}
		os.Exit(0)
	}
	if r != nil || err == nil || !a.es(continuacionC7RetirandoM38) || a.salida.primera.Load() == 0 ||
		a.custodia.lease.estado.Load() != 3 {
		os.Exit(13)
	}
	os.Exit(0)
}

func TestRevalidacionO3cLeaseInacreditableEsFatal(t *testing.T) {
	if os.Getenv("O3C_P2_FATAL") == "1" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		a := autoridadRealO3cP2PruebaM38(t)
		a.custodia.lease.auto = &leaseGuardiaO3aM38{}
		_, _ = revalidarAntesContO3cM38(a)
		os.Exit(99)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestRevalidacionO3cLeaseInacreditableEsFatal$")
	cmd.Env = append(os.Environ(), "O3C_P2_FATAL=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var salida *exec.ExitError
	if !errors.As(err, &salida) || salida.ExitCode() != estadoFallo || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("CF no fue 65/0/0: err=%v stdout=%d stderr=%d", err, stdout.Len(), stderr.Len())
	}
}

func TestRevalidacionO3cPrecedenciaYAusenciaP3(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta ausente")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion.go")
	f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	orden := make([]string, 0, 12)
	prohibidas := map[string]bool{"comenzarCritico": true, "Wait": true, "Signal": true, "Kill": true, "Sleep": true}
	for _, declaracion := range f.Decls {
		fn, ok := declaracion.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "revalidarAntesContO3cM38" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (id.Name == "SIGCONT" || id.Name == "sysPidfdSendSignal") {
				t.Fatalf("efecto P3+ prohibido: %s", id.Name)
			}
			llamada, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fn := llamada.Fun.(type) {
			case *ast.Ident:
				if fn.Name == "leerControlO3bM38" || fn.Name == "autoridadSenalO3cM38" || fn.Name == "identidadEjecucionO3cM38" ||
					fn.Name == "pidfdEInventarioO3cM38" || fn.Name == "identidadProcesoFinalO3cM38" || fn.Name == "segundaRondaO3cM38" ||
					fn.Name == "preasignacionO4aValidaO3cM38" {
					orden = append(orden, fn.Name)
				}
			case *ast.SelectorExpr:
				if prohibidas[fn.Sel.Name] {
					t.Fatalf("API P3+ prohibida: %s", fn.Sel.Name)
				}
			}
			return true
		})
	}
	esperado := []string{"leerControlO3bM38", "autoridadSenalO3cM38", "identidadEjecucionO3cM38", "pidfdEInventarioO3cM38", "identidadProcesoFinalO3cM38", "segundaRondaO3cM38", "preasignacionO4aValidaO3cM38"}
	if len(orden) < len(esperado) {
		t.Fatalf("precedencia incompleta: %v", orden)
	}
	for i := range esperado {
		if orden[len(orden)-len(esperado)+i] != esperado[i] {
			t.Fatalf("precedencia final incorrecta: %v", orden)
		}
	}
}
