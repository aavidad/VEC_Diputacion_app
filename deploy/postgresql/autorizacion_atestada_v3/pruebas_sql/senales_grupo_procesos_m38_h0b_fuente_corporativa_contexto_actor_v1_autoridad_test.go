//go:build ignore && linux && amd64

package main

import (
	"sync/atomic"
	"testing"
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
		resultadoEvidencia: p.resultado.evidencia,
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
		despues.resultadoEvidencia != antes.resultadoEvidencia || !p.resultado.observado.IsZero() {
		t.Fatal("O4B-P1 produjo un efecto o altero la autoridad prestada")
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
	forjada := &autorizacionEtapaO4aM38{}
	entrada := forjada
	tomada, err := tomarEntradaAutoridadO4bM38(&entrada)
	if err != nil || tomada != forjada || entrada != nil {
		t.Fatalf("transferencia no lineal: tomada=%p entrada=%p err=%v", tomada, entrada, err)
	}
	if clasificarEntradaAutoridadO4bM38(tomada) != entradaConsumidaAutoridadO4bM38 {
		t.Fatal("la forja separada no se clasifico como consumida")
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

func TestO4BP1AutoridadClonCopiaYForja(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaIncidente65O4aM38)
	antes := capturarHuellaCeroEfectosAutoridadO4bP1M38(e, p)
	entradas := map[string]*autorizacionEtapaO4aM38{
		"clon":    clonarAutorizacionAutoridadO4bP1M38(p, false),
		"copia":   clonarAutorizacionAutoridadO4bP1M38(p, true),
		"forjada": func() *autorizacionEtapaO4aM38 { f := &autorizacionEtapaO4aM38{}; f.auto = f; return f }(),
	}
	for nombre, entradaOriginal := range entradas {
		t.Run(nombre, func(t *testing.T) {
			entrada := entradaOriginal
			a, err := consumirAutoridadO4bM38(&entrada)
			if a != nil || entrada != nil || err != errUsoConsumidoAutoridadO4bM38 {
				t.Fatalf("aceptada: a=%p entrada=%p err=%v", a, entrada, err)
			}
		})
	}
	if p.estado.Load() != uint32(autorizacionEmitidaO4bM38) ||
		e.causa.estado.Load() != uint32(causaA4PermisoPreparadoM38) {
		t.Fatal("clon, copia o forja consumieron el slot real")
	}
	exigirCeroEfectosAutoridadO4bP1M38(t, e, p, antes)
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
		{"generacion_lease", func(e *autoridadEtapasO4aM38, _ *autorizacionEtapaO4aM38) { e.causa.sellos.lease.generacion++ }},
		{"tid", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.tid++ }},
		{"etapa", func(_ *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) { p.etapa = etapaMatarGrupoO4bM38 }},
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
			estadoPermiso, estadoCausa := p.estado.Load(), e.causa.estado.Load()
			if clasificarEntradaAutoridadO4bM38(p) != entradaFatalAutoridadO4bM38 {
				t.Fatal("la alteracion no se clasifico fatal")
			}
			if p.estado.Load() != estadoPermiso || e.causa.estado.Load() != estadoCausa ||
				p.resultado.estado.Load() != uint32(resultadoVacioO4bM38) {
				t.Fatal("la clasificacion adversarial produjo efectos")
			}
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
