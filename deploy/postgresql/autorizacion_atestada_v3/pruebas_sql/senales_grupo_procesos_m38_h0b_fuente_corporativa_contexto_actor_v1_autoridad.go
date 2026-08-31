//go:build ignore && linux && amd64

package main

import (
	"sync/atomic"
	"time"
)

type estadoAutoridadO4bM38 uint32

const (
	autoridadOB0RecibidaM38 estadoAutoridadO4bM38 = iota
	autoridadOB1ValidadaM38
	autoridadOB2ConsumiendoM38
	autoridadOB3PermisoPreparadoM38
	autoridadOB4SyscallIntentadoM38
	autoridadOB5ConsolidadoM38
	autoridadOB6EvidenciaM38
	autoridadOB7ResultadoSelladoM38
	autoridadOB8ConsumidoM38
	autoridadOBFFatalM38
)

type errorUsoAutoridadO4bM38 string

func (e errorUsoAutoridadO4bM38) Error() string { return string(e) }

const errUsoConsumidoAutoridadO4bM38 errorUsoAutoridadO4bM38 = "autorizacion O4b ya consumida"

type claseEntradaAutoridadO4bM38 uint8

const (
	entradaValidaAutoridadO4bM38 claseEntradaAutoridadO4bM38 = iota
	entradaConsumidaAutoridadO4bM38
	entradaFatalAutoridadO4bM38
)

// sellosAutoridadO4bM38 conserva la decisión y los recursos prestados por
// O4a. Es una autoridad privada: no expone getters ni ejecuta efectos.
type sellosAutoridadO4bM38 struct {
	etapas                 *autoridadEtapasO4aM38
	causa                  *autoridadCausaO4aM38
	autorizacion           *autorizacionEtapaO4aM38
	resultado              *resultadoEtapaO4bM38
	origen                 sellosO4aM38
	generacionAutorizacion uint64
	tid                    int
	etapa                  etapaO4bM38
	cardinalidadMaxima     uint8
	operacion              operacionEtapaO4bM38
	limite                 time.Time
	claseLimite            claseLimiteEtapaO4bM38
	rolPidfd               rolPidfdEtapaO4bM38
	causaPrimaria          causaPrimariaO4aM38
	incidente              uint32
	ownerLease             uint32
	ownerObservador        uint32
	estadoLease            uint32
	palabraObservador      uint64
}

type autoridadSenalesGrupoO4bM38 struct {
	auto   *autoridadSenalesGrupoO4bM38
	estado atomic.Uint32
	sellos sellosAutoridadO4bM38
}

func transicionAutoridadO4bM38(desde, hacia estadoAutoridadO4bM38) bool {
	if hacia == autoridadOBFFatalM38 {
		return desde >= autoridadOB0RecibidaM38 && desde <= autoridadOB7ResultadoSelladoM38
	}
	switch desde {
	case autoridadOB0RecibidaM38:
		return hacia == autoridadOB1ValidadaM38
	case autoridadOB1ValidadaM38:
		return hacia == autoridadOB2ConsumiendoM38
	case autoridadOB2ConsumiendoM38:
		return hacia == autoridadOB3PermisoPreparadoM38
	case autoridadOB3PermisoPreparadoM38:
		return hacia == autoridadOB4SyscallIntentadoM38
	case autoridadOB4SyscallIntentadoM38:
		return hacia == autoridadOB5ConsolidadoM38
	case autoridadOB5ConsolidadoM38:
		return hacia == autoridadOB3PermisoPreparadoM38 || hacia == autoridadOB6EvidenciaM38
	case autoridadOB6EvidenciaM38:
		return hacia == autoridadOB7ResultadoSelladoM38
	case autoridadOB7ResultadoSelladoM38:
		return hacia == autoridadOB8ConsumidoM38
	}
	return false
}

// tomarEntradaAutoridadO4bM38 separa la transferencia lineal de toda
// observación del permiso. El puntero del llamador queda anulado primero.
func tomarEntradaAutoridadO4bM38(entrada **autorizacionEtapaO4aM38) (*autorizacionEtapaO4aM38, error) {
	if entrada == nil || *entrada == nil {
		return nil, errUsoConsumidoAutoridadO4bM38
	}
	p := *entrada
	*entrada = nil
	return p, nil
}

func slotPropioAutoridadO4bM38(p *autorizacionEtapaO4aM38) (*autoridadEtapasO4aM38, bool) {
	if p == nil || p.autoridad == nil {
		return nil, false
	}
	e := p.autoridad
	for indice := range e.autorizaciones {
		if p == &e.autorizaciones[indice] {
			return e, true
		}
	}
	return nil, false
}

