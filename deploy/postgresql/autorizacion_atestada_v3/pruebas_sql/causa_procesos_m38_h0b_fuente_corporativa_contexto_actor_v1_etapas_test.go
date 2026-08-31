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
	"testing"
	"time"
)

func autoridadSinteticaEtapasO4aP4M38(t *testing.T, causa causaPrimariaO4aM38) *autoridadCausaO4aM38 {
	t.Helper()
	tid := 731
	registro := &registroAutoridadO3aM38{
		tid: tid, leases: make(map[*leaseGuardiaO3aM38]uint64),
		observadores: make(map[*observadorSenalO3aM38]uint64),
	}
	registro.auto = registro
	controlFD, terminal, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = controlFD.Close()
		_ = terminal.Close()
	})
	fdControl, fdTerminal := int(controlFD.Fd()), int(terminal.Fd())
	pidfdPrimario, pidfdReserva, pidfdOpaco := fdTerminal+10, fdTerminal+11, fdTerminal+12
	identidadPidfd := identidadFDO3aM38{dev: 1, ino: 7, fdflags: 1}
	fisico := snapshotFDO3aM38{limite: 64, mapa: map[int]huellaFDO3aM38{
		pidfdPrimario: {identidad: identidadPidfd, abierto: true},
		pidfdReserva:  {identidad: identidadPidfd, abierto: true},
		pidfdOpaco:    {identidad: identidadPidfd, abierto: true},
		fdControl:     {identidad: identidadFDO3aM38{dev: 2, ino: 8, fdflags: 1}, abierto: true},
		fdTerminal:    {identidad: identidadFDO3aM38{dev: 2, ino: 9, fdflags: 1}, abierto: true},
	}}
	lease := &leaseGuardiaO3aM38{registro: registro, generacion: 41, tid: tid, fisico: fisico}
	lease.auto = lease
	lease.estado.Store(3)
	registro.leases[lease] = lease.generacion
	observador := &observadorSenalO3aM38{registro: registro, generacion: 42}
	observador.auto = observador
	observador.palabra.Store(2)
	registro.observadores[observador] = observador.generacion
	control := &controladorPreinicioM38{}
	cmd := &exec.Cmd{Process: &os.Process{Pid: 801}}
	autoridadArranque := &autoridadEstadoO3aM38{estado: arranqueA6EntregadoM38}
	inicio := time.Now()
	custodia := &custodiaO3aM38{
		autoridad: autoridadArranque, control: control, controlFD: controlFD, terminal: terminal,
		lease: lease, observador: observador, baselineSenal: 2, tid: tid, ppid: 1, cmd: cmd,
		pidfdPrimario: pidfdPrimario, pidfdReserva: pidfdReserva, pidfdOpaco: pidfdOpaco,
		finBootstrap: inicio.Add(time.Minute),
	}
	custodia.consumida.Store(custodiaRecibidaO4aM38)
	autoridadCustodia := nuevaAutoridadCustodiaO3cM38()
	autoridadCustodia.ownerObservador.Store(uint32(propietarioO4aM38))
	autoridadCustodia.ownerLease.Store(uint32(propietarioO4aM38))
	origen := &agregadoO4aM38{autoridad: autoridadCustodia, custodia: custodia, ahoraCaso: inicio, finCaso: inicio.Add(duracionCasoO3cM38)}
	origen.auto = origen
	origen.primera.Store(uint32(observacionPidfdVacioO3cM38))
	identidad := muestraStatO3bM38{estado: 'T', pid: cmd.Process.Pid, ppid: 1, pgid: cmd.Process.Pid, sid: 1, inicio: 9}
	origen.identidad = identidad
	a := &autoridadCausaO4aM38{origen: origen}
	a.auto = a
	a.sellos = sellosO4aM38{
		autoridad: autoridadCustodia, autoridadArranque: autoridadArranque, custodia: custodia,
		lease: lease, observador: observador, registro: registro, control: control,
		controlFD: controlFD, terminal: terminal, cmd: cmd, proceso: cmd.Process,
		generacionLease: lease.generacion, generacionObservador: observador.generacion,
		tid: tid, ppid: 1, baselineSenal: 2, pidfd: [3]int{pidfdPrimario, pidfdReserva, pidfdOpaco},
		identidad: identidad, primera: uint32(observacionPidfdVacioO3cM38), palabraObservada: 2,
		canonControlRaw: controlRawVacioO4aM38, ahoraCaso: inicio, finCaso: origen.finCaso,
		fisico: copiaSnapshotO4aM38(fisico), huellaControl: fisico.mapa[fdControl], huellaTerminal: fisico.mapa[fdTerminal],
	}
	a.causa.Store(uint32(causa))
	a.estado.Store(uint32(causaA3CausaFijadaM38))
	return a
}

