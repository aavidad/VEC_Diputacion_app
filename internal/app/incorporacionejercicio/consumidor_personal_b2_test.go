package incorporacionejercicio

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pa "vec-diputacion-granada/internal/modules/personal/application"
	pd "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type planConsumidorB2Prueba struct {
	estado             pp.EstadoPlanIncorporacionCT
	preparaciones      int
	consultas          int
	ejecuciones        int
	falloConsulta      error
	falloEjecucion     error
	reciboAltaOriginal string
}

func (p *planConsumidorB2Prueba) ResolverSeleccion(context.Context, pp.SeleccionPlanIncorporacionCT) (pp.ResultadoSeleccionPlanIncorporacionCT, error) {
	return pp.ResultadoSeleccionPlanIncorporacionCT{}, errors.New("resolución de selección inesperada")
}

func (p *planConsumidorB2Prueba) PrepararPlan(_ context.Context, q pp.SolicitudPlanIncorporacionCT) (pp.EstadoPlanIncorporacionCT, error) {
	p.preparaciones++
	r := p.estado
	r.Plan.Datos = q.DatosPlanIncorporacionCT
	r.Plan.HuellaSHA256 = r.Plan.CalcularHuellaSHA256()
	return r, nil
}
func (p *planConsumidorB2Prueba) ConsultarPlan(_ context.Context, _ pp.ConsultaPlanIncorporacionCT) (pp.EstadoPlanIncorporacionCT, error) {
	p.consultas++
	return p.estado, p.falloConsulta
}
func (p *planConsumidorB2Prueba) EjecutarPlan(_ context.Context, _ pp.ConsultaPlanIncorporacionCT) (pp.EstadoPlanIncorporacionCT, error) {
	p.ejecuciones++
	if p.falloEjecucion != nil {
		return pp.EstadoPlanIncorporacionCT{}, p.falloEjecucion
	}
	p.estado.Estado = "ejecutado"
	p.estado.EjecucionReciboRef = "ejecucion:personal"
	p.estado.EjecucionHuellaSHA256 = strings.Repeat("a", 64)
	return p.estado, nil
}

type actoresConsumidorB2Prueba struct {
	actor core.ContextoActor
	fallo error
}

func (a actoresConsumidorB2Prueba) ActorPreparacionPlanB2(context.Context) (core.ContextoActor, error) {
	return a.actor, a.fallo
}
func (a actoresConsumidorB2Prueba) ActorConsultaPlanB2(context.Context) (core.ContextoActor, error) {
	return a.actor, a.fallo
}
func (a actoresConsumidorB2Prueba) ActorEjecucionPlanB2(context.Context) (core.ContextoActor, error) {
	return a.actor, a.fallo
}
func (a actoresConsumidorB2Prueba) ActorLecturaHechosB2(context.Context) (core.ContextoActor, error) {
	return a.actor, a.fallo
}

type autoridadRPTConsumidorB2Prueba struct {
	cruzar            bool
	falloReserva      error
	falloConfirmacion error
}

func (a *autoridadRPTConsumidorB2Prueba) AutorizarReservaPlanB2(_ context.Context, _ pp.PlanIncorporacionCT, m vp.MaterialReservaUsoCategoriaRPT) (vp.OrdenReservaUsoCategoriaRPT, error) {
	if a.falloReserva != nil {
		return vp.OrdenReservaUsoCategoriaRPT{}, a.falloReserva
	}
	if a.cruzar {
		m.UsoRef = "uso:ajeno"
	}
	return vp.OrdenReservaUsoCategoriaRPT{Material: m}, nil
}
func (a *autoridadRPTConsumidorB2Prueba) AutorizarConfirmacionPlanB2(_ context.Context, _ pp.PlanIncorporacionCT, m vp.MaterialTerminalUsoCategoriaRPT) (vp.OrdenConfirmacionUsoCategoriaRPT, error) {
	if a.falloConfirmacion != nil {
		return vp.OrdenConfirmacionUsoCategoriaRPT{}, a.falloConfirmacion
	}
	return vp.OrdenConfirmacionUsoCategoriaRPT{Material: m}, nil
}

