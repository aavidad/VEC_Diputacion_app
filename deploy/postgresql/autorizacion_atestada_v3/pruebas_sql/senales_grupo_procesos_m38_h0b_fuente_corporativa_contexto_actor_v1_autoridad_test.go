//go:build ignore && linux && amd64

package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

type huellaCeroEfectosAutoridadO4bP1M38 struct {
	origen              sellosO4aM38
	causa               uint32
	incidente           uint32
	ownerLease          uint32
	ownerObservador     uint32
	estadoLease         uint32
	secuenciaLease      uint64
	operacionLease      operacionGuardiaO3aM38
	cardinalLease       int
	objetivosLease      [2]int
	palabraObservador   uint64
	consumidaCustodia   uint32
	resultadoEstado     uint32
	resultadoCardinal   uint8
	resultadoRawPrimero int
	resultadoRawSegundo int
	resultadoEvidencia  claseEvidenciaEtapaO4bM38
	resultadoObservado  time.Time
	estadoCausa         uint32
	estadoAutorizacion  uint32
	pendiente           *autorizacionEtapaO4aM38
	emitidas            uint8
	historialLen        uint8
}

func capturarHuellaCeroEfectosAutoridadO4bP1M38(e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) huellaCeroEfectosAutoridadO4bP1M38 {
	s := e.causa.sellos
	return huellaCeroEfectosAutoridadO4bP1M38{
		origen: copiarSellosOrigenAutoridadO4bM38(s),
		causa:  e.causa.causa.Load(), incidente: e.causa.incidente.Load(),
		ownerLease: s.autoridad.ownerLease.Load(), ownerObservador: s.autoridad.ownerObservador.Load(),
		estadoLease: s.lease.estado.Load(), secuenciaLease: s.lease.secuencia,
		operacionLease: s.lease.operacion, cardinalLease: s.lease.cardinal, objetivosLease: s.lease.objetivos,
		palabraObservador: s.observador.palabra.Load(), consumidaCustodia: s.custodia.consumida.Load(),
		resultadoEstado: p.resultado.estado.Load(), resultadoCardinal: p.resultado.cardinalidad,
		resultadoRawPrimero: p.resultado.rawPrimero, resultadoRawSegundo: p.resultado.rawSegundo,
		resultadoEvidencia: p.resultado.evidencia, resultadoObservado: p.resultado.observado,
		estadoCausa: e.causa.estado.Load(), estadoAutorizacion: p.estado.Load(),
		pendiente: e.pendiente, emitidas: e.emitidas, historialLen: e.historialLen,
	}
}

func exigirCeroEfectosAutoridadO4bP1M38(t *testing.T, e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38, antes huellaCeroEfectosAutoridadO4bP1M38) {
	t.Helper()
	s := e.causa.sellos
	despues := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
	if !sellosOrigenExactosAutoridadO4bM38(&s, antes.origen) || despues.causa != antes.causa ||
		despues.incidente != antes.incidente || despues.ownerLease != antes.ownerLease ||
		despues.ownerObservador != antes.ownerObservador || despues.estadoLease != antes.estadoLease ||
		despues.secuenciaLease != antes.secuenciaLease || despues.operacionLease != antes.operacionLease ||
		despues.cardinalLease != antes.cardinalLease || despues.objetivosLease != antes.objetivosLease ||
		despues.palabraObservador != antes.palabraObservador ||
		despues.consumidaCustodia != antes.consumidaCustodia ||
		despues.resultadoEstado != antes.resultadoEstado ||
		despues.resultadoCardinal != antes.resultadoCardinal ||
		despues.resultadoRawPrimero != antes.resultadoRawPrimero ||
		despues.resultadoRawSegundo != antes.resultadoRawSegundo ||
		despues.resultadoEvidencia != antes.resultadoEvidencia ||
		despues.resultadoObservado != antes.resultadoObservado {
		t.Fatal("O4B-P1 produjo un efecto o altero la autoridad prestada")
	}
}

