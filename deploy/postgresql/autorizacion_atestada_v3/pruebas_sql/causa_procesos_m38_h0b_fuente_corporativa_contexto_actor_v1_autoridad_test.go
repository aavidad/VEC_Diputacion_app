//go:build ignore && linux && amd64

package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func sellosExactosO4aP1PruebaM38(a *autoridadCausaO4aM38, origen *agregadoO4aM38, fisico snapshotFDO3aM38) bool {
	if a == nil || origen == nil || origen.custodia == nil {
		return false
	}
	s, c := a.sellos, origen.custodia
	return s.autoridad == origen.autoridad && s.autoridadArranque == c.autoridad && s.custodia == c &&
		s.lease == c.lease && s.observador == c.observador && s.registro == c.lease.registro &&
		s.control == c.control && s.controlFD == c.controlFD && s.terminal == c.terminal &&
		s.cmd == c.cmd && s.proceso == c.cmd.Process && s.generacionLease == c.lease.generacion &&
		s.generacionObservador == c.observador.generacion && s.tid == c.tid && s.ppid == c.ppid &&
		s.baselineSenal == c.baselineSenal && s.pidfd == [3]int{c.pidfdPrimario, c.pidfdReserva, c.pidfdOpaco} &&
		s.identidad == origen.identidad && s.primera == origen.primera.Load() && s.retornoCont == origen.retornoCont &&
		maps.Equal(s.fisico.mapa, fisico.mapa) && s.fisico.limite == fisico.limite &&
		s.huellaControl == fisico.mapa[int(c.controlFD.Fd())] && s.huellaTerminal == fisico.mapa[int(c.terminal.Fd())]
}

func agregadoRealO4aP1PruebaM38(t *testing.T) *agregadoO4aM38 {
	t.Helper()
	_, a := autoridadRealO3cP4PruebaM38(t)
	a = observarInmediatoO3cM38(&a)
	if a == nil || !a.es(continuacionC3ObservadoM38) {
		t.Fatal("no llegó a C3")
	}
	return transferirHandoffO3cM38(&a)
}

