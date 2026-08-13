//go:build ignore && linux && amd64

package main

import (
	"errors"
	"sync/atomic"
	"time"
)

const (
	duracionParadaInicialO4aM38    = time.Second
	duracionGraciaO4aM38           = 2 * time.Second
	duracionParadaFinalO4aM38      = time.Second
	duracionDrenajeO4aM38          = 5 * time.Second
	maximoEtapasO4aM38             = 4
	incidenteCierreEnclavadoO4aM38 = 1
)

var errEtapasConsumidasO4aM38 = errors.New("etapas O4a no disponibles")

type etapaO4bM38 uint8

const (
	etapaParadaInicialO4bM38 etapaO4bM38 = iota + 1
	etapaTerminarReanudarO4bM38
	etapaParadaFinalO4bM38
	etapaMatarGrupoO4bM38
)

type operacionEtapaO4bM38 uint8

const (
	operacionStopO4bM38 operacionEtapaO4bM38 = iota + 1
	operacionTermContO4bM38
	operacionKillO4bM38
)

type claseLimiteEtapaO4bM38 uint8

const (
	limiteParadaInicialO4bM38 claseLimiteEtapaO4bM38 = iota + 1
	limiteGraciaO4bM38
	limiteParadaFinalO4bM38
	limiteDrenajeRapidoO4bM38
	limiteDrenajeCooperativoO4bM38
)

type rolPidfdEtapaO4bM38 uint8

const rolPidfdPrimarioO4bM38 rolPidfdEtapaO4bM38 = 1

type estadoAutorizacionEtapaO4bM38 uint32

const (
	autorizacionVaciaO4bM38 estadoAutorizacionEtapaO4bM38 = iota
	autorizacionEmitidaO4bM38
	autorizacionConsumiendoO4bM38
	autorizacionConsumidaO4bM38
)

type claseEvidenciaEtapaO4bM38 uint8

const (
	evidenciaSinO4bM38 claseEvidenciaEtapaO4bM38 = iota + 1
	evidenciaEstableO4bM38
	evidenciaTerminalO4bM38
	evidenciaGrupoPresenteO4bM38
	evidenciaNoEstableO4bM38
)

type estadoResultadoEtapaO4bM38 uint32

const (
	resultadoVacioO4bM38 estadoResultadoEtapaO4bM38 = iota
	resultadoSelladoO4bM38
	resultadoConsumiendoO4bM38
	resultadoConsumidoO4bM38
)

// autorizacionEtapaO4aM38 no contiene el descriptor: O4b recibe el agregado
// privado por la misma frontera y solo puede cotejar este permiso cardinal.
type autorizacionEtapaO4aM38 struct {
	auto         *autorizacionEtapaO4aM38
	autoridad    *autoridadEtapasO4aM38
	resultado    *resultadoEtapaO4bM38
	generacion   uint64
	tid          int
	etapa        etapaO4bM38
	cardinalidad uint8
	operacion    operacionEtapaO4bM38
	limite       time.Time
	claseLimite  claseLimiteEtapaO4bM38
	rolPidfd     rolPidfdEtapaO4bM38
	estado       atomic.Uint32
}

// resultadoEtapaO4bM38 es el sobre preasignado que O4b sellara. Sus raws no
// se interpretan hasta que la autorización y el resultado sean consumidos.
type resultadoEtapaO4bM38 struct {
	auto         *resultadoEtapaO4bM38
	autorizacion *autorizacionEtapaO4aM38
	generacion   uint64
	etapa        etapaO4bM38
	limite       time.Time
	claseLimite  claseLimiteEtapaO4bM38
	cardinalidad uint8
	rawPrimero   int
	rawSegundo   int
	evidencia    claseEvidenciaEtapaO4bM38
	observado    time.Time
	estado       atomic.Uint32
}

type plazosEtapasO4aM38 struct {
	finParadaInicial, finDrenajeRapido               time.Time
	finGracia, finParadaFinal, finDrenajeCooperativo time.Time
	finDrenajeNatural                                time.Time
}