func exigirClasificacionSinEfectosAutoridadO4bP1M38(t *testing.T, e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38, antes huellaCeroEfectosAutoridadO4bP1M38) {
	t.Helper()
	exigirCeroEfectosAutoridadO4bP1M38(t, e, p, antes)
	despues := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
	if despues.estadoCausa != antes.estadoCausa || despues.estadoAutorizacion != antes.estadoAutorizacion ||
		despues.pendiente != antes.pendiente || despues.emitidas != antes.emitidas ||
		despues.historialLen != antes.historialLen {
		t.Fatal("la clasificacion altero la maquina O4a")
	}
}

func clonarAutorizacionAutoridadO4bP1M38(p *autorizacionEtapaO4aM38, autoidentica bool) *autorizacionEtapaO4aM38 {
	c := &autorizacionEtapaO4aM38{
		auto: p.auto, autoridad: p.autoridad, resultado: p.resultado, generacion: p.generacion,
		tid: p.tid, etapa: p.etapa, cardinalidadMaxima: p.cardinalidadMaxima,
		operacion: p.operacion, limite: p.limite, claseLimite: p.claseLimite, rolPidfd: p.rolPidfd,
	}
	if autoidentica {
		c.auto = c
	}
	c.estado.Store(p.estado.Load())
	return c
}

func prepararEntradaAdversaAutoridadO4bP1M38(caso string, e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) (*autorizacionEtapaO4aM38, bool) {
	switch caso {
	case "clon_autoidentidad_rota":
		return clonarAutorizacionAutoridadO4bP1M38(p, false), true
	case "forja_sin_autoridad":
		forja := &autorizacionEtapaO4aM38{}
		forja.auto = forja
		return forja, true
	case "forja_autoidentica_autoridad_sin_slot":
		forja := clonarAutorizacionAutoridadO4bP1M38(p, true)
		forja.generacion = uint64(len(e.autorizaciones)) + 1
		return forja, true
	case "copia_consumida_alterada":
		p.estado.Store(uint32(autorizacionConsumidaO4bM38))
		copia := clonarAutorizacionAutoridadO4bP1M38(p, true)
		copia.tid++
		return copia, true
	case "estructura_ambigua":
		if len(e.autorizaciones) < 2 {
			return nil, false
		}
		otro := &e.autorizaciones[1]
		otro.generacion = p.generacion
		otro.resultado.generacion = p.generacion
		return p, true
	default:
		return nil, false
	}
}

func TestO4BP1AutoridadExitoOB0OB1(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
	antes := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
	entrada := p
	a, err := consumirAutoridadO4bM38(&entrada)
	if err != nil || entrada != nil || a == nil || !autoridadExactaO4bM38(a) ||
		a.estado.Load() != uint32(autoridadOB1ValidadaM38) ||
		p.estado.Load() != uint32(autorizacionConsumiendoO4bM38) ||
		e.causa.estado.Load() != uint32(causaA5EsperandoResultadoM38) {
		t.Fatalf("OB0->OB1 invalido: a=%p entrada=%p err=%v", a, entrada, err)
	}
	exigirCeroEfectosAutoridadO4bP1M38(t, e, p, antes)
}

func TestO4BP1AutoridadPunteroAnuladoAntesDeObservacion(t *testing.T) {
	_, permiso := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
	forja := &autorizacionEtapaO4aM38{}
	forja.auto = forja
	casos := []struct {
		nombre   string
		entrada  *autorizacionEtapaO4aM38
		esperada claseEntradaAutoridadO4bM38
	}{
		{"copia", clonarAutorizacionAutoridadO4bP1M38(permiso, true), entradaConsumidaAutoridadO4bM38},
		{"clon", clonarAutorizacionAutoridadO4bP1M38(permiso, false), entradaFatalAutoridadO4bM38},
		{"forja", forja, entradaFatalAutoridadO4bM38},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entrada := caso.entrada
			tomada, err := tomarEntradaAutoridadO4bM38(&entrada)
			if err != nil || tomada != caso.entrada || entrada != nil {
				t.Fatalf("transferencia no lineal: tomada=%p entrada=%p err=%v", tomada, entrada, err)
			}
			if clasificarEntradaAutoridadO4bM38(tomada) != caso.esperada {
				t.Fatalf("clasificacion inesperada para %s", caso.nombre)
			}
		})
	}
}

