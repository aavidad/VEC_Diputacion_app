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

func materialAltaB2Prueba(t *testing.T) domain.MaterialActoRegistroEmpleadoB2 {
	return materialAltaB2PruebaConOrganismo(t, "organismo:dipgra")
}

func materialAltaB2PruebaConOrganismo(t *testing.T, organismo string) domain.MaterialActoRegistroEmpleadoB2 {
	t.Helper()
	actor := ordenP(t).Material.Solicitud().Actor
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	persona := "per_" + strings.Repeat("a", 24)
	s := domain.SolicitudAltaEmpleadoB2{
		PersonaRef: persona, OrganismoRef: organismo, UnidadRef: "uni:prueba", RegimenRef: "reg:funcionario", ModalidadRef: "mod:interino", VigenteDesde: fecha, Actor: actor,
		Procedencia: domain.ProcedenciaActoEmpleadoB2{ActoRef: "acto:prueba", FuenteRef: "fuente:prueba", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64), IdempotenciaRef: "550e8400-e29b-41d4-a716-446655440000"},
	}
	m, err := domain.NuevoMaterialAltaEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func atestacionActoB2Prueba(t *testing.T, m domain.MaterialActoRegistroEmpleadoB2, accion, audiencia string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	actor := m.Actor()
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, m.Recurso().Referencia, h, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func reciboAltaB2Prueba(t *testing.T, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) []byte {
	t.Helper()
	instante := a.ResumenCapacidad().EmitidaEn().Add(time.Microsecond)
	r := ports.ResultadoAltaEmpleadoB2{Recibo: ports.ReciboActoRegistroEmpleadoB2{
		ReciboRef: "perrec_" + strings.Repeat("a", 32), EmpleadoRef: "emp_" + strings.Repeat("b", 24), RelacionRef: "rel_" + strings.Repeat("c", 24), ProyeccionRef: "pep_" + strings.Repeat("d", 24),
		Tipo: "alta", Version: 1, RegistradoEn: instante, DecisionRef: a.ResumenCapacidad().DecisionRef(), EfectoRef: a.ResumenCapacidad().EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:prueba",
	}, AccesoActual: ports.AccesoActualRegistroEmpleadoB2{DecisionRef: a.ResumenCapacidad().DecisionRef(), EfectoRef: a.ResumenCapacidad().EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: instante, EstadoReplay: "registrado"}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRegistroEmpleadoB2AltaConfirmaReciboNominal(t *testing.T) {
	m := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, m, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	tx := &txP{fila: filaP{vals: []any{reciboAltaB2Prueba(t, a)}}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	resultado, err := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: m, Autorizacion: a})
	if err != nil || resultado.Recibo.Tipo != "alta" || tx.commits != 1 || tx.q[1] != registrarEmpleadoB2SQL || len(tx.a[0]) != 11 {
		t.Fatal("alta nominal no confirmada", err)
	}
}

func TestRegistroEmpleadoB2AltaRevierteReciboAjeno(t *testing.T) {
	m := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, m, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	bruto := bytes.Replace(reciboAltaB2Prueba(t, a), []byte(`"decision_ref":"dec_prueba"`), []byte(`"decision_ref":"dec_ajena"`), 1)
	tx := &txP{fila: filaP{vals: []any{bruto}}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	_, err := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: m, Autorizacion: a})
	if !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("alta con recibo ajeno confirmada", err)
	}
}

func TestRegistroEmpleadoB2AltaRechazaFirmaInventada(t *testing.T) {
	m := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, m, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	base := reciboAltaB2Prueba(t, a)
	casos := map[string][]byte{
		"firma_verdadera": bytes.Replace(base, []byte(`"firma_oficial":false`), []byte(`"firma_oficial":true`), 1),
		"firma_omitida":   bytes.Replace(base, []byte(`,"firma_oficial":false`), nil, 1),
	}
	for nombre, bruto := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{bruto}}}
			r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
			_, err := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: m, Autorizacion: a})
			if !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("firma inventada confirmada", err)
			}
		})
	}
}

func TestRegistroEmpleadoB2AltaTraduceColision(t *testing.T) {
	m := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, m, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	tx := &txP{errQ: &pgconn.PgError{Code: "23505", Message: "detalle privado"}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	_, err := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: m, Autorizacion: a})
	if !errors.Is(err, domain.ErrRegistroEmpleadoB2Conflicto) || tx.commits != 0 || tx.rollbacks != 1 || strings.Contains(err.Error(), "privado") {
		t.Fatal("colisión no traducida", err)
	}
}

