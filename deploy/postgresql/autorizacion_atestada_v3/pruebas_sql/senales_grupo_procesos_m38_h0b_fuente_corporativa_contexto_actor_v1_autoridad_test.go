//go:build ignore && linux && amd64

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

// El canon materializa strings y escalares nuevos: nunca retiene punteros,
// mapas, slices o sellos mutables como oraculo posterior.
type canonProfundoAutoridadO4bP1M38 map[string]string

func referenciaSlotAutoridadO4bP1M38(e *autoridadEtapasO4aM38, p *autorizacionEtapaO4aM38) int {
	if p == nil {
		return -1
	}
	for i := range e.autorizaciones {
		if p == &e.autorizaciones[i] {
			return i
		}
	}
	return -2
}

func referenciaResultadoAutoridadO4bP1M38(e *autoridadEtapasO4aM38, r *resultadoEtapaO4bM38) int {
	if r == nil {
		return -1
	}
	for i := range e.resultados {
		if r == &e.resultados[i] {
			return i
		}
	}
	return -2
}

func canonSnapshotAutoridadO4bP1M38(s snapshotFDO3aM38) string {
	return fmt.Sprintf("%d|%#v", s.limite, s.mapa)
}

func canonFDAutoridadO4bP1M38(f *os.File) string {
	if f == nil {
		return "nil"
	}
	return fmt.Sprintf("%p|%d|%q", f, f.Fd(), f.Name())
}

