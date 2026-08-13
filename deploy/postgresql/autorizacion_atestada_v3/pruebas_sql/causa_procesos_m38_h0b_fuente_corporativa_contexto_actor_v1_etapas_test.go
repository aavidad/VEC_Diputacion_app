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
	"sync"
	"testing"
	"time"
)

func autoridadSinteticaEtapasO4aP4M38(causa causaPrimariaO4aM38) *autoridadCausaO4aM38 {
	tid := 731
	registro := &registroAutoridadO3aM38{tid: tid, leases: make(map[*leaseGuardiaO3aM38]uint64), observadores: make(map[*observadorSenalO3aM38]uint64)}
	registro.auto = registro
	fisico := snapshotFDO3aM38{limite: 64, mapa: map[int]huellaFDO3aM38{
		10: {identidad: identidadFDO3aM38{dev: 1, ino: 7, fdflags: 1}, abierto: true},
		11: {identidad: identidadFDO3aM38{dev: 1, ino: 7, fdflags: 1}, abierto: true},
		12: {identidad: identidadFDO3aM38{dev: 1, ino: 7, fdflags: 1}, abierto: true},
	}}
	lease := &leaseGuardiaO3aM38{registro: registro, generacion: 41, tid: tid, fisico: fisico}
	lease.auto = lease
	lease.estado.Store(3)
	registro.leases[lease] = lease.generacion
	observador := &observadorSenalO3aM38{registro: registro, generacion: 42}
	observador.auto = observador
	observador.palabra.Store(2)
	registro.observadores[observador] = observador.generacion
	control, controlFD, terminal := &controladorPreinicioM38{}, new(os.File), new(os.File)
	cmd := &exec.Cmd{Process: &os.Process{Pid: 801}}
	autoridadArranque := &autoridadEstadoO3aM38{estado: arranqueA6EntregadoM38}
	custodia := &custodiaO3aM38{
		autoridad: autoridadArranque, control: control, controlFD: controlFD, terminal: terminal,
		lease: lease, observador: observador, baselineSenal: 2, tid: tid, ppid: 1, cmd: cmd,
		pidfdPrimario: 10, pidfdReserva: 11, pidfdOpaco: 12,
	}
	autoridadCustodia := nuevaAutoridadCustodiaO3cM38()
	autoridadCustodia.ownerObservador.Store(uint32(propietarioO4aM38))
	autoridadCustodia.ownerLease.Store(uint32(propietarioO4aM38))
	inicio := time.Now()
	fin := inicio.Add(duracionCasoO3cM38)
	origen := &agregadoO4aM38{autoridad: autoridadCustodia, custodia: custodia, ahoraCaso: inicio, finCaso: fin}
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
		tid: tid, ppid: 1, baselineSenal: 2, pidfd: [3]int{10, 11, 12}, identidad: identidad,
		primera: uint32(observacionPidfdVacioO3cM38), palabraObservada: 2,
		canonControlRaw: controlRawVacioO4aM38, ahoraCaso: inicio, finCaso: fin,
		fisico: copiaSnapshotO4aM38(fisico),
	}
	a.causa.Store(uint32(causa))
	a.estado.Store(uint32(causaA3CausaFijadaM38))
	return a
}

