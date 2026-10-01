package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func datosPlanCTPrueba() ports.DatosPlanIncorporacionCT {
	return ports.DatosPlanIncorporacionCT{IdempotenciaRef: "10000000-0000-4000-8000-000000000001", OrigenCTRef: "ct:plan", OrigenCTReciboRef: "ct:recibo", OrigenCTHuellaSHA256: strings.Repeat("a", 64), ExpedienteRef: "exp:uno", ExpedienteVersion: 7, OrganismoRef: "org:uno", UnidadRef: "uni:uno", PersonaRef: "per_" + strings.Repeat("p", 24), PersonaVersion: 1, FuenteBolsaRef: "bolsa:persona", FuenteBolsaVersion: 2, FuenteBolsaReciboRef: "bolsa:recibo", FuenteBolsaHuellaSHA256: strings.Repeat("b", 64), Regimen: domain.EntradaCatalogoEmpleadoB2{Ref: "reg:uno", Version: 1}, Modalidad: domain.EntradaCatalogoEmpleadoB2{Ref: "mod:uno", Version: 2}, Desde: domain.FechaCivil("2026-10-01"), PlazaRef: "plaza:10000000-0000-4000-8000-000000000002", PuestoRef: "puesto:10000000-0000-4000-8000-000000000003", ClaseOcupacion: "temporal", VersionPlantillaRef: "plantilla:uno", VersionRPTRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 2, FuenteOrganizacionRef: "organizacion:uno", FuenteOrganizacionHuellaSHA256: strings.Repeat("c", 64), CatalogoRPTID: "rpt:catalogo", CatalogoRPTModulo: "contrataciontemporal", CatalogoRPTCategoria: "categoria:uno", CatalogoRPTVersion: 1, CatalogoRPTHuellaSHA256: strings.Repeat("d", 64), VinculoCTReciboRef: "vinculo:recibo", Procedencia: domain.ProcedenciaActoEmpleadoB2{ActoRef: "acto:incorporacion", FuenteRef: "fuente:ct", FuenteVersion: 7, FuenteHuellaSHA256: strings.Repeat("e", 64), IdempotenciaRef: "10000000-0000-4000-8000-000000000004"}}
}
func planCTPrueba(modo string) ports.PlanIncorporacionCT {
	p := ports.PlanIncorporacionCT{ClasesOcupacionCatalogoRef: "personal:clases_ocupacion_ct", ClasesOcupacionCatalogoVersion: 1, ClasesOcupacionCatalogoHuellaSHA256: strings.Repeat("f", 64), PlanRef: "perplan_" + strings.Repeat("a", 32), ReciboRef: "perplanrec_" + strings.Repeat("b", 32), Version: 1, Datos: datosPlanCTPrueba(), Modo: modo, ClaveAltaRelacion: "10000000-0000-4000-8000-000000000005", ClaveOcupacion: "10000000-0000-4000-8000-000000000006", UsoRPTRef: "uso:uno", ReservaRPTRef: "reserva:uno", ConfirmacionRPTRef: "confirmacion:uno"}
	if modo == "nueva_relacion" {
		p.EmpleadoExistenteRef = "emp_" + strings.Repeat("q", 24)
	}
	p.HuellaSHA256 = p.CalcularHuellaSHA256()
	return p
}

type autorizadorPlanCTPrueba struct {
	t   *testing.T
	err error
	n   int
}

func (a *autorizadorPlanCTPrueba) AutorizarPlanIncorporacionCT(_ context.Context, m domain.MaterialPlanIncorporacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.n++
	if a.err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.err
	}
	h, _ := m.HuellaSHA256()
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	resumen, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion(), m.Recurso().Referencia, h, domain.AudienciaPlanIncorporacionCT, ahora, ahora.Add(3*time.Second))
	if e != nil {
		a.t.Fatal(e)
	}
	actor := m.Actor()
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	x, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if e != nil {
		a.t.Fatal(e)
	}
	return x, nil
}

type repoPlanCTPrueba struct {
	catalogo  *domain.CatalogoClasesOcupacionCT
	seleccion *domain.SeleccionOrganizacionPlanCT
	estado    ports.EstadoPlanIncorporacionCT
	n         int
	corromper bool
}