type rptConsumidorB2Prueba struct {
	uso            *vp.UsoCategoriaRPT
	reservas       int
	confirmaciones int
	fallo          error
	terminal       vp.MaterialTerminalUsoCategoriaRPT
	despuesReserva func()
}

func (r *rptConsumidorB2Prueba) ReservarUsoCategoriaRPT(_ context.Context, o vp.OrdenReservaUsoCategoriaRPT) (vp.ResultadoUsoCategoriaRPT, error) {
	r.reservas++
	if r.despuesReserva != nil {
		r.despuesReserva()
	}
	if r.fallo != nil {
		return vp.ResultadoUsoCategoriaRPT{}, r.fallo
	}
	if r.uso == nil {
		m := o.Material
		r.uso = &vp.UsoCategoriaRPT{Consumidor: m.Consumidor, UsoRef: m.UsoRef, CategoriaID: m.CategoriaID,
			Publicacion: m.Publicacion, ReservaReciboRef: m.ReservaReciboRef, Estado: "reservado", Revision: 1}
	}
	return vp.ResultadoUsoCategoriaRPT{Encontrado: true, Uso: r.uso}, nil
}
func (r *rptConsumidorB2Prueba) ConfirmarUsoCategoriaRPT(_ context.Context, o vp.OrdenConfirmacionUsoCategoriaRPT) (vp.ResultadoUsoCategoriaRPT, error) {
	r.confirmaciones++
	r.terminal = o.Material
	r.uso.Estado = "confirmado"
	ref := o.Material.TerminalReciboRef
	instante := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	r.uso.TerminalReciboRef, r.uso.TerminalEn = &ref, &instante
	return vp.ResultadoUsoCategoriaRPT{Encontrado: true, Uso: r.uso}, nil
}
func (r *rptConsumidorB2Prueba) CancelarUsoCategoriaRPT(context.Context, vp.OrdenCancelacionUsoCategoriaRPT) (vp.ResultadoUsoCategoriaRPT, error) {
	return vp.ResultadoUsoCategoriaRPT{}, errors.New("cancelación inesperada")
}

type fichaConsumidorB2Prueba struct {
	ficha    pd.FichaEmpleadoB2
	lecturas int
}

func (f *fichaConsumidorB2Prueba) ConsultarFicha(_ context.Context, q pd.SolicitudFichaEmpleadoB2) (pp.ResultadoFichaEmpleadoB2, error) {
	f.lecturas++
	r := f.ficha
	r.Corte = q.Corte
	return pp.ResultadoFichaEmpleadoB2{Ficha: r, Evidencia: pp.EvidenciaRegistroEmpleadoB2{
		ReciboRef: "recibo:lectura", DecisionRef: "decision:lectura", AuditoriaRef: "auditoria:lectura",
		EfectoRef: q.EmpleadoRef, ConsumoHuellaSHA256: strings.Repeat("a", 64), ConsultadaEn: q.Corte.ConocidoEn}}, nil
}

func solicitudReservaConsumidorB2Prueba() SolicitudReservaPersonalB2 {
	x := contratoB2Prueba()
	x.PlazaRef = "plaza:11111111-1111-4111-8111-111111111111"
	x.PuestoRef = "puesto:22222222-2222-4222-8222-222222222222"
	return SolicitudReservaPersonalB2{Contrato: x,
		Persona: PersonaSeleccionadaBolsa{OrganizacionRef: x.OrganizacionRef, ExpedienteRef: x.ExpedienteRef,
			AceptacionRef: x.AceptacionRef, LlamamientoRef: x.LlamamientoRef, SeleccionRef: x.SeleccionRef,
			VersionSeleccion: x.VersionSeleccion, ReciboRef: x.SeleccionReciboRef, PersonaRef: "per_" + strings.Repeat("p", 24),
			PersonaVersion: 2, FuenteRef: "procedencia:bolsa", FuenteVersion: 1, FuenteSHA256: strings.Repeat("d", 64)},
		Puesto: PuestoRPTNominal{OrganizacionRef: x.OrganizacionRef, ExpedienteRef: x.ExpedienteRef, Fuente: x.FuenteRPT,
			PuestoRef: x.PuestoRef, PlazaRef: x.PlazaRef, CategoriaRef: x.CategoriaRef,
			VinculoRevision: x.VinculoRevision, VinculoReciboRef: x.VinculoReciboRef, Prospectivo: true}}
}