func TestO4BP1AutoridadNil(t *testing.T) {
	if a, err := consumirAutoridadO4bM38(nil); a != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("nil doble: a=%p err=%v", a, err)
	}
	var entrada *autorizacionEtapaO4aM38
	if a, err := consumirAutoridadO4bM38(&entrada); a != nil || entrada != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("nil opaco: a=%p entrada=%p err=%v", a, entrada, err)
	}
}

func TestO4BP1AutoridadAliasYReplay(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaPlazo65O4aM38)
	antes := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
	entrada, alias := p, p
	a, err := consumirAutoridadO4bM38(&entrada)
	if err != nil || a == nil || entrada != nil {
		t.Fatalf("ganador: a=%p entrada=%p err=%v", a, entrada, err)
	}
	repetida, err := consumirAutoridadO4bM38(&alias)
	if repetida != nil || alias != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("replay: a=%p alias=%p err=%v", repetida, alias, err)
	}
	exigirCeroEfectosAutoridadO4bP1M38(t, e, p, antes)
}

func TestO4BP1AutoridadReplayRealSlotConsumido(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
	sellarResultadoEtapaPruebaO4aP4M38(t, e, p, 1, 0, 0, evidenciaEstableO4bM38, p.limite.Add(-time.Nanosecond))
	antes := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
	if clasificarEntradaAutoridadO4bM38(p) != entradaConsumidaAutoridadO4bM38 {
		t.Fatal("el slot consumido estructuralmente valido no se clasifico como replay")
	}
	exigirClasificacionSinEfectosAutoridadO4bP1M38(t, e, p, antes)
	entrada := p
	a, err := consumirAutoridadO4bM38(&entrada)
	if a != nil || entrada != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("replay consumido: a=%p entrada=%p err=%v", a, entrada, err)
	}
	exigirClasificacionSinEfectosAutoridadO4bP1M38(t, e, p, antes)
}

func TestO4BP1AutoridadCopiaSinSlotPierdeSinEfectos(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaIncidente65O4aM38)
	antes := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
	entrada := clonarAutorizacionAutoridadO4bP1M38(p, true)
	if clasificarEntradaAutoridadO4bM38(entrada) != entradaConsumidaAutoridadO4bM38 {
		t.Fatal("la copia ordinaria exacta no se clasifico consumida")
	}
	exigirClasificacionSinEfectosAutoridadO4bP1M38(t, e, p, antes)
	a, err := consumirAutoridadO4bM38(&entrada)
	if a != nil || entrada != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("copia aceptada: a=%p entrada=%p err=%v", a, entrada, err)
	}
	if p.estado.Load() != uint32(autorizacionEmitidaO4bM38) ||
		e.causa.estado.Load() != uint32(causaA4PermisoPreparadoM38) {
		t.Fatal("la copia sin slot consumio la autoridad real")
	}
	exigirClasificacionSinEfectosAutoridadO4bP1M38(t, e, p, antes)
}

var casosFatalesAutoridadO4bP1M38 = [...]string{
	"clon_autoidentidad_rota",
	"forja_sin_autoridad",
	"forja_autoidentica_autoridad_sin_slot",
	"copia_consumida_alterada",
	"estructura_ambigua",
}

func TestO4BP1AutoridadMalformadasClasificanFatalSinEfectos(t *testing.T) {
	for _, nombre := range casosFatalesAutoridadO4bP1M38 {
		t.Run(nombre, func(t *testing.T) {
			e, p := iniciarEtapasPruebaO4aP4M38(t, causaIncidente65O4aM38)
			entrada, preparada := prepararEntradaAdversaAutoridadO4bP1M38(nombre, e, p)
			if !preparada {
				t.Fatal("caso adverso no preparado")
			}
			antes := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
			if clasificarEntradaAutoridadO4bM38(entrada) != entradaFatalAutoridadO4bM38 {
				t.Fatal("la entrada malformada no se clasifico fatal")
			}
			exigirClasificacionSinEfectosAutoridadO4bP1M38(t, e, p, antes)
		})
	}
}

