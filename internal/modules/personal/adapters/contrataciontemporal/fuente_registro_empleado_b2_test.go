package contrataciontemporal

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Dobles solo de prueba: el servicio B2 real debe volver a solicitar la
// atestación nominal; esto no acredita criptografía ni PostgreSQL.
type autorizadorFichaB2Prueba struct {
	t        *testing.T
	llamadas int
	denegado bool
}

func (a *autorizadorFichaB2Prueba) AutorizarConsultaRegistroEmpleadoB2(_ context.Context, m domain.MaterialConsultaRegistroEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	if a.denegado {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, domain.ErrRegistroEmpleadoB2Denegado
	}
	h, _ := m.HuellaSHA256()
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	r, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "contexto:prueba", strings.Repeat("c", 64), domain.AccionFichaEmpleadoB2, m.EmpleadoRef(), h, domain.AudienciaFichaEmpleadoB2, ahora, ahora.Add(3*time.Second))
	if err != nil {
		a.t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	actor := m.Actor()
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	x, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), r, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		a.t.Fatal(err)
	}
	return x, nil
}

type repositorioFichaB2Prueba struct {
	llamadas  int
	conHechos bool
}

func (r *repositorioFichaB2Prueba) ConsultarFichaRRHH(_ context.Context, o ports.OrdenFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	r.llamadas++
	x := o.Autorizacion.ResumenCapacidad()
	f := domain.FichaEmpleadoB2{EmpleadoRef: o.Material.EmpleadoRef(), OrganismoRef: o.Material.OrganismoRef(), PersonaRef: "per_" + strings.Repeat("p", 24), Corte: o.Material.Corte(), Version: 1}
	if r.conHechos {
		traza := domain.TrazaEmpleadoB2{Desde: "2026-09-01", RegistradaEn: f.Corte.ConocidoEn, Version: 1, ActoRef: "acto:uno", FuenteRef: "fuente:personal", FuenteVersion: 1}
		snapshot := func(tipo, ref string) *domain.SnapshotEntradaCatalogoEmpleadoB2 {
			return &domain.SnapshotEntradaCatalogoEmpleadoB2{OrganismoRef: f.OrganismoRef, Tipo: tipo, Ref: ref, Version: 1, Revision: 1, Denominacion: "Entrada sintética", HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: "2026-01-01", Estado: "publicada"}
		}
		relacionRef := "rel_" + strings.Repeat("r", 24)
		f.Relaciones = []domain.RelacionRegistroEmpleadoB2{{RelacionRef: relacionRef, OrganismoRef: f.OrganismoRef, UnidadRef: "unidad:uno", RegimenRef: "regimen:uno", ModalidadRef: "modalidad:uno", Estado: "vigente", Traza: traza, CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{Regimen: snapshot("regimen", "regimen:uno"), Modalidad: snapshot("modalidad", "modalidad:uno")}}}
		f.Ocupaciones = []domain.OcupacionEmpleadoB2{{OcupacionRef: "ocupacion:uno", RelacionRef: relacionRef, UnidadRef: "unidad:uno", PlazaRef: "plaza:uno", PuestoRef: "puesto:uno", ModalidadRef: "modalidad:uno", Clase: "temporal", Estado: "vigente", Traza: traza, CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{Modalidad: snapshot("modalidad", "modalidad:uno")}}}
	}
	e := ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "lectura:prueba", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}
	return ports.ResultadoFichaEmpleadoB2{Ficha: f, Evidencia: e}, nil
}
func (*repositorioFichaB2Prueba) ListarVacantesRRHH(context.Context, ports.OrdenVacantesB2) (ports.ResultadoVacantesB2, error) {
	return ports.ResultadoVacantesB2{}, errors.New("operación inesperada")
}

func solicitudFichaB2Prueba(t *testing.T) domain.SolicitudFichaEmpleadoB2 {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	instantanea := core.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := core.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return domain.SolicitudFichaEmpleadoB2{EmpleadoRef: "emp_" + strings.Repeat("e", 24), OrganismoRef: "organismo:dipgra", Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: ahora}, Actor: actor}
}

func TestFuenteRegistroEmpleadoB2RevalidaServicioPropietario(t *testing.T) {
	a := &autorizadorFichaB2Prueba{t: t}
	r := &repositorioFichaB2Prueba{}
	s, err := application.NuevoServicioRegistroEmpleadoB2(a, r)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NuevaFuenteRegistroEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	c := solicitudFichaB2Prueba(t)
	for i := 0; i < 2; i++ {
		v, err := f.ConsultarFicha(context.Background(), c)
		if err != nil || v.Ficha.EmpleadoRef != c.EmpleadoRef || v.Evidencia.EfectoRef != c.EmpleadoRef {
			t.Fatalf("consulta: %v", err)
		}
	}
	a.denegado = true
	if _, err := f.ConsultarFicha(context.Background(), c); !errors.Is(err, domain.ErrRegistroEmpleadoB2Denegado) {
		t.Fatal("autorización histórica reutilizada")
	}
	if a.llamadas != 3 || r.llamadas != 2 {
		t.Fatal("faltó autorización fresca o consultó tras denegación")
	}
	// La fuente tiene como consumidor el servicio que selecciona hechos;
	// esta ficha sin ocupación debe quedar pendiente, jamás como incorporación.
	consumidor, err := application.NuevoServicioConsultaIncorporacionCT(f)
	if err != nil {
		t.Fatal(err)
	}
	a.denegado = false
	x := ports.SeleccionHechosIncorporacionCT{EmpleadoRef: c.EmpleadoRef, PersonaRef: "per_" + strings.Repeat("p", 24), OrganismoRef: c.OrganismoRef, Corte: c.Corte, RelacionRef: "rel_" + strings.Repeat("r", 24), OcupacionRef: "ocupacion:uno", UnidadRef: "unidad:uno", PuestoRef: "puesto:uno", PlazaRef: "plaza:uno", VersionEmpleado: 1, VersionRelacion: 1, VersionOcupacion: 1}
	if _, err := consumidor.ConsultarHechosIncorporacionCT(context.Background(), ports.SolicitudHechosIncorporacionCT{Seleccion: x, Actor: c.Actor}); !errors.Is(err, domain.ErrRegistroEmpleadoB2Conflicto) {
		t.Fatalf("ficha sin hechos: %v", err)
	}
	r.conHechos = true
	v, err := consumidor.ConsultarHechosIncorporacionCT(context.Background(), ports.SolicitudHechosIncorporacionCT{Seleccion: x, Actor: c.Actor})
	if err != nil || v.Seleccion != x || v.Ocupacion.Hasta != "" || v.Evidencia.EfectoRef != x.EmpleadoRef || v.FirmaOficial || v.EficaciaAdministrativa {
		t.Fatalf("fuente real B2 → preparación mínima: %v", err)
	}
	if a.llamadas != 5 || r.llamadas != 4 {
		t.Fatal("faltó lectura nominal del consumidor")
	}
}

func TestFuenteRegistroEmpleadoB2NulaNoConsulta(t *testing.T) {
	if _, err := NuevaFuenteRegistroEmpleadoB2(nil); !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) {
		t.Fatal(err)
	}
	var f *FuenteRegistroEmpleadoB2
	if _, err := f.ConsultarFicha(context.Background(), domain.SolicitudFichaEmpleadoB2{}); !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) {
		t.Fatal(err)
	}
}