func resultadoVacioExactoAutoridadO4bM38(r *resultadoEtapaO4bM38) bool {
	return r != nil && r.auto == r && r.autorizacion != nil && r.autorizacion.resultado == r &&
		r.generacion == r.autorizacion.generacion && r.etapa == r.autorizacion.etapa &&
		r.limite == r.autorizacion.limite && r.claseLimite == r.autorizacion.claseLimite &&
		r.cardinalidad == 0 && r.rawPrimero == 0 && r.rawSegundo == 0 && r.evidencia == 0 &&
		r.observado.IsZero() && r.estado.Load() == uint32(resultadoVacioO4bM38)
}

func entradaExactaAutoridadO4bM38(e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) bool {
	return e != nil && e.auto == e && e.causa != nil && e.pendiente == p &&
		e.causa.estado.Load() == uint32(causaA4PermisoPreparadoM38) &&
		p.estado.Load() == uint32(autorizacionEmitidaO4bM38) &&
		autoridadBaseEtapasExactaO4aM38(e.causa) && autorizacionExactaEtapaO4aM38(e, p) &&
		resultadoVacioExactoAutoridadO4bM38(p.resultado)
}

func clasificarEntradaAutoridadO4bM38(p *autorizacionEtapaO4aM38) claseEntradaAutoridadO4bM38 {
	if p == nil || p.auto != p || p.autoridad == nil || p.autoridad.auto != p.autoridad {
		return entradaFatalAutoridadO4bM38
	}
	e, propia := slotPropioAutoridadO4bM38(p)
	if !propia {
		return entradaConsumidaAutoridadO4bM38
	}
	estado := estadoAutorizacionEtapaO4bM38(p.estado.Load())
	if estado == autorizacionConsumiendoO4bM38 || estado == autorizacionConsumidaO4bM38 {
		return entradaConsumidaAutoridadO4bM38
	}
	if estado != autorizacionEmitidaO4bM38 {
		return entradaFatalAutoridadO4bM38
	}
	if entradaExactaAutoridadO4bM38(e, p) {
		return entradaValidaAutoridadO4bM38
	}
	// Otro consumidor puede ganar mientras se comprueba la entrada. Esa
	// transición solo convierte esta copia en un replay consumido.
	estado = estadoAutorizacionEtapaO4bM38(p.estado.Load())
	if estado == autorizacionConsumiendoO4bM38 || estado == autorizacionConsumidaO4bM38 {
		return entradaConsumidaAutoridadO4bM38
	}
	return entradaFatalAutoridadO4bM38
}

func copiarSellosOrigenAutoridadO4bM38(origen sellosO4aM38) sellosO4aM38 {
	origen.fisico = copiaSnapshotO4aM38(origen.fisico)
	return origen
}

func sellosOrigenExactosAutoridadO4bM38(actual *sellosO4aM38, sellados sellosO4aM38) bool {
	return actual != nil && actual.autoridad == sellados.autoridad &&
		actual.autoridadArranque == sellados.autoridadArranque && actual.custodia == sellados.custodia &&
		actual.lease == sellados.lease && actual.observador == sellados.observador &&
		actual.registro == sellados.registro && actual.control == sellados.control &&
		actual.controlFD == sellados.controlFD && actual.terminal == sellados.terminal &&
		actual.cmd == sellados.cmd && actual.proceso == sellados.proceso &&
		actual.generacionLease == sellados.generacionLease &&
		actual.generacionObservador == sellados.generacionObservador && actual.tid == sellados.tid &&
		actual.ppid == sellados.ppid && actual.baselineSenal == sellados.baselineSenal &&
		actual.pidfd == sellados.pidfd && actual.identidad == sellados.identidad &&
		actual.primera == sellados.primera && actual.retornoCont == sellados.retornoCont &&
		actual.palabraObservada == sellados.palabraObservada &&
		actual.canonControlRaw == sellados.canonControlRaw && actual.ahoraCaso == sellados.ahoraCaso &&
		actual.finCaso == sellados.finCaso && actual.huellaControl == sellados.huellaControl &&
		actual.huellaTerminal == sellados.huellaTerminal &&
		snapshotsIgualesO3aM38(actual.fisico, sellados.fisico)
}

