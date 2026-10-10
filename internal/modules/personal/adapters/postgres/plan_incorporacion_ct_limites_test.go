package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ordenPlanCTConMaterialPrueba construye la orden del plan con su atestación
// VEC-AD-3 ligada al material concreto que recibe (operación, recurso y huella).
func ordenPlanCTConMaterialPrueba(t *testing.T, m domain.MaterialPlanIncorporacionCT) ports.OrdenPlanIncorporacionCT {
	t.Helper()
	actor := m.Actor()
	h, e := m.HuellaSHA256()
	if e != nil {
		t.Fatal(e)
	}
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	resumen, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion(), m.Recurso().Referencia, h, domain.AudienciaPlanIncorporacionCT, ahora, ahora.Add(3*time.Second))
	if e != nil {
		t.Fatal(e)
	}
	canon, e := actor.RepresentacionCanonicaVinculadaV2()
	if e != nil {
		t.Fatal(e)
	}
	raiz, e := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if e != nil {
		t.Fatal(e)
	}
	a, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if e != nil {
		t.Fatal(e)
	}
	return ports.OrdenPlanIncorporacionCT{Material: m, Autorizacion: a}
}

// evidenciaPlanCTPrueba construye la evidencia coherente con la capacidad de la
// atestación recibida, como la devuelve la función SQL nominal.
func evidenciaPlanCTPrueba(a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) ports.EvidenciaRegistroEmpleadoB2 {
	x := a.ResumenCapacidad()
	return ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "perplanacc_prueba", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "audit:actual", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}
}