func ejecutarAutoridadO4aP1AisladaM38(t *testing.T, caso string, estado int) {
	t.Helper()
	antes := hijosDirectosO3cP5Prueba()
	cmd := exec.Command(os.Args[0], "-test.run=^TestAutoridadO4aP1CasosAislados$")
	cmd.Env = append(os.Environ(), "O4A_P1_CASO="+caso)
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

func TestAutoridadO4aP1CasosAislados(t *testing.T) {
	caso := os.Getenv("O4A_P1_CASO")
	if caso == "" {
		for _, nombre := range []string{"positivo", "alias", "clon", "carrera", "nulo"} {
			ejecutarAutoridadO4aP1AisladaM38(t, nombre, 0)
		}
		for _, nombre := range []string{"owner", "owner_obs", "autoridad_auto", "autoridad_arranque", "lease", "registro", "registro_auto", "generacion", "tid", "lease_auto", "observador_auto", "baseline", "pidfd", "control", "terminal", "proceso", "identidad", "primera", "raw"} {
			ejecutarAutoridadO4aP1AisladaM38(t, nombre, estadoFallo)
		}
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if caso == "nulo" {
		var entrada *agregadoO4aM38
		if a, err := consumirAutoridadO4aM38(&entrada); a != nil || !errors.Is(err, errConsumoO4aM38) {
			os.Exit(10)
		}
		os.Exit(0)
	}
	entrada := agregadoRealO4aP1PruebaM38(t)
	origen := entrada
	clon := agregadoO4aM38{auto: entrada.auto, autoridad: entrada.autoridad, custodia: entrada.custodia,
		identidad: entrada.identidad, ahoraCaso: entrada.ahoraCaso, finCaso: entrada.finCaso, retornoCont: entrada.retornoCont}
	clon.primera.Store(entrada.primera.Load())
	primera, raw := entrada.primera.Load(), entrada.retornoCont
	fisico := copiaSnapshotO4aM38(entrada.custodia.lease.fisico)
	if caso == "carrera" {
		alias := entrada
		resultados := make(chan bool, 2)
		for _, candidato := range []*agregadoO4aM38{entrada, alias} {
			go func(x *agregadoO4aM38) {
				r, err := consumirAutoridadO4aM38(&x)
				resultados <- r != nil && err == nil
			}(candidato)
		}
		if primero, segundo := <-resultados, <-resultados; primero == segundo || origen.custodia.consumida.Load() != custodiaRecibidaO4aM38 {
			os.Exit(14)
		}
		os.Exit(0)
	}
	switch caso {
	case "owner":
		entrada.autoridad.ownerLease.Store(uint32(propietarioLiberadoO3cM38))
	case "owner_obs":
		entrada.autoridad.ownerObservador.Store(uint32(propietarioLiberadoO3cM38))
	case "autoridad_auto":
		entrada.autoridad.auto = nil
	case "autoridad_arranque":
		entrada.custodia.autoridad = nil
	case "lease":
		entrada.custodia.lease.estado.Store(2)
	case "registro":
		entrada.custodia.observador.registro = nil
	case "registro_auto":
		entrada.custodia.lease.registro.auto = nil
	case "generacion":
		entrada.custodia.lease.generacion++
	case "tid":
		entrada.custodia.lease.tid++
	case "lease_auto":
		entrada.custodia.lease.auto = nil
	case "observador_auto":
		entrada.custodia.observador.auto = nil
	case "baseline":
		entrada.custodia.baselineSenal += 1 << 10
	case "pidfd":
		entrada.custodia.pidfdReserva = entrada.custodia.pidfdPrimario
	case "control":
		entrada.custodia.control = nil
	case "terminal":
		entrada.custodia.terminal = nil
	case "proceso":
		entrada.custodia.cmd.Process = nil
	case "identidad":
		entrada.identidad.inicio = 0
	case "primera":
		entrada.primera.Store(99)
	case "raw":
		entrada.retornoCont = -1
	}
	if caso == "clon" {
		entradaClon := &clon
		if b, e := consumirAutoridadO4aM38(&entradaClon); b != nil || !errors.Is(e, errConsumoO4aM38) ||
			origen.custodia.consumida.Load() != custodiaEntregadaO3cM38 || entradaClon != nil {
			os.Exit(12)
		}
	}
	a, err := consumirAutoridadO4aM38(&entrada)
	if caso != "positivo" && caso != "alias" && caso != "clon" {
		os.Exit(99)
	}
	if err != nil || a == nil || entrada != nil || a.auto != a || a.origen != origen ||
		a.estado.Load() != uint32(causaA1ValidadoM38) || a.causa.Load() != uint32(causaVaciaO4aM38) ||
		a.incidente.Load() != 0 || origen.auto != origen || origen.custodia.consumida.Load() != custodiaRecibidaO4aM38 ||
		origen.primera.Load() != primera || origen.retornoCont != raw ||
		origen.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
		origen.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) ||
		!snapshotsIgualesO3aM38(origen.custodia.lease.fisico, fisico) || !sellosExactosO4aP1PruebaM38(a, origen, fisico) {
		os.Exit(11)
	}
	if caso == "alias" {
		alias := origen
		if b, e := consumirAutoridadO4aM38(&alias); b != nil || !errors.Is(e, errConsumoO4aM38) || alias != nil {
			os.Exit(13)
		}
	}
	os.Exit(0)
}

func TestAutoridadO4aP1EstructuraSinEfectos(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta prueba")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go")
	fset := token.NewFileSet()
	nodo, err := parser.ParseFile(fset, ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	prohibidas := []string{"Now", "Sleep", "Syscall", "Poll", "Kill", "Wait", "Close", "Write", "Print", "Log"}
	ast.Inspect(nodo, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.GoStmt, *ast.ChanType:
			t.Fatalf("concurrencia productiva prohibida %T", n)
		}
		return true
	})
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	for _, palabra := range prohibidas {
		if bytes.Contains(contenido, []byte(palabra+"(")) {
			t.Fatalf("efecto prohibido %s", palabra)
		}
	}
	if !bytes.Contains(contenido, []byte("maps.Clone")) {
		t.Fatal("snapshot no usa copia defensiva")
	}
}
