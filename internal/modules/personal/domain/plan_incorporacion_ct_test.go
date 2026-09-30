package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func datosDominioPlanCTPrueba() DatosPlanIncorporacionCT {
	return DatosPlanIncorporacionCT{IdempotenciaRef: "10000000-0000-4000-8000-000000000001", OrigenCTRef: "ct:plan", OrigenCTReciboRef: "ct:recibo", OrigenCTHuellaSHA256: strings.Repeat("a", 64), ExpedienteRef: "exp:uno", ExpedienteVersion: 7, OrganismoRef: "org:uno", UnidadRef: "uni:uno", PersonaRef: "per_" + strings.Repeat("p", 24), PersonaVersion: 1, FuenteBolsaRef: "bolsa:persona", FuenteBolsaVersion: 2, FuenteBolsaReciboRef: "bolsa:recibo", FuenteBolsaHuellaSHA256: strings.Repeat("b", 64), Regimen: EntradaCatalogoEmpleadoB2{Ref: "reg:uno", Version: 1}, Modalidad: EntradaCatalogoEmpleadoB2{Ref: "mod:uno", Version: 2}, Desde: FechaCivil("2026-10-01"), PlazaRef: "plaza:10000000-0000-4000-8000-000000000002", PuestoRef: "puesto:10000000-0000-4000-8000-000000000003", ClaseOcupacion: "temporal", VersionPlantillaRef: "plantilla:uno", VersionRPTRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 2, FuenteOrganizacionRef: "organizacion:uno", FuenteOrganizacionHuellaSHA256: strings.Repeat("c", 64), CatalogoRPTID: "rpt:catalogo", CatalogoRPTModulo: "contrataciontemporal", CatalogoRPTCategoria: "categoria:uno", CatalogoRPTVersion: 1, CatalogoRPTHuellaSHA256: strings.Repeat("d", 64), VinculoCTReciboRef: "vinculo:recibo", Procedencia: ProcedenciaActoEmpleadoB2{ActoRef: "acto:incorporacion", FuenteRef: "fuente:ct", FuenteVersion: 7, FuenteHuellaSHA256: strings.Repeat("e", 64), IdempotenciaRef: "10000000-0000-4000-8000-000000000004"}}
}
func TestPlanCTPeriodoCivilAbiertoYDatosAcreditados(t *testing.T) {
	d := datosDominioPlanCTPrueba()
	if e := d.Validar(); e != nil {
		t.Fatal("periodo abierto", e)
	}
	d.ClaseOcupacion = "reserva"
	if e := d.Validar(); e != nil {
		t.Fatal("gramática de clase del catálogo rechazada", e)
	}
	for _, f := range []func(*DatosPlanIncorporacionCT){func(d *DatosPlanIncorporacionCT) { d.Hasta = d.Desde }, func(d *DatosPlanIncorporacionCT) { d.Desde = "2026-02-30" }, func(d *DatosPlanIncorporacionCT) { d.PersonaRef = "" }, func(d *DatosPlanIncorporacionCT) { d.FuenteBolsaVersion = 0 }, func(d *DatosPlanIncorporacionCT) { d.RevisionPlaza = 0 }, func(d *DatosPlanIncorporacionCT) { d.CatalogoRPTHuellaSHA256 = "" }, func(d *DatosPlanIncorporacionCT) { d.ClaseOcupacion = "Reserva" }, func(d *DatosPlanIncorporacionCT) { d.ClaseOcupacion = "" }, func(d *DatosPlanIncorporacionCT) { d.ClaseOcupacion = "reserva;DROP" }} {
		otra := d
		f(&otra)
		if !errors.Is(otra.Validar(), ErrRegistroEmpleadoB2Invalido) {
			t.Fatal("dato no acreditado aceptado")
		}
	}
}
func TestPlanCTMaterialLigaActorActualPeroNegocioEsEstable(t *testing.T) {
	d := datosDominioPlanCTPrueba()
	a := actorAsignacionPrueba(t)
	m, e := NuevoMaterialPrepararPlanIncorporacionCT(SolicitudPlanIncorporacionCT{DatosPlanIncorporacionCT: d, Actor: a})
	if e != nil {
		t.Fatal(e)
	}
	a.Instantanea.VinculoVersion++
	n, e := NuevoMaterialPrepararPlanIncorporacionCT(SolicitudPlanIncorporacionCT{DatosPlanIncorporacionCT: d, Actor: a})
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Equal(m.Canonico(), n.Canonico()) || m.Datos().HuellaSHA256() != n.Datos().HuellaSHA256() {
		t.Fatal("identidad actual altero negocio o no liga material")
	}
	h, _ := m.HuellaSHA256()
	g, _ := n.HuellaSHA256()
	if h == g {
		t.Fatal("permiso no liga contexto actual")
	}
	canon := m.Canonico()
	canon[0] = 0
	if m.Canonico()[0] != '{' {
		t.Fatal("alias material mutable")
	}
	r := m.Recurso()
	r.Ambitos["organismo_ref"] = "otro"
	if m.Recurso().Ambitos["organismo_ref"] != d.OrganismoRef {
		t.Fatal("alias autoridad")
	}
}
func TestPlanCTModoYClavesSonParteDeLaIntencion(t *testing.T) {
	p := PlanIncorporacionCT{ClasesOcupacionCatalogoRef: "personal:clases_ocupacion_ct", ClasesOcupacionCatalogoVersion: 1, ClasesOcupacionCatalogoHuellaSHA256: strings.Repeat("f", 64), PlanRef: "perplan:uno", ReciboRef: "perplanrec:uno", Version: 1, Datos: datosDominioPlanCTPrueba(), Modo: "alta_empleado", ClaveAltaRelacion: "10000000-0000-4000-8000-000000000005", ClaveOcupacion: "10000000-0000-4000-8000-000000000006", UsoRPTRef: "uso:uno", ReservaRPTRef: "reserva:uno", ConfirmacionRPTRef: "confirmacion:uno"}
	p.HuellaSHA256 = p.CalcularHuellaSHA256()
	if e := p.Validar(); e != nil {
		t.Fatal(e)
	}
	p.Modo = "nueva_relacion"
	p.EmpleadoExistenteRef = "emp_" + strings.Repeat("q", 24)
	if p.Validar() == nil {
		t.Fatal("modo cambia sin nueva evidencia")
	}
	p.HuellaSHA256 = p.CalcularHuellaSHA256()
	if e := p.Validar(); e != nil {
		t.Fatal(e)
	}
	p.ClaveOcupacion = p.ClaveAltaRelacion
	p.HuellaSHA256 = p.CalcularHuellaSHA256()
	if p.Validar() == nil {
		t.Fatal("misma clave para dos actos")
	}
}

