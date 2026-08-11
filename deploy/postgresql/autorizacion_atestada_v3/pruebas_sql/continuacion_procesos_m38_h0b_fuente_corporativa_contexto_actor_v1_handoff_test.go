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
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func ejecutarHandoffO3cP5Aislado(t *testing.T, caso string, estado int) {
	t.Helper()
	var subreaper int32
	_, _, errno := syscall.Syscall6(syscall.SYS_PRCTL, 37, uintptr(unsafe.Pointer(&subreaper)), 0, 0, 0, 0)
	if errno != 0 {
		t.Fatalf("leer subreaper del arnés: %v", errno)
	}
	_, _, errno = syscall.Syscall6(syscall.SYS_PRCTL, 36, 1, 0, 0, 0, 0)
	if errno != 0 {
		t.Fatalf("subreaper del arnés: %v", errno)
	}
	defer syscall.Syscall6(syscall.SYS_PRCTL, 36, uintptr(subreaper), 0, 0, 0, 0)
	antes := hijosDirectosO3cP5Prueba()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHandoffO3cP5CasosAislados$")
	cmd.Env = append(os.Environ(), "O3C_P5_CASO="+caso)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	despues := hijosDirectosO3cP5Prueba()
	residuoAntesDeLimpiar := false
	for pid := range despues {
		residuoAntesDeLimpiar = residuoAntesDeLimpiar || !antes[pid]
	}
	if !retirarDescendientesO3cP5Prueba(antes) {
		t.Fatalf("caso %s dejó hijos o zombis", caso)
	}
	if (caso == "retirada" || caso == "retirada_terminal") && residuoAntesDeLimpiar {
		t.Fatalf("caso C8 %s dejó residuo antes de la limpieza defensiva", caso)
	}
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

func hijosDirectosO3cP5Prueba() map[int]bool {
	hijos := map[int]bool{}
	rutas, _ := filepath.Glob("/proc/self/task/*/children")
	for _, ruta := range rutas {
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			continue
		}
		for _, campo := range strings.Fields(string(contenido)) {
			pid, err := strconv.Atoi(campo)
			if err == nil && pid > 1 {
				hijos[pid] = true
			}
		}
	}
	return hijos
}