func capturarCanonProfundoAutoridadO4bP1M38(e *autoridadEtapasO4aM38, entrada *autorizacionEtapaO4aM38) canonProfundoAutoridadO4bP1M38 {
	c := make(canonProfundoAutoridadO4bP1M38)
	a, s, u := e.causa, e.causa.sellos, e.causa.sellos.custodia
	c["etapas.auto"] = fmt.Sprintf("%p|%t", e.auto, e.auto == e)
	c["etapas.contenido"] = fmt.Sprintf("%t|%t|%#v", e.extincion, e.terminalidad, e.plazos)
	c["causa.estado"] = fmt.Sprint(a.estado.Load())
	c["causa.contenido"] = fmt.Sprintf("%p|%t|%p|%d|%d", a.auto, a.auto == a, a.origen, a.causa.Load(), a.incidente.Load())
	c["pendiente"] = fmt.Sprint(referenciaSlotAutoridadO4bP1M38(e, e.pendiente))
	c["emitidas"] = fmt.Sprint(e.emitidas)
	c["historial_len"] = fmt.Sprint(e.historialLen)
	c["owners"] = fmt.Sprintf("%d|%d", s.autoridad.ownerLease.Load(), s.autoridad.ownerObservador.Load())
	c["origen"] = fmt.Sprintf("%p|%t|%p|%p|%#v|%d|%#v|%#v|%d", a.origen.auto,
		a.origen.auto == a.origen, a.origen.autoridad, a.origen.custodia, a.origen.identidad,
		a.origen.primera.Load(), a.origen.ahoraCaso, a.origen.finCaso, a.origen.retornoCont)
	c["sellos"] = fmt.Sprintf("%p|%p|%p|%p|%p|%p|%p|%p|%p|%p|%p|%d|%d|%d|%d|%d|%#v|%d|%d|%#v|%d|%#v|%#v|%#v|%#v",
		s.autoridad, s.autoridadArranque, s.custodia, s.lease, s.observador, s.registro, s.control,
		s.controlFD, s.terminal, s.cmd, s.proceso, s.generacionLease, s.generacionObservador,
		s.tid, s.ppid, s.baselineSenal, s.identidad, s.primera, s.retornoCont, s.palabraObservada,
		s.canonControlRaw, s.ahoraCaso, s.finCaso, s.huellaControl, s.huellaTerminal)
	c["custodia"] = fmt.Sprintf("%p|%p|%p|%p|%p|%d|%p|%d|%#v|%d|%d|%#v|%#v|%d|%v|%v|%#v",
		u.autoridad, u.control, u.controlFD, u.terminal, u.observador, u.baselineSenal, u.reloj,
		u.vueltaInicio, u.finBootstrap, u.tid, u.ppid, u.formaRaiz, u.formaRunner,
		u.consumida.Load(), u.primera, u.secundarios, u.primeraCausa)
	c["recursos"] = fmt.Sprintf("%p|%p|%d|%d|%d|%d|%#v|%#v", u.cmd, u.cmd.Process,
		u.cmd.Process.Pid, u.pidfdPrimario, u.pidfdReserva, u.pidfdOpaco, u.destinados, u.huellasDestinadas)
	c["fds"] = fmt.Sprintf("%s|%s|%s|%s", canonFDAutoridadO4bP1M38(s.controlFD),
		canonFDAutoridadO4bP1M38(s.terminal), canonFDAutoridadO4bP1M38(u.ticketEscritor), canonFDAutoridadO4bP1M38(u.ticketLector))
	c["control"] = fmt.Sprintf("%p|%#v", s.control, *s.control)
	c["lease"] = fmt.Sprintf("%p|%t|%p|%d|%d|%d|%d|%d|%d|%#v", s.lease.auto,
		s.lease.auto == s.lease, s.lease.registro, s.lease.generacion, s.lease.tid, s.lease.estado.Load(),
		s.lease.secuencia, s.lease.operacion, s.lease.cardinal, s.lease.objetivos)
	c["observador"] = fmt.Sprintf("%p|%t|%p|%d|%d", s.observador.auto, s.observador.auto == s.observador,
		s.observador.registro, s.observador.generacion, s.observador.palabra.Load())
	c["registro"] = fmt.Sprintf("%p|%t|%d|%d|%#v|%#v|%#v", s.registro.auto, s.registro.auto == s.registro,
		s.registro.tid, s.registro.generacion, s.registro.preflight, s.registro.leases, s.registro.observadores)
	c["snapshots"] = fmt.Sprintf("%s|%s|%s|%s|%s", canonSnapshotAutoridadO4bP1M38(s.fisico),
		canonSnapshotAutoridadO4bP1M38(s.lease.fisico), canonSnapshotAutoridadO4bP1M38(s.lease.pre),
		canonSnapshotAutoridadO4bP1M38(u.snapshot), canonSnapshotAutoridadO4bP1M38(u.baseline))
	for i := range e.autorizaciones {
		p, r := &e.autorizaciones[i], &e.resultados[i]
		prefijoP, prefijoR := fmt.Sprintf("slot.%d.", i), fmt.Sprintf("resultado.%d.", i)
		c[prefijoP+"auto"] = fmt.Sprintf("%p|%t", p.auto, p.auto == p)
		c[prefijoP+"autoridad"] = fmt.Sprintf("%p|%t", p.autoridad, p.autoridad == e)
		c[prefijoP+"resultado"] = fmt.Sprint(referenciaResultadoAutoridadO4bP1M38(e, p.resultado))
		c[prefijoP+"generacion"] = fmt.Sprint(p.generacion)
		c[prefijoP+"tid"] = fmt.Sprint(p.tid)
		c[prefijoP+"etapa"] = fmt.Sprint(p.etapa)
		c[prefijoP+"cardinalidad"] = fmt.Sprint(p.cardinalidadMaxima)
		c[prefijoP+"operacion"] = fmt.Sprint(p.operacion)
		c[prefijoP+"limite"] = fmt.Sprintf("%#v", p.limite)
		c[prefijoP+"clase_limite"] = fmt.Sprint(p.claseLimite)
		c[prefijoP+"rol_pidfd"] = fmt.Sprint(p.rolPidfd)
		c[prefijoP+"estado"] = fmt.Sprint(p.estado.Load())
		c[prefijoR+"auto"] = fmt.Sprintf("%p|%t", r.auto, r.auto == r)
		c[prefijoR+"autorizacion"] = fmt.Sprint(referenciaSlotAutoridadO4bP1M38(e, r.autorizacion))
		c[prefijoR+"generacion"] = fmt.Sprint(r.generacion)
		c[prefijoR+"etapa"] = fmt.Sprint(r.etapa)
		c[prefijoR+"limite"] = fmt.Sprintf("%#v", r.limite)
		c[prefijoR+"clase_limite"] = fmt.Sprint(r.claseLimite)
		c[prefijoR+"cardinalidad"] = fmt.Sprint(r.cardinalidad)
		c[prefijoR+"raw_primero"] = fmt.Sprint(r.rawPrimero)
		c[prefijoR+"raw_segundo"] = fmt.Sprint(r.rawSegundo)
		c[prefijoR+"evidencia"] = fmt.Sprint(r.evidencia)
		c[prefijoR+"observado"] = fmt.Sprintf("%#v", r.observado)
		c[prefijoR+"estado"] = fmt.Sprint(r.estado.Load())
		c[fmt.Sprintf("historial.%d", i)] = fmt.Sprintf("%d|%d", referenciaSlotAutoridadO4bP1M38(e, e.historial[i].autorizacion), referenciaResultadoAutoridadO4bP1M38(e, e.historial[i].resultado))
	}
	c["entrada.relacion"] = fmt.Sprint(referenciaSlotAutoridadO4bP1M38(e, entrada))
	if entrada != nil && referenciaSlotAutoridadO4bP1M38(e, entrada) == -2 {
		c["entrada.auto"] = fmt.Sprintf("%p|%t", entrada.auto, entrada.auto == entrada)
		c["entrada.autoridad"] = fmt.Sprintf("%p|%t", entrada.autoridad, entrada.autoridad == e)
		c["entrada.resultado"] = fmt.Sprint(referenciaResultadoAutoridadO4bP1M38(e, entrada.resultado))
		c["entrada.generacion"] = fmt.Sprint(entrada.generacion)
		c["entrada.contenido"] = fmt.Sprintf("%d|%d|%d|%d|%#v|%d|%d|%d", entrada.tid, entrada.etapa,
			entrada.cardinalidadMaxima, entrada.operacion, entrada.limite, entrada.claseLimite, entrada.rolPidfd, entrada.estado.Load())
	}
	return c
}

