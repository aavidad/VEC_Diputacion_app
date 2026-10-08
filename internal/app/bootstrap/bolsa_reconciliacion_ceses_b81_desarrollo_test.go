package bootstrap

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type pasoReconciliacionB81 struct {
	funcion string
	args    []any
	fila    filaCeseB81
}

type consultasReconciliacionB81 struct {
	t        *testing.T
	pasos    []pasoReconciliacionB81
	llamadas int
}

var huellaReconciliacionB81 = strings.Repeat("a", 64)

func (q *consultasReconciliacionB81) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	q.t.Helper()
	if q.llamadas >= len(q.pasos) {
		q.t.Fatalf("consulta adicional: %s", sql)
	}
	paso := q.pasos[q.llamadas]
	q.llamadas++
	if !strings.Contains(sql, paso.funcion) || !reflect.DeepEqual(args, paso.args) {
		q.t.Fatalf("consulta %d: esperada=%s %v, actual=%s %v", q.llamadas, paso.funcion, paso.args, sql, args)
	}
	return paso.fila
}

func filaPendientesB81(ref string, posicion int64) filaCeseB81 {
	if ref == "" {
		return filaCeseB81{valores: []any{[]byte(`[]`)}}
	}
	return filaCeseB81{valores: []any{[]byte(fmt.Sprintf(`[{"origen_ref":%q,"huella_sha256":%q,"origen_posicion":%d}]`, ref, huellaReconciliacionB81, posicion))}}
}

func filaB45B81(reutilizada bool) filaCeseB81 {
	return filaCeseB81{valores: []any{reutilizada, "recibo:1", "candidato:1", time.Now(), int64(1)}}
}

func TestReconciliacionB81RecuperaCeseAnteriorAlCursorCT(t *testing.T) {
	ref := "ref:outbox:antiguo"
	q := &consultasReconciliacionB81{t: t, pasos: []pasoReconciliacionB81{
		{"cursor_restriccion_cese_bolsa_v1", nil, filaCeseB81{valores: []any{int64(500), "ref:outbox:reciente"}}},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81(ref, 7)},
		{"registrar_restriccion_cese_bolsa_v1", []any{ref, huellaReconciliacionB81, int64(7)}, filaB45B81(false)},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81("", 0)},
	}}
	lector := lectorCesesB81{&lectorContratosCTPrueba{}}
	principal := &entregaCesesCTBolsa{lector: lector, pool: q, lote: 1}
	resultado, err := entregarCesesConReconciliacionB81(context.Background(), principal, &reconciliacionCesesB81{pool: q, lote: 1})
	if err != nil || resultado.nuevos != 1 || q.llamadas != len(q.pasos) {
		t.Fatalf("cese antiguo: resultado=%+v error=%v consultas=%d", resultado, err, q.llamadas)
	}
	if len(lector.desdes) != 1 || lector.desdes[0] != (puertosct.CursorPublicacionContratosBolsa{Posicion: 500, OrigenRef: "ref:outbox:reciente"}) {
		t.Fatalf("cursor CT modificado: %v", lector.desdes)
	}
}

func TestReconciliacionB81EsperaVinculoYNoDuplicaReplay(t *testing.T) {
	ref := "ref:outbox:pendiente"
	q := &consultasReconciliacionB81{t: t, pasos: []pasoReconciliacionB81{
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81("", 0)},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81(ref, 2)},
		{"registrar_restriccion_cese_bolsa_v1", []any{ref, huellaReconciliacionB81, int64(2)}, filaB45B81(true)},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81("", 0)},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81("", 0)},
	}}
	r := &reconciliacionCesesB81{pool: q, lote: 1}
	primero, err := r.entregar(context.Background())
	if err != nil || primero != (resultadoEntregaContratosCT{}) {
		t.Fatalf("antes del vínculo: %+v, %v", primero, err)
	}
	segundo, err := r.entregar(context.Background())
	if err != nil || segundo.reentregas != 1 || segundo.nuevos != 0 {
		t.Fatalf("replay B45: %+v, %v", segundo, err)
	}
	tercero, err := r.entregar(context.Background())
	if err != nil || tercero != (resultadoEntregaContratosCT{}) || q.llamadas != len(q.pasos) {
		t.Fatalf("duplicación tras replay: %+v, %v, consultas=%d", tercero, err, q.llamadas)
	}
}