func escenarioConsumidorB2(t *testing.T, modo string) (*ConsumidorPersonalB2, *planConsumidorB2Prueba, *rptConsumidorB2Prueba, *fichaConsumidorB2Prueba) {
	t.Helper()
	ahora := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	ctx, _, _ := autoridadFixtureContexto(t, ahora, "p", "c")
	s := solicitudReservaConsumidorB2Prueba()
	d, err := datosPlanPropietarioB2(s)
	if err != nil {
		t.Fatal(err)
	}
	p := pp.PlanIncorporacionCT{ClasesOcupacionCatalogoRef: "personal:clases_ocupacion_ct", ClasesOcupacionCatalogoVersion: 1,
		ClasesOcupacionCatalogoHuellaSHA256: strings.Repeat("f", 64), PlanRef: "plan:personal", ReciboRef: "recibo:plan-personal", Version: 1,
		Datos: d, Modo: modo, ClaveAltaRelacion: "33333333-3333-4333-8333-333333333333", ClaveOcupacion: "44444444-4444-4444-8444-444444444444",
		UsoRPTRef: "uso:personal", ReservaRPTRef: "reserva:personal", ConfirmacionRPTRef: "confirmacion:personal"}
	emp := "emp_" + strings.Repeat("e", 24)
	if modo == "nueva_relacion" {
		p.EmpleadoExistenteRef = emp
	}
	p.HuellaSHA256 = p.CalcularHuellaSHA256()
	if p.Validar() != nil {
		t.Fatal("plan fixture inválido")
	}
	rel := "rel_" + strings.Repeat("r", 24)
	tipo := "alta"
	if modo == "nueva_relacion" {
		tipo = "relacion"
	}
	a := &pp.ReciboActoRegistroEmpleadoB2{ReciboRef: "perrec_" + strings.Repeat("a", 32), EmpleadoRef: emp, RelacionRef: rel,
		Tipo: tipo, Version: 1, RegistradoEn: ahora, EfectoRef: emp}
	if modo == "alta_empleado" {
		a.EfectoRef = d.PersonaRef
	}
	o := &pp.ReciboActoRegistroEmpleadoB2{ReciboRef: "perrec_" + strings.Repeat("b", 32), EmpleadoRef: emp, RelacionRef: rel,
		Tipo: "ocupacion", HechoRef: "ocu_" + strings.Repeat("o", 24), Version: 1, RegistradoEn: ahora, EfectoRef: emp}
	planes := &planConsumidorB2Prueba{estado: pp.EstadoPlanIncorporacionCT{Plan: p, Estado: "relacion_registrada", ReciboAltaRelacion: a, ReciboOcupacion: o}}
	snapshot := func(tipo, ref string) *pd.SnapshotEntradaCatalogoEmpleadoB2 {
		return &pd.SnapshotEntradaCatalogoEmpleadoB2{OrganismoRef: d.OrganismoRef, Tipo: tipo, Ref: ref, Version: 1, Revision: 1,
			Estado: "publicada", Denominacion: "Entrada sintética", VigenteDesde: d.Desde, HuellaSHA256: strings.Repeat("a", 64)}
	}
	traza := pd.TrazaEmpleadoB2{Desde: d.Desde, Hasta: d.Hasta, RegistradaEn: ahora, Version: 1, ActoRef: d.Procedencia.ActoRef,
		FuenteRef: d.Procedencia.FuenteRef, FuenteVersion: d.Procedencia.FuenteVersion}
	ficha := &fichaConsumidorB2Prueba{ficha: pd.FichaEmpleadoB2{EmpleadoRef: emp, PersonaRef: d.PersonaRef, OrganismoRef: d.OrganismoRef, Version: 3,
		Relaciones: []pd.RelacionRegistroEmpleadoB2{{RelacionRef: rel, OrganismoRef: d.OrganismoRef, UnidadRef: d.UnidadRef,
			RegimenRef: d.Regimen.Ref, ModalidadRef: d.Modalidad.Ref, Estado: "vigente", Traza: traza,
			CatalogoSnapshot: pd.SnapshotCatalogoEmpleadoB2{Regimen: snapshot("regimen", d.Regimen.Ref), Modalidad: snapshot("modalidad", d.Modalidad.Ref)}}},
		Ocupaciones: []pd.OcupacionEmpleadoB2{{OcupacionRef: o.HechoRef, RelacionRef: rel, UnidadRef: d.UnidadRef, PuestoRef: d.PuestoRef,
			PlazaRef: d.PlazaRef, ModalidadRef: d.Modalidad.Ref, Estado: "vigente", Clase: d.ClaseOcupacion, Traza: traza,
			CatalogoSnapshot: pd.SnapshotCatalogoEmpleadoB2{Modalidad: snapshot("modalidad", d.Modalidad.Ref)}}}}}
	hechos, err := pa.NuevoServicioConsultaIncorporacionCT(ficha)
	if err != nil {
		t.Fatal(err)
	}
	rpt := &rptConsumidorB2Prueba{}
	c, err := NuevoConsumidorPersonalB2(ConfiguracionConsumidorPersonalB2{Personal: planes, Ficha: ficha, Hechos: hechos,
		RPT: rpt, AutoridadRPT: &autoridadRPTConsumidorB2Prueba{}, Actores: actoresConsumidorB2Prueba{actor: ctx.Resultado.Contexto}, Reloj: autoridadRelojDoble{instante: ahora}})
	if err != nil {
		t.Fatal(err)
	}
	return c, planes, rpt, ficha
}