func diferenciasCanonAutoridadO4bP1M38(antes, despues canonProfundoAutoridadO4bP1M38) []string {
	claves := make(map[string]struct{}, len(antes)+len(despues))
	for k := range antes {
		claves[k] = struct{}{}
	}
	for k := range despues {
		claves[k] = struct{}{}
	}
	var diferencias []string
	for k := range claves {
		if antes[k] != despues[k] {
			diferencias = append(diferencias, k)
		}
	}
	sort.Strings(diferencias)
	return diferencias
}

func cambiosCanonExactosAutoridadO4bP1M38(antes, despues canonProfundoAutoridadO4bP1M38, esperados ...string) bool {
	sort.Strings(esperados)
	reales := diferenciasCanonAutoridadO4bP1M38(antes, despues)
	return fmt.Sprint(reales) == fmt.Sprint(esperados)
}

func exigirCambiosCanonAutoridadO4bP1M38(t *testing.T, antes, despues canonProfundoAutoridadO4bP1M38, esperados ...string) {
	t.Helper()
	if !cambiosCanonExactosAutoridadO4bP1M38(antes, despues, esperados...) {
		t.Fatalf("cambios profundos: obtenidos=%v esperados=%v", diferenciasCanonAutoridadO4bP1M38(antes, despues), esperados)
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

type modoFixtureFatalAutoridadO4bP1M38 uint8

const (
	fixtureEmitidaAutoridadO4bP1M38 modoFixtureFatalAutoridadO4bP1M38 = iota
	fixtureConsumiendoAutoridadO4bP1M38
	fixtureConsumidaAutoridadO4bP1M38
	fixtureHistoricaAutoridadO4bP1M38
	fixtureCopiaEmitidaAutoridadO4bP1M38
)

type casoFatalAutoridadO4bP1M38 struct {
	nombre, dimension string
	modo              modoFixtureFatalAutoridadO4bP1M38
}

type fixtureFatalAutoridadO4bP1M38 struct {
	e       *autoridadEtapasO4aM38
	p       *autorizacionEtapaO4aM38
	entrada *autorizacionEtapaO4aM38
}

func prepararFixtureFatalAutoridadO4bP1M38(t *testing.T, modo modoFixtureFatalAutoridadO4bP1M38) fixtureFatalAutoridadO4bP1M38 {
	t.Helper()
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaIncidente65O4aM38)
	if modo == fixtureConsumiendoAutoridadO4bP1M38 {
		if !p.estado.CompareAndSwap(uint32(autorizacionEmitidaO4bM38), uint32(autorizacionConsumiendoO4bM38)) || !confirmarConsumoAutorizacionEtapaO4aM38(e, p) {
			t.Fatal("fixture CONSUMIENDO no preparada")
		}
	}
	if modo == fixtureConsumidaAutoridadO4bP1M38 || modo == fixtureHistoricaAutoridadO4bP1M38 {
		sellarResultadoEtapaPruebaO4aP4M38(t, e, p, 1, 0, 0, evidenciaEstableO4bM38, p.limite.Add(-time.Nanosecond))
	}
	if modo == fixtureHistoricaAutoridadO4bP1M38 {
		p.resultado.estado.Store(uint32(resultadoConsumidoO4bM38))
		e.historial[0] = registroEtapaO4aM38{autorizacion: p, resultado: p.resultado}
		e.historialLen, e.pendiente = 1, nil
		e.causa.estado.Store(uint32(causaA7EntregaO4cPreparadaM38))
	}
	entrada := p
	if modo == fixtureCopiaEmitidaAutoridadO4bP1M38 {
		entrada = clonarAutorizacionAutoridadO4bP1M38(p, true)
	}
	return fixtureFatalAutoridadO4bP1M38{e: e, p: p, entrada: entrada}
}

func aplicarMutacionFatalAutoridadO4bP1M38(caso string, f fixtureFatalAutoridadO4bP1M38) bool {
	e, p, r := f.e, f.p, f.p.resultado
	switch caso {
	case "clon_autoidentidad_rota":
		f.entrada.auto = nil
	case "forja_sin_autoridad":
		f.entrada.autoridad = nil
	case "forja_generacion_fuera":
		f.entrada.generacion = uint64(len(e.autorizaciones)) + 1
	case "autoidentidad_permiso":
		p.auto = nil
	case "autoidentidad_autoridad":
		e.auto = nil
	case "identidad":
		e.causa.origen.identidad.inicio++
	case "generacion_permiso":
		p.generacion++
	case "resultado_autoidentidad":
		r.auto = nil
	case "generacion_lease":
		e.causa.sellos.lease.generacion++
	case "tid":
		p.tid++
	case "etapa":
		p.etapa = etapaMatarGrupoO4bM38
	case "operacion":
		p.operacion++
	case "limite":
		p.limite = p.limite.Add(time.Second)
	case "clase_limite":
		p.claseLimite++
	case "rol_pidfd":
		p.rolPidfd++
	case "emitidas":
		e.emitidas++
	case "recurso":
		e.causa.sellos.custodia.pidfdPrimario++
	case "owner_lease":
		e.causa.sellos.autoridad.ownerLease.Store(uint32(propietarioO3cM38))
	case "owner_observador":
		e.causa.sellos.autoridad.ownerObservador.Store(uint32(propietarioO3cM38))
	case "lease":
		e.causa.sellos.lease.estado.Store(2)
	case "observador":
		e.causa.sellos.observador.palabra.Store(1)
	case "estructura_ambigua":
		e.autorizaciones[1].generacion = p.generacion
	case "consumiendo_ligadura", "consumido_ligadura":
		r.autorizacion = &e.autorizaciones[1]
	case "consumiendo_cardinalidad", "consumido_cardinalidad":
		r.cardinalidad++
	case "consumiendo_raw_primero", "consumido_raw_primero":
		r.rawPrimero++
	case "consumiendo_raw_segundo", "consumido_raw_segundo":
		r.rawSegundo++
	case "emitido_causa_a5":
		e.causa.estado.Store(uint32(causaA5EsperandoResultadoM38))
	case "emitido_causa_a7", "consumiendo_causa_a7", "consumido_causa_a7":
		e.causa.estado.Store(uint32(causaA7EntregaO4cPreparadaM38))
	case "consumido_causa_a4", "historico_causa_a4":
		e.causa.estado.Store(uint32(causaA4PermisoPreparadoM38))
	case "pendiente_nil":
		e.pendiente = nil
	case "pendiente_ajeno":
		e.pendiente = &e.autorizaciones[1]
	case "pendiente_historico":
		e.pendiente = p
	case "historial_incompleto":
		e.historial[0].resultado = nil
	case "historial_duplicado":
		e.historial[1] = e.historial[0]
	case "slot_historico_no_consumido":
		p.estado.Store(uint32(autorizacionConsumiendoO4bM38))
	case "estado_permiso_fuera":
		p.estado.Store(97)
	case "estado_resultado_fuera":
		r.estado.Store(98)
	case "estado_causa_fuera":
		e.causa.estado.Store(99)
	default:
		return false
	}
	return true
}

func TestO4BP1AutoridadExitoOB0OB1(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
	antes := capturarCanonProfundoAutoridadO4bP1M38(e, p)
	entrada := p
	a, err := consumirAutoridadO4bM38(&entrada)
	if err != nil || entrada != nil || a == nil || !autoridadExactaO4bM38(a) ||
		a.estado.Load() != uint32(autoridadOB1ValidadaM38) ||
		p.estado.Load() != uint32(autorizacionConsumiendoO4bM38) ||
		e.causa.estado.Load() != uint32(causaA5EsperandoResultadoM38) {
		t.Fatalf("OB0->OB1 invalido: a=%p entrada=%p err=%v", a, entrada, err)
	}
	exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p), "causa.estado", "slot.0.estado")
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
	antes := capturarCanonProfundoAutoridadO4bP1M38(e, p)
	entrada, alias := p, p
	a, err := consumirAutoridadO4bM38(&entrada)
	if err != nil || a == nil || entrada != nil {
		t.Fatalf("ganador: a=%p entrada=%p err=%v", a, entrada, err)
	}
	repetida, err := consumirAutoridadO4bM38(&alias)
	if repetida != nil || alias != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("replay: a=%p alias=%p err=%v", repetida, alias, err)
	}
	exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p), "causa.estado", "slot.0.estado")
}