func iniciarEtapasPruebaO4aP4M38(t *testing.T, causa causaPrimariaO4aM38) (*autoridadEtapasO4aM38, *autorizacionEtapaO4aM38) {
	t.Helper()
	a := autoridadSinteticaEtapasO4aP4M38(t, causa)
	e, p, err := iniciarEtapasO4aM38(&a)
	if err != nil || e == nil || a != nil {
		t.Fatalf("inicio: e=%p p=%p err=%v a=%p", e, p, err, a)
	}
	return e, p
}

func sellarResultadoEtapaPruebaO4aP4M38(t *testing.T, e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38, cardinal uint8, raw1, raw2 int, evidencia claseEvidenciaEtapaO4bM38, observado time.Time) *resultadoEtapaO4bM38 {
	t.Helper()
	if p == nil || !p.estado.CompareAndSwap(uint32(autorizacionEmitidaO4bM38), uint32(autorizacionConsumiendoO4bM38)) ||
		!confirmarConsumoAutorizacionEtapaO4aM38(e, p) {
		t.Fatal("consumo permiso")
	}
	r := p.resultado
	r.cardinalidad, r.rawPrimero, r.rawSegundo, r.evidencia, r.observado = cardinal, raw1, raw2, evidencia, observado
	if !r.estado.CompareAndSwap(uint32(resultadoVacioO4bM38), uint32(resultadoSelladoO4bM38)) ||
		!p.estado.CompareAndSwap(uint32(autorizacionConsumiendoO4bM38), uint32(autorizacionConsumidaO4bM38)) {
		t.Fatal("sellado resultado")
	}
	return r
}

func aplicarResultadoEnPruebaO4aP4M38(t *testing.T, e *autoridadEtapasO4aM38, r *resultadoEtapaO4bM38, ahora time.Time) *autorizacionEtapaO4aM38 {
	t.Helper()
	if !r.estado.CompareAndSwap(uint32(resultadoSelladoO4bM38), uint32(resultadoConsumiendoO4bM38)) {
		t.Fatal("resultado no disponible")
	}
	return aplicarResultadoEnO4aM38(e, r, ahora)
}

func permisoExactoEtapaPruebaO4aP4M38(p *autorizacionEtapaO4aM38, e *autoridadEtapasO4aM38, etapa etapaO4bM38, operacion operacionEtapaO4bM38, maxima uint8, limite time.Time, clase claseLimiteEtapaO4bM38) bool {
	return p != nil && p.auto == p && p.autoridad == e && p.resultado != nil && p.resultado.auto == p.resultado &&
		p.resultado.autorizacion == p && p.generacion > 0 && p.resultado.generacion == p.generacion &&
		p.tid == e.causa.sellos.tid && p.etapa == etapa && p.operacion == operacion &&
		p.cardinalidadMaxima == maxima && p.limite == limite && p.claseLimite == clase &&
		p.rolPidfd == rolPidfdPrimarioO4bM38 && p.estado.Load() == uint32(autorizacionEmitidaO4bM38)
}