type registroEtapaO4aM38 struct {
	autorizacion *autorizacionEtapaO4aM38
	resultado    *resultadoEtapaO4bM38
}

type autoridadEtapasO4aM38 struct {
	auto           *autoridadEtapasO4aM38
	causa          *autoridadCausaO4aM38
	autorizaciones [maximoEtapasO4aM38]autorizacionEtapaO4aM38
	resultados     [maximoEtapasO4aM38]resultadoEtapaO4bM38
	historial      [maximoEtapasO4aM38]registroEtapaO4aM38
	plazos         plazosEtapasO4aM38
	pendiente      *autorizacionEtapaO4aM38
	emitidas       uint8
	historialLen   uint8
	extincion      bool
	terminalidad   bool
}

func fatalEtapasO4aM38(e *autoridadEtapasO4aM38) {
	if e != nil && e.causa != nil {
		e.causa.estado.Store(uint32(causaAFFatalM38))
	}
	fatalO3cM38()
}

func autoridadBaseEtapasExactaO4aM38(a *autoridadCausaO4aM38) bool {
	if a == nil || a.auto != a || a.origen == nil || a.origen.auto != a.origen ||
		a.origen.autoridad == nil || a.sellos.autoridad != a.origen.autoridad ||
		a.sellos.custodia == nil || a.sellos.custodia != a.origen.custodia ||
		a.sellos.lease == nil || a.sellos.observador == nil || a.sellos.registro == nil ||
		a.sellos.lease != a.sellos.custodia.lease || a.sellos.observador != a.sellos.custodia.observador ||
		a.sellos.control != a.sellos.custodia.control || a.sellos.controlFD != a.sellos.custodia.controlFD ||
		a.sellos.terminal != a.sellos.custodia.terminal || a.sellos.cmd != a.sellos.custodia.cmd ||
		a.sellos.control == nil || a.sellos.controlFD == nil || a.sellos.terminal == nil ||
		a.sellos.cmd == nil || a.sellos.cmd.Process == nil || a.sellos.proceso != a.sellos.cmd.Process ||
		a.sellos.autoridad.ownerLease.Load() != uint32(propietarioO4aM38) ||
		a.sellos.autoridad.ownerObservador.Load() != uint32(propietarioO4aM38) ||
		a.sellos.lease.auto != a.sellos.lease || a.sellos.observador.auto != a.sellos.observador ||
		a.sellos.registro.auto != a.sellos.registro || a.sellos.lease.registro != a.sellos.registro ||
		a.sellos.observador.registro != a.sellos.registro || a.sellos.lease.estado.Load() != 3 ||
		a.sellos.autoridadArranque != a.sellos.custodia.autoridad ||
		a.sellos.registro.leases[a.sellos.lease] != a.sellos.generacionLease ||
		a.sellos.registro.observadores[a.sellos.observador] != a.sellos.generacionObservador ||
		a.sellos.lease.generacion != a.sellos.generacionLease || a.sellos.observador.generacion != a.sellos.generacionObservador ||
		a.sellos.lease.tid != a.sellos.tid || a.sellos.registro.tid != a.sellos.tid ||
		a.sellos.pidfd != [3]int{a.sellos.custodia.pidfdPrimario, a.sellos.custodia.pidfdReserva, a.sellos.custodia.pidfdOpaco} ||
		a.sellos.identidad != a.origen.identidad || a.sellos.primera != a.origen.primera.Load() ||
		a.sellos.retornoCont != a.origen.retornoCont || a.sellos.baselineSenal != a.sellos.custodia.baselineSenal ||
		a.sellos.ahoraCaso != a.origen.ahoraCaso || a.sellos.finCaso != a.origen.finCaso ||
		!tiempoSelladoArbitrajeO4aM38(a) || !snapshotsIgualesO3aM38(a.sellos.fisico, a.sellos.lease.fisico) {
		return false
	}
	p := a.sellos.pidfd
	if p[0] < 0 || p[1] < 0 || p[2] < 0 || p[0] == p[1] || p[0] == p[2] || p[1] == p[2] {
		return false
	}
	h0, ok0 := a.sellos.fisico.mapa[p[0]]
	h1, ok1 := a.sellos.fisico.mapa[p[1]]
	h2, ok2 := a.sellos.fisico.mapa[p[2]]
	if !ok0 || !ok1 || !ok2 || !h0.abierto || !h1.abierto || !h2.abierto ||
		h0.identidad != h1.identidad || h1.identidad != h2.identidad {
		return false
	}
	c := causaPrimariaO4aM38(a.causa.Load())
	i := a.incidente.Load()
	return causaPrimariaValidaO4aM38(c) && (i == 0 || i == incidenteCierreEnclavadoO4aM38)
}