func TestO4BP1AutoridadReplayRealSlotConsumido(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
	sellarResultadoEtapaPruebaO4aP4M38(t, e, p, 1, 0, 0, evidenciaEstableO4bM38, p.limite.Add(-time.Nanosecond))
	antes := capturarCanonProfundoAutoridadO4bP1M38(e, p)
	if clasificarEntradaAutoridadO4bM38(p) != entradaConsumidaAutoridadO4bM38 {
		t.Fatal("el slot consumido estructuralmente valido no se clasifico como replay")
	}
	exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p))
	entrada := p
	a, err := consumirAutoridadO4bM38(&entrada)
	if a != nil || entrada != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("replay consumido: a=%p entrada=%p err=%v", a, entrada, err)
	}
	exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p))
	copia := clonarAutorizacionAutoridadO4bP1M38(p, true)
	if clasificarEntradaAutoridadO4bM38(copia) != entradaConsumidaAutoridadO4bM38 {
		t.Fatal("la copia exacta del slot consumido valido no se clasifico como replay")
	}
	if a, err := consumirAutoridadO4bM38(&copia); a != nil || copia != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("copia de replay consumido: a=%p copia=%p err=%v", a, copia, err)
	}
	exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p))
}

func TestO4BP1AutoridadReplayHistoricoBienFormado(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
	sellarResultadoEtapaPruebaO4aP4M38(t, e, p, 1, 0, 0, evidenciaEstableO4bM38, p.limite.Add(-time.Nanosecond))
	p.resultado.estado.Store(uint32(resultadoConsumidoO4bM38))
	e.historial[0] = registroEtapaO4aM38{autorizacion: p, resultado: p.resultado}
	e.historialLen, e.pendiente = 1, nil
	e.causa.estado.Store(uint32(causaA7EntregaO4cPreparadaM38))
	antes := capturarCanonProfundoAutoridadO4bP1M38(e, p)
	for _, entrada := range []*autorizacionEtapaO4aM38{p, clonarAutorizacionAutoridadO4bP1M38(p, true)} {
		if clasificarEntradaAutoridadO4bM38(entrada) != entradaConsumidaAutoridadO4bM38 {
			t.Fatal("el replay historico bien formado no se clasifico consumido")
		}
		exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p))
	}
}

