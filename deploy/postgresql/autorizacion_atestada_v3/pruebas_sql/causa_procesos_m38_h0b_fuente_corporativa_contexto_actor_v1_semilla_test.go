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
)

func ejecutarSemillaO4aP2AisladaM38(t *testing.T, caso string, estado int) {
	t.Helper()
	antes := hijosDirectosO3cP5Prueba()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSemillaO4aP2CasosAislados$")
	cmd.Env = append(os.Environ(), "O4A_P2_CASO="+caso)
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

func autoridadSemillaO4aP2PruebaM38(t *testing.T, caso string) *autoridadCausaO4aM38 {
	t.Helper()
	e := agregadoRealO4aP1PruebaM38(t)
	base := e.custodia.baselineSenal
	palabra := func(delta uint64, senal syscall.Signal) uint64 {
		return ((base>>10)+delta)<<10 | uint64(uint8(senal))<<2 | 2
	}
	switch caso {
	case "cont", "cont_control", "cont_discriminante", "cont_canon", "cont_palabra", "cont_pareja":
		e.retornoCont = 5
		if caso == "cont_control" {
			e.primera.Store(uint32(observacionControlRawO3cM38))
			e.custodia.control.enclavarCausa("CANCELADO", "65")
			e.custodia.primeraCausa = e.custodia.control.causa
		}
	case "control_cancelado", "control_protocolo", "control_int", "control_term":
		causa, estado := "CANCELADO", "65"
		if caso == "control_protocolo" {
			causa = "PROTOCOLO"
		}
		if caso == "control_int" {
			causa, estado = "SENAL_INT", "130"
		}
		if caso == "control_term" {
			causa, estado = "SENAL_TERM", "143"
		}
		e.primera.Store(uint32(observacionControlRawO3cM38))
		e.custodia.control.enclavarCausa(causa, estado)
		e.custodia.primeraCausa = e.custodia.control.causa
	case "senal_int":
		e.primera.Store(uint32(observacionSenalRawO3cM38))
		e.custodia.observador.palabra.Store(palabra(1, syscall.SIGINT))
	case "senal_term":
		e.primera.Store(uint32(observacionSenalRawO3cM38))
		e.custodia.observador.palabra.Store(palabra(1, syscall.SIGTERM))
	case "senal_cero":
		e.primera.Store(uint32(observacionSenalRawO3cM38))
	case "senal_multiple":
		e.primera.Store(uint32(observacionSenalRawO3cM38))
		e.custodia.observador.palabra.Store(palabra(2, syscall.SIGTERM))
	case "senal_ajena":
		e.primera.Store(uint32(observacionSenalRawO3cM38))
		e.custodia.observador.palabra.Store(palabra(1, syscall.SIGUSR1))
	case "natural":
		e.primera.Store(uint32(observacionPidfdTerminalNaturalO3cM38))
	case "infra", "carrera_causa":
		e.primera.Store(uint32(observacionPidfdInfraestructuraO3cM38))
	}
	a, err := consumirAutoridadO4aM38(&e)
	if err != nil || a == nil || e != nil {
		t.Fatal("autoridad P1C")
	}
	return a
}

func TestSemillaO4aP2CasosAislados(t *testing.T) {
	caso := os.Getenv("O4A_P2_CASO")
	if caso == "" {
		for _, n := range []string{"nulo", "vacio", "cont", "cont_control", "cont_discriminante", "cont_canon", "cont_palabra", "cont_pareja", "control_cancelado", "control_protocolo", "control_int", "control_term", "senal_int", "senal_term", "senal_cero", "senal_multiple", "senal_ajena", "natural", "infra", "replay", "previa", "carrera", "carrera_causa", "inmutable"} {
			ejecutarSemillaO4aP2AisladaM38(t, n, 0)
		}
		for _, n := range []string{"canon_vacio", "canon_ajeno", "canon_no_control", "palabra_estado", "palabra_regresiva", "palabra_pidfd", "pareja_corrupta", "discriminante", "raw_negativo", "causa_corrupta", "estado_corrupto", "auto"} {
			ejecutarSemillaO4aP2AisladaM38(t, n, estadoFallo)
		}
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if caso == "nulo" {
		if !errors.Is(sembrarCausaO4aM38(nil), errSemillaConsumidaO4aM38) {
			os.Exit(16)
		}
		os.Exit(0)
	}
	a := autoridadSemillaO4aP2PruebaM38(t, caso)
	switch caso {
	case "cont_discriminante":
		a.sellos.primera = 99
	case "cont_canon":
		a.sellos.canonControlRaw = 99
	case "cont_palabra":
		a.sellos.palabraObservada = 3
	case "cont_pareja":
		a.sellos.baselineSenal, a.sellos.palabraObservada = 3, 3
	case "canon_vacio":
		a.sellos.primera = uint32(observacionControlRawO3cM38)
		a.sellos.canonControlRaw = controlRawVacioO4aM38
	case "canon_ajeno":
		a.sellos.primera = uint32(observacionControlRawO3cM38)
		a.sellos.canonControlRaw = 99
	case "canon_no_control":
		a.sellos.canonControlRaw = controlRawCancelado65O4aM38
	case "palabra_estado":
		a.sellos.primera = uint32(observacionSenalRawO3cM38)
		a.sellos.palabraObservada = 3
	case "palabra_regresiva":
		a.sellos.primera = uint32(observacionSenalRawO3cM38)
		a.sellos.baselineSenal = 1<<10 | 2
	case "palabra_pidfd":
		a.sellos.palabraObservada += 1 << 10
	case "pareja_corrupta":
		a.sellos.baselineSenal, a.sellos.palabraObservada = 3, 3
	case "discriminante":
		a.sellos.primera = 99
	case "raw_negativo":
		a.sellos.retornoCont = -1
	case "causa_corrupta":
		a.causa.Store(99)
	case "estado_corrupto":
		a.estado.Store(uint32(causaA8EntregadoO4cM38))
	case "auto":
		a.auto = nil
	case "previa":
		a.sellos.primera, a.sellos.canonControlRaw, a.sellos.retornoCont = 99, 99, 5
		a.causa.Store(uint32(causaCancelado65O4aM38))
		if !errors.Is(sembrarCausaO4aM38(a), errSemillaConsumidaO4aM38) || a.causa.Load() != uint32(causaCancelado65O4aM38) {
			os.Exit(18)
		}
		os.Exit(0)
	case "carrera", "carrera_causa":
		fisico := copiaSnapshotO4aM38(a.origen.custodia.lease.fisico)
		resultados := make(chan error, 2)
		go func() { resultados <- sembrarCausaO4aM38(a) }()
		go func() { resultados <- sembrarCausaO4aM38(a) }()
		e1, e2 := <-resultados, <-resultados
		estado, causa := uint32(causaA2ObservandoM38), uint32(0)
		if caso == "carrera_causa" {
			estado, causa = uint32(causaA3CausaFijadaM38), uint32(causaIncidente65O4aM38)
		}
		if (e1 == nil) == (e2 == nil) || a.estado.Load() != estado || a.causa.Load() != causa ||
			a.origen.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
			a.origen.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) ||
			!snapshotsIgualesO3aM38(a.origen.custodia.lease.fisico, fisico) {
			os.Exit(19)
		}
		os.Exit(0)
	}
	primera, raw, palabra, canon := a.sellos.primera, a.sellos.retornoCont, a.sellos.palabraObservada, a.sellos.canonControlRaw
	fisico := copiaSnapshotO4aM38(a.origen.custodia.lease.fisico)
	err := sembrarCausaO4aM38(a)
	if err != nil {
		os.Exit(10)
	}
	esperada, estado := causaIncidente65O4aM38, causaA3CausaFijadaM38
	switch caso {
	case "vacio", "replay", "inmutable":
		esperada, estado = causaVaciaO4aM38, causaA2ObservandoM38
	case "control_cancelado":
		esperada = causaCancelado65O4aM38
	case "control_protocolo":
		esperada = causaProtocolo65O4aM38
	case "control_int", "senal_int":
		esperada = causaSenalInt130O4aM38
	case "control_term", "senal_term":
		esperada = causaSenalTerm143O4aM38
	case "natural":
		esperada = causaSalidaO4aM38
	}
	if a.causa.Load() != uint32(esperada) || a.estado.Load() != uint32(estado) || a.incidente.Load() != 0 ||
		a.sellos.primera != primera || a.sellos.retornoCont != raw || a.sellos.palabraObservada != palabra || a.sellos.canonControlRaw != canon ||
		a.origen.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
		a.origen.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) ||
		!snapshotsIgualesO3aM38(a.origen.custodia.lease.fisico, fisico) {
		os.Exit(11)
	}
	if caso == "replay" && !errors.Is(sembrarCausaO4aM38(a), errSemillaConsumidaO4aM38) {
		os.Exit(12)
	}
	if caso == "inmutable" {
		a.origen.custodia.observador.palabra.Add(4)
		a.origen.custodia.control.causa = causaPreinicioM38{causa: "CANCELADO", estado: "65"}
		if a.sellos.primera != primera || a.sellos.palabraObservada != palabra || a.sellos.canonControlRaw != canon {
			os.Exit(13)
		}
	}
	os.Exit(0)
}

func TestSemillaO4aP2EstructuraCerrada(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_semilla.go")
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
	for _, p := range []string{".origen.", ".control.", ".observador.", "Syscall(", "Poll(", "Now(", "Sleep(", "Kill(", "Wait(", "Close(", "Write(", "Print(", "Log("} {
		if bytes.Contains(contenido, []byte(p)) {
			t.Fatalf("lectura/efecto prohibido %s", p)
		}
	}
	if strings.Count(string(contenido), "a.causa.CompareAndSwap(") != 1 {
		t.Fatal("CAS causa no único")
	}
}