const marcadorPreparacionFatalAutoridadO4bP1M38 = "O4B-P1-R2:fixture-validada\n"

func ejecutarEntradaFatalAutoridadO4bP1M38(t *testing.T, caso string) {
	t.Helper()
	lector, escritor, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lector.Close() })
	cmd := exec.Command(os.Args[0], "-test.run=^TestO4BP1AutoridadEntradasMalformadasFatales$")
	cmd.Env = append(os.Environ(), "O4B_P1_FATAL="+caso)
	cmd.ExtraFiles = []*os.File{escritor}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Start(); err != nil {
		_ = escritor.Close()
		t.Fatal(err)
	}
	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	preparacion, errLectura := io.ReadAll(lector)
	var salida *exec.ExitError
	if errLectura != nil || !bytes.Equal(preparacion, []byte(marcadorPreparacionFatalAutoridadO4bP1M38)) ||
		!errors.As(err, &salida) || salida.ExitCode() != estadoFallo || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("fatal %s: err=%v preparacion=%q err_lectura=%v stdout=%d stderr=%d", caso, err, preparacion, errLectura, stdout.Len(), stderr.Len())
	}
}

func TestO4BP1AutoridadEntradasMalformadasFatales(t *testing.T) {
	caso := os.Getenv("O4B_P1_FATAL")
	if caso == "" {
		for _, nombre := range casosFatalesAutoridadO4bP1M38 {
			t.Run(nombre, func(t *testing.T) {
				ejecutarEntradaFatalAutoridadO4bP1M38(t, nombre)
			})
		}
		return
	}
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaIncidente65O4aM38)
	if clasificarEntradaAutoridadO4bM38(p) != entradaValidaAutoridadO4bM38 {
		os.Exit(10)
	}
	entrada, preparada := prepararEntradaAdversaAutoridadO4bP1M38(caso, e, p)
	if !preparada || clasificarEntradaAutoridadO4bM38(entrada) != entradaFatalAutoridadO4bM38 {
		os.Exit(11)
	}
	preparacion := os.NewFile(3, "o4b-p1-r2-preparacion")
	if preparacion == nil {
		os.Exit(12)
	}
	escritos, err := io.WriteString(preparacion, marcadorPreparacionFatalAutoridadO4bP1M38)
	if err != nil || escritos != len(marcadorPreparacionFatalAutoridadO4bP1M38) || preparacion.Close() != nil {
		os.Exit(13)
	}
	_, _ = consumirAutoridadO4bM38(&entrada)
	os.Exit(14)
}

func TestO4BP1AutoridadCarreraUnGanador(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
	antes := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
	var ganadores atomic.Int32
	var consumidas atomic.Int32
	t.Cleanup(func() {
		if ganadores.Load() != 1 || consumidas.Load() != 1 ||
			p.estado.Load() != uint32(autorizacionConsumiendoO4bM38) ||
			e.causa.estado.Load() != uint32(causaA5EsperandoResultadoM38) {
			t.Errorf("carrera: ganadores=%d consumidas=%d permiso=%d causa=%d", ganadores.Load(), consumidas.Load(), p.estado.Load(), e.causa.estado.Load())
		}
		exigirCeroEfectosAutoridadO4bP1M38(t, e, p, antes)
	})
	for _, nombre := range []string{"primera", "segunda"} {
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			entrada := p
			a, err := consumirAutoridadO4bM38(&entrada)
			if entrada != nil {
				t.Fatal("el puntero del contendiente no fue anulado")
			}
			switch {
			case err == nil && a != nil && autoridadExactaO4bM38(a):
				ganadores.Add(1)
			case err == errUsoConsumidoAutoridadO4bM38 && a == nil:
				consumidas.Add(1)
			default:
				t.Fatalf("resultado inesperado: a=%p err=%v", a, err)
			}
		})
	}
}

