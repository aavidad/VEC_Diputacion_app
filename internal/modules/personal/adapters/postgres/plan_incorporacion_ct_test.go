package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenPlanCTPrueba(t *testing.T) ports.OrdenPlanIncorporacionCT {
	actor := materialAltaB2Prueba(t).Actor()
	m, e := domain.NuevoMaterialConsultarPlanIncorporacionCT(ports.ConsultaPlanIncorporacionCT{PlanRef: "perplan_prueba", OrganismoRef: "org:uno", Actor: actor}, "consultar")
	if e != nil {
		t.Fatal(e)
	}
	h, _ := m.HuellaSHA256()
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	resumen, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion(), m.PlanRef(), h, domain.AudienciaPlanIncorporacionCT, ahora, ahora.Add(3*time.Second))
	if e != nil {
		t.Fatal(e)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	a, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if e != nil {
		t.Fatal(e)
	}
	return ports.OrdenPlanIncorporacionCT{Material: m, Autorizacion: a}
}
func estadoPlanCTPrueba(t *testing.T, o ports.OrdenPlanIncorporacionCT) ports.EstadoPlanIncorporacionCT {
	d := domain.DatosPlanIncorporacionCT{IdempotenciaRef: "10000000-0000-4000-8000-000000000001", OrigenCTRef: "ct:plan", OrigenCTReciboRef: "ct:recibo", OrigenCTHuellaSHA256: strings.Repeat("a", 64), ExpedienteRef: "exp:uno", ExpedienteVersion: 7, OrganismoRef: "org:uno", UnidadRef: "uni:uno", PersonaRef: "per_" + strings.Repeat("p", 24), PersonaVersion: 1, FuenteBolsaRef: "bolsa:persona", FuenteBolsaVersion: 2, FuenteBolsaReciboRef: "bolsa:recibo", FuenteBolsaHuellaSHA256: strings.Repeat("b", 64), Regimen: domain.EntradaCatalogoEmpleadoB2{Ref: "reg:uno", Version: 1}, Modalidad: domain.EntradaCatalogoEmpleadoB2{Ref: "mod:uno", Version: 2}, Desde: domain.FechaCivil("2026-10-01"), PlazaRef: "plaza:10000000-0000-4000-8000-000000000002", PuestoRef: "puesto:10000000-0000-4000-8000-000000000003", ClaseOcupacion: "temporal", VersionPlantillaRef: "plantilla:uno", VersionRPTRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 2, FuenteOrganizacionRef: "organizacion:uno", FuenteOrganizacionHuellaSHA256: strings.Repeat("c", 64), CatalogoRPTID: "rpt:catalogo", CatalogoRPTModulo: "contrataciontemporal", CatalogoRPTCategoria: "categoria:uno", CatalogoRPTVersion: 1, CatalogoRPTHuellaSHA256: strings.Repeat("d", 64), VinculoCTReciboRef: "vinculo:recibo", Procedencia: domain.ProcedenciaActoEmpleadoB2{ActoRef: "acto:incorporacion", FuenteRef: "fuente:ct", FuenteVersion: 7, FuenteHuellaSHA256: strings.Repeat("e", 64), IdempotenciaRef: "10000000-0000-4000-8000-000000000004"}}
	p := ports.PlanIncorporacionCT{ClasesOcupacionCatalogoRef: "personal:clases_ocupacion_ct", ClasesOcupacionCatalogoVersion: 1, ClasesOcupacionCatalogoHuellaSHA256: strings.Repeat("f", 64), PlanRef: o.Material.PlanRef(), ReciboRef: "perplanrec_prueba", Version: 1, Datos: d, Modo: "alta_empleado", ClaveAltaRelacion: "10000000-0000-4000-8000-000000000005", ClaveOcupacion: "10000000-0000-4000-8000-000000000006", UsoRPTRef: "uso:uno", ReservaRPTRef: "reserva:uno", ConfirmacionRPTRef: "confirmacion:uno"}
	p.HuellaSHA256 = p.CalcularHuellaSHA256()
	if e := p.Validar(); e != nil {
		t.Fatal(e)
	}
	x := o.Autorizacion.ResumenCapacidad()
	return ports.EstadoPlanIncorporacionCT{Plan: p, Estado: "preparado", Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "perplanacc_prueba", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "audit:actual", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}}
}
func TestPlanCTPostgresConfirmaSoloRespuestaLigada(t *testing.T) {
	o := ordenPlanCTPrueba(t)
	s := estadoPlanCTPrueba(t, o)
	b, _ := json.Marshal(s)
	tx := &txP{fila: filaP{vals: []any{b}}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	got, e := r.ConsultarPlan(context.Background(), o)
	if e != nil || got.Plan.PlanRef != s.Plan.PlanRef || tx.commits != 1 || tx.q[1] != planIncorporacionCTSQL || len(tx.a[0]) != 11 {
		t.Fatal("transaccion plan", e)
	}
}
func TestPlanCTPostgresSalidaCorruptaRevierte(t *testing.T) {
	for _, modo := range []string{"sha", "recibo_ajeno", "contexto", "duplicado", "estado"} {
		t.Run(modo, func(t *testing.T) {
			o := ordenPlanCTPrueba(t)
			s := estadoPlanCTPrueba(t, o)
			switch modo {
			case "sha":
				s.Plan.ClaveAltaRelacion = "10000000-0000-4000-8000-000000000007"
			case "recibo_ajeno":
				s.ReciboAltaRelacion = &ports.ReciboActoRegistroEmpleadoB2{ReciboRef: "perrec_" + strings.Repeat("a", 32)}
			case "contexto":
				s.Evidencia.DecisionRef = "dec_otro"
			case "estado":
				s.Estado = "ejecutado"
			}
			b, _ := json.Marshal(s)
			if modo == "duplicado" {
				b = bytes.Replace(b, []byte(`"estado":"preparado"`), []byte(`"estado":"preparado","estado":"ejecutado"`), 1)
			}
			tx := &txP{fila: filaP{vals: []any{b}}}
			r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
			if _, e := r.ConsultarPlan(context.Background(), o); !errors.Is(e, domain.ErrRegistroEmpleadoB2NoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("salida insegura confirma", e)
			}
		})
	}
}
func TestPlanCTPostgresDenegacionNoEsDependencia(t *testing.T) {
	for _, caso := range []struct {
		codigo   string
		esperado error
	}{{"42501", domain.ErrRegistroEmpleadoB2Denegado}, {"55000", domain.ErrRegistroEmpleadoB2NoDisponible}, {"P7404", domain.ErrRegistroEmpleadoB2NoEncontrado}, {"23505", domain.ErrRegistroEmpleadoB2Conflicto}} {
		o := ordenPlanCTPrueba(t)
		tx := &txP{fila: filaP{err: &pgconn.PgError{Code: caso.codigo}}}
		r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
		if _, e := r.ConsultarPlan(context.Background(), o); !errors.Is(e, caso.esperado) || tx.commits != 0 || tx.rollbacks != 1 {
			t.Fatal(caso.codigo, e)
		}
	}
}