func iniciarEtapasPruebaO4aP4M38(t *testing.T, causa causaPrimariaO4aM38) (*autoridadEtapasO4aM38, *autorizacionEtapaO4aM38) {
	t.Helper()
	a := autoridadSinteticaEtapasO4aP4M38(causa)
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

func permisoExactoEtapaPruebaO4aP4M38(p *autorizacionEtapaO4aM38, e *autoridadEtapasO4aM38, etapa etapaO4bM38, operacion operacionEtapaO4bM38, cardinal uint8, limite time.Time, clase claseLimiteEtapaO4bM38) bool {
	return p != nil && p.auto == p && p.autoridad == e && p.resultado != nil && p.resultado.auto == p.resultado &&
		p.resultado.autorizacion == p && p.generacion > 0 && p.resultado.generacion == p.generacion &&
		p.tid == e.causa.sellos.tid && p.etapa == etapa && p.operacion == operacion && p.cardinalidad == cardinal &&
		p.limite == limite && p.claseLimite == clase && p.rolPidfd == rolPidfdPrimarioO4bM38 &&
		p.estado.Load() == uint32(autorizacionEmitidaO4bM38)
}

func TestEtapasO4aP4InicioExactoYSalidaNatural(t *testing.T) {
	ahora := time.Now()
	parada, rapido, ok := plazosInicialesO4aM38(ahora)
	if !ok || parada.Sub(ahora) != duracionParadaInicialO4aM38 || rapido.Sub(parada) != duracionDrenajeO4aM38 {
		t.Fatal("plazos iniciales no exactos")
	}
	gracia, final, cooperativo, ok := plazosCooperativosO4aM38(ahora)
	if !ok || gracia.Sub(ahora) != duracionGraciaO4aM38 || final.Sub(gracia) != duracionParadaFinalO4aM38 ||
		cooperativo.Sub(final) != duracionDrenajeO4aM38 {
		t.Fatal("plazos cooperativos no exactos")
	}
	if _, ok := sumarPlazoEtapaO4aM38(time.Time{}, time.Second); ok {
		t.Fatal("marca cero admitida")
	}
	if _, ok := sumarPlazoEtapaO4aM38(ahora.Round(0), time.Second); ok {
		t.Fatal("marca civil admitida")
	}
	for _, causa := range []causaPrimariaO4aM38{causaCancelado65O4aM38, causaProtocolo65O4aM38, causaSenalInt130O4aM38, causaSenalTerm143O4aM38, causaPlazo65O4aM38, causaIncidente65O4aM38} {
		e, p := iniciarEtapasPruebaO4aP4M38(t, causa)
		if e.causa.causa.Load() != uint32(causa) || !e.extincion || e.terminalidad ||
			e.plazos.finParadaInicial.Sub(e.causa.sellos.ahoraCaso) <= 0 ||
			e.plazos.finDrenajeRapido.Sub(e.plazos.finParadaInicial) != duracionDrenajeO4aM38 ||
			!permisoExactoEtapaPruebaO4aP4M38(p, e, etapaParadaInicialO4bM38, operacionStopO4bM38, 1, e.plazos.finParadaInicial, limiteParadaInicialO4bM38) ||
			e.causa.estado.Load() != uint32(causaA4PermisoPreparadoM38) || e.causa.incidente.Load() != 0 {
			t.Fatalf("inicio extinción divergente causa=%d", causa)
		}
	}
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaSalidaO4aM38)
	if p != nil || !e.terminalidad || e.extincion || !tiempoMonotonoO3cM38(e.plazos.finDrenajeNatural) ||
		e.plazos.finDrenajeNatural.Sub(e.causa.sellos.ahoraCaso) <= 0 ||
		e.causa.estado.Load() != uint32(causaA7EntregaO4cPreparadaM38) || e.emitidas != 0 {
		t.Fatal("SALIDA no convergió sin señal")
	}
}

