package composicion

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type identidadR15Prueba struct{ llamadas int }

func (i *identidadR15Prueba) ResolverIdentidadRegistradaBorrador(context.Context) (IdentidadRegistradaBorrador, error) {
	i.llamadas++
	return IdentidadRegistradaBorrador{}, errors.New("no debe usarse")
}

type proveedorR15Prueba struct{ llamadas int }

func (p *proveedorR15Prueba) AutorizarBorradorPropio(context.Context, dietasports.EfectoAutorizacionBorrador) (dietasports.AutorizacionBorradorDurable, error) {
	p.llamadas++
	return dietasports.AutorizacionBorradorDurable{}, nil
}

type proveedorPersonalR15Prueba struct{}

func (proveedorPersonalR15Prueba) AutorizarConsultaRelacionPropia(context.Context, personaldomain.MaterialConsultaRelacionPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("no debe usarse")
}

type repositorioPersonalR15Prueba struct{}

func (repositorioPersonalR15Prueba) ConsultarRelacionesPropiasDietas(context.Context, personalports.OrdenConsultaRelacionPropia) (personalports.ResultadoConsultaRelacionPropia, error) {
	return personalports.ResultadoConsultaRelacionPropia{}, errors.New("no debe usarse")
}

func servicioPersonalR15Prueba(t *testing.T) *personalapp.ServicioConsultaRelacionEmpleado {
	t.Helper()
	s, e := personalapp.NuevoServicioConsultaRelacionEmpleado(proveedorPersonalR15Prueba{}, repositorioPersonalR15Prueba{})
	if e != nil {
		t.Fatal(e)
	}
	return s
}

func TestResolutorR15RechazaCancelacionAntesDeTodaLectura(t *testing.T) {
	i := &identidadR15Prueba{}
	a := &proveedorR15Prueba{}
	r, e := NuevoResolutorIdentidadEfectivaBorrador(i, servicioPersonalR15Prueba(t), a)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e = r.ResolverIdentidadEfectivaBorrador(ctx, dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionConsultarBorrador})
	if !errors.Is(e, context.Canceled) || i.llamadas != 0 || a.llamadas != 0 {
		t.Fatalf("err=%v identidad=%d autorizacion=%d", e, i.llamadas, a.llamadas)
	}
}

func TestResolutorR15RechazaDependenciasNulas(t *testing.T) {
	if r, e := NuevoResolutorIdentidadEfectivaBorrador(nil, servicioPersonalR15Prueba(t), &proveedorR15Prueba{}); r != nil || !errors.Is(e, ErrIdentidadEfectivaBorradorNoDisponible) {
		t.Fatalf("r=%v err=%v", r, e)
	}
}

// Las pruebas construyen el recibo registrado y el vínculo mediante las
// fábricas públicas; no hay una identidad alternativa de prueba.
type fuenteContextoR15 struct {
	identidad IdentidadRegistradaBorrador
	llamadas  int
	err       error
}

func (f *fuenteContextoR15) ResolverIdentidadRegistradaBorrador(context.Context) (IdentidadRegistradaBorrador, error) {
	f.llamadas++
	return f.identidad, f.err
}

type proveedorConsultaR15 struct {
	a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (p proveedorConsultaR15) AutorizarConsultaRelacionPropia(_ context.Context, m personaldomain.MaterialConsultaRelacionPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return p.a, nil
}

type repoConsultaR15 struct {
	r        personalports.ResultadoConsultaRelacionPropia
	llamadas int
	despues  func()
}

func (r *repoConsultaR15) ConsultarRelacionesPropiasDietas(context.Context, personalports.OrdenConsultaRelacionPropia) (personalports.ResultadoConsultaRelacionPropia, error) {
	r.llamadas++
	if r.despues != nil {
		r.despues()
	}
	return r.r, nil
}

type autorizadorCompletoR15 struct {
	a        dietasports.AutorizacionBorradorDurable
	llamadas int
	err      error
}

type revalidadorVinculoR15 struct {
	a vecdomain.AutenticacionRevalidadaV1
}

func (r revalidadorVinculoR15) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	return r.a, nil
}