func nuevaAutoridadEtapasO4aM38(a *autoridadCausaO4aM38) *autoridadEtapasO4aM38 {
	e := &autoridadEtapasO4aM38{causa: a}
	e.auto = e
	for indice := range e.autorizaciones {
		p, r := &e.autorizaciones[indice], &e.resultados[indice]
		p.auto, p.autoridad, p.resultado, p.generacion = p, e, r, uint64(indice+1)
		r.auto, r.autorizacion, r.generacion = r, p, p.generacion
	}
	return e
}

func sumarPlazoEtapaO4aM38(inicio time.Time, duracion time.Duration) (time.Time, bool) {
	if !tiempoMonotonoO3cM38(inicio) || duracion <= 0 {
		return time.Time{}, false
	}
	fin := inicio.Add(duracion)
	return fin, tiempoMonotonoO3cM38(fin) && fin.After(inicio) && fin.Sub(inicio) == duracion
}

func plazosInicialesO4aM38(ahora time.Time) (time.Time, time.Time, bool) {
	parada, ok := sumarPlazoEtapaO4aM38(ahora, duracionParadaInicialO4aM38)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	drenaje, ok := sumarPlazoEtapaO4aM38(parada, duracionDrenajeO4aM38)
	return parada, drenaje, ok
}

func plazosCooperativosO4aM38(ahora time.Time) (time.Time, time.Time, time.Time, bool) {
	gracia, ok := sumarPlazoEtapaO4aM38(ahora, duracionGraciaO4aM38)
	if !ok {
		return time.Time{}, time.Time{}, time.Time{}, false
	}
	parada, ok := sumarPlazoEtapaO4aM38(gracia, duracionParadaFinalO4aM38)
	if !ok {
		return time.Time{}, time.Time{}, time.Time{}, false
	}
	drenaje, ok := sumarPlazoEtapaO4aM38(parada, duracionDrenajeO4aM38)
	return gracia, parada, drenaje, ok
}

func especificacionEtapaO4aM38(etapa etapaO4bM38, limite time.Time, clase claseLimiteEtapaO4bM38) (operacionEtapaO4bM38, uint8, bool) {
	if !tiempoMonotonoO3cM38(limite) {
		return 0, 0, false
	}
	switch etapa {
	case etapaParadaInicialO4bM38:
		return operacionStopO4bM38, 1, clase == limiteParadaInicialO4bM38
	case etapaTerminarReanudarO4bM38:
		return operacionTermContO4bM38, 2, clase == limiteGraciaO4bM38
	case etapaParadaFinalO4bM38:
		return operacionStopO4bM38, 1, clase == limiteParadaFinalO4bM38
	case etapaMatarGrupoO4bM38:
		return operacionKillO4bM38, 1, clase == limiteDrenajeRapidoO4bM38 || clase == limiteDrenajeCooperativoO4bM38
	}
	return 0, 0, false
}