func TestConsumidorPersonalB2RetomaConMismosRecibosYReleeHechos(t *testing.T) {
	for _, modo := range []string{"alta_empleado", "nueva_relacion"} {
		t.Run(modo, func(t *testing.T) {
			c, p, rpt, f := escenarioConsumidorB2(t, modo)
			var recibo string
			for i := 0; i < 2; i++ {
				r, err := c.ConfirmarPersonalB2(context.Background(), p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
				if err != nil {
					t.Fatal(err)
				}
				if i == 0 {
					recibo = r.Estado.ReciboAltaRelacion.ReciboRef
				} else if r.Estado.ReciboAltaRelacion.ReciboRef != recibo {
					t.Fatal("cambió el recibo original")
				}
				if r.Hechos.Relacion.Hasta != "" || r.Hechos.Ocupacion.Hasta != "" || r.Uso.Uso.Estado != "confirmado" ||
					rpt.terminal.EvidenciaRef != p.estado.EjecucionReciboRef || rpt.terminal.EvidenciaSHA256 != p.estado.EjecucionHuellaSHA256 {
					t.Fatal("periodo o evidencia terminal alterados")
				}
			}
			if p.ejecuciones != 1 || p.preparaciones != 0 || f.lecturas != 4 || rpt.reservas != 2 || rpt.confirmaciones != 2 {
				t.Fatalf("duplicó ejecución o perdió lectura actual: %+v, lecturas=%d", p, f.lecturas)
			}
		})
	}
}

func TestConsumidorPersonalB2GETNoPreparaNiReservaNiEjecuta(t *testing.T) {
	c, p, rpt, f := escenarioConsumidorB2(t, "alta_empleado")
	_, err := c.ConsultarPersonalB2(context.Background(), p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
	if err != nil || p.consultas != 1 || p.preparaciones != 0 || p.ejecuciones != 0 || rpt.reservas != 0 || rpt.confirmaciones != 0 || f.lecturas != 0 {
		t.Fatalf("GET produjo efectos: %v", err)
	}
}

func TestConsumidorPersonalB2VersionesLimiteAntesDePreparar(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*SolicitudReservaPersonalB2)
		valido  bool
	}{
		{"expediente_maximo", func(s *SolicitudReservaPersonalB2) {
			s.Contrato.VersionExpediente = ct.MaximoEnteroSeguroOperacionAnalisis
		}, true},
		{"expediente_fuera_de_contrato", func(s *SolicitudReservaPersonalB2) {
			s.Contrato.VersionExpediente = ct.MaximoEnteroSeguroOperacionAnalisis + 1
		}, false},
		{"expediente_desborda_int64", func(s *SolicitudReservaPersonalB2) { s.Contrato.VersionExpediente = uint64(math.MaxInt64) + 1 }, false},
		{"persona_maxima", func(s *SolicitudReservaPersonalB2) {
			s.Contrato.PersonaVersion = ct.MaximoEnteroSeguroOperacionAnalisis
			s.Persona.PersonaVersion = s.Contrato.PersonaVersion
		}, true},
		{"persona_fuera_de_contrato", func(s *SolicitudReservaPersonalB2) {
			s.Contrato.PersonaVersion = ct.MaximoEnteroSeguroOperacionAnalisis + 1
			s.Persona.PersonaVersion = s.Contrato.PersonaVersion
		}, false},
		{"persona_desborda_int64", func(s *SolicitudReservaPersonalB2) {
			s.Contrato.PersonaVersion = uint64(math.MaxInt64) + 1
			s.Persona.PersonaVersion = s.Contrato.PersonaVersion
		}, false},
		{"catalogo_maximo", func(s *SolicitudReservaPersonalB2) { s.Contrato.DatosPersonal.CatalogoRPTVersion = math.MaxInt32 }, true},
		{"catalogo_fuera_de_contrato", func(s *SolicitudReservaPersonalB2) { s.Contrato.DatosPersonal.CatalogoRPTVersion = math.MaxInt32 + 1 }, false},
		{"catalogo_desborda_int64", func(s *SolicitudReservaPersonalB2) {
			s.Contrato.DatosPersonal.CatalogoRPTVersion = uint64(math.MaxInt64) + 1
		}, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s := solicitudReservaConsumidorB2Prueba()
			caso.cambiar(&s)
			c, p, rpt, _ := escenarioConsumidorB2(t, "alta_empleado")
			_, err := c.ReservarPersonalB2(context.Background(), s)
			if caso.valido && (err != nil || p.preparaciones != 1) {
				t.Fatalf("versión válida rechazada: %v, preparaciones=%d", err, p.preparaciones)
			}
			if !caso.valido && (!errors.Is(err, ct.ErrIntencionIncorporacionAplicacion) || p.preparaciones != 0 || rpt.reservas != 0) {
				t.Fatalf("versión inválida produjo efecto o error inesperado: %v, preparaciones=%d, reservas=%d", err, p.preparaciones, rpt.reservas)
			}
		})
	}
}