func hastaTermContPruebaO4aP4M38(t *testing.T) (*autoridadEtapasO4aM38, *autorizacionEtapaO4aM38) {
	t.Helper()
	e, stop := iniciarEtapasPruebaO4aP4M38(t, causaSenalTerm143O4aM38)
	observado := stop.limite.Add(-time.Nanosecond)
	r := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, 1, 0, 0, evidenciaEstableO4bM38, observado)
	term := aplicarResultadoEnPruebaO4aP4M38(t, e, r, stop.limite)
	if !permisoExactoEtapaPruebaO4aP4M38(term, e, etapaTerminarReanudarO4bM38, operacionTermContO4bM38, 2, e.plazos.finGracia, limiteGraciaO4bM38) {
		t.Fatal("TERM-CONT no emitido")
	}
	return e, term
}

func hastaParadaFinalPruebaO4aP4M38(t *testing.T) (*autoridadEtapasO4aM38, *autorizacionEtapaO4aM38) {
	t.Helper()
	e, term := hastaTermContPruebaO4aP4M38(t)
	r := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 2, 0, 0, evidenciaGrupoPresenteO4bM38, e.plazos.finGracia)
	final := aplicarResultadoEnPruebaO4aP4M38(t, e, r, e.plazos.finGracia)
	if !permisoExactoEtapaPruebaO4aP4M38(final, e, etapaParadaFinalO4bM38, operacionStopO4bM38, 1, e.plazos.finParadaFinal, limiteParadaFinalO4bM38) {
		t.Fatal("PARADA_FINAL condicional no emitida")
	}
	return e, final
}

func TestEtapasO4aP4PlazosMaximosYSalida(t *testing.T) {
	ahora := time.Now()
	parada, rapido, ok := plazosInicialesO4aM38(ahora)
	if !ok || parada.Sub(ahora) != time.Second || rapido.Sub(parada) != 5*time.Second {
		t.Fatal("plazos iniciales divergentes")
	}
	gracia, final, cooperativo, ok := plazosCooperativosO4aM38(ahora)
	if !ok || gracia.Sub(ahora) != 2*time.Second || final.Sub(gracia) != time.Second || cooperativo.Sub(final) != 5*time.Second {
		t.Fatal("plazos cooperativos divergentes")
	}
	if _, ok := sumarPlazoEtapaO4aM38(time.Time{}, time.Second); ok {
		t.Fatal("marca cero admitida")
	}
	for _, causa := range []causaPrimariaO4aM38{causaCancelado65O4aM38, causaProtocolo65O4aM38, causaSenalInt130O4aM38, causaSenalTerm143O4aM38, causaPlazo65O4aM38, causaIncidente65O4aM38} {
		e, p := iniciarEtapasPruebaO4aP4M38(t, causa)
		if !e.extincion || e.terminalidad || e.causa.incidente.Load() != 0 ||
			!permisoExactoEtapaPruebaO4aP4M38(p, e, etapaParadaInicialO4bM38, operacionStopO4bM38, 1, e.plazos.finParadaInicial, limiteParadaInicialO4bM38) {
			t.Fatalf("inicio divergente causa=%d", causa)
		}
	}
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaSalidaO4aM38)
	if p != nil || !e.terminalidad || e.extincion || e.emitidas != 0 ||
		e.causa.estado.Load() != uint32(causaA7EntregaO4cPreparadaM38) || !tiempoMonotonoO3cM38(e.plazos.finDrenajeNatural) {
		t.Fatal("SALIDA no convergió sin autorización")
	}
}