func emitirEtapaO4aM38(e *autoridadEtapasO4aM38, desde estadoCausaO4aM38, etapa etapaO4bM38, limite time.Time, clase claseLimiteEtapaO4bM38) (*autorizacionEtapaO4aM38, bool) {
	operacion, cardinalidad, valida := especificacionEtapaO4aM38(etapa, limite, clase)
	if !valida || e == nil || e.auto != e || e.pendiente != nil || e.emitidas >= maximoEtapasO4aM38 {
		fatalEtapasO4aM38(e)
	}
	p, r := &e.autorizaciones[e.emitidas], &e.resultados[e.emitidas]
	if p.auto != p || p.autoridad != e || p.resultado != r || r.auto != r || r.autorizacion != p ||
		p.estado.Load() != uint32(autorizacionVaciaO4bM38) || r.estado.Load() != uint32(resultadoVacioO4bM38) {
		fatalEtapasO4aM38(e)
	}
	p.tid, p.etapa, p.cardinalidad, p.operacion = e.causa.sellos.tid, etapa, cardinalidad, operacion
	p.limite, p.claseLimite, p.rolPidfd = limite, clase, rolPidfdPrimarioO4bM38
	r.etapa, r.limite, r.claseLimite = etapa, limite, clase
	if !e.causa.estado.CompareAndSwap(uint32(desde), uint32(causaA4PermisoPreparadoM38)) {
		return nil, false
	}
	if !p.estado.CompareAndSwap(uint32(autorizacionVaciaO4bM38), uint32(autorizacionEmitidaO4bM38)) {
		fatalEtapasO4aM38(e)
	}
	e.pendiente = p
	e.emitidas++
	return p, true
}

func iniciarEtapasO4aM38(entrada **autoridadCausaO4aM38) (*autoridadEtapasO4aM38, *autorizacionEtapaO4aM38, error) {
	if entrada == nil || *entrada == nil {
		return nil, nil, errEtapasConsumidasO4aM38
	}
	a := *entrada
	*entrada = nil
	if !autoridadBaseEtapasExactaO4aM38(a) {
		fatalEtapasO4aM38(&autoridadEtapasO4aM38{causa: a})
	}
	if a.estado.Load() != uint32(causaA3CausaFijadaM38) {
		return nil, nil, errEtapasConsumidasO4aM38
	}
	if a.incidente.Load() != 0 {
		fatalEtapasO4aM38(&autoridadEtapasO4aM38{causa: a})
	}
	e := nuevaAutoridadEtapasO4aM38(a)
	ahora := time.Now()
	if !tiempoMonotonoO3cM38(ahora) || ahora.Before(a.sellos.ahoraCaso) {
		fatalEtapasO4aM38(e)
	}
	if causaPrimariaO4aM38(a.causa.Load()) == causaSalidaO4aM38 {
		fin, ok := sumarPlazoEtapaO4aM38(ahora, duracionDrenajeO4aM38)
		if !ok {
			fatalEtapasO4aM38(e)
		}
		if !a.estado.CompareAndSwap(uint32(causaA3CausaFijadaM38), uint32(causaA7EntregaO4cPreparadaM38)) {
			return nil, nil, errEtapasConsumidasO4aM38
		}
		e.plazos.finDrenajeNatural, e.terminalidad = fin, true
		return e, nil, nil
	}
	parada, drenaje, ok := plazosInicialesO4aM38(ahora)
	if !ok {
		fatalEtapasO4aM38(e)
	}
	e.plazos.finParadaInicial, e.plazos.finDrenajeRapido, e.extincion = parada, drenaje, true
	p, emitida := emitirEtapaO4aM38(e, causaA3CausaFijadaM38, etapaParadaInicialO4bM38, parada, limiteParadaInicialO4bM38)
	if !emitida {
		return nil, nil, errEtapasConsumidasO4aM38
	}
	return e, p, nil
}