type resolutorVinculoR15 struct {
	r vecdomain.ResultadoContextoActorRegistradoV2
}

func (r resolutorVinculoR15) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	return r.r, nil
}

type relojVinculoR15 struct{ ahora time.Time }

func (r relojVinculoR15) Ahora() time.Time { return r.ahora }

func (a *autorizadorCompletoR15) AutorizarBorradorPropio(context.Context, dietasports.EfectoAutorizacionBorrador) (dietasports.AutorizacionBorradorDurable, error) {
	a.llamadas++
	return a.a, a.err
}

func TestResolutorR15RelacionUnicaYCierresAntesDeDietas(t *testing.T) {
	base, material := identidadYMaterialR15(t)
	uno := relacionR15(base.Contexto.Contexto.PersonaRef, "a")
	for _, caso := range []struct {
		nombre     string
		relaciones []personaldomain.RelacionEmpleado
		want       error
		auth       int
	}{
		{"cero", nil, dietasports.ErrRelacionNoDisponible, 0}, {"una", []personaldomain.RelacionEmpleado{uno}, nil, 1}, {"muchas", []personaldomain.RelacionEmpleado{uno, relacionR15(base.Contexto.Contexto.PersonaRef, "b")}, dietasports.ErrRelacionAmbigua, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &repoConsultaR15{r: resultadoR15(material, uno)}
			repo.r.Relaciones = caso.relaciones
			s, _ := personalapp.NuevoServicioConsultaRelacionEmpleado(proveedorConsultaR15{material}, repo)
			a := &autorizadorCompletoR15{a: autorizacionR15(t, base, uno, "dietas.borrador.propio.consultar", "dietas:borradores:propios", "consultar_borrador_propio")}
			r, _ := NuevoResolutorIdentidadEfectivaBorrador(&fuenteContextoR15{identidad: base}, s, a)
			got, err := r.ResolverIdentidadEfectivaBorrador(context.Background(), dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionConsultarBorrador, Consulta: dietasports.ConsultaBorradoresPropios{Limite: 20}})
			if caso.want != nil {
				if !errors.Is(err, caso.want) || a.llamadas != caso.auth {
					t.Fatalf("err=%v auth=%d", err, a.llamadas)
				}
				return
			}
			if err != nil || got.Relacion.RelacionRef != uno.RelacionRef || got.Relacion.UnidadRef != uno.UnidadRef || a.llamadas != 1 {
				t.Fatalf("err=%v identidad=%#v auth=%d", err, got, a.llamadas)
			}
		})
	}
}

func TestCrearBorradorSinSelectorUsaSoloRelacionUnicaAcreditada(t *testing.T) {
	base, material := identidadYMaterialR15(t)
	uno := relacionR15(base.Contexto.Contexto.PersonaRef, "a")
	for _, caso := range []struct {
		nombre     string
		relaciones []personaldomain.RelacionEmpleado
		esperado   error
	}{
		{"una", []personaldomain.RelacionEmpleado{uno}, nil},
		{"muchas", []personaldomain.RelacionEmpleado{uno, relacionR15(base.Contexto.Contexto.PersonaRef, "b")}, dietasports.ErrRelacionAmbigua},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &repoConsultaR15{r: resultadoR15(material, uno)}
			repo.r.Relaciones = caso.relaciones
			consulta, err := personalapp.NuevoServicioConsultaRelacionEmpleado(proveedorConsultaR15{material}, repo)
			if err != nil {
				t.Fatal(err)
			}
			a := &autorizadorCompletoR15{a: autorizacionR15(t, base, uno, "dietas.borrador.propio.crear", "dietas:borradores:propios", "crear_borrador_propio")}
			r, err := NuevoResolutorIdentidadEfectivaBorrador(&fuenteContextoR15{identidad: base}, consulta, a)
			if err != nil {
				t.Fatal(err)
			}
			s := dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionCrearBorrador, Crear: dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_0123456789abcdef", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita técnica", CodigosRuta: []string{}}}
			got, err := r.ResolverIdentidadEfectivaBorrador(context.Background(), s)
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("error=%v; esperado=%v", err, caso.esperado)
			}
			if caso.esperado == nil && (got.Relacion.RelacionRef != uno.RelacionRef || a.llamadas != 1) {
				t.Fatalf("relacion=%q; autorizaciones=%d", got.Relacion.RelacionRef, a.llamadas)
			}
			if caso.esperado != nil && a.llamadas != 0 {
				t.Fatalf("se autorizó relación ambigua: %d", a.llamadas)
			}
		})
	}
}

