package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/dietas/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestBorradorSQLMecanica usa una BD nueva y un DOBLE AD3 explícito. Comprueba
// atomicidad, ACL, RLS e idempotencia de Dietas; NO verifica COSE, revocación
// central, ni acredita integración V3 real. Requiere un clúster desechable sin
// roles Dietas instalados y DSN privado de DBA; nunca registra dicho DSN.
func TestBorradorSQLMecanica(t *testing.T) {
	dsn := os.Getenv("VEC_DIETAS_SQL_MECANICA_DSN")
	if dsn == "" {
		t.Skip("requiere clúster PostgreSQL desechable mediante VEC_DIETAS_SQL_MECANICA_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal("no se pudo conectar al clúster desechable")
	}
	defer admin.Close(context.Background())
	var existentes int
	err = admin.QueryRow(ctx, `SELECT count(*) FROM pg_roles WHERE rolname IN ('vec_dietas_v1_propietario','vec_dietas_v1_ejecutor')`).Scan(&existentes)
	if err != nil || existentes != 0 {
		t.Fatal("el clúster debe carecer de roles Dietas")
	}
	var sufijo [12]byte
	if _, err = rand.Read(sufijo[:]); err != nil {
		t.Fatal("no se pudo generar nombre de prueba")
	}
	nombre := "vec_dietas_test_" + hex.EncodeToString(sufijo[:])
	identificador := pgx.Identifier{nombre}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+identificador); err != nil {
		falloSQLDietas(t, "crear BD propia", err)
	}
	rolesCreados := false
	defer func() {
		limpieza, fin := context.WithTimeout(context.Background(), 15*time.Second)
		defer fin()
		if _, e := admin.Exec(limpieza, "DROP DATABASE "+identificador+" WITH (FORCE)"); e != nil {
			falloSQLDietas(t, "retirar exclusivamente BD de prueba", e)
		}
		if rolesCreados {
			if _, e := admin.Exec(limpieza, "DROP ROLE vec_dietas_v1_ejecutor,vec_dietas_v1_propietario"); e != nil {
				falloSQLDietas(t, "retirar roles creados por prueba", e)
			}
		}
	}()
	cfg := admin.Config().Copy()
	cfg.Database = nombre
	bd, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal("no se pudo abrir BD de prueba")
	}
	defer bd.Close(context.Background())
	if _, err = bd.Exec(ctx, "CREATE SCHEMA vec_autorizacion_atestada_v3"); err != nil {
		falloSQLDietas(t, "esquema del doble", err)
	}
	_, archivo, _, _ := runtime.Caller(0)
	raiz := filepath.Clean(filepath.Join(filepath.Dir(archivo), "../../../../.."))
	leer := func(sufijo string) string {
		t.Helper()
		b, e := os.ReadFile(filepath.Join(raiz, "deploy/postgresql/dietas_v1/migraciones/000001_borrador_propio."+sufijo+".sql"))
		if e != nil {
			t.Fatal("no se pudo leer migración Dietas")
		}
		return strings.ReplaceAll(string(b), "\\set ON_ERROR_STOP on\n", "")
	}
	up, down := leer("up"), leer("down")
	if _, err = bd.Exec(ctx, up); err != nil {
		falloSQLDietas(t, "UP vacío", err)
	}
	rolesCreados = true
	if _, err = bd.Exec(ctx, down); err != nil {
		falloSQLDietas(t, "DOWN vacío", err)
	}
	rolesCreados = false
	if _, err = bd.Exec(ctx, up); err != nil {
		falloSQLDietas(t, "UP tras reversión vacía", err)
	}
	rolesCreados = true
	if _, err = bd.Exec(ctx, dobleConsumoDietasSQL); err != nil {
		falloSQLDietas(t, "instalar doble mecánico", err)
	}
	var indebidos int
	err = bd.QueryRow(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
	 WHERE n.nspname='vec_dietas_v1' AND c.relkind='r' AND
	 (NOT c.relrowsecurity OR NOT c.relforcerowsecurity OR
	 has_table_privilege('vec_dietas_v1_ejecutor',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'))`).Scan(&indebidos)
	if err != nil || indebidos != 0 {
		t.Fatal("ACL o RLS de tablas incorrectas")
	}
	err = bd.QueryRow(ctx, `SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
	 WHERE n.nspname='vec_dietas_v1' AND has_function_privilege('vec_dietas_v1_ejecutor',p.oid,'EXECUTE')`).Scan(&indebidos)
	if err != nil || indebidos != 3 {
		t.Fatal("ejecutor debe tener sólo las tres fachadas")
	}
	var secuencia atomic.Uint64
	actor := map[string]any{"actor_ref": "persona:sintetica", "perfil_ref": "perfil:sintetico", "persona_ref": "persona:sintetica", "empleado_ref": "empleado:sintetico"}
	borrador := map[string]any{
		"persona_ref": "persona:sintetica", "objeto": "Prueba sintética",
		"inicio": "2026-09-20T09:00:00Z", "fin": "2026-09-20T10:00:00Z",
		"fecha": "2026-09-20", "fecha_fin": "2026-09-20", "hora_inicio": "11:00", "hora_fin": "12:00", "zona_horaria": "Europe/Madrid",
		"ruta_etiquetas":   []string{"Origen sintético", "Destino sintético"},
		"procedencia_ruta": "declarada_no_verificada", "revalidacion_ruta_requerida": true,
		"vehiculo_propio": true, "estado": "borrador", "contexto_personal": "contexto_personal_pendiente", "liquidable": false,
		"ruta": map[string]any{"fuente": "osrm_interno", "version": "version_sintetica", "referencia": "ruta:sintetica",
			"catalogo_version": "catalogo_sintetico", "alternativa_ref": "ruta:principal", "kilometros": "10.5000",
			"recomendada": true, "motivo_alternativa": "", "liquidable": false,
			"paradas": []any{map[string]any{"codigo": "origen", "nombre": "Origen sintético", "latitud": 37.0, "longitud": -3.0},
				map[string]any{"codigo": "destino", "nombre": "Destino sintético", "latitud": 37.1, "longitud": -3.1}},
			"tramos": []any{map[string]any{"origen_codigo": "origen", "destino_codigo": "destino", "kilometros": "10.0000",
				"duracion_minutos": 20, "ajuste_kilometros": "0.5000", "motivo_ajuste": "Desvío sintético"}},
			"trazado": []any{[]float64{37.0, -3.0}, []float64{37.1, -3.1}}},
		"gastos":               map[string]any{"manutencion_eur": "12.34", "alojamiento_eur": "20.00", "otros_eur": "1.00"},
		"politica_kilometraje": map[string]any{"referencia": "politica:sintetica", "version": "version_sintetica", "tarifa_eur_km": "0.2600"},
		"desglose":             map[string]any{"kilometraje_eur": "2.73", "manutencion_eur": "12.34", "alojamiento_eur": "20.00", "otros_eur": "1.00", "total_eur": "36.07"},
	}
	material := func(clave string) map[string]any {
		return map[string]any{"actor": actor, "clave_operacion": clave, "version_esperada": 0, "borrador": borrador}
	}
	llamar := func(op string, m map[string]any, denegar bool) (map[string]any, error) {
		conn, e := pgx.ConnectConfig(ctx, cfg.Copy())
		if e != nil {
			return nil, e
		}
		defer conn.Close(context.Background())
		tx, e := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			return nil, e
		}
		defer tx.Rollback(context.Background())
		if _, e = tx.Exec(ctx, `SET LOCAL ROLE vec_dietas_v1_ejecutor; SET LOCAL timezone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'`); e != nil {
			return nil, e
		}
		a := m["actor"].(map[string]any)
		ref, _ := m["comision_ref"].(string)
		if op == "crear" {
			ref = "dietas:borrador:" + m["clave_operacion"].(string)
		}
		accion, funcion := "dietas.borrador."+op+"_propio", op+"_borrador_propio_v1"
		if op == "listar" {
			ref, accion, funcion = "dietas:borradores:propios", "dietas.borrador.listar_propios", "listar_borradores_propios_v1"
		}
		bytes, _ := json.Marshal(m)
		h := sha256.Sum256(bytes)
		cr, _ := json.Marshal(map[string]any{"ambitos": map[string]any{"empleado_ref": a["empleado_ref"], "persona_ref": a["persona_ref"]}, "atributos": map[string]any{"material_sha256": hex.EncodeToString(h[:])}})
		hc := sha256.Sum256(cr)
		id := fmt.Sprintf("decision:prueba:%d", secuencia.Add(1))
		vence := time.Now().UTC().Add(time.Hour).Format("2006-01-02T15:04:05.000000Z")
		d, _ := json.Marshal(map[string]any{"valida_hasta": vence, "decision_ref": id, "accion": accion, "modulo_id": "dietas", "tipo_recurso": "borrador_comision", "finalidad": "gestionar_borrador_propio", "recurso_ref": ref, "principal_id": a["actor_ref"], "perfil_activo_ref": a["perfil_ref"], "contexto_recurso_huella_sha256": hex.EncodeToString(hc[:]), "correlacion_ref": "correlacion:prueba", "prueba_denegada": denegar})
		x, _ := json.Marshal(map[string]any{"vigente_hasta": vence, "esquema": "vec.contexto-actor.vinculado.v2", "principal_ref": a["actor_ref"], "perfil_activo_ref": a["perfil_ref"], "persona_ref": a["persona_ref"], "vinculos": []any{map[string]any{"tipo": "empleado", "estado": "activo", "referencia": a["empleado_ref"], "vigente_desde": time.Now().UTC().Add(-time.Hour).Format("2006-01-02T15:04:05.000000Z"), "vigente_hasta": time.Now().UTC().Add(time.Hour).Format("2006-01-02T15:04:05.000000Z")}}})
		capacidad, _ := json.Marshal(map[string]any{"expira_en": vence, "decision_valida_hasta": vence, "configuracion_expira_en": vence, "raiz_valida_hasta": vence})
		var salida []byte
		e = tx.QueryRow(ctx, "SELECT vec_dietas_v1."+funcion+"($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)", string(bytes), capacidad, d, []byte("doble"), x, 1, 1, []byte("doble"), []byte("doble"), []byte("doble"), []byte("doble")).Scan(&salida)
		if e != nil {
			return nil, e
		}
		if e = tx.Commit(ctx); e != nil {
			return nil, e
		}
		var resultado map[string]any
		e = json.Unmarshal(salida, &resultado)
		return resultado, e
	}
	clave := "operacion_sintetica_0001"
	primero, err := llamar("crear", material(clave), false)
	if err != nil {
		falloSQLDietas(t, "crear", err)
	}
	replay, err := llamar("crear", material(clave), false)
	if err != nil {
		falloSQLDietas(t, "replay", err)
	}
	if replay["repeticion"] != true || replay["recibo_ref"] != primero["recibo_ref"] || replay["registrado_en"] != primero["registrado_en"] {
		t.Fatal("replay no conserva recibo")
	}
	recuperado, err := llamar("recuperar", map[string]any{"actor": actor, "comision_ref": primero["comision_ref"]}, false)
	if err != nil {
		falloSQLDietas(t, "recuperar tras reconectar", err)
	}
	if recuperado["recibo"].(map[string]any)["recibo_ref"] != primero["recibo_ref"] {
		t.Fatal("recuperación cambia recibo")
	}
	esperadoJSON, _ := json.Marshal(borrador)
	guardadoJSON, _ := json.Marshal(recuperado["borrador"])
	if string(esperadoJSON) != string(guardadoJSON) {
		t.Fatal("recuperación altera ruta, gastos o desglose")
	}
	actorAjeno := map[string]any{"actor_ref": "persona:ajena", "perfil_ref": "perfil:ajeno", "persona_ref": "persona:ajena", "empleado_ref": "empleado:ajeno"}
	_, err = llamar("recuperar", map[string]any{"actor": actorAjeno, "comision_ref": primero["comision_ref"]}, false)
	exigirEstadoDietasSQL(t, err, "PDI04")
	borrador["objeto"] = "Material divergente"
	_, err = llamar("crear", material(clave), false)
	exigirEstadoDietasSQL(t, err, "PDI01")
	borrador["objeto"] = "Prueba sintética"
	_, err = llamar("crear", material("operacion_denegada_01"), true)
	exigirEstadoDietasSQL(t, err, "PDI03")
	desglose := borrador["desglose"].(map[string]any)
	desglose["total_eur"] = "36.08"
	_, err = llamar("crear", material("operacion_total_invalido"), false)
	exigirEstadoDietasSQL(t, err, "PDI00")
	desglose["total_eur"] = "36.07"
	borrador["hora_inicio"] = "09:00"
	_, err = llamar("crear", material("operacion_hora_invalida"), false)
	exigirEstadoDietasSQL(t, err, "PDI00")
	borrador["hora_inicio"] = "11:00"
	for _, limite := range []int{0, 21} {
		_, err = llamar("listar", map[string]any{"actor": actor, "limite": limite, "despues": ""}, false)
		exigirEstadoDietasSQL(t, err, "PDI00")
	}
	// Dos conexiones concurrentes: una fila durable; el segundo consumo puede
	// confirmar replay o requerir reintento por aislamiento serializable.
	var wg sync.WaitGroup
	resultados := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := llamar("crear", material("operacion_concurrente_01"), false)
			resultados <- e
		}()
	}
	wg.Wait()
	close(resultados)
	confirmadas := 0
	for e := range resultados {
		if e == nil {
			confirmadas++
			continue
		}
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || (pg.Code != "40001" && pg.Code != "PDI01") {
			falloSQLDietas(t, "concurrencia", e)
		}
	}
	if confirmadas < 1 {
		t.Fatal("ninguna petición concurrente confirmó")
	}
	if _, err = llamar("crear", material("operacion_concurrente_01"), false); err != nil {
		falloSQLDietas(t, "replay después de concurrencia", err)
	}
	pagina, err := llamar("listar", map[string]any{"actor": actor, "limite": 1, "despues": ""}, false)
	if err != nil {
		falloSQLDietas(t, "primera página propia", err)
	}
	filas := pagina["borradores"].([]any)
	if len(filas) != 1 || pagina["siguiente"] != filas[0].(map[string]any)["recibo"].(map[string]any)["comision_ref"] {
		t.Fatal("página propia no conserva cursor del último recibo")
	}
	segunda, err := llamar("listar", map[string]any{"actor": actor, "limite": 1, "despues": pagina["siguiente"]}, false)
	if err != nil {
		falloSQLDietas(t, "segunda página propia", err)
	}
	filas2 := segunda["borradores"].([]any)
	if len(filas2) != 1 || segunda["siguiente"] != "" ||
		filas2[0].(map[string]any)["recibo"].(map[string]any)["comision_ref"].(string) <= pagina["siguiente"].(string) {
		t.Fatal("cursor repite u omite borrador propio")
	}
	ajena, err := llamar("listar", map[string]any{"actor": actorAjeno, "limite": 20, "despues": ""}, false)
	if err != nil {
		falloSQLDietas(t, "listado ajeno vacío", err)
	}
	if len(ajena["borradores"].([]any)) != 0 || ajena["siguiente"] != "" {
		t.Fatal("listado revela borradores ajenos")
	}
	_, err = llamar("listar", map[string]any{"actor": actor, "limite": 20, "despues": ""}, true)
	exigirEstadoDietasSQL(t, err, "PDI03")
	var revisiones, eventos, consumos, accesos int
	err = bd.QueryRow(ctx, `SELECT (SELECT count(*) FROM vec_dietas_v1.borrador_revision),(SELECT count(*) FROM vec_dietas_v1.borrador_outbox),(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumos_prueba),(SELECT count(*) FROM vec_dietas_v1.borrador_acceso)`).Scan(&revisiones, &eventos, &consumos, &accesos)
	if err != nil {
		falloSQLDietas(t, "contar efectos", err)
	}
	if revisiones != 2 || eventos != 2 || consumos != accesos {
		t.Fatal("duplicación o consumo separado del efecto")
	}
	// Frontera real JSONB del borrador con el doble AD3 explícito de este test.
	// No constituye aceptación V3/COSE; protege almacenamiento y proyección.
	t.Run("ruta_maxima_recuperable_y_lista_ligera", func(t *testing.T) {
		ruta := borrador["ruta"].(map[string]any)
		anterior := ruta["trazado"]
		defer func() { ruta["trazado"] = anterior }()
		trazado := make([][2]float64, 2000)
		for i := range trazado {
			trazado[i] = [2]float64{37.123456789012, -3.123456789012}
		}
		ruta["trazado"] = trazado
		var primerRef string
		for i := 0; i < 20; i++ {
			clave := fmt.Sprintf("zz_limite_%08d", i)
			bytes, _ := json.Marshal(material(clave))
			if len(bytes) <= 65536 || len(bytes) > ports.MaxBytesMaterialBorrador {
				t.Fatal("frontera material no ejercitada", len(bytes))
			}
			r, e := llamar("crear", material(clave), false)
			if e != nil {
				falloSQLDietas(t, "crear ruta máxima", e)
			}
			if i == 0 {
				primerRef = r["comision_ref"].(string)
			}
		}
		recuperada, e := llamar("recuperar", map[string]any{"actor": actor, "comision_ref": primerRef}, false)
		if e != nil {
			falloSQLDietas(t, "recuperar ruta máxima", e)
		}
		if len(recuperada["borrador"].(map[string]any)["ruta"].(map[string]any)["trazado"].([]any)) != 2000 {
			t.Fatal("trazado perdido")
		}
		var tamano int
		if e = bd.QueryRow(ctx, "SELECT octet_length(jsonb_build_object('borrador',borrador,'recibo',recibo_json)::text) FROM vec_dietas_v1.borrador_revision WHERE comision_ref=$1", primerRef).Scan(&tamano); e != nil || tamano <= 65536 || tamano > ports.MaxBytesDetalleBorrador {
			t.Fatal("límite detalle incompatible", tamano)
		}
		p, e := llamar("listar", map[string]any{"actor": actor, "limite": 20, "despues": "dietas:borrador:zz_limite_000000"}, false)
		if e != nil {
			falloSQLDietas(t, "listar veinte rutas máximas", e)
		}
		items := p["borradores"].([]any)
		if len(items) != 20 || p["siguiente"] != "" {
			t.Fatal("página incompleta")
		}
		for _, item := range items {
			b := item.(map[string]any)["borrador"].(map[string]any)
			r := b["ruta"].(map[string]any)
			for _, pesado := range []string{"paradas", "tramos", "trazado"} {
				if _, ok := r[pesado]; ok {
					t.Fatal("listado incluye geometría", pesado)
				}
			}
			if len(b["ruta_etiquetas"].([]any)) != 2 || b["procedencia_ruta"] != "declarada_no_verificada" || b["revalidacion_ruta_requerida"] != true {
				t.Fatal("resumen pierde ruta o advertencia")
			}
		}
		bytes, _ := json.Marshal(p)
		if len(bytes) > ports.MaxBytesListadoBorrador {
			t.Fatal("lista supera límite de transporte")
		}
		for i := range trazado {
			trazado[i] = [2]float64{math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64}
		}
		_, e = llamar("crear", material("zz_expansion_decimal_01"), false)
		exigirEstadoDietasSQL(t, e, "PDI00")
		borrador["procedencia_ruta"] = "osrm_verificada"
		_, e = llamar("crear", material("zz_origen_inventado_01"), false)
		exigirEstadoDietasSQL(t, e, "PDI00")
		borrador["procedencia_ruta"] = "declarada_no_verificada"
	})
	// Propietario también queda sujeto a RLS sin contexto; no existe política true.
	tx, err := bd.Begin(ctx)
	if err != nil {
		falloSQLDietas(t, "abrir RLS", err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SET LOCAL ROLE vec_dietas_v1_propietario"); err != nil {
		falloSQLDietas(t, "rol RLS", err)
	}
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM vec_dietas_v1.borrador_revision").Scan(&revisiones); err != nil || revisiones != 0 {
		t.Fatal("RLS revela filas sin contexto")
	}
	_ = tx.Rollback(ctx)
	var unico bool
	for _, civil := range []struct{ instante, fecha, hora string }{
		{"2026-10-25T00:30:00Z", "2026-10-25", "02:30"},
		{"2026-03-29T01:30:00Z", "2026-03-29", "02:30"},
	} {
		err = bd.QueryRow(ctx, "SELECT vec_dietas_v1.instante_civil_unico_v1($1::timestamptz,$2,$3,'Europe/Madrid')",
			civil.instante, civil.fecha, civil.hora).Scan(&unico)
		if err != nil || unico {
			t.Fatal("hora ambigua o inexistente aceptada")
		}
	}
	// DOWN no elimina historia ni siquiera después de retirar el consumidor.
	tx, err = bd.Begin(ctx)
	if err != nil {
		falloSQLDietas(t, "abrir comprobación DOWN", err)
	}
	_, err = tx.Exec(ctx, "DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_borrador_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)")
	if err != nil {
		falloSQLDietas(t, "retirar doble dentro de rollback", err)
	}
	sinTransaccion := strings.ReplaceAll(strings.ReplaceAll(down, "BEGIN;\n", ""), "COMMIT;\n", "")
	_, err = tx.Exec(ctx, sinTransaccion)
	exigirEstadoDietasSQL(t, err, "55000")
	_ = tx.Rollback(ctx)
	for _, sql := range []string{"UPDATE vec_dietas_v1.borrador_revision SET estado=estado", "DELETE FROM vec_dietas_v1.borrador_acceso", "TRUNCATE vec_dietas_v1.borrador_outbox"} {
		_, err = bd.Exec(ctx, sql)
		exigirEstadoDietasSQL(t, err, "55000")
	}
}

func exigirEstadoDietasSQL(t *testing.T, err error, codigo string) {
	t.Helper()
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != codigo {
		falloSQLDietas(t, "SQLSTATE esperado "+codigo, err)
	}
}
func falloSQLDietas(t *testing.T, paso string, err error) {
	t.Helper()
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		t.Fatalf("%s: SQLSTATE %s", paso, pg.Code)
	}
	t.Fatalf("%s: fallo sin detalles de conexión ni material", paso)
}

const dobleConsumoDietasSQL = `
CREATE TABLE vec_autorizacion_atestada_v3.consumos_prueba(decision_ref text PRIMARY KEY);
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_borrador_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF (d->>'prueba_denegada')::boolean THEN RAISE EXCEPTION 'doble deniega' USING ERRCODE='PDI03'; END IF;
 INSERT INTO vec_autorizacion_atestada_v3.consumos_prueba VALUES(d->>'decision_ref');
 RETURN QUERY SELECT d->>'decision_ref',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
  encode(sha256(p_decision),'hex'),'auditoria:'||(d->>'decision_ref'),clock_timestamp(),true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_borrador_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_dietas_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_borrador_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_v1_propietario;
`
