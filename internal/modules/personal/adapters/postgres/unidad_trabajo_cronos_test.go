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

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenUnidadCronosPrueba(t *testing.T) ports.OrdenUnidadTrabajoCronos {
	t.Helper()
	b := ordenFichaPropiaPrueba(t)
	m, err := domain.NuevoMaterialUnidadTrabajoCronos(b.Material.Actor(), b.Material.Corte().VigenteEn, b.Material.Corte().ConocidoEn)
	if err != nil {
		t.Fatal(err)
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	actor := m.Actor()
	x, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_unidad", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_unidad", strings.Repeat("c", 64), ports.AccionUnidadTrabajoPropiaCronos, m.EmpleadoRef(), h, ports.AudienciaUnidadTrabajoPropiaCronos, actor.ResueltoEn, actor.ResueltoEn.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), x, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenUnidadTrabajoCronos{Material: m, Autorizacion: a}
}

func respuestaUnidadCronosPrueba(t *testing.T, o ports.OrdenUnidadTrabajoCronos) []byte {
	t.Helper()
	x := o.Autorizacion.ResumenCapacidad()
	huella := strings.Repeat("d", 64)
	b, err := json.Marshal(ports.ResultadoUnidadTrabajoPropiaCronos{Cobertura: "completa",
		Unidad:    &ports.UnidadTrabajoCronos{EmpleadoRef: o.Material.EmpleadoRef(), RelacionRef: "rel_" + strings.Repeat("r", 24), OrganismoRef: "org_prueba", UnidadClave: "unidad_prueba", RegimenRef: "regimen:prueba", Desde: "2026-01-01", FuenteRef: "fuente:personal", FuenteVersionRef: "2", Version: 3},
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "unidadcronos:01234567-89ab-cdef-0123-456789abcdef", DecisionRef: x.DecisionRef(), EfectoRef: o.Material.EmpleadoRef(), ConsumoHuellaSHA256: huella, AuditoriaRef: "aud_v3_" + huella[:32], ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestUnidadTrabajoCronosPostgresConfirmaLecturaNominal(t *testing.T) {
	o := ordenUnidadCronosPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaUnidadCronosPrueba(t, o)}}}
	pool := &poolP{tx: tx}
	r, _ := nuevoRepositorioUnidadTrabajoCronos(pool)
	got, err := r.ConsultarUnidadTrabajoPropiaCronos(context.Background(), o)
	if err != nil || got.Unidad == nil || got.Unidad.UnidadClave != "unidad_prueba" || pool.o.IsoLevel != pgx.Serializable || pool.o.AccessMode != pgx.ReadWrite || tx.commits != 1 || tx.rollbacks != 0 || tx.q[1] != consultaUnidadTrabajoCronosSQL || len(tx.a[0]) != 11 {
		t.Fatal("lectura nominal no confirmada", err)
	}
}

func TestUnidadTrabajoCronosPostgresRevierteRespuestaIncoherente(t *testing.T) {
	o := ordenUnidadCronosPrueba(t)
	base := respuestaUnidadCronosPrueba(t, o)
	for nombre, bruto := range map[string][]byte{
		"unidad_ajena": bytes.Replace(base, []byte(o.Material.EmpleadoRef()), []byte("emp_"+strings.Repeat("x", 24)), 1),
		"sin_unidad":   bytes.Replace(base, []byte(`"cobertura":"completa"`), []byte(`"cobertura":"no_acreditada"`), 1),
		"campo_extra":  bytes.Replace(base, []byte(`"unidad":{`), []byte(`"unidad":{"dni":"ficticio",`), 1),
		"sin_recibo":   bytes.Replace(base, []byte(`"recibo_ref":"unidadcronos:01234567-89ab-cdef-0123-456789abcdef"`), []byte(`"recibo_ref":""`), 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{bruto}}}
			r, _ := nuevoRepositorioUnidadTrabajoCronos(&poolP{tx: tx})
			_, err := r.ConsultarUnidadTrabajoPropiaCronos(context.Background(), o)
			if !errors.Is(err, domain.ErrUnidadTrabajoCronosNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("respuesta divergente confirmada", err)
			}
		})
	}
}