func TestEtapasO4aP4TerminalPostContYBordeGracia(t *testing.T) {
	for _, delta := range []time.Duration{-time.Nanosecond, 0} {
		t.Run(delta.String(), func(t *testing.T) {
			e, term := hastaTermContPruebaO4aP4M38(t)
			observado := e.plazos.finGracia.Add(delta)
			r := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 2, 0, 0, evidenciaTerminalO4bM38, observado)
			if p := aplicarResultadoEnPruebaO4aP4M38(t, e, r, observado); p != nil || !e.terminalidad ||
				e.emitidas != 2 || e.causa.incidente.Load() != 0 || e.causa.estado.Load() != uint32(causaA7EntregaO4cPreparadaM38) {
				t.Fatal("terminalidad post-CONT no llegó limpia a A7")
			}
		})
	}
	t.Run("presente_antes", func(t *testing.T) {
		e, term := hastaTermContPruebaO4aP4M38(t)
		observado := e.plazos.finGracia.Add(-time.Nanosecond)
		r := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 2, 0, 0, evidenciaGrupoPresenteO4bM38, observado)
		if p := aplicarResultadoEnPruebaO4aP4M38(t, e, r, observado); p != nil || e.causa.estado.Load() != uint32(causaA3CausaFijadaM38) {
			t.Fatal("presencia anterior no esperó en A3")
		}
		if continuarEtapasEnO4aM38(e, observado) != nil {
			t.Fatal("presencia anterior emitió STOP")
		}
		final := continuarEtapasEnO4aM38(e, e.plazos.finGracia)
		if !permisoExactoEtapaPruebaO4aP4M38(final, e, etapaParadaFinalO4bM38, operacionStopO4bM38, 1, e.plazos.finParadaFinal, limiteParadaFinalO4bM38) {
			t.Fatal("borde no emitió una PARADA_FINAL condicional")
		}
	})
	t.Run("presente_igual", func(t *testing.T) {
		e, term := hastaTermContPruebaO4aP4M38(t)
		r := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 2, 0, 0, evidenciaGrupoPresenteO4bM38, e.plazos.finGracia)
		final := aplicarResultadoEnPruebaO4aP4M38(t, e, r, e.plazos.finGracia)
		if !permisoExactoEtapaPruebaO4aP4M38(final, e, etapaParadaFinalO4bM38, operacionStopO4bM38, 1, e.plazos.finParadaFinal, limiteParadaFinalO4bM38) || e.emitidas != 3 {
			t.Fatal("igualdad de gracia no produjo una única autorización")
		}
	})
}

func TestEtapasO4aP4TerminalFinalCardinalRealCeroOUno(t *testing.T) {
	for _, cardinal := range []uint8{0, 1} {
		t.Run(string(rune('0'+cardinal)), func(t *testing.T) {
			e, final := hastaParadaFinalPruebaO4aP4M38(t)
			if final.cardinalidadMaxima != 1 || final.resultado.cardinalidad != 0 {
				t.Fatal("máximo y cardinalidad real se confundieron")
			}
			r := sellarResultadoEtapaPruebaO4aP4M38(t, e, final, cardinal, 0, 0, evidenciaTerminalO4bM38, e.plazos.finParadaFinal)
			if p := aplicarResultadoEnPruebaO4aP4M38(t, e, r, e.plazos.finParadaFinal); p != nil ||
				!e.terminalidad || e.causa.incidente.Load() != 0 || e.emitidas != 3 || e.historialLen != 3 ||
				e.causa.estado.Load() != uint32(causaA7EntregaO4cPreparadaM38) {
				t.Fatalf("terminal final cardinal=%d no llegó limpia a A7", cardinal)
			}
		})
	}
}

func TestEtapasO4aP4RamasFinalesExactas(t *testing.T) {
	casos := []struct {
		nombre    string
		raw       int
		evidencia claseEvidenciaEtapaO4bM38
		observado func(*autoridadEtapasO4aM38) time.Time
		incidente uint32
	}{
		{"estable", 0, evidenciaEstableO4bM38, func(e *autoridadEtapasO4aM38) time.Time { return e.plazos.finParadaFinal.Add(-time.Nanosecond) }, 0},
		{"no_estable", 0, evidenciaNoEstableO4bM38, func(e *autoridadEtapasO4aM38) time.Time { return e.plazos.finParadaFinal }, 1},
		{"raw_error", 4, evidenciaSinO4bM38, func(*autoridadEtapasO4aM38) time.Time { return time.Time{} }, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e, final := hastaParadaFinalPruebaO4aP4M38(t)
			r := sellarResultadoEtapaPruebaO4aP4M38(t, e, final, 1, caso.raw, 0, caso.evidencia, caso.observado(e))
			kill := aplicarResultadoEnPruebaO4aP4M38(t, e, r, e.plazos.finParadaFinal)
			if !permisoExactoEtapaPruebaO4aP4M38(kill, e, etapaMatarGrupoO4bM38, operacionKillO4bM38, 1, e.plazos.finDrenajeCooperativo, limiteDrenajeCooperativoO4bM38) ||
				e.causa.incidente.Load() != caso.incidente {
				t.Fatalf("rama final divergente: incidente=%d", e.causa.incidente.Load())
			}
		})
	}
}