func identidadYMaterialR15(t *testing.T) (IdentidadRegistradaBorrador, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	ahora := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	in := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, e := vecdomain.NuevoContextoActor(cuenta, in, ahora)
	if e != nil {
		t.Fatal(e)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: in.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, AcreditacionProcedenciaComponenteContextoActorV1: ac}}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if res.Validar() != nil {
		t.Fatal("resultado")
	}
	f, _ := personaldomain.NuevaFechaCivil("2026-09-21")
	s := personaldomain.SolicitudConsultaRelacionPropia{FechaReferencia: f, Operacion: personaldomain.OperacionListaRelacionPropia, Actor: actor}
	m, _ := personaldomain.NuevoMaterialConsultaRelacionPropia(s)
	h, _ := m.HuellaSHA256()
	sum, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), "personal.relacion.propia.consultar_dietas", m.Recurso().Referencia, h, "vec_personal.relacion_propia.consultar_dietas.v1", ahora, ahora.Add(3*time.Second))
	if e != nil {
		t.Fatal(e)
	}
	root, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	exp, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), sum, []byte("d"), []byte("m"), canon, 1, 1, []byte("p"), []byte("s"), []byte("e"), root)
	if e != nil {
		t.Fatal(e)
	}
	auth := vecdomain.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, MetodoObservado: vecdomain.AuthMethodCertificate, GarantiaObservada: vecdomain.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Minute)}
	v, e := vecdomain.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorVinculoR15{auth}, vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorVinculoR15{res}, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojVinculoR15{ahora})
	if e != nil {
		t.Fatal(e)
	}
	return IdentidadRegistradaBorrador{Contexto: res, Vinculo: v, FechaReferencia: f}, exp
}
func relacionR15(persona, suf string) personaldomain.RelacionEmpleado {
	d, _ := personaldomain.NuevaFechaCivil("2026-01-01")
	return personaldomain.RelacionEmpleado{PersonaRef: persona, EmpleadoRef: "emp_" + strings.TrimPrefix(persona, "per_"), RelacionRef: "rel_" + strings.Repeat(suf, 24), UnidadRef: "unidad:x", Estado: "activa", Desde: d, Version: 1, ProcedenciaActoRef: "acto:x", FuenteRef: "fuente:x", FuenteVersion: 1}
}
func resultadoR15(a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r personaldomain.RelacionEmpleado) personalports.ResultadoConsultaRelacionPropia {
	return personalports.ResultadoConsultaRelacionPropia{Relaciones: []personaldomain.RelacionEmpleado{r}, Evidencia: personalports.EvidenciaConsultaRelacionPropia{ReciboRef: "rpd_" + strings.Repeat("a", 32), DecisionRef: a.ResumenCapacidad().DecisionRef(), EfectoRef: a.ResumenCapacidad().EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "aud", ConsultadaEn: a.ResumenCapacidad().EmitidaEn().Add(time.Microsecond)}}
}
func autorizacionR15(t *testing.T, b IdentidadRegistradaBorrador, r personaldomain.RelacionEmpleado, accion, recurso, fin string) dietasports.AutorizacionBorradorDurable {
	t.Helper()
	_, a := identidadYMaterialR15(t)
	return dietasports.AutorizacionBorradorDurable{Material: a, Accion: accion, RecursoRef: recurso, Finalidad: fin, Revalidacion: selloRelacion(r, b.FechaReferencia)}
}