// brutoEstadoPlanCTPrueba devuelve la respuesta nominal del plan con la
// evidencia ligada a la orden recibida. En preparar la referencia del plan la
// asigna SQL y el adaptador solo coteja la huella de los datos del material.
func brutoEstadoPlanCTPrueba(t *testing.T, o ports.OrdenPlanIncorporacionCT) []byte {
	t.Helper()
	s := estadoPlanCTPrueba(t, ordenPlanCTPrueba(t))
	x := o.Autorizacion.ResumenCapacidad()
	s.Evidencia.DecisionRef = x.DecisionRef()
	s.Evidencia.EfectoRef = x.EfectoRef()
	s.Evidencia.ConsultadaEn = x.EmitidaEn().Add(time.Microsecond)
	b, e := json.Marshal(s)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

// TestPlanCTPostgresCadaTransaccionFijaLimitesAntesDeConsultar fija que toda
// transacción del plan de incorporación (preparar, consultar, ejecutar,
// confirmar, seleccionar, clases de ocupación y sus actos de alta y hecho)
// emite como primera sentencia los dos topes que el núcleo V3 exige para el
// consumo VEC-AD-3 —statement_timeout de hasta 15 s e
// idle_in_transaction_session_timeout de hasta 20 s—, antes de su primera
// consulta. Sin ellos el núcleo rechaza con 22023 «límites VEC-AD-3 ausentes».
// La función común ejecutarRegistroEmpleadoB2 los aplica una sola vez en el
// punto por el que pasan todas las transacciones del plan.
func TestPlanCTPostgresCadaTransaccionFijaLimitesAntesDeConsultar(t *testing.T) {
	actor := materialAltaB2Prueba(t).Actor()
	datos := estadoPlanCTPrueba(t, ordenPlanCTPrueba(t)).Plan.Datos
	desde, e := domain.NuevaFechaCivil("2026-10-01")
	if e != nil {
		t.Fatal(e)
	}
	plaza := "plaza:10000000-0000-4000-8000-000000000002"
	puesto := "puesto:10000000-0000-4000-8000-000000000003"
	nuevoPlan := func(op string) domain.MaterialPlanIncorporacionCT {
		m, e := domain.NuevoMaterialConsultarPlanIncorporacionCT(ports.ConsultaPlanIncorporacionCT{PlanRef: "perplan_prueba", OrganismoRef: "org:uno", Actor: actor}, op)
		if e != nil {
			t.Fatal(e)
		}
		return m
	}
	mPreparar, e := domain.NuevoMaterialPrepararPlanIncorporacionCT(ports.SolicitudPlanIncorporacionCT{DatosPlanIncorporacionCT: datos, Actor: actor})
	if e != nil {
		t.Fatal(e)
	}
	mSeleccion, e := domain.NuevoMaterialSeleccionPlanIncorporacionCT(ports.SeleccionPlanIncorporacionCT{SelectorOrganizacionPlanCT: domain.SelectorOrganizacionPlanCT{PlazaRef: plaza, PuestoRef: puesto, Desde: desde}, OrganismoRef: "org:uno", Actor: actor})
	if e != nil {
		t.Fatal(e)
	}
	mClases, e := domain.NuevoMaterialClasesOcupacionCT(ports.ConsultaClasesOcupacionCT{OrganismoRef: "org:uno", Actor: actor})
	if e != nil {
		t.Fatal(e)
	}
	mAlta := materialAltaB2Prueba(t)
	aAlta := atestacionActoB2Prueba(t, mAlta, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	fecha, e := domain.NuevaFechaCivil("2026-09-20")
	if e != nil {
		t.Fatal(e)
	}
	empleado := "emp_" + strings.Repeat("b", 24)
	relacion := "rel_" + strings.Repeat("c", 24)
	mHecho, e := domain.NuevoMaterialHechoEmpleadoB2(domain.SolicitudHechoEmpleadoB2{
		Tipo: "situacion", EmpleadoRef: empleado, OrganismoRef: "organismo:dipgra", RelacionRef: relacion, RevisionEsperada: 1, RelacionVersionEsperada: 1,
		Situacion: domain.EntradaCatalogoEmpleadoB2{Ref: "sit:servicio_activo", Version: 1}, Estado: "vigente", VigenteDesde: fecha, Actor: actor,
		Procedencia: domain.ProcedenciaActoEmpleadoB2{ActoRef: "acto:prueba", FuenteRef: "fuente:prueba", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64), IdempotenciaRef: "550e8400-e29b-41d4-a716-446655440000"},
	})
	if e != nil {
		t.Fatal(e)
	}
	aHecho := atestacionActoB2Prueba(t, mHecho, domain.AccionHechoEmpleadoB2, domain.AudienciaHechoEmpleadoB2)
	xHecho := aHecho.ResumenCapacidad()
	instante := xHecho.EmitidaEn().Add(time.Microsecond)
	brutoHecho, e := json.Marshal(ports.ResultadoHechoEmpleadoB2{Recibo: ports.ReciboActoRegistroEmpleadoB2{
		ReciboRef: "perrec_" + strings.Repeat("a", 32), EmpleadoRef: empleado, RelacionRef: relacion, HechoRef: "sit_" + strings.Repeat("d", 24), Tipo: "situacion", Version: 1, RegistradoEn: instante, DecisionRef: xHecho.DecisionRef(), EfectoRef: xHecho.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:prueba",
	}, AccesoActual: ports.AccesoActualRegistroEmpleadoB2{DecisionRef: xHecho.DecisionRef(), EfectoRef: xHecho.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: instante, EstadoReplay: "registrado"}})
	if e != nil {
		t.Fatal(e)
	}
	seleccion := domain.SeleccionOrganizacionPlanCT{PlantillaFuenteRef: "plantilla:fuente", RPTFuenteRef: "rpt:fuente", RevisionPlantilla: 1, RevisionRPT: 2, UnidadRef: "uni:uno", OrganismoRef: "org:uno", PlazaRef: plaza, PuestoRef: puesto, Desde: desde, RevisionPlaza: 1, RevisionPuesto: 2, VersionPlantillaRef: "plantilla:uno", VersionRPTRef: "rpt:uno", PlantillaHuellaSHA256: strings.Repeat("a", 64), RPTHuellaSHA256: strings.Repeat("b", 64), FuenteOrganizacionRef: "organizacion:uno", FuenteOrganizacionHuellaSHA256: strings.Repeat("c", 64)}
	clases := domain.CatalogoClasesOcupacionCT{Ref: "personal:clases_ocupacion_ct", Version: 1, HuellaSHA256: strings.Repeat("f", 64), Opciones: []domain.OpcionClaseOcupacionCT{{Valor: "temporal", TextoClave: "personal.clases_ocupacion.temporal"}}}
	llamarPlan := func(m domain.MaterialPlanIncorporacionCT, llamar func(*testing.T, *RepositorioRegistroEmpleadoB2PostgreSQL, ports.OrdenPlanIncorporacionCT) error) func(*testing.T, *poolP) error {
		return func(t *testing.T, pool *poolP) error {
			r, e := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool)
			if e != nil {
				return e
			}
			return llamar(t, r, ordenPlanCTConMaterialPrueba(t, m))
		}
	}
	llamarActo := func(llamar func(*RepositorioActosPlanIncorporacionCTPostgreSQL) error) func(*testing.T, *poolP) error {
		return func(_ *testing.T, pool *poolP) error {
			r, e := nuevoRepositorioActosPlanCT(pool)
			if e != nil {
				return e
			}
			return llamar(r)
		}
	}
	casos := []struct {
		nombre string
		sql    string
		bruto  []byte
		llamar func(*testing.T, *poolP) error
	}{
		{"preparar", planIncorporacionCTSQL, brutoEstadoPlanCTPrueba(t, ordenPlanCTConMaterialPrueba(t, mPreparar)), llamarPlan(mPreparar, func(t *testing.T, r *RepositorioRegistroEmpleadoB2PostgreSQL, o ports.OrdenPlanIncorporacionCT) error {
			_, e := r.PrepararPlan(context.Background(), o)
			return e
		})},
		{"consultar", planIncorporacionCTSQL, brutoEstadoPlanCTPrueba(t, ordenPlanCTConMaterialPrueba(t, nuevoPlan("consultar"))), llamarPlan(nuevoPlan("consultar"), func(t *testing.T, r *RepositorioRegistroEmpleadoB2PostgreSQL, o ports.OrdenPlanIncorporacionCT) error {
			_, e := r.ConsultarPlan(context.Background(), o)
			return e
		})},
		{"ejecutar", planIncorporacionCTSQL, brutoEstadoPlanCTPrueba(t, ordenPlanCTConMaterialPrueba(t, nuevoPlan("ejecutar"))), llamarPlan(nuevoPlan("ejecutar"), func(t *testing.T, r *RepositorioRegistroEmpleadoB2PostgreSQL, o ports.OrdenPlanIncorporacionCT) error {
			_, e := r.ConsultarPlan(context.Background(), o)
			return e
		})},
		{"confirmar", planIncorporacionCTSQL, brutoEstadoPlanCTPrueba(t, ordenPlanCTConMaterialPrueba(t, nuevoPlan("confirmar"))), llamarPlan(nuevoPlan("confirmar"), func(t *testing.T, r *RepositorioRegistroEmpleadoB2PostgreSQL, o ports.OrdenPlanIncorporacionCT) error {
			_, e := r.ConfirmarPlan(context.Background(), o)
			return e
		})},
		{"seleccionar", planIncorporacionCTSQL, func() []byte {
			b, e := json.Marshal(ports.ResultadoSeleccionPlanIncorporacionCT{Seleccion: seleccion, Evidencia: evidenciaPlanCTPrueba(ordenPlanCTConMaterialPrueba(t, mSeleccion).Autorizacion)})
			if e != nil {
				t.Fatal(e)
			}
			return b
		}(), llamarPlan(mSeleccion, func(t *testing.T, r *RepositorioRegistroEmpleadoB2PostgreSQL, o ports.OrdenPlanIncorporacionCT) error {
			_, e := r.ResolverSeleccion(context.Background(), o)
			return e
		})},
		{"clases_ocupacion", planIncorporacionCTSQL, func() []byte {
			b, e := json.Marshal(ports.ResultadoClasesOcupacionCT{Catalogo: clases, Evidencia: evidenciaPlanCTPrueba(ordenPlanCTConMaterialPrueba(t, mClases).Autorizacion)})
			if e != nil {
				t.Fatal(e)
			}
			return b
		}(), llamarPlan(mClases, func(t *testing.T, r *RepositorioRegistroEmpleadoB2PostgreSQL, o ports.OrdenPlanIncorporacionCT) error {
			_, e := r.ConsultarClasesOcupacion(context.Background(), o)
			return e
		})},
		{"acto_alta", registrarAltaPlanCTSQL, reciboAltaB2Prueba(t, aAlta), llamarActo(func(r *RepositorioActosPlanIncorporacionCTPostgreSQL) error {
			_, e := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: mAlta, Autorizacion: aAlta})
			return e
		})},
		{"acto_hecho", registrarHechoPlanCTSQL, brutoHecho, llamarActo(func(r *RepositorioActosPlanIncorporacionCTPostgreSQL) error {
			_, e := r.RegistrarHechoEmpleadoRRHH(context.Background(), ports.OrdenHechoEmpleadoB2{Material: mHecho, Autorizacion: aHecho})
			return e
		})},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{c.bruto}}}
			e := c.llamar(t, &poolP{tx: tx})
			if e != nil {
				t.Fatalf("%s: recorrido nominal fallado: %v", c.nombre, e)
			}
			if len(tx.q) != 2 || tx.q[1] != c.sql {
				t.Fatalf("%s: la consulta nominal no llega justo después de los límites: %v", c.nombre, tx.q)
			}
			if !strings.Contains(tx.q[0], "set_config('statement_timeout','15s',true)") || !strings.Contains(tx.q[0], "set_config('idle_in_transaction_session_timeout','20s',true)") {
				t.Fatalf("%s: la transacción no fija statement_timeout e idle_in_transaction_session_timeout antes de consultar: %v", c.nombre, tx.q[0])
			}
			if len(tx.a) != 1 || len(tx.a[0]) != 11 || tx.commits != 1 {
				t.Fatalf("%s: la primera consulta está incompleta o la transacción no se confirmó", c.nombre)
			}
		})
	}
}