func TestReconciliacionB81ReintentaFalloB45(t *testing.T) {
	ref := "ref:outbox:reintentar"
	q := &consultasReconciliacionB81{t: t, pasos: []pasoReconciliacionB81{
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81(ref, 3)},
		{"registrar_restriccion_cese_bolsa_v1", []any{ref, huellaReconciliacionB81, int64(3)}, filaCeseB81{err: &pgconn.PgError{Code: "08006"}}},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81(ref, 3)},
		{"registrar_restriccion_cese_bolsa_v1", []any{ref, huellaReconciliacionB81, int64(3)}, filaB45B81(false)},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81("", 0)},
	}}
	r := &reconciliacionCesesB81{pool: q, lote: 1}
	primero, err := r.entregar(context.Background())
	if err == nil || primero != (resultadoEntregaContratosCT{}) {
		t.Fatalf("fallo B45 tragado: %+v, %v", primero, err)
	}
	segundo, err := r.entregar(context.Background())
	if err != nil || segundo.nuevos != 1 || q.llamadas != len(q.pasos) {
		t.Fatalf("pendiente perdido: %+v, %v, consultas=%d", segundo, err, q.llamadas)
	}
}

func TestReconciliacionB81ContinuaSiLectorCTFalla(t *testing.T) {
	ref := "ref:outbox:antiguo"
	q := &consultasReconciliacionB81{t: t, pasos: []pasoReconciliacionB81{
		{"cursor_restriccion_cese_bolsa_v1", nil, filaCeseB81{valores: []any{int64(500), "ref:outbox:reciente"}}},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81(ref, 7)},
		{"registrar_restriccion_cese_bolsa_v1", []any{ref, huellaReconciliacionB81, int64(7)}, filaB45B81(false)},
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaPendientesB81("", 0)},
	}}
	lector := lectorCesesB81{&lectorContratosCTPrueba{err: fmt.Errorf("fuente CT no disponible")}}
	principal := &entregaCesesCTBolsa{lector: lector, pool: q, lote: 1}
	resultado, err := entregarCesesConReconciliacionB81(context.Background(), principal, &reconciliacionCesesB81{pool: q, lote: 1})
	if err == nil || resultado.nuevos != 1 || q.llamadas != len(q.pasos) {
		t.Fatalf("recuperación dependió del lector CT: %+v, %v, consultas=%d", resultado, err, q.llamadas)
	}
}

func TestRelevoCesesExigeGuardasB90(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		fila   filaCeseB81
		pasa   bool
	}{
		{"ambas_instaladas", filaCeseB81{valores: []any{true, true}}, true},
		{"falta_B90", filaCeseB81{valores: []any{false, true}}, false},
		{"falta_lote_B90", filaCeseB81{valores: []any{true, false}}, false},
		{"consulta_fallida", filaCeseB81{err: pgx.ErrNoRows}, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			q := &consultasReconciliacionB81{t: t, pasos: []pasoReconciliacionB81{
				{"candidatos_cese_pendiente_b90", nil, caso.fila},
			}}
			err := verificarGuardasCesePendienteB90(context.Background(), q)
			if (err == nil) != caso.pasa || q.llamadas != 1 {
				t.Fatalf("guardas B90: error=%v consultas=%d", err, q.llamadas)
			}
		})
	}
}

func TestReconciliacionB81NoConfundeRespuestaNulaConListaVacia(t *testing.T) {
	q := &consultasReconciliacionB81{t: t, pasos: []pasoReconciliacionB81{
		{"listar_ceses_sin_candidato_pendientes_v1", []any{1}, filaCeseB81{valores: []any{[]byte(`null`)}}},
	}}
	resultado, err := (&reconciliacionCesesB81{pool: q, lote: 1}).entregar(context.Background())
	if err == nil || resultado != (resultadoEntregaContratosCT{}) || q.llamadas != 1 {
		t.Fatalf("respuesta nula aceptada: %+v, %v, consultas=%d", resultado, err, q.llamadas)
	}
}
