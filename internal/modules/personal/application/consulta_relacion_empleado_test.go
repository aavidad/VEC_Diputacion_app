package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorP struct {
	a   vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	err error
	n   int
	m   personaldomain.MaterialConsultaRelacionPropia
}

func (p *proveedorP) AutorizarConsultaRelacionPropia(_ context.Context, m personaldomain.MaterialConsultaRelacionPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.n++
	p.m = m
	return p.a, p.err
}

type repoP struct {
	r personalports.ResultadoConsultaRelacionPropia
	n int
	o personalports.OrdenConsultaRelacionPropia
}

func (p *repoP) ConsultarRelacionesPropiasDietas(_ context.Context, o personalports.OrdenConsultaRelacionPropia) (personalports.ResultadoConsultaRelacionPropia, error) {
	p.n++
	p.o = o
	return p.r, nil
}
func solicitudP(t *testing.T) personaldomain.SolicitudConsultaRelacionPropia {
	t.Helper()
	z := strings.Repeat("a", 24)
	ah := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	cu := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	in := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cu.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ah.Add(-time.Hour), VigenteHasta: ah.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ah.Add(-time.Hour), VigenteHasta: ah.Add(time.Hour)}}}
	a, e := vecdomain.NuevoContextoActor(cu, in, ah)
	if e != nil {
		t.Fatal(e)
	}
	f, _ := personaldomain.NuevaFechaCivil("2026-09-20")
	return personaldomain.SolicitudConsultaRelacionPropia{FechaReferencia: f, Operacion: personaldomain.OperacionListaRelacionPropia, Actor: a}
}
func exportP(t *testing.T, m personaldomain.MaterialConsultaRelacionPropia) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, e := m.HuellaSHA256()
	if e != nil {
		t.Fatal(e)
	}
	r := m.Recurso()
	n, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), "personal.relacion.propia.consultar_dietas", r.Referencia, h, "vec_personal.relacion_propia.consultar_dietas.v1", time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), time.Date(2026, 9, 20, 10, 0, 3, 0, time.UTC))
	if e != nil {
		t.Fatal(e)
	}
	raiz, e := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if e != nil {
		t.Fatal(e)
	}
	ctx, e := m.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	if e != nil {
		t.Fatal(e)
	}
	x, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), n, []byte("d"), []byte("m"), ctx, 1, 1, []byte("p"), []byte("s"), []byte("e"), raiz)
	if e != nil {
		t.Fatal(e)
	}
	return x
}
func exportPAlterada(t *testing.T, m personaldomain.MaterialConsultaRelacionPropia, op, aud, ef, hh string, personaVersion, perfilVersion uint64) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, _ := m.HuellaSHA256()
	if hh != "" {
		h = hh
	}
	n, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), op, ef, h, aud, time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), time.Date(2026, 9, 20, 10, 0, 3, 0, time.UTC))
	if e != nil {
		t.Fatal(e)
	}
	root, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	ctx, _ := m.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	x, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), n, []byte("d"), []byte("m"), ctx, personaVersion, perfilVersion, []byte("p"), []byte("s"), []byte("e"), root)
	if e != nil {
		t.Fatal(e)
	}
	return x
}
func TestMaterialPropioCanonYClonDefensivo(t *testing.T) {
	s := solicitudP(t)
	m, e := personaldomain.NuevoMaterialConsultaRelacionPropia(s)
	if e != nil {
		t.Fatal(e)
	}
	b := m.Canonico()
	if !bytes.Contains(b, []byte(`"operacion":"lista"`)) || m.Recurso().Atributos["relacion_ref"] != "sin_seleccion" {
		t.Fatal("material/recurso")
	}
	s.Actor.Instantanea.Vinculos[0].Referencia = "emp_" + strings.Repeat("z", 24)
	if bytes.Equal(b, []byte{}) || m.Solicitud().Actor.Instantanea.Vinculos[0].Referencia != "emp_"+strings.Repeat("a", 24) {
		t.Fatal("alias actor")
	}
	r := m.Recurso()
	r.Atributos["operacion"] = "otra"
	if m.Recurso().Atributos["operacion"] != "lista" {
		t.Fatal("alias recurso")
	}
}
func TestServicioOrdenaUnaVezYNoEntregaAutorizacionInvalida(t *testing.T) {
	s := solicitudP(t)
	m, e := personaldomain.NuevoMaterialConsultaRelacionPropia(s)
	if e != nil {
		t.Fatal(e)
	}
	p := &proveedorP{a: exportP(t, m)}
	repo := &repoP{r: personalports.ResultadoConsultaRelacionPropia{Evidencia: personalports.EvidenciaConsultaRelacionPropia{ReciboRef: "rpd_" + strings.Repeat("a", 32), DecisionRef: p.a.ResumenCapacidad().DecisionRef(), EfectoRef: p.a.ResumenCapacidad().EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "a", ConsultadaEn: p.a.ResumenCapacidad().EmitidaEn().Add(time.Microsecond)}}}
	serv, e := NuevoServicioConsultaRelacionEmpleado(p, repo)
	if e != nil {
		t.Fatal(e)
	}
	_, e = serv.ConsultarPropiasParaDietas(context.Background(), s)
	if e != nil || p.n != 1 || repo.n != 1 || !bytes.Equal(p.m.Canonico(), repo.o.Material.Canonico()) {
		t.Fatalf("e=%v p=%d r=%d", e, p.n, repo.n)
	}
}
func TestServicioConsultaPropiaRechazaDependenciasNulas(t *testing.T) {
	s, e := NuevoServicioConsultaRelacionEmpleado(nil, nil)
	if s != nil || !errors.Is(e, personalports.ErrRelacionEmpleadoNoDisponible) {
		t.Fatalf("servicio=%v error=%v", s, e)
	}
}