func autorizacionExactaEtapaO4aM38(e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) bool {
	if e == nil || e.auto != e || e.causa == nil || p == nil || p.auto != p || p.autoridad != e ||
		p.resultado == nil || p.resultado.auto != p.resultado || p.resultado.autorizacion != p ||
		p.generacion == 0 || p.generacion > uint64(e.emitidas) || p.generacion > maximoEtapasO4aM38 ||
		p != &e.autorizaciones[p.generacion-1] || p.resultado != &e.resultados[p.generacion-1] ||
		p.resultado.generacion != p.generacion || p.resultado.etapa != p.etapa ||
		p.resultado.limite != p.limite || p.resultado.claseLimite != p.claseLimite ||
		p.tid != e.causa.sellos.tid || p.rolPidfd != rolPidfdPrimarioO4bM38 {
		return false
	}
	operacion, cardinalidad, valida := especificacionEtapaO4aM38(p.etapa, p.limite, p.claseLimite)
	if !valida || p.operacion != operacion || p.cardinalidad != cardinalidad {
		return false
	}
	switch p.claseLimite {
	case limiteParadaInicialO4bM38:
		return p.limite == e.plazos.finParadaInicial
	case limiteGraciaO4bM38:
		return p.limite == e.plazos.finGracia
	case limiteParadaFinalO4bM38:
		return p.limite == e.plazos.finParadaFinal
	case limiteDrenajeRapidoO4bM38:
		return p.limite == e.plazos.finDrenajeRapido
	case limiteDrenajeCooperativoO4bM38:
		return p.limite == e.plazos.finDrenajeCooperativo
	}
	return false
}

// confirmarConsumoAutorizacionEtapaO4aM38 pertenece a O4a: O4b la invoca
// tras ganar EMITIDO->CONSUMIENDO y antes de intentar el primer efecto.
func confirmarConsumoAutorizacionEtapaO4aM38(e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) bool {
	return autorizacionExactaEtapaO4aM38(e, p) && e.pendiente == p &&
		p.estado.Load() == uint32(autorizacionConsumiendoO4bM38) &&
		e.causa.estado.CompareAndSwap(uint32(causaA4PermisoPreparadoM38), uint32(causaA5EsperandoResultadoM38))
}

func resultadoExactoEtapaO4aM38(e *autoridadEtapasO4aM38, r *resultadoEtapaO4bM38) bool {
	p := e.pendiente
	return autorizacionExactaEtapaO4aM38(e, p) && r != nil && r.auto == r && r.autorizacion == p && p.resultado == r &&
		r.generacion == p.generacion && r.etapa == p.etapa && r.limite == p.limite &&
		r.claseLimite == p.claseLimite && r.rawPrimero >= 0 && r.rawSegundo >= 0 &&
		p.estado.Load() == uint32(autorizacionConsumidaO4bM38) &&
		e.causa.estado.Load() == uint32(causaA5EsperandoResultadoM38)
}

func marcaResultadoO4aM38(e *autoridadEtapasO4aM38, r *resultadoEtapaO4bM38, antesDelLimite bool) bool {
	if !tiempoMonotonoO3cM38(r.observado) || r.observado.Before(e.causa.sellos.ahoraCaso) {
		return false
	}
	return !antesDelLimite || r.observado.Before(r.limite)
}

func enclavarIncidenteCierreO4aM38(e *autoridadEtapasO4aM38) {
	i := e.causa.incidente.Load()
	if i == incidenteCierreEnclavadoO4aM38 {
		return
	}
	if i != 0 || !e.causa.incidente.CompareAndSwap(0, incidenteCierreEnclavadoO4aM38) {
		fatalEtapasO4aM38(e)
	}
}

func consumirResultadoO4aM38(e *autoridadEtapasO4aM38, r *resultadoEtapaO4bM38) {
	if !r.estado.CompareAndSwap(uint32(resultadoConsumiendoO4bM38), uint32(resultadoConsumidoO4bM38)) ||
		e.historialLen >= maximoEtapasO4aM38 {
		fatalEtapasO4aM38(e)
	}
	e.historial[e.historialLen] = registroEtapaO4aM38{autorizacion: e.pendiente, resultado: r}
	e.historialLen++
	e.pendiente = nil
}

func emitirKillO4aM38(e *autoridadEtapasO4aM38, desde estadoCausaO4aM38, ahora, limite time.Time, clase claseLimiteEtapaO4bM38) *autorizacionEtapaO4aM38 {
	if !tiempoMonotonoO3cM38(ahora) || ahora.Before(e.causa.sellos.ahoraCaso) || !ahora.Before(limite) {
		fatalEtapasO4aM38(e)
	}
	p, ok := emitirEtapaO4aM38(e, desde, etapaMatarGrupoO4bM38, limite, clase)
	if !ok {
		fatalEtapasO4aM38(e)
	}
	return p
}