func TestO4BP1AutoridadCopiaSinSlotPierdeSinEfectos(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaIncidente65O4aM38)
	antes := capturarCanonProfundoAutoridadO4bP1M38(e, p)
	entrada := clonarAutorizacionAutoridadO4bP1M38(p, true)
	if clasificarEntradaAutoridadO4bM38(entrada) != entradaConsumidaAutoridadO4bM38 {
		t.Fatal("la copia ordinaria exacta no se clasifico consumida")
	}
	exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p))
	a, err := consumirAutoridadO4bM38(&entrada)
	if a != nil || entrada != nil || err != errUsoConsumidoAutoridadO4bM38 {
		t.Fatalf("copia aceptada: a=%p entrada=%p err=%v", a, entrada, err)
	}
	if p.estado.Load() != uint32(autorizacionEmitidaO4bM38) ||
		e.causa.estado.Load() != uint32(causaA4PermisoPreparadoM38) {
		t.Fatal("la copia sin slot consumio la autoridad real")
	}
	exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p))
}

func TestO4BP1AutoridadCrucesCanonicosA4A5A7(t *testing.T) {
	tipo := []struct {
		nombre   string
		modo     modoFixtureFatalAutoridadO4bP1M38
		causa    estadoCausaO4aM38
		casSolo  bool
		esperada claseEntradaAutoridadO4bM38
	}{
		{"emitido_a4", fixtureEmitidaAutoridadO4bP1M38, causaA4PermisoPreparadoM38, false, entradaValidaAutoridadO4bM38},
		{"consumiendo_a4", fixtureEmitidaAutoridadO4bP1M38, causaA4PermisoPreparadoM38, true, entradaConsumidaAutoridadO4bM38},
		{"consumiendo_a5", fixtureConsumiendoAutoridadO4bP1M38, causaA5EsperandoResultadoM38, false, entradaConsumidaAutoridadO4bM38},
		{"consumido_a5", fixtureConsumidaAutoridadO4bP1M38, causaA5EsperandoResultadoM38, false, entradaConsumidaAutoridadO4bM38},
		{"historico_a5", fixtureHistoricaAutoridadO4bP1M38, causaA5EsperandoResultadoM38, false, entradaConsumidaAutoridadO4bM38},
		{"historico_a7", fixtureHistoricaAutoridadO4bP1M38, causaA7EntregaO4cPreparadaM38, false, entradaConsumidaAutoridadO4bM38},
	}
	for _, caso := range tipo {
		t.Run(caso.nombre, func(t *testing.T) {
			f := prepararFixtureFatalAutoridadO4bP1M38(t, caso.modo)
			f.e.causa.estado.Store(uint32(caso.causa))
			if caso.casSolo && !f.p.estado.CompareAndSwap(uint32(autorizacionEmitidaO4bM38), uint32(autorizacionConsumiendoO4bM38)) {
				t.Fatal("CAS legitimo no preparado")
			}
			antes := capturarCanonProfundoAutoridadO4bP1M38(f.e, f.entrada)
			if clasificarEntradaAutoridadO4bM38(f.entrada) != caso.esperada {
				t.Fatal("cruce canonico rechazado")
			}
			exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(f.e, f.entrada))
		})
	}
}