func TestConsumidorPersonalB2PlanConVersionInvalidaNoSeDevuelveComoReserva(t *testing.T) {
	c, p, rpt, _ := escenarioConsumidorB2(t, "alta_empleado")
	p.estado.Plan.Version = -1
	_, err := c.ReservarPersonalB2(context.Background(), solicitudReservaConsumidorB2Prueba())
	if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || rpt.reservas != 0 {
		t.Fatalf("plan con versión inválida admitido: %v", err)
	}
}

func TestConsumidorPersonalB2DenegacionCaidaYCrucesNoEjecutan(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		fallo  error
		want   error
	}{
		{"denegado", pd.ErrRegistroEmpleadoB2Denegado, ct.ErrDenegadaIncorporacionAplicacion},
		{"caido", pd.ErrRegistroEmpleadoB2NoDisponible, ct.ErrComposicionIncorporacionAplicacion},
		{"sin_plan", pd.ErrRegistroEmpleadoB2NoEncontrado, ct.ErrPreparacionIncorporacionPendiente},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c, p, rpt, _ := escenarioConsumidorB2(t, "alta_empleado")
			p.falloConsulta = caso.fallo
			_, err := c.ConfirmarPersonalB2(context.Background(), p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
			if !errors.Is(err, caso.want) || p.ejecuciones != 0 || rpt.reservas != 0 {
				t.Fatalf("clasificación o efecto: %v", err)
			}
		})
	}
	c, p, rpt, _ := escenarioConsumidorB2(t, "alta_empleado")
	c.c.AutoridadRPT = &autoridadRPTConsumidorB2Prueba{cruzar: true}
	_, err := c.ConfirmarPersonalB2(context.Background(), p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
	if !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || rpt.reservas != 0 || p.ejecuciones != 0 {
		t.Fatalf("autoridad RPT cruzada: %v", err)
	}
}