func TestServicioDistingueDenegacionDeFalloSinConsultarRelaciones(t *testing.T) {
	s := solicitudP(t)
	for _, caso := range []struct {
		nombre        string
		err, esperado error
	}{
		{"denegacion probada", personalports.ErrRelacionEmpleadoDenegada, personalports.ErrRelacionEmpleadoDenegada},
		{"dependencia caida", errors.New("fallo interno"), personalports.ErrRelacionEmpleadoNoDisponible},
		{"denegacion con timeout", errors.Join(personalports.ErrRelacionEmpleadoDenegada, context.DeadlineExceeded), context.DeadlineExceeded},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p, repo := &proveedorP{err: caso.err}, &repoP{}
			servicio, err := NuevoServicioConsultaRelacionEmpleado(p, repo)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := servicio.ConsultarPropiasParaDietas(context.Background(), s)
			if !errors.Is(err, caso.esperado) || repo.n != 0 || len(resultado.Relaciones) != 0 {
				t.Fatalf("error=%v, consultas=%d, relaciones=%d", err, repo.n, len(resultado.Relaciones))
			}
		})
	}
}
func TestServicioRechazaResumenAlteradoAntesDeRepositorio(t *testing.T) {
	s := solicitudP(t)
	m, _ := personaldomain.NuevoMaterialConsultaRelacionPropia(s)
	r := m.Recurso()
	h, _ := m.HuellaSHA256()
	for _, x := range []struct {
		op, aud, ef, hh               string
		personaVersion, perfilVersion uint64
	}{{"otra", "vec_personal.relacion_propia.consultar_dietas.v1", r.Referencia, h, 1, 1}, {"personal.relacion.propia.consultar_dietas", "otra", r.Referencia, h, 1, 1}, {"personal.relacion.propia.consultar_dietas", "vec_personal.relacion_propia.consultar_dietas.v1", "otro", h, 1, 1}, {"personal.relacion.propia.consultar_dietas", "vec_personal.relacion_propia.consultar_dietas.v1", r.Referencia, strings.Repeat("0", 64), 1, 1}, {"personal.relacion.propia.consultar_dietas", "vec_personal.relacion_propia.consultar_dietas.v1", r.Referencia, h, 2, 1}, {"personal.relacion.propia.consultar_dietas", "vec_personal.relacion_propia.consultar_dietas.v1", r.Referencia, h, 1, 2}} {
		p := &proveedorP{a: exportPAlterada(t, m, x.op, x.aud, x.ef, x.hh, x.personaVersion, x.perfilVersion)}
		repo := &repoP{}
		serv, _ := NuevoServicioConsultaRelacionEmpleado(p, repo)
		_, e := serv.ConsultarPropiasParaDietas(context.Background(), s)
		if !errors.Is(e, personalports.ErrRelacionEmpleadoNoDisponible) || repo.n != 0 {
			t.Fatalf("e=%v repo=%d", e, repo.n)
		}
	}
}

func TestServicioListaAceptaCeroUnaOMuchasSinSeleccionar(t *testing.T) {
	s := solicitudP(t)
	m, err := personaldomain.NuevoMaterialConsultaRelacionPropia(s)
	if err != nil {
		t.Fatal(err)
	}
	a := exportP(t, m)
	base := resultadoPropioValido(t, m, a)
	for _, caso := range []struct {
		nombre     string
		relaciones []personaldomain.RelacionEmpleado
	}{
		{"cero", nil},
		{"una", base.Relaciones},
		{"muchas", append(append([]personaldomain.RelacionEmpleado(nil), base.Relaciones...), relacionPropiaAlterna(t, base.Relaciones[0]))},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &repoP{r: personalports.ResultadoConsultaRelacionPropia{Relaciones: caso.relaciones, Evidencia: base.Evidencia}}
			servicio, _ := NuevoServicioConsultaRelacionEmpleado(&proveedorP{a: a}, repo)
			resultado, got := servicio.ConsultarPropiasParaDietas(context.Background(), s)
			if got != nil || len(resultado.Relaciones) != len(caso.relaciones) || repo.o.Material.Solicitud().Operacion != personaldomain.OperacionListaRelacionPropia || repo.o.Material.Solicitud().RelacionRef != "" {
				t.Fatalf("err=%v resultado=%d orden=%+v", got, len(resultado.Relaciones), repo.o.Material.Solicitud())
			}
		})
	}
}