func aplicarResultadoEnO4aM38(e *autoridadEtapasO4aM38, r *resultadoEtapaO4bM38, ahora time.Time) *autorizacionEtapaO4aM38 {
	if e == nil || e.auto != e || e.causa == nil {
		fatalEtapasO4aM38(e)
	}
	p := e.pendiente
	if !autoridadBaseEtapasExactaO4aM38(e.causa) || !resultadoExactoEtapaO4aM38(e, r) ||
		!tiempoMonotonoO3cM38(ahora) || ahora.Before(e.causa.sellos.ahoraCaso) {
		fatalEtapasO4aM38(e)
	}
	switch p.etapa {
	case etapaParadaInicialO4bM38:
		valido := r.cardinalidad == 1 && r.rawSegundo == 0
		estable := valido && r.rawPrimero == 0 && r.evidencia == evidenciaEstableO4bM38 && marcaResultadoO4aM38(e, r, true)
		noEstable := valido && r.rawPrimero == 0 && r.evidencia == evidenciaNoEstableO4bM38 && marcaResultadoO4aM38(e, r, false)
		fallo := valido && r.rawPrimero != 0 && r.evidencia == evidenciaSinO4bM38 && r.observado.IsZero()
		if !estable && !noEstable && !fallo {
			fatalEtapasO4aM38(e)
		}
		consumirResultadoO4aM38(e, r)
		if estable {
			gracia, parada, drenaje, ok := plazosCooperativosO4aM38(ahora)
			if !ok {
				fatalEtapasO4aM38(e)
			}
			e.plazos.finGracia, e.plazos.finParadaFinal, e.plazos.finDrenajeCooperativo = gracia, parada, drenaje
			q, ok := emitirEtapaO4aM38(e, causaA5EsperandoResultadoM38, etapaTerminarReanudarO4bM38, gracia, limiteGraciaO4bM38)
			if !ok {
				fatalEtapasO4aM38(e)
			}
			return q
		}
		enclavarIncidenteCierreO4aM38(e)
		return emitirKillO4aM38(e, causaA5EsperandoResultadoM38, ahora, e.plazos.finDrenajeRapido, limiteDrenajeRapidoO4bM38)
	case etapaTerminarReanudarO4bM38:
		falloTerm := r.cardinalidad == 1 && r.rawPrimero != 0 && r.rawSegundo == 0 && r.evidencia == evidenciaSinO4bM38 && r.observado.IsZero()
		falloCont := r.cardinalidad == 2 && r.rawPrimero == 0 && r.rawSegundo != 0 && r.evidencia == evidenciaSinO4bM38 && r.observado.IsZero()
		terminal := r.cardinalidad == 2 && r.rawPrimero == 0 && r.rawSegundo == 0 && r.evidencia == evidenciaTerminalO4bM38 && marcaResultadoO4aM38(e, r, true)
		presente := r.cardinalidad == 2 && r.rawPrimero == 0 && r.rawSegundo == 0 && r.evidencia == evidenciaGrupoPresenteO4bM38 && marcaResultadoO4aM38(e, r, true)
		if !falloTerm && !falloCont && !terminal && !presente {
			fatalEtapasO4aM38(e)
		}
		consumirResultadoO4aM38(e, r)
		if terminal {
			e.terminalidad = true
			if !e.causa.estado.CompareAndSwap(uint32(causaA5EsperandoResultadoM38), uint32(causaA7EntregaO4cPreparadaM38)) {
				fatalEtapasO4aM38(e)
			}
			return nil
		}
		if presente {
			if !e.causa.estado.CompareAndSwap(uint32(causaA5EsperandoResultadoM38), uint32(causaA3CausaFijadaM38)) {
				fatalEtapasO4aM38(e)
			}
			return continuarEtapasEnO4aM38(e, ahora)
		}
		enclavarIncidenteCierreO4aM38(e)
		return emitirKillO4aM38(e, causaA5EsperandoResultadoM38, ahora, e.plazos.finDrenajeCooperativo, limiteDrenajeCooperativoO4bM38)
	case etapaParadaFinalO4bM38:
		valido := r.cardinalidad == 1 && r.rawSegundo == 0
		estable := valido && r.rawPrimero == 0 && r.evidencia == evidenciaEstableO4bM38 && marcaResultadoO4aM38(e, r, true)
		noEstable := valido && r.rawPrimero == 0 && r.evidencia == evidenciaNoEstableO4bM38 && marcaResultadoO4aM38(e, r, false)
		fallo := valido && r.rawPrimero != 0 && r.evidencia == evidenciaSinO4bM38 && r.observado.IsZero()
		if !estable && !noEstable && !fallo {
			fatalEtapasO4aM38(e)
		}
		consumirResultadoO4aM38(e, r)
		if !estable {
			enclavarIncidenteCierreO4aM38(e)
		}
		return emitirKillO4aM38(e, causaA5EsperandoResultadoM38, ahora, e.plazos.finDrenajeCooperativo, limiteDrenajeCooperativoO4bM38)
	case etapaMatarGrupoO4bM38:
		if r.cardinalidad != 1 || r.rawSegundo != 0 || r.evidencia != evidenciaSinO4bM38 || !r.observado.IsZero() {
			fatalEtapasO4aM38(e)
		}
		consumirResultadoO4aM38(e, r)
		if r.rawPrimero != 0 {
			enclavarIncidenteCierreO4aM38(e)
		}
		if !e.causa.estado.CompareAndSwap(uint32(causaA5EsperandoResultadoM38), uint32(causaA7EntregaO4cPreparadaM38)) {
			fatalEtapasO4aM38(e)
		}
		return nil
	}
	fatalEtapasO4aM38(e)
	return nil
}