func TestRegistroEmpleadoB2AltaNoEnviaOrganismoAjeno(t *testing.T) {
	materialConcedido := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, materialConcedido, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	materialAjeno := materialAltaB2PruebaConOrganismo(t, "organismo:ajeno")
	pool := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool)
	_, err := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: materialAjeno, Autorizacion: a})
	if !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) || pool.n != 0 {
		t.Fatal("atestación de alta de otro organismo llegó a SQL", err)
	}
}

func TestRegistroEmpleadoB2AltaObjetivoNoAcreditadoDeniegaSinDetalle(t *testing.T) {
	m := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, m, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	for _, caso := range []string{"ausente", "revocada", "caducada"} {
		t.Run(caso, func(t *testing.T) {
			tx := &txP{errQ: &pgconn.PgError{Code: "P0002", Message: "persona " + caso + " con identificador privado"}}
			r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
			_, err := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: m, Autorizacion: a})
			if !errors.Is(err, domain.ErrRegistroEmpleadoB2Denegado) || err.Error() != domain.ErrRegistroEmpleadoB2Denegado.Error() || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("acreditación B1 revelada o confirmada", err)
			}
		})
	}
}

func TestRegistroEmpleadoB2ReplayConservaFechaOriginal(t *testing.T) {
	m := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, m, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	var respuesta ports.ResultadoAltaEmpleadoB2
	if err := json.Unmarshal(reciboAltaB2Prueba(t, a), &respuesta); err != nil {
		t.Fatal(err)
	}
	respuesta.AccesoActual.EstadoReplay = "replay"
	respuesta.Recibo.RegistradoEn = a.ResumenCapacidad().EmitidaEn().Add(-time.Hour)
	respuesta.Recibo.DecisionRef = "dec_original"
	respuesta.Recibo.ConsumoHuellaSHA256 = strings.Repeat("f", 64)
	respuesta.Recibo.AuditoriaRef = "auditoria:original"
	bruto, _ := json.Marshal(respuesta)
	tx := &txP{fila: filaP{vals: []any{bruto}}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	resultado, err := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: m, Autorizacion: a})
	if err != nil || resultado.AccesoActual.EstadoReplay != "replay" || !resultado.Recibo.RegistradoEn.Equal(respuesta.Recibo.RegistradoEn) || resultado.Recibo.DecisionRef != "dec_original" || resultado.AccesoActual.DecisionRef != a.ResumenCapacidad().DecisionRef() || tx.commits != 1 {
		t.Fatal("replay no conservó recibo con consumo actual", err)
	}
}

func TestRegistroEmpleadoB2HechoUsaRelacionExplicita(t *testing.T) {
	actor := ordenP(t).Material.Solicitud().Actor
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	empleado := "emp_" + strings.Repeat("b", 24)
	relacion := "rel_" + strings.Repeat("c", 24)
	s := domain.SolicitudHechoEmpleadoB2{
		Tipo: "situacion", EmpleadoRef: empleado, OrganismoRef: "organismo:dipgra", RelacionRef: relacion, RevisionEsperada: 1, RelacionVersionEsperada: 1, ClaseRef: "sit:servicio_activo", VigenteDesde: fecha, Actor: actor,
		Procedencia: domain.ProcedenciaActoEmpleadoB2{ActoRef: "acto:prueba", FuenteRef: "fuente:prueba", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64), IdempotenciaRef: "550e8400-e29b-41d4-a716-446655440000"},
	}
	m, err := domain.NuevoMaterialHechoEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	a := atestacionActoB2Prueba(t, m, domain.AccionHechoEmpleadoB2, domain.AudienciaHechoEmpleadoB2)
	instante := a.ResumenCapacidad().EmitidaEn().Add(time.Microsecond)
	bruto, _ := json.Marshal(ports.ResultadoHechoEmpleadoB2{Recibo: ports.ReciboActoRegistroEmpleadoB2{
		ReciboRef: "perrec_" + strings.Repeat("a", 32), EmpleadoRef: empleado, RelacionRef: relacion, HechoRef: "sit_" + strings.Repeat("d", 24), Tipo: "situacion", Version: 1, RegistradoEn: instante, DecisionRef: a.ResumenCapacidad().DecisionRef(), EfectoRef: a.ResumenCapacidad().EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:prueba",
	}, AccesoActual: ports.AccesoActualRegistroEmpleadoB2{DecisionRef: a.ResumenCapacidad().DecisionRef(), EfectoRef: a.ResumenCapacidad().EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: instante, EstadoReplay: "registrado"}})
	tx := &txP{fila: filaP{vals: []any{bruto}}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	resultado, err := r.RegistrarHechoEmpleadoRRHH(context.Background(), ports.OrdenHechoEmpleadoB2{Material: m, Autorizacion: a})
	if err != nil || resultado.Recibo.RelacionRef != relacion || tx.commits != 1 || tx.q[1] != registrarHechoEmpleadoB2SQL {
		t.Fatal("hecho sin relación explícita", err)
	}
}