func TestEtapasO4aP4ParadaInicialPermaneceCardinalUno(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		raw       int
		evidencia claseEvidenciaEtapaO4bM38
		observado func(*autorizacionEtapaO4aM38) time.Time
	}{
		{"no_estable", 0, evidenciaNoEstableO4bM38, func(p *autorizacionEtapaO4aM38) time.Time { return p.limite }},
		{"raw_error", 4, evidenciaSinO4bM38, func(*autorizacionEtapaO4aM38) time.Time { return time.Time{} }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, stop := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
			r := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, 1, caso.raw, 0, caso.evidencia, caso.observado(stop))
			kill := aplicarResultadoEnPruebaO4aP4M38(t, e, r, stop.limite)
			if !permisoExactoEtapaPruebaO4aP4M38(kill, e, etapaMatarGrupoO4bM38, operacionKillO4bM38, 1, e.plazos.finDrenajeRapido, limiteDrenajeRapidoO4bM38) ||
				e.causa.incidente.Load() != incidenteCierreEnclavadoO4aM38 || !e.plazos.finGracia.IsZero() {
				t.Fatal("PARADA_INICIAL no convergió a KILL rápido")
			}
		})
	}
}

func TestEtapasO4aP4AliasCarreraYReplay(t *testing.T) {
	a := autoridadSinteticaEtapasO4aP4M38(t, causaCancelado65O4aM38)
	a1, a2 := a, a
	type inicio struct {
		e   *autoridadEtapasO4aM38
		p   *autorizacionEtapaO4aM38
		err error
	}
	inicios := make(chan inicio, 2)
	for _, entrada := range []**autoridadCausaO4aM38{&a1, &a2} {
		go func(x **autoridadCausaO4aM38) {
			e, p, err := iniciarEtapasO4aM38(x)
			inicios <- inicio{e, p, err}
		}(entrada)
	}
	i1, i2 := <-inicios, <-inicios
	ganador := i1
	if ganador.err != nil {
		ganador = i2
	}
	if (i1.err == nil) == (i2.err == nil) || ganador.e == nil || ganador.p == nil || a1 != nil || a2 != nil {
		t.Fatal("alias de autoridad no dejó un ganador")
	}
	p := ganador.p
	consumos := make(chan bool, 2)
	for range 2 {
		go func() {
			ok := p.estado.CompareAndSwap(uint32(autorizacionEmitidaO4bM38), uint32(autorizacionConsumiendoO4bM38))
			if ok {
				ok = confirmarConsumoAutorizacionEtapaO4aM38(ganador.e, p)
			}
			consumos <- ok
		}()
	}
	if c1, c2 := <-consumos, <-consumos; c1 == c2 {
		t.Fatal("alias de permiso no fue one-shot")
	}
	r := p.resultado
	r.cardinalidad, r.rawPrimero, r.evidencia = 1, 4, evidenciaSinO4bM38
	if !r.estado.CompareAndSwap(0, uint32(resultadoSelladoO4bM38)) ||
		!p.estado.CompareAndSwap(uint32(autorizacionConsumiendoO4bM38), uint32(autorizacionConsumidaO4bM38)) {
		t.Fatal("sellado tras carrera")
	}
	r1, r2 := r, r
	type salida struct {
		p   *autorizacionEtapaO4aM38
		err error
	}
	salidas := make(chan salida, 2)
	for _, entrada := range []**resultadoEtapaO4bM38{&r1, &r2} {
		go func(x **resultadoEtapaO4bM38) {
			q, err := aplicarResultadoEtapaO4aM38(ganador.e, x)
			salidas <- salida{q, err}
		}(entrada)
	}
	s1, s2 := <-salidas, <-salidas
	if (s1.err == nil) == (s2.err == nil) || (s1.p == nil) == (s2.p == nil) || r1 != nil || r2 != nil || ganador.e.historialLen != 1 {
		t.Fatal("alias de resultado no dejó un ganador")
	}
	replay := r
	if q, err := aplicarResultadoEtapaO4aM38(ganador.e, &replay); !errors.Is(err, errEtapasConsumidasO4aM38) || q != nil || replay != nil || ganador.e.historialLen != 1 {
		t.Fatal("replay no falló cerrado")
	}
	e2, p2 := iniciarEtapasPruebaO4aP4M38(t, causaProtocolo65O4aM38)
	forjado := &autorizacionEtapaO4aM38{autoridad: e2, resultado: p2.resultado, generacion: p2.generacion, tid: p2.tid, etapa: p2.etapa, cardinalidadMaxima: p2.cardinalidadMaxima, operacion: p2.operacion, limite: p2.limite, claseLimite: p2.claseLimite, rolPidfd: p2.rolPidfd}
	forjado.auto = forjado
	forjado.estado.Store(uint32(autorizacionConsumiendoO4bM38))
	if confirmarConsumoAutorizacionEtapaO4aM38(e2, forjado) || e2.causa.estado.Load() != uint32(causaA4PermisoPreparadoM38) {
		t.Fatal("forja de autorización adquirió autoridad")
	}
}