func sellarAutoridadO4bM38(e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) sellosAutoridadO4bM38 {
	a := e.causa
	return sellosAutoridadO4bM38{
		etapas: e, causa: a, autorizacion: p, resultado: p.resultado,
		origen:                 copiarSellosOrigenAutoridadO4bM38(a.sellos),
		generacionAutorizacion: p.generacion, tid: p.tid, etapa: p.etapa,
		cardinalidadMaxima: p.cardinalidadMaxima, operacion: p.operacion, limite: p.limite,
		claseLimite: p.claseLimite, rolPidfd: p.rolPidfd,
		causaPrimaria: causaPrimariaO4aM38(a.causa.Load()), incidente: a.incidente.Load(),
		ownerLease:        a.sellos.autoridad.ownerLease.Load(),
		ownerObservador:   a.sellos.autoridad.ownerObservador.Load(),
		estadoLease:       a.sellos.lease.estado.Load(),
		palabraObservador: a.sellos.observador.palabra.Load(),
	}
}

func autoridadExactaO4bM38(a *autoridadSenalesGrupoO4bM38) bool {
	if a == nil || a.auto != a || a.estado.Load() != uint32(autoridadOB1ValidadaM38) {
		return false
	}
	s := &a.sellos
	e, p := s.etapas, s.autorizacion
	if e == nil || e.auto != e || e.causa != s.causa || e.pendiente != p || p == nil || p.auto != p ||
		p.autoridad != e || p.resultado != s.resultado || p.estado.Load() != uint32(autorizacionConsumiendoO4bM38) ||
		e.causa.estado.Load() != uint32(causaA5EsperandoResultadoM38) ||
		!autoridadBaseEtapasExactaO4aM38(e.causa) || !autorizacionExactaEtapaO4aM38(e, p) ||
		!resultadoVacioExactoAutoridadO4bM38(p.resultado) ||
		!sellosOrigenExactosAutoridadO4bM38(&e.causa.sellos, s.origen) {
		return false
	}
	return p.generacion == s.generacionAutorizacion && p.tid == s.tid && p.etapa == s.etapa &&
		p.cardinalidadMaxima == s.cardinalidadMaxima && p.operacion == s.operacion &&
		p.limite == s.limite && p.claseLimite == s.claseLimite && p.rolPidfd == s.rolPidfd &&
		causaPrimariaO4aM38(e.causa.causa.Load()) == s.causaPrimaria && e.causa.incidente.Load() == s.incidente &&
		e.causa.sellos.autoridad.ownerLease.Load() == s.ownerLease && s.ownerLease == uint32(propietarioO4aM38) &&
		e.causa.sellos.autoridad.ownerObservador.Load() == s.ownerObservador && s.ownerObservador == uint32(propietarioO4aM38) &&
		e.causa.sellos.lease.estado.Load() == s.estadoLease && s.estadoLease == 3 &&
		e.causa.sellos.observador.palabra.Load() == s.palabraObservador &&
		s.palabraObservador&mascaraEstadoObservadorO3aM38 == 2
}

func fatalAutoridadO4bM38(a *autoridadSenalesGrupoO4bM38, e *autoridadEtapasO4aM38) {
	if a != nil && a.auto == a {
		desde := estadoAutoridadO4bM38(a.estado.Load())
		if desde == autoridadOB0RecibidaM38 || desde == autoridadOB1ValidadaM38 {
			a.estado.CompareAndSwap(uint32(desde), uint32(autoridadOBFFatalM38))
		}
	}
	if e != nil && e.causa != nil {
		e.causa.estado.Store(uint32(causaAFFatalM38))
	}
	fatalO3cM38()
}

func consumirAutoridadO4bM38(entrada **autorizacionEtapaO4aM38) (*autoridadSenalesGrupoO4bM38, error) {
	p, err := tomarEntradaAutoridadO4bM38(entrada)
	if err != nil {
		return nil, err
	}
	clase := clasificarEntradaAutoridadO4bM38(p)
	if clase == entradaConsumidaAutoridadO4bM38 {
		return nil, errUsoConsumidoAutoridadO4bM38
	}
	e, _ := slotPropioAutoridadO4bM38(p)
	a := &autoridadSenalesGrupoO4bM38{}
	a.auto = a
	if clase != entradaValidaAutoridadO4bM38 {
		fatalAutoridadO4bM38(a, e)
	}
	if !p.estado.CompareAndSwap(uint32(autorizacionEmitidaO4bM38), uint32(autorizacionConsumiendoO4bM38)) {
		return nil, errUsoConsumidoAutoridadO4bM38
	}
	if !confirmarConsumoAutorizacionEtapaO4aM38(e, p) {
		fatalAutoridadO4bM38(a, e)
	}
	a.sellos = sellarAutoridadO4bM38(e, p)
	if !a.estado.CompareAndSwap(uint32(autoridadOB0RecibidaM38), uint32(autoridadOB1ValidadaM38)) ||
		!autoridadExactaO4bM38(a) {
		fatalAutoridadO4bM38(a, e)
	}
	return a, nil
}