func (r *repoPlanCTPrueba) salida(o ports.OrdenPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	r.n++
	s := r.estado
	x := o.Autorizacion.ResumenCapacidad()
	s.Evidencia = ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "perplanacc_prueba", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "audit:actual", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}
	if r.corromper {
		s.Plan.Datos.PersonaRef = "per_" + strings.Repeat("z", 24)
	}
	return s, nil
}
func (r *repoPlanCTPrueba) PrepararPlan(_ context.Context, o ports.OrdenPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	return r.salida(o)
}
func (r *repoPlanCTPrueba) ConsultarPlan(_ context.Context, o ports.OrdenPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	return r.salida(o)
}
func (r *repoPlanCTPrueba) ConfirmarPlan(_ context.Context, o ports.OrdenPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	r.estado.Estado = "ejecutado"
	r.estado.EjecucionReciboRef = "perplanejec_prueba"
	h := sha256.Sum256([]byte(r.estado.Plan.HuellaSHA256 + "|" + r.estado.ReciboAltaRelacion.ReciboRef + "|" + r.estado.ReciboOcupacion.ReciboRef))
	r.estado.EjecucionHuellaSHA256 = hex.EncodeToString(h[:])
	return r.salida(o)
}

type fuenteReservaPlanCTPrueba struct {
	err   error
	ajena bool
}

func (f *fuenteReservaPlanCTPrueba) AcreditarReservaPlanCT(_ context.Context, p ports.PlanIncorporacionCT, _ core.ContextoActor) (ports.EvidenciaReservaRPTPlanCT, error) {
	if f.err != nil {
		return ports.EvidenciaReservaRPTPlanCT{}, f.err
	}
	d := p.Datos
	e := ports.EvidenciaReservaRPTPlanCT{UsoRPTRef: p.UsoRPTRef, ReservaRPTRef: p.ReservaRPTRef, PlanHuellaSHA256: p.HuellaSHA256, PlazaRef: d.PlazaRef, PuestoRef: d.PuestoRef, CatalogoID: d.CatalogoRPTID, Modulo: d.CatalogoRPTModulo, Categoria: d.CatalogoRPTCategoria, Version: d.CatalogoRPTVersion, HuellaSHA256: d.CatalogoRPTHuellaSHA256, ReciboRef: "rpt:recibo"}
	if f.ajena {
		e.PlazaRef = "plaza:otra"
	}
	return e, nil
}

type actosPlanCTPrueba struct {
	r              *repoPlanCTPrueba
	altas          int
	relaciones     int
	ocupaciones    int
	fallarTrasAlta bool
	claves         []string
	actorOcupacion core.ContextoActor
}

