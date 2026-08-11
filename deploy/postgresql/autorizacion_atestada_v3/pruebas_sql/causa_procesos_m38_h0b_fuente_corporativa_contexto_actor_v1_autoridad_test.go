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
	"strings"
	"syscall"
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
		s.palabraObservada == c.observador.palabra.Load() &&
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
		for _, nombre := range []string{"positivo", "alias", "clon", "carrera", "nulo", "senal_int", "senal_term", "senal_delta_cero", "senal_signo_cero", "senal_signo_ajeno", "senal_multiple", "control_cancelado", "control_protocolo", "control_int", "control_term", "control_activo", "inmutable"} {
			ejecutarAutoridadO4aP1AisladaM38(t, nombre, 0)
		}
		for _, nombre := range []string{"owner", "owner_obs", "autoridad_auto", "autoridad_arranque", "custodia_nil", "lease", "registro", "registro_auto", "generacion", "tid", "lease_auto", "observador_auto", "baseline", "pidfd", "control", "terminal", "proceso", "identidad", "primera", "raw", "senal_regresiva", "senal_estado", "palabra_no_senal", "control_interno", "control_s2", "control_primera_adversa", "control_funcional_adverso", "control_funcional_invalido"} {
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
	baseline := entrada.custodia.baselineSenal
	palabraSenal := func(delta uint64, senal syscall.Signal) uint64 {
		return ((baseline>>10)+delta)<<10 | uint64(uint8(senal))<<2 | 2
	}
	if caso == "carrera" {
		alias := entrada
		resultados := make(chan *autoridadCausaO4aM38, 2)
		for _, candidato := range []*agregadoO4aM38{entrada, alias} {
			go func(x *agregadoO4aM38) {
				r, err := consumirAutoridadO4aM38(&x)
				if err != nil {
					r = nil
				}
				resultados <- r
			}(candidato)
		}
		primero, segundo := <-resultados, <-resultados
		ganador := primero
		if ganador == nil {
			ganador = segundo
		}
		if (primero == nil) == (segundo == nil) || ganador.origen != origen ||
			origen.custodia.consumida.Load() != custodiaRecibidaO4aM38 ||
			origen.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
			origen.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) ||
			origen.primera.Load() != primera || origen.retornoCont != raw ||
			origen.custodia.observador.palabra.Load() != baseline ||
			!snapshotsIgualesO3aM38(origen.custodia.lease.fisico, fisico) || !sellosExactosO4aP1PruebaM38(ganador, origen, fisico) {
			os.Exit(14)
		}
		os.Exit(0)
	}
	switch caso {
	case "senal_int":
		entrada.primera.Store(uint32(observacionSenalRawO3cM38))
		entrada.custodia.observador.palabra.Store(palabraSenal(1, syscall.SIGINT))
	case "senal_term":
		entrada.primera.Store(uint32(observacionSenalRawO3cM38))
		entrada.custodia.observador.palabra.Store(palabraSenal(1, syscall.SIGTERM))
	case "senal_delta_cero":
		entrada.primera.Store(uint32(observacionSenalRawO3cM38))
	case "senal_signo_cero":
		entrada.primera.Store(uint32(observacionSenalRawO3cM38))
		entrada.custodia.observador.palabra.Store(palabraSenal(1, 0))
	case "senal_signo_ajeno":
		entrada.primera.Store(uint32(observacionSenalRawO3cM38))
		entrada.custodia.observador.palabra.Store(palabraSenal(1, syscall.SIGUSR1))
	case "senal_multiple":
		entrada.primera.Store(uint32(observacionSenalRawO3cM38))
		entrada.custodia.observador.palabra.Store(palabraSenal(2, syscall.SIGTERM))
	case "senal_regresiva":
		entrada.primera.Store(uint32(observacionSenalRawO3cM38))
		entrada.custodia.baselineSenal = (1 << 10) | 2
	case "senal_estado":
		entrada.primera.Store(uint32(observacionSenalRawO3cM38))
		entrada.custodia.observador.palabra.Store(3)
	case "palabra_no_senal":
		entrada.custodia.observador.palabra.Store(palabraSenal(1, syscall.SIGINT))
	case "control_cancelado", "control_protocolo", "control_int", "control_term", "inmutable":
		causa, estado := "CANCELADO", "65"
		switch caso {
		case "control_protocolo":
			causa = "PROTOCOLO"
		case "control_int":
			causa, estado = "SENAL_INT", "130"
		case "control_term":
			causa, estado = "SENAL_TERM", "143"
		}
		entrada.primera.Store(uint32(observacionControlRawO3cM38))
		entrada.custodia.control.enclavarCausa(causa, estado)
		entrada.custodia.primeraCausa = entrada.custodia.control.causa
	case "control_activo":
		entrada.primera.Store(uint32(observacionControlRawO3cM38))
	case "control_interno":
		entrada.primera.Store(uint32(observacionControlRawO3cM38))
		entrada.custodia.control.limpiarConFallo(errInvarianteControlPreinicioM38)
	case "control_s2":
		entrada.primera.Store(uint32(observacionControlRawO3cM38))
		entrada.custodia.control.fase = controlPreinicioS2M38
	case "control_primera_adversa":
		entrada.primera.Store(uint32(observacionControlRawO3cM38))
		entrada.custodia.primeraCausa = causaPreinicioM38{faseOrigen: controlPreinicioS3M38, causa: "CANCELADO", estado: "65"}
	case "control_funcional_adverso", "control_funcional_invalido":
		entrada.primera.Store(uint32(observacionControlRawO3cM38))
		entrada.custodia.control.enclavarCausa("CANCELADO", "65")
		entrada.custodia.primeraCausa = entrada.custodia.control.causa
		if caso == "control_funcional_adverso" {
			entrada.custodia.primeraCausa.causa = "PROTOCOLO"
		} else {
			entrada.custodia.control.causa.estado = "64"
		}
	case "owner":
		entrada.autoridad.ownerLease.Store(uint32(propietarioLiberadoO3cM38))
	case "owner_obs":
		entrada.autoridad.ownerObservador.Store(uint32(propietarioLiberadoO3cM38))
	case "autoridad_auto":
		entrada.autoridad.auto = nil
	case "autoridad_arranque":
		entrada.custodia.autoridad = nil
	case "custodia_nil":
		entrada.custodia = nil
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
	primera, raw = entrada.primera.Load(), entrada.retornoCont
	if caso == "clon" {
		entradaClon := &clon
		if b, e := consumirAutoridadO4aM38(&entradaClon); b != nil || !errors.Is(e, errConsumoO4aM38) ||
			origen.custodia.consumida.Load() != custodiaEntregadaO3cM38 || entradaClon != nil {
			os.Exit(12)
		}
	}
	a, err := consumirAutoridadO4aM38(&entrada)
	casoValido := caso == "positivo" || caso == "alias" || caso == "clon" || strings.HasPrefix(caso, "senal_") && caso != "senal_regresiva" && caso != "senal_estado" || strings.HasPrefix(caso, "control_") && caso != "control_interno" && caso != "control_s2" && caso != "control_primera_adversa" && caso != "control_funcional_adverso" && caso != "control_funcional_invalido" || caso == "inmutable"
	if !casoValido {
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
	esperadoCanon := controlRawVacioO4aM38
	switch caso {
	case "control_cancelado", "inmutable":
		esperadoCanon = controlRawCancelado65O4aM38
	case "control_protocolo", "control_activo":
		esperadoCanon = controlRawProtocolo65O4aM38
	case "control_int":
		esperadoCanon = controlRawSenalInt130O4aM38
	case "control_term":
		esperadoCanon = controlRawSenalTerm143O4aM38
	}
	if a.sellos.canonControlRaw != esperadoCanon || a.sellos.primera != origen.primera.Load() ||
		a.sellos.palabraObservada != origen.custodia.observador.palabra.Load() {
		os.Exit(15)
	}
	if caso == "inmutable" {
		palabraSellada, canonSellado := a.sellos.palabraObservada, a.sellos.canonControlRaw
		origen.custodia.observador.palabra.Add(1 << 10)
		origen.custodia.control.causa = causaPreinicioM38{}
		if a.sellos.palabraObservada != palabraSellada || a.sellos.canonControlRaw != canonSellado {
			os.Exit(16)
		}
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
	if strings.Count(string(contenido), ".primera.Load()") != 1 || strings.Count(string(contenido), ".palabra.Load()") != 1 {
		t.Fatal("captura raw no es única")
	}
	posBase := bytes.Index(contenido, []byte("entradaBaseExactaO4aM38(a)"))
	posCaptura, posCAS := bytes.Index(contenido, []byte("a.primera.Load()")), bytes.Index(contenido, []byte("CompareAndSwap(custodiaEntregadaO3cM38"))
	if posBase < 0 || posCaptura <= posBase || posCAS <= posCaptura {
		t.Fatal("orden prevalidación-captura-CAS divergente")
	}
}