var casosFatalesAutoridadO4bP1M38 = [...]casoFatalAutoridadO4bP1M38{
	{"clon_autoidentidad_rota", "entrada.auto", fixtureCopiaEmitidaAutoridadO4bP1M38},
	{"forja_sin_autoridad", "entrada.autoridad", fixtureCopiaEmitidaAutoridadO4bP1M38},
	{"forja_generacion_fuera", "entrada.generacion", fixtureCopiaEmitidaAutoridadO4bP1M38},
	{"autoidentidad_permiso", "slot.0.auto", fixtureEmitidaAutoridadO4bP1M38},
	{"autoidentidad_autoridad", "etapas.auto", fixtureEmitidaAutoridadO4bP1M38},
	{"identidad", "origen", fixtureEmitidaAutoridadO4bP1M38},
	{"generacion_permiso", "slot.0.generacion", fixtureEmitidaAutoridadO4bP1M38},
	{"resultado_autoidentidad", "resultado.0.auto", fixtureEmitidaAutoridadO4bP1M38},
	{"generacion_lease", "lease", fixtureEmitidaAutoridadO4bP1M38},
	{"tid", "slot.0.tid", fixtureEmitidaAutoridadO4bP1M38},
	{"etapa", "slot.0.etapa", fixtureEmitidaAutoridadO4bP1M38},
	{"operacion", "slot.0.operacion", fixtureEmitidaAutoridadO4bP1M38},
	{"limite", "slot.0.limite", fixtureEmitidaAutoridadO4bP1M38},
	{"clase_limite", "slot.0.clase_limite", fixtureEmitidaAutoridadO4bP1M38},
	{"rol_pidfd", "slot.0.rol_pidfd", fixtureEmitidaAutoridadO4bP1M38},
	{"emitidas", "emitidas", fixtureEmitidaAutoridadO4bP1M38},
	{"recurso", "recursos", fixtureEmitidaAutoridadO4bP1M38},
	{"owner_lease", "owners", fixtureEmitidaAutoridadO4bP1M38},
	{"owner_observador", "owners", fixtureEmitidaAutoridadO4bP1M38},
	{"lease", "lease", fixtureEmitidaAutoridadO4bP1M38},
	{"observador", "observador", fixtureEmitidaAutoridadO4bP1M38},
	{"estructura_ambigua", "slot.1.generacion", fixtureEmitidaAutoridadO4bP1M38},
	{"consumiendo_ligadura", "resultado.0.autorizacion", fixtureConsumiendoAutoridadO4bP1M38},
	{"consumiendo_cardinalidad", "resultado.0.cardinalidad", fixtureConsumiendoAutoridadO4bP1M38},
	{"consumiendo_raw_primero", "resultado.0.raw_primero", fixtureConsumiendoAutoridadO4bP1M38},
	{"consumiendo_raw_segundo", "resultado.0.raw_segundo", fixtureConsumiendoAutoridadO4bP1M38},
	{"consumido_ligadura", "resultado.0.autorizacion", fixtureConsumidaAutoridadO4bP1M38},
	{"consumido_cardinalidad", "resultado.0.cardinalidad", fixtureConsumidaAutoridadO4bP1M38},
	{"consumido_raw_primero", "resultado.0.raw_primero", fixtureConsumidaAutoridadO4bP1M38},
	{"consumido_raw_segundo", "resultado.0.raw_segundo", fixtureConsumidaAutoridadO4bP1M38},
	{"emitido_causa_a5", "causa.estado", fixtureEmitidaAutoridadO4bP1M38},
	{"emitido_causa_a7", "causa.estado", fixtureEmitidaAutoridadO4bP1M38},
	{"consumiendo_causa_a7", "causa.estado", fixtureConsumiendoAutoridadO4bP1M38},
	{"consumido_causa_a4", "causa.estado", fixtureConsumidaAutoridadO4bP1M38},
	{"consumido_causa_a7", "causa.estado", fixtureConsumidaAutoridadO4bP1M38},
	{"historico_causa_a4", "causa.estado", fixtureHistoricaAutoridadO4bP1M38},
	{"pendiente_nil", "pendiente", fixtureEmitidaAutoridadO4bP1M38},
	{"pendiente_ajeno", "pendiente", fixtureEmitidaAutoridadO4bP1M38},
	{"pendiente_historico", "pendiente", fixtureHistoricaAutoridadO4bP1M38},
	{"historial_incompleto", "historial.0", fixtureHistoricaAutoridadO4bP1M38},
	{"historial_duplicado", "historial.1", fixtureHistoricaAutoridadO4bP1M38},
	{"slot_historico_no_consumido", "slot.0.estado", fixtureHistoricaAutoridadO4bP1M38},
	{"estado_permiso_fuera", "slot.0.estado", fixtureEmitidaAutoridadO4bP1M38},
	{"estado_resultado_fuera", "resultado.0.estado", fixtureEmitidaAutoridadO4bP1M38},
	{"estado_causa_fuera", "causa.estado", fixtureEmitidaAutoridadO4bP1M38},
}