func TestEtapasO4aP4SecuenciaCooperativaCompleta(t *testing.T) {
	e, paradaInicial := iniciarEtapasPruebaO4aP4M38(t, causaSenalTerm143O4aM38)
	observadoInicial := paradaInicial.limite.Add(-time.Nanosecond)
	rInicial := sellarResultadoEtapaPruebaO4aP4M38(t, e, paradaInicial, 1, 0, 0, evidenciaEstableO4bM38, observadoInicial)
	ahoraCooperativo := observadoInicial.Add(time.Nanosecond)
	termCont := aplicarResultadoEnPruebaO4aP4M38(t, e, rInicial, ahoraCooperativo)
	if !permisoExactoEtapaPruebaO4aP4M38(termCont, e, etapaTerminarReanudarO4bM38, operacionTermContO4bM38, 2, e.plazos.finGracia, limiteGraciaO4bM38) ||
		e.plazos.finGracia.Sub(ahoraCooperativo) != duracionGraciaO4aM38 ||
		e.plazos.finParadaFinal.Sub(e.plazos.finGracia) != duracionParadaFinalO4aM38 ||
		e.plazos.finDrenajeCooperativo.Sub(e.plazos.finParadaFinal) != duracionDrenajeO4aM38 {
		t.Fatal("subplazos cooperativos divergentes")
	}
	observadoPresente := ahoraCooperativo.Add(time.Nanosecond)
	rTerm := sellarResultadoEtapaPruebaO4aP4M38(t, e, termCont, 2, 0, 0, evidenciaGrupoPresenteO4bM38, observadoPresente)
	if p := aplicarResultadoEnPruebaO4aP4M38(t, e, rTerm, observadoPresente); p != nil || e.causa.estado.Load() != uint32(causaA3CausaFijadaM38) {
		t.Fatal("presencia anterior a gracia no esperó")
	}
	paradaFinal := continuarEtapasEnO4aM38(e, e.plazos.finGracia)
	if !permisoExactoEtapaPruebaO4aP4M38(paradaFinal, e, etapaParadaFinalO4bM38, operacionStopO4bM38, 1, e.plazos.finParadaFinal, limiteParadaFinalO4bM38) {
		t.Fatal("parada final no emitida en gracia")
	}
	rFinal := sellarResultadoEtapaPruebaO4aP4M38(t, e, paradaFinal, 1, 0, 0, evidenciaEstableO4bM38, paradaFinal.limite.Add(-time.Nanosecond))
	kill := aplicarResultadoEnPruebaO4aP4M38(t, e, rFinal, paradaFinal.limite)
	if !permisoExactoEtapaPruebaO4aP4M38(kill, e, etapaMatarGrupoO4bM38, operacionKillO4bM38, 1, e.plazos.finDrenajeCooperativo, limiteDrenajeCooperativoO4bM38) {
		t.Fatal("KILL cooperativo no quedó opaco")
	}
	rKill := sellarResultadoEtapaPruebaO4aP4M38(t, e, kill, 1, 0, 0, evidenciaSinO4bM38, time.Time{})
	if siguiente := aplicarResultadoEnPruebaO4aP4M38(t, e, rKill, paradaFinal.limite.Add(time.Nanosecond)); siguiente != nil ||
		e.causa.estado.Load() != uint32(causaA7EntregaO4cPreparadaM38) || e.historialLen != 4 ||
		e.causa.causa.Load() != uint32(causaSenalTerm143O4aM38) || e.causa.incidente.Load() != 0 {
		t.Fatal("secuencia cooperativa no convergió")
	}
}

func TestEtapasO4aP4RamasDeFalloCerradas(t *testing.T) {
	t.Run("stop_error_a_kill_rapido", func(t *testing.T) {
		e, stop := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
		r := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, 1, 4, 0, evidenciaSinO4bM38, time.Time{})
		kill := aplicarResultadoEnPruebaO4aP4M38(t, e, r, stop.limite)
		if !permisoExactoEtapaPruebaO4aP4M38(kill, e, etapaMatarGrupoO4bM38, operacionKillO4bM38, 1, e.plazos.finDrenajeRapido, limiteDrenajeRapidoO4bM38) ||
			e.causa.incidente.Load() != incidenteCierreEnclavadoO4aM38 || !e.plazos.finGracia.IsZero() {
			t.Fatal("STOP fallido abrió gracia")
		}
		rKill := sellarResultadoEtapaPruebaO4aP4M38(t, e, kill, 1, 4, 0, evidenciaSinO4bM38, time.Time{})
		if aplicarResultadoEnPruebaO4aP4M38(t, e, rKill, stop.limite.Add(time.Nanosecond)) != nil ||
			e.causa.estado.Load() != uint32(causaA7EntregaO4cPreparadaM38) || e.causa.incidente.Load() != 1 {
			t.Fatal("KILL error no convergió con incidente único")
		}
	})
	t.Run("term_error_omite_cont", func(t *testing.T) {
		e, stop := iniciarEtapasPruebaO4aP4M38(t, causaPlazo65O4aM38)
		rStop := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, 1, 0, 0, evidenciaEstableO4bM38, stop.limite.Add(-time.Nanosecond))
		term := aplicarResultadoEnPruebaO4aP4M38(t, e, rStop, stop.limite)
		rTerm := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 1, 4, 0, evidenciaSinO4bM38, time.Time{})
		kill := aplicarResultadoEnPruebaO4aP4M38(t, e, rTerm, term.limite)
		if kill == nil || kill.claseLimite != limiteDrenajeCooperativoO4bM38 || e.causa.incidente.Load() != 1 || e.historialLen != 2 {
			t.Fatal("TERM fallido no convergió a KILL")
		}
	})
	t.Run("cont_error_conserva_dos_raw", func(t *testing.T) {
		e, stop := iniciarEtapasPruebaO4aP4M38(t, causaIncidente65O4aM38)
		rStop := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, 1, 0, 0, evidenciaEstableO4bM38, stop.limite.Add(-time.Nanosecond))
		term := aplicarResultadoEnPruebaO4aP4M38(t, e, rStop, stop.limite)
		rTerm := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 2, 0, 4, evidenciaSinO4bM38, time.Time{})
		kill := aplicarResultadoEnPruebaO4aP4M38(t, e, rTerm, term.limite)
		if kill == nil || rTerm.rawPrimero != 0 || rTerm.rawSegundo != 4 || e.causa.causa.Load() != uint32(causaIncidente65O4aM38) {
			t.Fatal("raw CONT no se conservó")
		}
	})
	t.Run("terminal_sin_otra_senal", func(t *testing.T) {
		e, stop := iniciarEtapasPruebaO4aP4M38(t, causaProtocolo65O4aM38)
		rStop := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, 1, 0, 0, evidenciaEstableO4bM38, stop.limite.Add(-time.Nanosecond))
		term := aplicarResultadoEnPruebaO4aP4M38(t, e, rStop, stop.limite)
		rTerm := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 2, 0, 0, evidenciaTerminalO4bM38, term.limite.Add(-time.Nanosecond))
		if p := aplicarResultadoEnPruebaO4aP4M38(t, e, rTerm, term.limite.Add(-time.Nanosecond)); p != nil || !e.terminalidad || e.emitidas != 2 || e.causa.estado.Load() != uint32(causaA7EntregaO4cPreparadaM38) {
			t.Fatal("terminalidad emitió señal adicional")
		}
	})
}

