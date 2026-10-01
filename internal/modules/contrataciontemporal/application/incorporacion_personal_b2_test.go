package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type fuentePlanB2Prueba struct{ llamadas int }

func (f *fuentePlanB2Prueba) ResolverPlanNominalB2(context.Context, ports.SolicitudPlanNominalB2, ports.ActorIncorporacionPersonalB2) (domain.PlanIncorporacionPersonalB2, error) {
	f.llamadas++
	return domain.PlanIncorporacionPersonalB2{}, ports.ErrPlanNominalB2NoDisponible
}
func (f *fuentePlanB2Prueba) VerificarHechosPersonalB2(context.Context, ports.ContratoPlanNominalB2, ports.HechosPersonalIncorporacionB2, ports.ActorIncorporacionPersonalB2) (ports.HechosPersonalIncorporacionB2, error) {
	f.llamadas++
	return ports.HechosPersonalIncorporacionB2{}, ports.ErrPlanNominalB2NoDisponible
}

type autoridadPlanB2Prueba struct {
	llamadas int
	accion   string
	permitir bool
}

func (a *autoridadPlanB2Prueba) AutorizarPlanNominalB2(_ context.Context, accion string, _ []byte, _ ports.ActorIncorporacionPersonalB2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	a.accion = accion
	if a.permitir {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
	}
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrPlanNominalB2Denegado
}

type repoPlanB2Prueba struct {
	llamadas       int
	lecturaErr     error
	confirmaciones int
}

func (r *repoPlanB2Prueba) RegistrarPlanNominalB2(context.Context, ports.RegistroPlanNominalB2, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ContratoPlanNominalB2, error) {
	r.llamadas++
	return ports.ContratoPlanNominalB2{}, nil
}
func (r *repoPlanB2Prueba) LeerContratoPlanNominal(context.Context, string, string, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ...string) (ports.ContratoPlanNominalB2, error) {
	r.llamadas++
	return ports.ContratoPlanNominalB2{}, r.lecturaErr
}
func (r *repoPlanB2Prueba) ConfirmarOrigenIncorporacionB2(context.Context, ports.ConfirmacionOrigenIncorporacionB2, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.OrigenIncorporacionPersonalB2, error) {
	r.llamadas++
	r.confirmaciones++
	return ports.OrigenIncorporacionPersonalB2{}, nil
}
func (r *repoPlanB2Prueba) LeerOrigenIncorporacionB2(context.Context, string, string, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ...string) (ports.OrigenIncorporacionPersonalB2, bool, error) {
	r.llamadas++
	return ports.OrigenIncorporacionPersonalB2{}, false, nil
}
func TestPlanPersonalB2RevalidacionNominalEnCadaLectura(t *testing.T) {
	f, a, r := &fuentePlanB2Prueba{}, &autoridadPlanB2Prueba{}, &repoPlanB2Prueba{}
	s, e := NuevoServicioPlanNominalB2(f, a, r)
	if e != nil {
		t.Fatal(e)
	}
	for range 2 {
		if _, e = s.LeerContratoPlanNominal(context.Background(), "org:uno", "expediente:uno"); !errors.Is(e, ports.ErrPlanNominalB2Denegado) {
			t.Fatal(e)
		}
	}
	if _, _, e = s.LeerOrigenIncorporacionB2(context.Background(), "org:uno", "expediente:uno"); !errors.Is(e, ports.ErrPlanNominalB2Denegado) {
		t.Fatal(e)
	}
	if a.llamadas != 3 || a.accion != ports.AccionLeerPlanNominalB2 || r.llamadas != 0 || f.llamadas != 0 {
		t.Fatal("lectura/replay evita concesión nominal actual o ejecuta un efecto")
	}
}
func TestPlanPersonalB2NoReservaPersonalParaActorSinResolver(t *testing.T) {
	f, a, r := &fuentePlanB2Prueba{}, &autoridadPlanB2Prueba{}, &repoPlanB2Prueba{}
	s, _ := NuevoServicioPlanNominalB2(f, a, r)
	_, e := s.RegistrarPlanNominalB2(context.Background(), ports.SolicitudPlanNominalB2{}, ports.ActorIncorporacionPersonalB2{})
	if !errors.Is(e, ports.ErrPlanNominalB2Invalido) || f.llamadas+a.llamadas+r.llamadas != 0 {
		t.Fatal("petición sin actor llega a una reserva")
	}
}

func (f *fuentePlanB2Prueba) ResolverUnidadPlanNominalB2(context.Context, string, string) (string, error) {
	return "unidad:rrhh", nil
}

func TestPlanPersonalB2PrimerPOSTSinPlanNoConfirmaOrigen(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	cuenta := vd.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vd.AuthMethodCertificate, Garantia: vd.AuthAssuranceHigh}
	instantanea := vd.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1,
		Estado: vd.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, e := vd.NuevoContextoActor(cuenta, instantanea, ahora)
	if e != nil {
		t.Fatal(e)
	}
	f, a, r := &fuentePlanB2Prueba{}, &autoridadPlanB2Prueba{permitir: true}, &repoPlanB2Prueba{lecturaErr: ports.ErrPlanNominalB2NoEncontrado}
	s, e := NuevoServicioPlanNominalB2(f, a, r)
	if e != nil {
		t.Fatal(e)
	}
	m := ports.ConfirmacionOrigenIncorporacionB2{OrganizacionRef: "org:uno", ExpedienteRef: "expediente:uno", PlanRef: "plan:uno", PlanVersion: 1, PlanSHA256: strings.Repeat("a", 64)}
	_, e = s.ConfirmarOrigenIncorporacionB2(context.Background(), m, actor)
	if !errors.Is(e, ports.ErrPlanNominalB2NoEncontrado) || a.llamadas != 1 || a.accion != ports.AccionLeerPlanNominalB2 || r.llamadas != 1 || r.confirmaciones != 0 {
		t.Fatalf("primer POST sin plan no detuvo confirmación: err=%v autoridad=%d repo=%d confirma=%d", e, a.llamadas, r.llamadas, r.confirmaciones)
	}
}