func ejecutarFatalEtapasO4aP4M38(t *testing.T, caso string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestEtapasO4aP4Fatales$")
	cmd.Env = append(os.Environ(), "O4A_P4_FATAL="+caso)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var salida *exec.ExitError
	if !errors.As(err, &salida) || salida.ExitCode() != estadoFallo || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("fatal %s: err=%v stdout=%d stderr=%d", caso, err, stdout.Len(), stderr.Len())
	}
}

func TestEtapasO4aP4Fatales(t *testing.T) {
	caso := os.Getenv("O4A_P4_FATAL")
	if caso == "" {
		for _, nombre := range []string{"post_cont_tardio", "inicial_terminal", "inicial_cardinal_cero", "final_cardinal_dos", "final_terminal_raw", "final_marca_tardia", "marca_futura", "resultado_forjado", "permiso_maximo_forjado", "final_estable_en_borde", "kill_en_borde"} {
			ejecutarFatalEtapasO4aP4M38(t, nombre)
		}
		return
	}
	switch caso {
	case "post_cont_tardio":
		e, term := hastaTermContPruebaO4aP4M38(t)
		observado := e.plazos.finGracia.Add(time.Nanosecond)
		r := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 2, 0, 0, evidenciaGrupoPresenteO4bM38, observado)
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, r, observado)
	case "inicial_terminal", "inicial_cardinal_cero":
		e, stop := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
		cardinal := uint8(1)
		if caso == "inicial_cardinal_cero" {
			cardinal = 0
		}
		r := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, cardinal, 0, 0, evidenciaTerminalO4bM38, stop.limite.Add(-time.Nanosecond))
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, r, stop.limite)
	case "final_cardinal_dos", "final_terminal_raw", "final_marca_tardia", "marca_futura", "final_estable_en_borde":
		e, final := hastaParadaFinalPruebaO4aP4M38(t)
		cardinal, raw, evidencia, observado, ahora := uint8(0), 0, evidenciaTerminalO4bM38, e.plazos.finParadaFinal, e.plazos.finParadaFinal
		switch caso {
		case "final_cardinal_dos":
			cardinal = 2
		case "final_terminal_raw":
			cardinal, raw = 1, 4
		case "final_marca_tardia":
			observado, ahora = observado.Add(time.Nanosecond), ahora.Add(time.Nanosecond)
		case "marca_futura":
			ahora = ahora.Add(-time.Nanosecond)
		case "final_estable_en_borde":
			cardinal, evidencia = 1, evidenciaEstableO4bM38
		}
		r := sellarResultadoEtapaPruebaO4aP4M38(t, e, final, cardinal, raw, 0, evidencia, observado)
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, r, ahora)
	case "resultado_forjado":
		e, stop := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
		if !stop.estado.CompareAndSwap(uint32(autorizacionEmitidaO4bM38), uint32(autorizacionConsumiendoO4bM38)) || !confirmarConsumoAutorizacionEtapaO4aM38(e, stop) {
			os.Exit(20)
		}
		falso := &resultadoEtapaO4bM38{autorizacion: stop, generacion: stop.generacion, etapa: stop.etapa, limite: stop.limite, claseLimite: stop.claseLimite, cardinalidad: 1, rawPrimero: 4, evidencia: evidenciaSinO4bM38}
		falso.auto = falso
		falso.estado.Store(uint32(resultadoSelladoO4bM38))
		stop.estado.Store(uint32(autorizacionConsumidaO4bM38))
		_, _ = aplicarResultadoEtapaO4aM38(e, &falso)
	case "permiso_maximo_forjado":
		e, stop := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
		r := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, 1, 4, 0, evidenciaSinO4bM38, time.Time{})
		stop.cardinalidadMaxima = 0
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, r, stop.limite)
	case "kill_en_borde":
		e, final := hastaParadaFinalPruebaO4aP4M38(t)
		r := sellarResultadoEtapaPruebaO4aP4M38(t, e, final, 1, 0, 0, evidenciaEstableO4bM38, final.limite.Add(-time.Nanosecond))
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, r, e.plazos.finDrenajeCooperativo)
	}
	os.Exit(90)
}