func TestEtapasO4aP4OneShotCarrerasYReplay(t *testing.T) {
	a := autoridadSinteticaEtapasO4aP4M38(causaCancelado65O4aM38)
	alias1, alias2 := a, a
	type inicio struct {
		e   *autoridadEtapasO4aM38
		p   *autorizacionEtapaO4aM38
		err error
	}
	resultados := make(chan inicio, 2)
	for _, entrada := range []**autoridadCausaO4aM38{&alias1, &alias2} {
		go func(x **autoridadCausaO4aM38) {
			e, p, err := iniciarEtapasO4aM38(x)
			resultados <- inicio{e, p, err}
		}(entrada)
	}
	i1, i2 := <-resultados, <-resultados
	ganador := i1
	if ganador.err != nil {
		ganador = i2
	}
	if (i1.err == nil) == (i2.err == nil) || ganador.e == nil || ganador.p == nil || alias1 != nil || alias2 != nil {
		t.Fatal("inicio concurrente no tuvo un ganador")
	}
	p := ganador.p
	ganadores := make(chan bool, 2)
	for range 2 {
		go func() {
			ok := p.estado.CompareAndSwap(uint32(autorizacionEmitidaO4bM38), uint32(autorizacionConsumiendoO4bM38))
			if ok {
				ok = confirmarConsumoAutorizacionEtapaO4aM38(ganador.e, p)
			}
			ganadores <- ok
		}()
	}
	if g1, g2 := <-ganadores, <-ganadores; g1 == g2 {
		t.Fatal("permiso concurrente no fue one-shot")
	}
	r := p.resultado
	r.cardinalidad, r.rawPrimero, r.evidencia = 1, 4, evidenciaSinO4bM38
	if !r.estado.CompareAndSwap(0, uint32(resultadoSelladoO4bM38)) || !p.estado.CompareAndSwap(uint32(autorizacionConsumiendoO4bM38), uint32(autorizacionConsumidaO4bM38)) {
		t.Fatal("sellado concurrente")
	}
	r1, r2 := r, r
	type salida struct {
		p   *autorizacionEtapaO4aM38
		err error
	}
	salidas := make(chan salida, 2)
	var wg sync.WaitGroup
	for _, entrada := range []**resultadoEtapaO4bM38{&r1, &r2} {
		wg.Add(1)
		go func(x **resultadoEtapaO4bM38) {
			defer wg.Done()
			q, err := aplicarResultadoEtapaO4aM38(ganador.e, x)
			salidas <- salida{q, err}
		}(entrada)
	}
	wg.Wait()
	s1, s2 := <-salidas, <-salidas
	if (s1.err == nil) == (s2.err == nil) || (s1.p == nil) == (s2.p == nil) || r1 != nil || r2 != nil || ganador.e.historialLen != 1 {
		t.Fatal("resultado concurrente no fue one-shot")
	}
	natural := autoridadSinteticaEtapasO4aP4M38(causaSalidaO4aM38)
	n1, n2 := natural, natural
	naturales := make(chan error, 2)
	for _, entrada := range []**autoridadCausaO4aM38{&n1, &n2} {
		go func(x **autoridadCausaO4aM38) {
			_, _, err := iniciarEtapasO4aM38(x)
			naturales <- err
		}(entrada)
	}
	if e1, e2 := <-naturales, <-naturales; (e1 == nil) == (e2 == nil) || n1 != nil || n2 != nil || natural.estado.Load() != uint32(causaA7EntregaO4cPreparadaM38) {
		t.Fatal("SALIDA concurrente no fue one-shot")
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
		for _, nombre := range []string{"owner", "causa", "permiso_adulterado", "resultado_forjado", "resultado_incompatible", "kill_en_borde", "parada_final_tardia"} {
			ejecutarFatalEtapasO4aP4M38(t, nombre)
		}
		return
	}
	a := autoridadSinteticaEtapasO4aP4M38(causaCancelado65O4aM38)
	if caso == "owner" {
		a.origen.autoridad.ownerLease.Store(uint32(propietarioLiberadoO3cM38))
		_, _, _ = iniciarEtapasO4aM38(&a)
		os.Exit(10)
	}
	if caso == "causa" {
		a.causa.Store(99)
		_, _, _ = iniciarEtapasO4aM38(&a)
		os.Exit(11)
	}
	e, stop, err := iniciarEtapasO4aM38(&a)
	if err != nil {
		os.Exit(12)
	}
	r := sellarResultadoEtapaPruebaO4aP4M38(t, e, stop, 1, 4, 0, evidenciaSinO4bM38, time.Time{})
	switch caso {
	case "permiso_adulterado":
		stop.operacion = operacionKillO4bM38
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, r, stop.limite)
	case "resultado_forjado":
		falso := &resultadoEtapaO4bM38{autorizacion: stop, generacion: r.generacion, etapa: r.etapa, limite: r.limite, claseLimite: r.claseLimite, cardinalidad: 1, rawPrimero: 4, evidencia: evidenciaSinO4bM38}
		falso.auto = falso
		falso.estado.Store(uint32(resultadoSelladoO4bM38))
		_, _ = aplicarResultadoEtapaO4aM38(e, &falso)
	case "resultado_incompatible":
		r.cardinalidad = 2
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, r, stop.limite)
	case "kill_en_borde":
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, r, e.plazos.finDrenajeRapido)
	case "parada_final_tardia":
		r.rawPrimero, r.evidencia, r.observado = 0, evidenciaEstableO4bM38, stop.limite.Add(-time.Nanosecond)
		term := aplicarResultadoEnPruebaO4aP4M38(t, e, r, stop.limite)
		rTerm := sellarResultadoEtapaPruebaO4aP4M38(t, e, term, 2, 0, 0, evidenciaGrupoPresenteO4bM38, term.limite.Add(-time.Nanosecond))
		_ = aplicarResultadoEnPruebaO4aP4M38(t, e, rTerm, e.plazos.finGracia.Add(-time.Nanosecond))
		_ = continuarEtapasEnO4aM38(e, e.plazos.finParadaFinal)
	}
	os.Exit(13)
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
	ast.Inspect(nodo, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name.IsExported() {
				t.Fatalf("API exportada %s", x.Name.Name)
			}
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
	for _, prohibida := range []string{"syscall.", "Signal(", "Kill(", "Wait(", "Wait4(", "Waitid(", "Close(", "Write(", "Print(", "Log(", "Sleep(", "NewTimer(", "time.After(", "context.", "pidfd_open", "F_DUPFD", "/proc/", "ownerLease.Store", "ownerObservador.Store"} {
		if strings.Contains(texto, prohibida) {
			t.Fatalf("efecto prohibido %s", prohibida)
		}
	}
	for _, nombre := range []string{"etapaParadaInicialO4bM38", "etapaTerminarReanudarO4bM38", "etapaParadaFinalO4bM38", "etapaMatarGrupoO4bM38", "rolPidfdPrimarioO4bM38"} {
		if !strings.Contains(texto, nombre) {
			t.Fatalf("tipo cerrado ausente %s", nombre)
		}
	}
	if strings.Count(texto, "time.Now()") != 3 || strings.Count(texto, "CompareAndSwap(uint32(autorizacionVaciaO4bM38), uint32(autorizacionEmitidaO4bM38))") != 1 ||
		strings.Count(texto, "CompareAndSwap(0, incidenteCierreEnclavadoO4aM38)") != 1 {
		t.Fatal("reloj, emisión o latch divergente")
	}
}