func aplicarResultadoEtapaO4aM38(e *autoridadEtapasO4aM38, entrada **resultadoEtapaO4bM38) (*autorizacionEtapaO4aM38, error) {
	if entrada == nil || *entrada == nil {
		return nil, errEtapasConsumidasO4aM38
	}
	r := *entrada
	*entrada = nil
	if r.estado.Load() == uint32(resultadoConsumiendoO4bM38) || r.estado.Load() == uint32(resultadoConsumidoO4bM38) {
		return nil, errEtapasConsumidasO4aM38
	}
	if !r.estado.CompareAndSwap(uint32(resultadoSelladoO4bM38), uint32(resultadoConsumiendoO4bM38)) {
		if estado := r.estado.Load(); estado == uint32(resultadoConsumiendoO4bM38) || estado == uint32(resultadoConsumidoO4bM38) {
			return nil, errEtapasConsumidasO4aM38
		}
		fatalEtapasO4aM38(e)
	}
	return aplicarResultadoEnO4aM38(e, r, time.Now()), nil
}

func continuarEtapasEnO4aM38(e *autoridadEtapasO4aM38, ahora time.Time) *autorizacionEtapaO4aM38 {
	if e == nil || e.auto != e || !autoridadBaseEtapasExactaO4aM38(e.causa) || e.pendiente != nil ||
		e.causa.estado.Load() != uint32(causaA3CausaFijadaM38) || e.plazos.finGracia.IsZero() ||
		!tiempoMonotonoO3cM38(ahora) || ahora.Before(e.causa.sellos.ahoraCaso) {
		fatalEtapasO4aM38(e)
	}
	if ahora.Before(e.plazos.finGracia) {
		return nil
	}
	if !ahora.Before(e.plazos.finParadaFinal) {
		fatalEtapasO4aM38(e)
	}
	p, ok := emitirEtapaO4aM38(e, causaA3CausaFijadaM38, etapaParadaFinalO4bM38, e.plazos.finParadaFinal, limiteParadaFinalO4bM38)
	if !ok {
		fatalEtapasO4aM38(e)
	}
	return p
}

func continuarEtapasO4aM38(e *autoridadEtapasO4aM38) *autorizacionEtapaO4aM38 {
	return continuarEtapasEnO4aM38(e, time.Now())
}
