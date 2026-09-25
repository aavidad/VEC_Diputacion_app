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

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenEmpleadosB2Prueba(t *testing.T) ports.OrdenEmpleadosB2 {
	t.Helper()
	actor := ordenP(t).Material.Solicitud().Actor
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	instante := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	m, err := domain.NuevoMaterialEmpleadosB2(domain.SolicitudEmpleadosB2{OrganismoRef: "org:dipgra", Corte: domain.CorteEmpleadoB2{VigenteEn: fecha, ConocidoEn: instante}, Limite: 25, Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	h, _ := m.HuellaSHA256()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionEmpleadosB2, m.Recurso().Referencia, h, domain.AudienciaEmpleadosB2, instante, instante.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenEmpleadosB2{Material: m, Autorizacion: a}
}

func respuestaEmpleadosB2Prueba(t *testing.T, o ports.OrdenEmpleadosB2) []byte {
	t.Helper()
	m := o.Material
	desde, _ := domain.NuevaFechaCivil("2026-01-01")
	r := ports.ResultadoEmpleadosB2{
		Pagina: domain.PaginaEmpleadosB2{OrganismoRef: m.OrganismoRef(), Corte: m.Corte(), Limite: m.Limite(), Empleados: []domain.EmpleadoOrganismoB2{
			{EmpleadoRef: "emp_" + strings.Repeat("a", 24), Relaciones: []domain.RelacionVigenteEmpleadoB2{{
				RelacionRef: "rel_" + strings.Repeat("a", 24), Estado: "vigente", UnidadRef: "uni:prueba",
				UnidadDenominacion: "Servicio sintético", PuestoDenominacion: "Técnico sintético", RegimenDenominacion: "Funcionario", ModalidadDenominacion: "Interino",
				Traza: domain.TrazaEmpleadoB2{Desde: desde, RegistradaEn: m.Corte().ConocidoEn.Add(-time.Hour), Version: 1, ActoRef: "acto:prueba", FuenteRef: "fuente:prueba", FuenteVersion: 1},
			}}},
			{EmpleadoRef: "emp_" + strings.Repeat("b", 24), Relaciones: []domain.RelacionVigenteEmpleadoB2{}},
		}},
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:prueba", DecisionRef: o.Autorizacion.ResumenCapacidad().DecisionRef(), EfectoRef: m.OrganismoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: m.Corte().ConocidoEn.Add(time.Microsecond)},
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRegistroEmpleadoB2ListaEmpleadosNominal(t *testing.T) {
	o := ordenEmpleadosB2Prueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaEmpleadosB2Prueba(t, o)}}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	resultado, err := r.ListarEmpleadosRRHH(context.Background(), o)
	if err != nil || len(resultado.Pagina.Empleados) != 2 || tx.commits != 1 || len(tx.a[0]) != 11 || tx.q[1] != consultaEmpleadosB2SQL {
		t.Fatal("lista de empleados no confirmada", err)
	}
}

func TestRegistroEmpleadoB2ListaRevierteFormaAjena(t *testing.T) {
	o := ordenEmpleadosB2Prueba(t)
	base := respuestaEmpleadosB2Prueba(t, o)
	casos := map[string][]byte{
		"persona_filtrada":  bytes.Replace(base, []byte(`{"empleado_ref":"emp_aaa`), []byte(`{"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaa","empleado_ref":"emp_aaa`), 1),
		"relaciones_nulas":  bytes.Replace(base, []byte(`"relaciones":[]`), []byte(`"relaciones":null`), 1),
		"orden_invertido":   bytes.Replace(bytes.Replace(base, []byte("emp_"+strings.Repeat("a", 24)), []byte("emp_"+strings.Repeat("c", 24)), 1), []byte("emp_"+strings.Repeat("b", 24)), []byte("emp_"+strings.Repeat("a", 24)), 1),
		"relacion_extinta":  bytes.Replace(base, []byte(`"estado":"vigente"`), []byte(`"estado":"finalizada"`), 1),
		"organismo_ajeno":   bytes.Replace(base, []byte(`"organismo_ref":"org:dipgra"`), []byte(`"organismo_ref":"org:ajeno"`), 1),
		"clave_desconocida": bytes.Replace(base, []byte(`"pagina":{`), []byte(`"pagina":{"total":2,`), 1),
		"control_en_texto":  bytes.Replace(base, []byte(`Servicio sintético`), []byte(`Servicio\u0007`), 1),
	}
	for nombre, bruto := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{bruto}}}
			r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
			_, err := r.ListarEmpleadosRRHH(context.Background(), o)
			if !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("respuesta alterada confirmada", err)
			}
		})
	}
}

func TestRegistroEmpleadoB2ListaNoEnviaAtestacionDeOtraOperacion(t *testing.T) {
	o := ordenEmpleadosB2Prueba(t)
	o.Autorizacion = ordenVacantesB2Prueba(t).Autorizacion
	p := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(p)
	if _, err := r.ListarEmpleadosRRHH(context.Background(), o); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) || p.n != 0 {
		t.Fatal("atestación de vacantes llegó a la lista", err)
	}
}