func retirarDescendientesO3cP5Prueba(antes map[int]bool) bool {
	fin := time.Now().Add(2 * time.Second)
	for time.Now().Before(fin) {
		nuevos := hijosDirectosO3cP5Prueba()
		for pid := range nuevos {
			if !antes[pid] {
				_ = syscall.Kill(pid, syscall.SIGKILL)
				var estado syscall.WaitStatus
				_, _ = syscall.Wait4(pid, &estado, syscall.WNOHANG, nil)
			}
		}
		restantes := hijosDirectosO3cP5Prueba()
		limpio := true
		for pid := range restantes {
			limpio = limpio && antes[pid]
		}
		if limpio {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return false
}

func TestHandoffO3cP5CasosAislados(t *testing.T) {
	caso := os.Getenv("O3C_P5_CASO")
	if caso == "" {
		for _, nombre := range []string{"positivo", "retirada", "retirada_terminal"} {
			ejecutarHandoffO3cP5Aislado(t, nombre, 0)
		}
		for _, nombre := range []string{"reuso", "particion", "retirada_sin_ref", "retirada_plazo"} {
			ejecutarHandoffO3cP5Aislado(t, nombre, estadoFallo)
		}
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if strings.HasPrefix(caso, "retirada") {
		probarRetiradaO3cP5(t, caso)
		return
	}
	_, a := autoridadRealO3cP4PruebaM38(t)
	a = observarInmediatoO3cM38(&a)
	if a == nil || !a.es(continuacionC3ObservadoM38) {
		os.Exit(10)
	}
	alias := a
	autoridad, identidad := a.autoridad, a.identidad
	if caso == "particion" {
		a.estado = continuacionC4TTransfiriendoM38
		a.autoridad.ownerLease.Store(uint32(propietarioLiberadoO3cM38))
		_ = consolidarHandoffO3cM38(a)
		os.Exit(99)
	}
	salida := transferirHandoffO3cM38(&a)
	if salida == nil || a != nil || salida.auto != salida || salida.autoridad != autoridad ||
		salida.custodia == nil || salida.identidad != identidad || salida.primera.Load() == 0 ||
		salida.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
		salida.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) || alias.estado != continuacionC5EntregadoM38 ||
		salida.custodia.lease.estado.Load() != 3 || salida.custodia.observador.palabra.Load()&mascaraEstadoObservadorO3aM38 != 2 {
		os.Exit(11)
	}
	if caso == "reuso" {
		_ = transferirHandoffO3cM38(&alias)
		os.Exit(99)
	}
	os.Exit(0)
}

func autoridadRetiradaRealO3cP5(t *testing.T) *autoridadContinuacionO3cM38 {
	t.Helper()
	a := autoridadRealO3cP2PruebaM38(t)
	a.custodia.finBootstrap = a.sellos.finBootstrap
	if _, err := retirarPreContO3cM38(a, preContControlO3cM38); err == nil || !a.es(continuacionC7RetirandoM38) {
		t.Fatal("no llegó a C7")
	}
	return a
}

func probarRetiradaO3cP5(t *testing.T, caso string) {
	a := autoridadRetiradaRealO3cP5(t)
	if caso == "retirada_terminal" {
		if enviarKillRetiradaO3cM38(a.custodia, a.custodia.pidfdPrimario) != nil ||
			!esperarTerminalRetiradaO3cM38(a.custodia, a.custodia.pidfdPrimario, time.Now().Add(time.Second)) {
			os.Exit(21)
		}
	}
	if caso == "retirada_sin_ref" {
		_ = syscallCloseO3cP5Prueba(a.custodia.pidfdPrimario)
		_ = syscallCloseO3cP5Prueba(a.custodia.pidfdReserva)
	}
	if caso == "retirada_plazo" {
		a.custodia.finBootstrap = a.sellos.finBootstrap.Add(-24 * time.Hour)
	}
	alias := a
	err := retirarAntesContO3cM38(&a)
	if err == nil || a != nil || !alias.es(continuacionC8RetiradoM38) || alias.custodia != nil || alias.salida != nil || alias.autoridad != nil {
		os.Exit(20)
	}
	os.Exit(0)
}

func syscallCloseO3cP5Prueba(fd int) error { return syscall.Close(fd) }

func TestHandoffO3cP5SeleccionReserva(t *testing.T) {
	fd, viva, ok := seleccionarPidfdRetiradaO3cM38(
		referenciaRetiradaO3cM38{fd: 10, integra: false, viva: false},
		referenciaRetiradaO3cM38{fd: 11, integra: true, viva: true})
	if !ok || fd != 11 || !viva {
		t.Fatal("reserva fiable no seleccionada")
	}
	if _, _, ok := seleccionarPidfdRetiradaO3cM38(referenciaRetiradaO3cM38{}, referenciaRetiradaO3cM38{}); ok {
		t.Fatal("aceptó referencias no fiables")
	}
}

func TestHandoffO3cP5OrdenYAusenciaP6(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta prueba")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go")
	fset := token.NewFileSet()
	nodo, err := parser.ParseFile(fset, ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var orden []string
	llamadas := map[string]int{}
	ast.Inspect(nodo, func(n ast.Node) bool {
		llamada, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		nombre := ""
		switch f := llamada.Fun.(type) {
		case *ast.Ident:
			nombre = f.Name
		case *ast.SelectorExpr:
			nombre = f.Sel.Name
		}
		llamadas[nombre]++
		if nombre == "CompareAndSwap" {
			orden = append(orden, fset.Position(llamada.Pos()).String())
		}
		return true
	})
	if len(orden) < 4 || llamadas["esperarConLeaseO3aM38"] != 1 || llamadas["Wait4"] != 1 || llamadas["liberar"] != 2 {
		t.Fatalf("estructura incompleta: CAS=%d Wait=%d Wait4=%d liberar=%d", len(orden), llamadas["esperarConLeaseO3aM38"], llamadas["Wait4"], llamadas["liberar"])
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)
	if llamadas["enviarKillRetiradaO3cM38"] != 1 || !strings.Contains(texto, "vivo && enviarKillRetiradaO3cM38") {
		t.Fatal("SIGKILL no está dominado por vivo o no es único")
	}
	if llamadas["seleccionarPidfdRetiradaO3cM38"] != 1 ||
		!strings.Contains(texto, "return seleccionarPidfdRetiradaO3cM38(") {
		t.Fatal("selector primario/reserva no alimenta la retirada real")
	}
	retirada := texto[strings.Index(texto, "func retirarAntesContO3cM38"):]
	ordenRetirada := []string{"esperarConLeaseO3aM38", "drenarAdoptadosO3cM38", "grupoAusenteO3cM38", "cerrarRecursosRetiradaO3cM38", "inventarioLiberadoO3cM38", "liberarRetiradaO3cM38"}
	posicion := -1
	for _, nombre := range ordenRetirada {
		actual := strings.Index(retirada, nombre)
		if actual <= posicion {
			t.Fatalf("orden retirada roto en %s", nombre)
		}
		posicion = actual
	}
	liberacion := texto[strings.Index(texto, "func liberarRetiradaO3cM38"):strings.Index(texto, "func retirarAntesContO3cM38")]
	if strings.Index(liberacion, "c.observador.liberar") >= strings.Index(liberacion, "c.lease.liberar") {
		t.Fatal("lease no fue la última capacidad subyacente")
	}
	for _, prohibido := range []string{"waitid", "SIGCONT", "SIGTERM", "Process.Signal", "Process.Kill", "TERMINAL.Write", "http.", "sql."} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("P6/O4 prohibido: %s", prohibido)
		}
	}
}