func TestConsumidorPersonalB2DenegacionV3RegistradaDifiereDeFalloDeRegistro(t *testing.T) {
	casos := []struct {
		nombre string
		fallo  error
		want   error
	}{
		{"denegacion_registrada", errors.Join(core.ErrAutorizacionDenegada, vp.ErrDenegacionExplicitaAutorizacionLigadaV3), ct.ErrDenegadaIncorporacionAplicacion},
		{"registro_denegacion_caido", errors.Join(core.ErrAutorizacionDenegada, vp.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible), ct.ErrComposicionIncorporacionAplicacion},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, p, rpt, _ := escenarioConsumidorB2(t, "alta_empleado")
			c.c.AutoridadRPT = &autoridadRPTConsumidorB2Prueba{falloReserva: caso.fallo}
			_, err := c.ConfirmarPersonalB2(context.Background(), p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
			if !errors.Is(err, caso.want) || rpt.reservas != 0 || rpt.confirmaciones != 0 || p.ejecuciones != 0 {
				t.Fatalf("reserva RPT produjo clasificación o efectos indebidos: %v, reserva=%d, confirmacion=%d, ejecucion=%d", err, rpt.reservas, rpt.confirmaciones, p.ejecuciones)
			}
		})
	}
	c, p, rpt, _ := escenarioConsumidorB2(t, "alta_empleado")
	c.c.AutoridadRPT = &autoridadRPTConsumidorB2Prueba{falloConfirmacion: errors.Join(core.ErrAutorizacionDenegada, vp.ErrDenegacionExplicitaAutorizacionLigadaV3)}
	_, err := c.ConfirmarPersonalB2(context.Background(), p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
	if !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || rpt.confirmaciones != 0 {
		t.Fatalf("confirmación RPT denegada no conservó 403 ni frenó el efecto: %v, confirmaciones=%d", err, rpt.confirmaciones)
	}
}

func TestConsumidorPersonalB2HechosCruzadosNoConfirmanRPT(t *testing.T) {
	c, p, rpt, f := escenarioConsumidorB2(t, "nueva_relacion")
	f.ficha.PersonaRef = "per_" + strings.Repeat("z", 24)
	_, err := c.ConfirmarPersonalB2(context.Background(), p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
	if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || rpt.confirmaciones != 0 {
		t.Fatalf("confirmó hechos ajenos: %v", err)
	}
}

func TestConsumidorPersonalB2NoConfirmaCatalogoOPrecedenciaDistintos(t *testing.T) {
	for _, campo := range []string{"catalogo", "procedencia"} {
		t.Run(campo, func(t *testing.T) {
			c, p, rpt, f := escenarioConsumidorB2(t, "alta_empleado")
			if campo == "catalogo" {
				f.ficha.Relaciones[0].CatalogoSnapshot.Regimen.Version++
			} else {
				f.ficha.Ocupaciones[0].Traza.FuenteRef = "fuente:otra"
			}
			_, err := c.ConfirmarPersonalB2(context.Background(), p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
			if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || rpt.confirmaciones != 0 {
				t.Fatalf("confirmó %s distinto: %v", campo, err)
			}
		})
	}
}

func TestConsumidorPersonalB2CancelacionTrasReservaNoEjecuta(t *testing.T) {
	c, p, rpt, _ := escenarioConsumidorB2(t, "alta_empleado")
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	rpt.despuesReserva = cancelar
	_, err := c.ConfirmarPersonalB2(ctx, p.estado.Plan.PlanRef, p.estado.Plan.Datos.OrganismoRef)
	if !errors.Is(err, context.Canceled) || p.ejecuciones != 0 || rpt.confirmaciones != 0 {
		t.Fatalf("ejecutó tras cancelar: %v", err)
	}
}