func (a *actosPlanCTPrueba) recibo(tipo, empleado, relacion, efecto string) ports.ReciboActoRegistroEmpleadoB2 {
	r := ports.ReciboActoRegistroEmpleadoB2{ReciboRef: "perrec_" + strings.Repeat("a", 32), EmpleadoRef: empleado, RelacionRef: relacion, Tipo: tipo, Version: 1, RegistradoEn: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), DecisionRef: "dec_historica", EfectoRef: efecto, ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "audit:historica"}
	if tipo == "alta" {
		r.ProyeccionRef = "pep_" + strings.Repeat("r", 24)
	} else {
		r.HechoRef = relacion
	}
	return r
}
func (a *actosPlanCTPrueba) RegistrarEmpleado(_ context.Context, s domain.SolicitudAltaEmpleadoB2) (ports.ResultadoAltaEmpleadoB2, error) {
	a.altas++
	a.claves = append(a.claves, s.Procedencia.IdempotenciaRef)
	r := a.recibo("alta", "emp_"+strings.Repeat("q", 24), "rel_"+strings.Repeat("r", 24), s.PersonaRef)
	a.r.estado.ReciboAltaRelacion = &r
	a.r.estado.Estado = "relacion_registrada"
	if a.fallarTrasAlta {
		a.fallarTrasAlta = false
		return ports.ResultadoAltaEmpleadoB2{}, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return ports.ResultadoAltaEmpleadoB2{Recibo: r}, nil
}
func (a *actosPlanCTPrueba) RegistrarHecho(_ context.Context, s domain.SolicitudHechoEmpleadoB2) (ports.ResultadoHechoEmpleadoB2, error) {
	a.claves = append(a.claves, s.Procedencia.IdempotenciaRef)
	if s.Tipo == "relacion" {
		a.relaciones++
		r := a.recibo("relacion", s.EmpleadoRef, "rel_"+strings.Repeat("r", 24), s.EmpleadoRef)
		a.r.estado.ReciboAltaRelacion = &r
		a.r.estado.Estado = "relacion_registrada"
		return ports.ResultadoHechoEmpleadoB2{Recibo: r}, nil
	}
	a.ocupaciones++
	a.actorOcupacion = s.Actor
	r := a.recibo("ocupacion", s.EmpleadoRef, s.RelacionRef, s.EmpleadoRef)
	r.ReciboRef = "perrec_" + strings.Repeat("b", 32)
	r.HechoRef = "ocu_" + strings.Repeat("o", 24)
	a.r.estado.ReciboOcupacion = &r
	a.r.estado.Estado = "ocupacion_registrada"
	return ports.ResultadoHechoEmpleadoB2{Recibo: r}, nil
}

func TestPlanCTRecuperaAltaDurableConContextoActualSinRepetir(t *testing.T) {
	p := planCTPrueba("alta_empleado")
	repo := &repoPlanCTPrueba{estado: ports.EstadoPlanIncorporacionCT{Plan: p, Estado: "preparado"}}
	actos := &actosPlanCTPrueba{r: repo, fallarTrasAlta: true}
	a := &autorizadorPlanCTPrueba{t: t}
	s, _ := NuevoServicioPlanIncorporacionCT(a, repo, actos, &fuenteReservaPlanCTPrueba{})
	q := ports.ConsultaPlanIncorporacionCT{PlanRef: p.PlanRef, OrganismoRef: p.Datos.OrganismoRef, Actor: solicitudP(t).Actor}
	if _, e := s.EjecutarPlan(context.Background(), q); !errors.Is(e, domain.ErrRegistroEmpleadoB2NoDisponible) || actos.altas != 1 || actos.ocupaciones != 0 {
		t.Fatal("fallo tras alta durable", e)
	}
	q.Actor.Instantanea.VinculoVersion++
	nuevo, _ := NuevoServicioPlanIncorporacionCT(a, repo, actos, &fuenteReservaPlanCTPrueba{})
	estado, e := nuevo.EjecutarPlan(context.Background(), q)
	if e != nil || estado.Estado != "ejecutado" || actos.altas != 1 || actos.ocupaciones != 1 || actos.actorOcupacion.Instantanea.VinculoVersion != q.Actor.Instantanea.VinculoVersion {
		t.Fatal("recuperacion nueva identidad repite efecto", e)
	}
	if len(actos.claves) != 2 || actos.claves[0] != p.ClaveAltaRelacion || actos.claves[1] != p.ClaveOcupacion {
		t.Fatal("claves no conservadas")
	}
	segundo, e := nuevo.EjecutarPlan(context.Background(), q)
	if e != nil || segundo.EjecucionReciboRef != estado.EjecucionReciboRef || actos.altas != 1 || actos.ocupaciones != 1 {
		t.Fatal("reintento terminal duplica", e)
	}
}
func TestPlanCTNuevaRelacionNoCreaEmpleado(t *testing.T) {
	p := planCTPrueba("nueva_relacion")
	repo := &repoPlanCTPrueba{estado: ports.EstadoPlanIncorporacionCT{Plan: p, Estado: "preparado"}}
	actos := &actosPlanCTPrueba{r: repo}
	s, _ := NuevoServicioPlanIncorporacionCT(&autorizadorPlanCTPrueba{t: t}, repo, actos, &fuenteReservaPlanCTPrueba{})
	e, err := s.EjecutarPlan(context.Background(), ports.ConsultaPlanIncorporacionCT{PlanRef: p.PlanRef, OrganismoRef: p.Datos.OrganismoRef, Actor: solicitudP(t).Actor})
	if err != nil || e.Plan.Modo != "nueva_relacion" || actos.altas != 0 || actos.relaciones != 1 || actos.ocupaciones != 1 {
		t.Fatal("nueva relacion no conserva empleado", err)
	}
}
func TestPlanCTDenegacionYReservaAjenaNoEjecutan(t *testing.T) {
	for _, modo := range []string{"denegado", "dependencia", "ajena", "salida_corrupta"} {
		t.Run(modo, func(t *testing.T) {
			p := planCTPrueba("alta_empleado")
			repo := &repoPlanCTPrueba{estado: ports.EstadoPlanIncorporacionCT{Plan: p, Estado: "preparado"}}
			actos := &actosPlanCTPrueba{r: repo}
			a := &autorizadorPlanCTPrueba{t: t}
			rpt := &fuenteReservaPlanCTPrueba{}
			esperado := domain.ErrRegistroEmpleadoB2Denegado
			switch modo {
			case "denegado":
				a.err = esperado
			case "dependencia":
				rpt.err = domain.ErrRegistroEmpleadoB2NoDisponible
				esperado = rpt.err
			case "ajena":
				rpt.ajena = true
			case "salida_corrupta":
				repo.corromper = true
				esperado = domain.ErrRegistroEmpleadoB2NoDisponible
			}
			s, _ := NuevoServicioPlanIncorporacionCT(a, repo, actos, rpt)
			_, e := s.EjecutarPlan(context.Background(), ports.ConsultaPlanIncorporacionCT{PlanRef: p.PlanRef, OrganismoRef: p.Datos.OrganismoRef, Actor: solicitudP(t).Actor})
			if !errors.Is(e, esperado) || actos.altas != 0 || actos.ocupaciones != 0 {
				t.Fatal("efecto sin autoridad", e)
			}
			if modo == "denegado" && repo.n != 0 {
				t.Fatal("consulta anterior a autorizacion")
			}
		})
	}
}
func TestPlanCTConsultaNoPreparaNiReserva(t *testing.T) {
	p := planCTPrueba("alta_empleado")
	repo := &repoPlanCTPrueba{estado: ports.EstadoPlanIncorporacionCT{Plan: p, Estado: "preparado"}}
	actos := &actosPlanCTPrueba{r: repo}
	s, _ := NuevoServicioPlanIncorporacionCT(&autorizadorPlanCTPrueba{t: t}, repo, actos, &fuenteReservaPlanCTPrueba{err: domain.ErrRegistroEmpleadoB2NoDisponible})
	if _, e := s.ConsultarPlan(context.Background(), ports.ConsultaPlanIncorporacionCT{PlanRef: p.PlanRef, OrganismoRef: p.Datos.OrganismoRef, Actor: solicitudP(t).Actor}); e != nil || actos.altas != 0 {
		t.Fatal("consulta toca dependencia efectos", e)
	}
}

func (r *repoPlanCTPrueba) ResolverSeleccion(_ context.Context, o ports.OrdenPlanIncorporacionCT) (ports.ResultadoSeleccionPlanIncorporacionCT, error) {
	if r.seleccion == nil {
		return ports.ResultadoSeleccionPlanIncorporacionCT{}, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	s, _ := r.salida(o)
	return ports.ResultadoSeleccionPlanIncorporacionCT{Seleccion: *r.seleccion, Evidencia: s.Evidencia}, nil
}

func TestPlanCTSeleccionPropietariaNoHaceEfectos(t *testing.T) {
	d := datosPlanCTPrueba()
	q := ports.SeleccionPlanIncorporacionCT{SelectorOrganizacionPlanCT: ports.SelectorOrganizacionPlanCT{PlazaRef: d.PlazaRef, PuestoRef: d.PuestoRef, Desde: d.Desde}, OrganismoRef: d.OrganismoRef, Actor: solicitudP(t).Actor}
	snapshot := domain.SeleccionOrganizacionPlanCT{PlantillaFuenteRef: "fuente:plantilla", RPTFuenteRef: "fuente:rpt", RevisionPlantilla: 1, RevisionRPT: 2, UnidadRef: d.UnidadRef, OrganismoRef: d.OrganismoRef, PlazaRef: d.PlazaRef, PuestoRef: d.PuestoRef, Desde: d.Desde, RevisionPlaza: 1, RevisionPuesto: 2, VersionPlantillaRef: d.VersionPlantillaRef, VersionRPTRef: d.VersionRPTRef, PlantillaHuellaSHA256: strings.Repeat("a", 64), RPTHuellaSHA256: strings.Repeat("b", 64), FuenteOrganizacionRef: d.FuenteOrganizacionRef, FuenteOrganizacionHuellaSHA256: d.FuenteOrganizacionHuellaSHA256}
	repo := &repoPlanCTPrueba{seleccion: &snapshot}
	actos := &actosPlanCTPrueba{r: repo}
	s, _ := NuevoServicioPlanIncorporacionCT(&autorizadorPlanCTPrueba{t: t}, repo, actos, &fuenteReservaPlanCTPrueba{err: domain.ErrRegistroEmpleadoB2NoDisponible})
	r, e := s.ResolverSeleccion(context.Background(), q)
	if e != nil || r.Seleccion.RevisionPuesto != 2 || actos.altas != 0 || repo.n != 1 {
		t.Fatal("selección no propietaria", e)
	}
	snapshot.PlazaRef = "plaza:10000000-0000-4000-8000-000000000008"
	if _, e := s.ResolverSeleccion(context.Background(), q); !errors.Is(e, domain.ErrRegistroEmpleadoB2NoDisponible) {
		t.Fatal("selección ajena", e)
	}
}

func (r *repoPlanCTPrueba) ConsultarClasesOcupacion(_ context.Context, o ports.OrdenPlanIncorporacionCT) (ports.ResultadoClasesOcupacionCT, error) {
	if r.catalogo == nil {
		return ports.ResultadoClasesOcupacionCT{}, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	ev, _ := r.salida(o)
	return ports.ResultadoClasesOcupacionCT{Catalogo: *r.catalogo, Evidencia: ev.Evidencia}, nil
}

func TestPlanCTClasesProcedenDeLaColeccionVersionada(t *testing.T) {
	c := domain.CatalogoClasesOcupacionCT{Ref: "personal:clases:publicacion", Version: 2, HuellaSHA256: strings.Repeat("e", 64), Opciones: []domain.OpcionClaseOcupacionCT{{Valor: "provisional", TextoClave: "personal.clases.provisional"}}}
	repo := &repoPlanCTPrueba{catalogo: &c}
	s, _ := NuevoServicioPlanIncorporacionCT(&autorizadorPlanCTPrueba{t: t}, repo, &actosPlanCTPrueba{r: repo}, &fuenteReservaPlanCTPrueba{err: domain.ErrRegistroEmpleadoB2NoDisponible})
	r, e := s.ConsultarClasesOcupacion(context.Background(), ports.ConsultaClasesOcupacionCT{OrganismoRef: "org:uno", Actor: solicitudP(t).Actor})
	if e != nil || r.Catalogo.Version != 2 || len(r.Catalogo.Opciones) != 1 || r.Catalogo.Opciones[0].Valor != "provisional" {
		t.Fatal("colección no procede del propietario", e)
	}
	c.Opciones[0].Valor = "desconocida"
	r, e = s.ConsultarClasesOcupacion(context.Background(), ports.ConsultaClasesOcupacionCT{OrganismoRef: "org:uno", Actor: solicitudP(t).Actor})
	if e != nil || len(r.Catalogo.Opciones) != 1 || r.Catalogo.Opciones[0].Valor != "desconocida" {
		t.Fatal("la colección versionada no admite una clase válida nueva", e)
	}
	c.Opciones = append(c.Opciones, domain.OpcionClaseOcupacionCT{Valor: "desconocida", TextoClave: "personal.clases.duplicada"})
	if _, e := s.ConsultarClasesOcupacion(context.Background(), ports.ConsultaClasesOcupacionCT{OrganismoRef: "org:uno", Actor: solicitudP(t).Actor}); !errors.Is(e, domain.ErrRegistroEmpleadoB2NoDisponible) {
		t.Fatal("colección con opción duplicada aceptada", e)
	}
}