const marcadorPreparacionFatalAutoridadO4bP1M38 = "O4B-P1-R4:positiva+mutacion-unica\n"

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
		for _, candidato := range casosFatalesAutoridadO4bP1M38 {
			t.Run(candidato.nombre, func(t *testing.T) {
				ejecutarEntradaFatalAutoridadO4bP1M38(t, candidato.nombre)
			})
		}
		return
	}
	var elegido *casoFatalAutoridadO4bP1M38
	for i := range casosFatalesAutoridadO4bP1M38 {
		if casosFatalesAutoridadO4bP1M38[i].nombre == caso {
			elegido = &casosFatalesAutoridadO4bP1M38[i]
			break
		}
	}
	if elegido == nil {
		os.Exit(20)
	}
	f := prepararFixtureFatalAutoridadO4bP1M38(t, elegido.modo)
	esperada := entradaConsumidaAutoridadO4bM38
	if elegido.modo == fixtureEmitidaAutoridadO4bP1M38 {
		esperada = entradaValidaAutoridadO4bM38
	}
	if clasificarEntradaAutoridadO4bM38(f.entrada) != esperada {
		os.Exit(21)
	}
	antes := capturarCanonProfundoAutoridadO4bP1M38(f.e, f.entrada)
	if !aplicarMutacionFatalAutoridadO4bP1M38(elegido.nombre, f) {
		os.Exit(22)
	}
	mutada := capturarCanonProfundoAutoridadO4bP1M38(f.e, f.entrada)
	if !cambiosCanonExactosAutoridadO4bP1M38(antes, mutada, elegido.dimension) {
		os.Exit(23)
	}
	if clasificarEntradaAutoridadO4bM38(f.entrada) != entradaFatalAutoridadO4bM38 {
		os.Exit(24)
	}
	if !cambiosCanonExactosAutoridadO4bP1M38(mutada, capturarCanonProfundoAutoridadO4bP1M38(f.e, f.entrada)) {
		os.Exit(25)
	}
	preparacion := os.NewFile(3, "o4b-p1-r4-preparacion")
	if preparacion == nil {
		os.Exit(26)
	}
	escritos, err := io.WriteString(preparacion, marcadorPreparacionFatalAutoridadO4bP1M38)
	if err != nil || escritos != len(marcadorPreparacionFatalAutoridadO4bP1M38) || preparacion.Close() != nil {
		os.Exit(27)
	}
	_, _ = consumirAutoridadO4bM38(&f.entrada)
	os.Exit(28)
}