func TestPlanPersonalB2CotejaVersionesNativasNoUUIDInstrumento(t *testing.T) {
	h := strings.Repeat("a", 64)
	fuente := domain.FuentePlanPersonalB2{Ref: "fuente:original", Version: 1, SHA256: h}
	p := domain.PlanIncorporacionPersonalB2{
		OrganizacionRef: "org:uno", UnidadCTRef: "unidad:rrhh", ExpedienteRef: "expediente:uno", VersionExpediente: 8,
		AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis", AnalisisSHA256: h, PropuestaReciboRef: "recibo:propuesta",
		AceptacionRef: "aceptacion:uno", AceptacionReciboRef: "recibo:aceptacion",
		Bolsa: domain.SelectorBolsaPlanB2{UnidadRef: "unidad:uno", CategoriaRef: "categoria:uno", NecesidadRef: "necesidad:uno",
			AceptacionOperacionRef: "operacion:aceptacion", AceptacionRegistroSHA256: h, AperturaOperacionRef: "operacion:apertura",
			AperturaRegistroSHA256: h, LlamamientoRef: "llamamiento:uno", PropuestaRef: "propuesta:uno"},
		PersonaRef: "per_aaaaaaaaaaaaaaaaaaaaaaaa", PersonaVersion: 1, PersonaFuente: fuente, PersonaReciboBolsaRef: "recibo:bolsa",
		OrganismoRef: "organismo:uno", UnidadRef: "unidad:uno", FuenteOrganizacion: domain.FuenteSinVersionPlanB2{Ref: fuente.Ref, SHA256: h},
		FuentePlantilla: domain.InstrumentoPlanPersonalB2{Ref: "11111111-1111-4111-8111-111111111111", Revision: 1, FuenteRef: fuente.Ref, FuenteSHA256: h},
		FuenteRPT:       domain.InstrumentoPlanPersonalB2{Ref: "22222222-2222-4222-8222-222222222222", Revision: 1, FuenteRef: fuente.Ref, FuenteSHA256: h},
		VersionPlazaRef: "plantilla:uno", VersionPuestoRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 1,
		PuestoRef: "puesto:uno", PlazaRef: "plaza:uno", CatalogoRPTID: "catalogo:rpt", CatalogoRPTModulo: "personal",
		CatalogoRPTVersion: 1, CatalogoRPTSHA256: h, CategoriaRef: "categoria:uno", VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo",
		Regimen: domain.EntradaPlanPersonalB2{Ref: "regimen:uno", Version: 1}, Modalidad: domain.EntradaPlanPersonalB2{Ref: "modalidad:uno", Version: 1},
		ClaseOcupacion: "temporal", Desde: "2026-09-30", MotivoClave: "incorporacion", DocumentoRef: "documento:uno", DocumentoSHA256: h,
		EjercicioSintetico: true,
	}
	if err := p.Validar(); err != nil {
		t.Fatalf("plan de prueba inválido: %v", err)
	}
	s := ports.SolicitudPlanNominalB2{OrganizacionRef: p.OrganizacionRef, ExpedienteRef: p.ExpedienteRef,
		VersionExpediente: p.VersionExpediente, PuestoRef: p.PuestoRef, PlazaRef: p.PlazaRef,
		VersionPlantillaRef: p.VersionPlazaRef, VersionRPTRef: p.VersionPuestoRef,
		Regimen: p.Regimen, Modalidad: p.Modalidad, ClaseOcupacion: p.ClaseOcupacion,
		Desde: p.Desde, Hasta: p.Hasta, MotivoClave: p.MotivoClave, DocumentoRef: p.DocumentoRef, DocumentoSHA256: p.DocumentoSHA256,
		ClaveIdempotencia: "33333333-3333-4333-8333-333333333333"}
	if !validarSolicitudPlanB2(s) {
		t.Fatal("selección canónica rechazada")
	}
	s.Desde = "0000-09-30"
	if validarSolicitudPlanB2(s) {
		t.Fatal("año cero admitido")
	}
	s.Desde = p.Desde
	s.ClaseOcupacion = ""
	if validarSolicitudPlanB2(s) {
		t.Fatal("clase ausente admitida")
	}
	s.ClaseOcupacion = p.ClaseOcupacion
	if !planCoincideSeleccionB2(p, s) {
		t.Fatal("la selección válida rechazó versiones nativas distintas del UUID de instrumento")
	}
	s.VersionPlantillaRef = "plantilla:otra"
	if planCoincideSeleccionB2(p, s) {
		t.Fatal("versión de plaza ajena admitida")
	}
	s.VersionPlantillaRef = p.VersionPlazaRef
	s.VersionRPTRef = "rpt:otro"
	if planCoincideSeleccionB2(p, s) {
		t.Fatal("versión de puesto ajena admitida")
	}
}