func TestO4BP1AutoridadAlteracionesSonFatalesSinEfecto(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*autoridadEtapasO4aM38, *autorizacionEtapaO4aM38)
	}{
		{"autoidentidad_permiso", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.auto = nil }},
		{"autoidentidad_autoridad", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.auto = nil }},
		{"identidad", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.causa.origen.identidad.inicio++ }},
		{"generacion_permiso", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.generacion++ }},
		{"resultado_autoidentidad", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.resultado.auto = nil }},
		{"resultado_ligadura", func(e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) {
			p.resultado.autorizacion = &e.autorizaciones[1]
		}},
		{"generacion_lease", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.causa.sellos.lease.generacion++ }},
		{"tid", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.tid++ }},
		{"etapa", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.etapa = etapaMatarGrupoO4bM38 }},
		{"operacion", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.operacion++ }},
		{"cardinalidad", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.cardinalidadMaxima++ }},
		{"limite", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.limite = p.limite.Add(time.Second) }},
		{"clase_limite", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.claseLimite++ }},
		{"rol_pidfd", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.rolPidfd++ }},
		{"emitidas", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.emitidas++ }},
		{"recurso", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.causa.sellos.custodia.pidfdPrimario++ }},
		{"owner_lease", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) {
			e.causa.sellos.autoridad.ownerLease.Store(uint32(propietarioO3cM38))
		}},
		{"owner_observador", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) {
			e.causa.sellos.autoridad.ownerObservador.Store(uint32(propietarioO3cM38))
		}},
		{"lease", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.causa.sellos.lease.estado.Store(2) }},
		{"observador", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.causa.sellos.observador.palabra.Store(1) }},
		{"slot_pendiente", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.pendiente = nil }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e, p := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
			caso.mutar(e, p)
			antes := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
			if clasificarEntradaAutoridadO4bM38(p) != entradaFatalAutoridadO4bM38 {
				t.Fatal("la alteracion no se clasifico fatal")
			}
			exigirClasificacionSinEfectosAutoridadO4bP1M38(t, e, p, antes)
		})
	}
}

func TestO4BP1AutoridadMaquinaTotal(t *testing.T) {
	validas := map[[2]estadoAutoridadO4bM38]bool{
		{autoridadOB0RecibidaM38, autoridadOB1ValidadaM38}:                 true,
		{autoridadOB1ValidadaM38, autoridadOB2ConsumiendoM38}:              true,
		{autoridadOB2ConsumiendoM38, autoridadOB3PermisoPreparadoM38}:      true,
		{autoridadOB3PermisoPreparadoM38, autoridadOB4SyscallIntentadoM38}: true,
		{autoridadOB4SyscallIntentadoM38, autoridadOB5ConsolidadoM38}:      true,
		{autoridadOB5ConsolidadoM38, autoridadOB3PermisoPreparadoM38}:      true,
		{autoridadOB5ConsolidadoM38, autoridadOB6EvidenciaM38}:             true,
		{autoridadOB6EvidenciaM38, autoridadOB7ResultadoSelladoM38}:        true,
		{autoridadOB7ResultadoSelladoM38, autoridadOB8ConsumidoM38}:        true,
	}
	for desde := autoridadOB0RecibidaM38; desde <= autoridadOBFFatalM38; desde++ {
		for hacia := autoridadOB0RecibidaM38; hacia <= autoridadOBFFatalM38; hacia++ {
			esperada := validas[[2]estadoAutoridadO4bM38{desde, hacia}] ||
				hacia == autoridadOBFFatalM38 && desde <= autoridadOB7ResultadoSelladoM38
			if transicionAutoridadO4bM38(desde, hacia) != esperada {
				t.Fatalf("transicion %d->%d", desde, hacia)
			}
		}
	}
}