func TestPlanCTSeleccionLigaRevisionesFuenteYActor(t *testing.T) {
	q := SolicitudSeleccionPlanIncorporacionCT{SelectorOrganizacionPlanCT: SelectorOrganizacionPlanCT{PlazaRef: "plaza:10000000-0000-4000-8000-000000000002", PuestoRef: "puesto:10000000-0000-4000-8000-000000000003", Desde: "2026-10-01"}, OrganismoRef: "org:uno", Actor: actorAsignacionPrueba(t)}
	m, e := NuevoMaterialSeleccionPlanIncorporacionCT(q)
	if e != nil {
		t.Fatal(e)
	}
	s := SeleccionOrganizacionPlanCT{PlantillaFuenteRef: "fuente:plantilla", RPTFuenteRef: "fuente:rpt", RevisionPlantilla: 1, RevisionRPT: 2, UnidadRef: "uni:uno", OrganismoRef: q.OrganismoRef, PlazaRef: q.PlazaRef, PuestoRef: q.PuestoRef, Desde: q.Desde, RevisionPlaza: 1, RevisionPuesto: 2, VersionPlantillaRef: "plantilla:uno", VersionRPTRef: "rpt:uno", PlantillaHuellaSHA256: strings.Repeat("a", 64), RPTHuellaSHA256: strings.Repeat("b", 64), FuenteOrganizacionRef: "fuente:org", FuenteOrganizacionHuellaSHA256: strings.Repeat("c", 64)}
	if e := s.ValidarPara(m); e != nil {
		t.Fatal(e)
	}
	s.PuestoRef = "puesto:10000000-0000-4000-8000-000000000004"
	if s.ValidarPara(m) == nil {
		t.Fatal("otro puesto acreditado como selector")
	}
	if m.Recurso().Referencia != q.PlazaRef || m.Accion() != "personal.plan_incorporacion_ct.seleccionar" {
		t.Fatal("permiso de selección nominal")
	}
	q.PlazaRef = "10000000-0000-4000-8000-000000000002"
	if _, e := NuevoMaterialSeleccionPlanIncorporacionCT(q); e == nil {
		t.Fatal("referencia fuente no canónica")
	}
}

func TestPlanCTClasesSinVersionODuplicadasNoSonCatalogo(t *testing.T) {
	c := CatalogoClasesOcupacionCT{Ref: "personal:clases:publicacion", Version: 1, HuellaSHA256: strings.Repeat("e", 64), Opciones: []OpcionClaseOcupacionCT{
		{Valor: "temporal", TextoClave: "personal.clases.temporal"},
		{Valor: "titular", TextoClave: "personal.clases.titular"},
		{Valor: "provisional", TextoClave: "personal.clases.provisional"},
	}}
	if e := c.Validar(); e != nil {
		t.Fatal(e)
	}
	c.Opciones = append(c.Opciones, OpcionClaseOcupacionCT{Valor: "reserva", TextoClave: "personal.clases.reserva"})
	if c.Validar() == nil {
		t.Fatal("reserva no es incorporación efectiva")
	}
	c.Opciones = c.Opciones[:len(c.Opciones)-1]
	c.Version = 0
	if c.Validar() == nil {
		t.Fatal("fuente sin versión")
	}
	c.Version = 1
	c.Opciones = append(c.Opciones, c.Opciones[0])
	if c.Validar() == nil {
		t.Fatal("opciones duplicadas")
	}
	c.Opciones[len(c.Opciones)-1].Valor = "Reserva"
	if c.Validar() == nil {
		t.Fatal("opción sin gramática canónica")
	}
}