func TestServicioDetalleRechazaRelacionDistinta(t *testing.T) {
	s := solicitudP(t)
	s.Operacion = personaldomain.OperacionDetalleRelacionPropia
	s.RelacionRef = "rel_" + strings.Repeat("a", 24)
	m, err := personaldomain.NuevoMaterialConsultaRelacionPropia(s)
	if err != nil {
		t.Fatal(err)
	}
	a := exportP(t, m)
	resultado := resultadoPropioValido(t, m, a)
	resultado.Relaciones[0] = relacionPropiaAlterna(t, resultado.Relaciones[0])
	repo := &repoP{r: resultado}
	servicio, _ := NuevoServicioConsultaRelacionEmpleado(&proveedorP{a: a}, repo)
	_, got := servicio.ConsultarPropiasParaDietas(context.Background(), s)
	if !errors.Is(got, personalports.ErrRelacionEmpleadoNoDisponible) || repo.n != 1 {
		t.Fatalf("err=%v llamadas=%d", got, repo.n)
	}
}

func TestServicioRechazaResultadoAdversoDelRepositorio(t *testing.T) {
	s := solicitudP(t)
	m, err := personaldomain.NuevoMaterialConsultaRelacionPropia(s)
	if err != nil {
		t.Fatal(err)
	}
	a := exportP(t, m)
	result := resultadoPropioValido(t, m, a)
	for _, mutar := range []struct {
		nombre  string
		aplicar func(*personalports.ResultadoConsultaRelacionPropia)
	}{
		{"recibo", func(r *personalports.ResultadoConsultaRelacionPropia) { r.Evidencia.ReciboRef = "recibo" }},
		{"hash", func(r *personalports.ResultadoConsultaRelacionPropia) {
			r.Evidencia.ConsumoHuellaSHA256 = strings.Repeat("A", 64)
		}},
		{"instante", func(r *personalports.ResultadoConsultaRelacionPropia) {
			r.Evidencia.ConsultadaEn = a.ResumenCapacidad().ExpiraEn()
		}},
		{"zona no UTC", func(r *personalports.ResultadoConsultaRelacionPropia) {
			r.Evidencia.ConsultadaEn = r.Evidencia.ConsultadaEn.In(time.FixedZone("UTC+0", 0))
		}},
		{"persona", func(r *personalports.ResultadoConsultaRelacionPropia) {
			r.Relaciones[0].PersonaRef = "per_" + strings.Repeat("b", 24)
		}},
		{"empleado", func(r *personalports.ResultadoConsultaRelacionPropia) {
			r.Relaciones[0].EmpleadoRef = "emp_" + strings.Repeat("b", 24)
		}},
		{"vigencia", func(r *personalports.ResultadoConsultaRelacionPropia) {
			f, _ := personaldomain.NuevaFechaCivil("2026-09-21")
			r.Relaciones[0].Desde = f
		}},
	} {
		t.Run(mutar.nombre, func(t *testing.T) {
			r := result
			r.Relaciones = append([]personaldomain.RelacionEmpleado(nil), result.Relaciones...)
			mutar.aplicar(&r)
			repo := &repoP{r: r}
			servicio, _ := NuevoServicioConsultaRelacionEmpleado(&proveedorP{a: a}, repo)
			_, got := servicio.ConsultarPropiasParaDietas(context.Background(), s)
			if !errors.Is(got, personalports.ErrRelacionEmpleadoNoDisponible) || repo.n != 1 {
				t.Fatalf("err=%v llamadas=%d", got, repo.n)
			}
		})
	}
}

func resultadoPropioValido(t *testing.T, m personaldomain.MaterialConsultaRelacionPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) personalports.ResultadoConsultaRelacionPropia {
	t.Helper()
	c := m.Solicitud()
	empleados, err := c.Actor.Referencias("empleado")
	if err != nil {
		t.Fatal(err)
	}
	desde, _ := personaldomain.NuevaFechaCivil("2026-01-01")
	relacion := c.RelacionRef
	if relacion == "" {
		relacion = "rel_" + strings.Repeat("a", 24)
	}
	return personalports.ResultadoConsultaRelacionPropia{
		Relaciones: []personaldomain.RelacionEmpleado{{PersonaRef: c.Actor.PersonaRef, EmpleadoRef: empleados[0], RelacionRef: relacion, UnidadRef: "unidad:x", Estado: "activa", Desde: desde, Version: 1, ProcedenciaActoRef: "acto:x", FuenteRef: "fuente:x", FuenteVersion: 1}},
		Evidencia:  personalports.EvidenciaConsultaRelacionPropia{ReciboRef: "rpd_" + strings.Repeat("a", 32), DecisionRef: a.ResumenCapacidad().DecisionRef(), EfectoRef: a.ResumenCapacidad().EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "aud", ConsultadaEn: a.ResumenCapacidad().EmitidaEn().Add(time.Microsecond)},
	}
}

func relacionPropiaAlterna(t *testing.T, original personaldomain.RelacionEmpleado) personaldomain.RelacionEmpleado {
	t.Helper()
	copia := original
	copia.RelacionRef = "rel_" + strings.Repeat("b", 24)
	return copia
}
