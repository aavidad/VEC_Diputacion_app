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
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenFichaPropiaPrueba(t *testing.T) ports.OrdenFichaPropia {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "pep_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	m, err := domain.NuevoMaterialFichaPropia(domain.SolicitudFichaPropia{Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: ahora.Add(-time.Second)}, Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	h, _ := m.HuellaSHA256()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionFichaPropia, m.EmpleadoRef(), h, domain.AudienciaFichaPropia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, 1, 1, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenFichaPropia{Material: m, Autorizacion: a}
}

func respuestaFichaPropiaPrueba(t *testing.T, o ports.OrdenFichaPropia) []byte {
	t.Helper()
	consultada := o.Autorizacion.ResumenCapacidad().EmitidaEn().Add(time.Microsecond).Format("2006-01-02T15:04:05.000000Z")
	return []byte(`{"ficha":{"corte":{"vigente_en":"2026-09-25","conocido_en":"` + o.Material.Corte().ConocidoEn.Format("2006-01-02T15:04:05.000000Z") + `"},` +
		`"relaciones":[{"inicio":"2026-01-01","fin":"","estado":"vigente","regimen":"Funcionario interino","modalidad":"Vacante","unidad":"Servicio de Personal","puesto":"Técnico/a","situacion":"Servicio activo"}],` +
		`"servicios":[{"inicio":"2019-01-01","fin":"2019-12-31","clase":"Servicios previos","dias":365,"estado":"reconocido"}]},` +
		`"evidencia":{"recibo_ref":"fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100","decision_ref":"dec_prueba","efecto_ref":"` + o.Material.EmpleadoRef() + `","consumo_huella_sha256":"` + strings.Repeat("d", 64) + `","auditoria_ref":"auditoria:prueba","consultada_en":"` + consultada + `"}}`)
}

func TestFichaPropiaConsultaConfirmaRespuestaExacta(t *testing.T) {
	o := ordenFichaPropiaPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaFichaPropiaPrueba(t, o)}}}
	pool := &poolP{tx: tx}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool)
	resultado, err := r.ConsultarFichaPropia(context.Background(), o)
	if err != nil || tx.commits != 1 || tx.rollbacks != 0 || pool.o.IsoLevel != pgx.Serializable || tx.q[1] != consultaFichaPropiaSQL || len(tx.a[0]) != 11 {
		t.Fatalf("consulta no confirmada: %v", err)
	}
	if len(resultado.Ficha.Relaciones) != 1 || resultado.Ficha.Relaciones[0].Unidad != "Servicio de Personal" || resultado.Ficha.Servicios[0].Dias != 365 {
		t.Fatalf("ficha inesperada: %+v", resultado.Ficha)
	}
	if !bytes.Equal([]byte(tx.a[0][0].(string)), o.Material.Canonico()) {
		t.Fatal("no se envió el material canónico")
	}
}

func TestFichaPropiaRevierteRespuestaSinFormaExacta(t *testing.T) {
	o := ordenFichaPropiaPrueba(t)
	base := respuestaFichaPropiaPrueba(t, o)
	casos := map[string][]byte{
		"referencia_interna": bytes.Replace(base, []byte(`"situacion":"Servicio activo"`), []byte(`"situacion":"Servicio activo","relacion_ref":"rel_x"`), 1),
		"clave_ficha":        bytes.Replace(base, []byte(`"ficha":{`), []byte(`"ficha":{"persona_ref":"per_x",`), 1),
		"servicio_sin_dias":  bytes.Replace(base, []byte(`"dias":365,`), nil, 1),
		"corte_ajeno":        bytes.Replace(base, []byte(`"vigente_en":"2026-09-25"`), []byte(`"vigente_en":"2026-09-24"`), 1),
		"decision_ajena":     bytes.Replace(base, []byte(`"decision_ref":"dec_prueba"`), []byte(`"decision_ref":"dec_otra"`), 1),
		"estado_libre":       bytes.Replace(base, []byte(`"estado":"vigente"`), []byte(`"estado":"activo"`), 1),
		"relaciones_nulas":   bytes.Replace(base, []byte(`"relaciones":[`), []byte(`"relaciones":null,"x":[`), 1),
	}
	for nombre, bruto := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{bruto}}}
			r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
			_, err := r.ConsultarFichaPropia(context.Background(), o)
			if !errors.Is(err, domain.ErrFichaPropiaNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("respuesta alterada confirmada", err)
			}
		})
	}
}

func TestFichaPropiaDenegacionSQLOpaca(t *testing.T) {
	o := ordenFichaPropiaPrueba(t)
	tx := &txP{errQ: &pgconn.PgError{Code: "42501", Message: "empleado ajeno a la persona"}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	_, err := r.ConsultarFichaPropia(context.Background(), o)
	if !errors.Is(err, domain.ErrFichaPropiaDenegada) || strings.Contains(err.Error(), "ajeno") || tx.commits != 0 {
		t.Fatal("denegación SQL no opaca", err)
	}
	tx = &txP{errQ: &pgconn.PgError{Code: "55000", Message: "ficha propia incoherente"}}
	r, _ = nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	if _, err = r.ConsultarFichaPropia(context.Background(), o); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) {
		t.Fatal("incoherencia no tratada como no disponible", err)
	}
	// 54000: más filas de las que admite el contrato; estado propio, sin confirmar.
	tx = &txP{errQ: &pgconn.PgError{Code: "54000", Message: "ficha propia excede límite"}}
	r, _ = nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	if _, err = r.ConsultarFichaPropia(context.Background(), o); !errors.Is(err, domain.ErrFichaPropiaExcedeLimite) || tx.commits != 0 {
		t.Fatal("exceso de filas no distinguido", err)
	}
}

func TestFichaPropiaRechazaAtestacionAjenaAntesDeSQL(t *testing.T) {
	o := ordenFichaPropiaPrueba(t)
	ajena := ordenFichaB2Prueba(t).Autorizacion
	tx := &txP{}
	pool := &poolP{tx: tx}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool)
	if _, err := r.ConsultarFichaPropia(context.Background(), ports.OrdenFichaPropia{Material: o.Material, Autorizacion: ajena}); !errors.Is(err, domain.ErrFichaPropiaInvalida) || pool.n != 0 {
		t.Fatal("atestación de otra operación enviada a SQL", err)
	}
}

func TestDecodificarFichaPropiaSinReferenciasEnJSON(t *testing.T) {
	o := ordenFichaPropiaPrueba(t)
	r, err := decodificarFichaPropia(respuestaFichaPropiaPrueba(t, o), o)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(r.Ficha)
	if bytes.Contains(b, []byte("emp_")) || bytes.Contains(b, []byte("per_")) {
		t.Fatalf("la ficha serializada contiene referencias: %s", b)
	}
}