func TestO4BP1AutoridadCarreraUnGanador(t *testing.T) {
	e, p := iniciarEtapasPruebaO4aP4M38(t, causaCancelado65O4aM38)
	antes := capturarCanonProfundoAutoridadO4bP1M38(e, p)
	var ganadores atomic.Int32
	var consumidas atomic.Int32
	t.Cleanup(func() {
		if ganadores.Load() != 1 || consumidas.Load() != 1 ||
			p.estado.Load() != uint32(autorizacionConsumiendoO4bM38) ||
			e.causa.estado.Load() != uint32(causaA5EsperandoResultadoM38) {
			t.Errorf("carrera: ganadores=%d consumidas=%d permiso=%d causa=%d", ganadores.Load(), consumidas.Load(), p.estado.Load(), e.causa.estado.Load())
		}
		exigirCambiosCanonAutoridadO4bP1M38(t, antes, capturarCanonProfundoAutoridadO4bP1M38(e, p), "causa.estado", "slot.0.estado")
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

func nombreLlamadaAutoridadO4bP1M38(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return nombreLlamadaAutoridadO4bP1M38(x.X) + "." + x.Sel.Name
	}
	return "?"
}

func TestO4BP1AutoridadFatalidadSoloEnclavaContrato(t *testing.T) {
	_, actual, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("fuente de prueba no localizada")
	}
	ruta := filepath.Join(filepath.Dir(actual), "senales_grupo_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go")
	fset := token.NewFileSet()
	archivo, err := parser.ParseFile(fset, ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	esperadas := map[string]int{
		"a.estado.Load": 1, "a.estado.CompareAndSwap": 1,
		"e.causa.estado.Store": 1, "fatalO3cM38": 1,
		"estadoAutoridadO4bM38": 1, "uint32": 3,
	}
	var funcion *ast.FuncDecl
	for _, declaracion := range archivo.Decls {
		if candidata, ok := declaracion.(*ast.FuncDecl); ok && candidata.Name.Name == "fatalAutoridadO4bM38" {
			funcion = candidata
			break
		}
	}
	if funcion == nil {
		t.Fatal("fatalidad O4B-P1 ausente")
	}
	actuales, estructuraValida := make(map[string]int), true
	ast.Inspect(funcion.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			actuales[nombreLlamadaAutoridadO4bP1M38(x.Fun)]++
		case *ast.AssignStmt:
			estructuraValida = estructuraValida && x.Tok == token.DEFINE
		case *ast.IncDecStmt, *ast.GoStmt, *ast.DeferStmt, *ast.SendStmt:
			estructuraValida = false
		}
		return true
	})
	if !estructuraValida || fmt.Sprint(actuales) != fmt.Sprint(esperadas) {
		t.Fatalf("fatalidad con efectos no contractuales: estructura=%t llamadas=%v", estructuraValida, actuales)
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