func TestEtapasO4aP4EstructuraOpacaSinEfectos(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go")
	nodo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tipos := make(map[string]map[string]bool)
	ast.Inspect(nodo, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name.IsExported() {
				t.Fatalf("API exportada %s", x.Name.Name)
			}
		case *ast.GoStmt, *ast.ChanType:
			t.Fatalf("concurrencia productiva %T", n)
		case *ast.TypeSpec:
			estructura, existe := x.Type.(*ast.StructType)
			if !existe {
				break
			}
			campos := make(map[string]bool)
			for _, campo := range estructura.Fields.List {
				for _, nombre := range campo.Names {
					campos[nombre.Name] = true
				}
			}
			tipos[x.Name.Name] = campos
		}
		return true
	})
	if !tipos["autorizacionEtapaO4aM38"]["cardinalidadMaxima"] || tipos["autorizacionEtapaO4aM38"]["cardinalidad"] ||
		!tipos["resultadoEtapaO4bM38"]["cardinalidad"] || tipos["resultadoEtapaO4bM38"]["cardinalidadMaxima"] {
		t.Fatal("máximo autorizado y cardinalidad real no están separados")
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)
	for _, prohibida := range []string{"syscall.", ".Fd()", "Signal(", "Kill(", "Wait(", "Wait4(", "Waitid(", "Close(", "Write(", "Sleep(", "NewTimer(", "time.After(", "context.", "pidfd_open", "F_DUPFD", "/proc/", "parser.", "ownerLease.Store", "ownerObservador.Store"} {
		if strings.Contains(texto, prohibida) {
			t.Fatalf("efecto físico prohibido %s", prohibida)
		}
	}
	if strings.Count(texto, "time.Now()") != 3 ||
		strings.Count(texto, "CompareAndSwap(uint32(autorizacionVaciaO4bM38), uint32(autorizacionEmitidaO4bM38))") != 1 ||
		strings.Count(texto, "CompareAndSwap(0, incidenteCierreEnclavadoO4aM38)") != 1 {
		t.Fatal("reloj, emisión o latch divergente")
	}
	for _, nombre := range []string{"etapaParadaInicialO4bM38", "etapaTerminarReanudarO4bM38", "etapaParadaFinalO4bM38", "etapaMatarGrupoO4bM38"} {
		if !strings.Contains(texto, nombre) {
			t.Fatalf("etapa cerrada ausente %s", nombre)
		}
	}
}
