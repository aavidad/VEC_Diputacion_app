package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const variableDSNConsumoC1O404D = "VEC_O404D_ADMIN_DSN"
const variableAislamientoConsumoC1O404D = "VEC_O404D_PG18_AISLADO"

func TestConsumoC1CoberturaO404DPostgreSQL18Real(t *testing.T) {
	dsn := os.Getenv(variableDSNConsumoC1O404D)
	if dsn == "" {
		t.Skip("PostgreSQL 18 efímero O4-04D no solicitado")
	}
	if os.Getenv(variableAislamientoConsumoC1O404D) != "1" {
		t.Fatal("la suite O4-04D exige PostgreSQL 18 efímero aislado")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelar()
	raiz := raizRepositorioPreparacionDecisionCobertura(t)
	base, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	var version int
	var baseDatos string
	if err := base.QueryRow(ctx, `
		SELECT current_setting('server_version_num')::integer / 10000,
		       current_database()`).Scan(&version, &baseDatos); err != nil {
		t.Fatal(err)
	}
	if version != 18 || baseDatos != "postgres" {
		t.Fatalf("entorno O4-04D inválido: PG=%d DB=%q", version, baseDatos)
	}
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())
	nombreBase := "vec_o404d_" + sufijo
	login := "vec_o404d_runtime_" + sufijo
	asegurarClusterLimpioPreparacionDecisionCobertura(
		t,
		ctx,
		base,
		nombreBase,
	)
	crearBasePreparacionDecisionCobertura(t, ctx, base, nombreBase)
	admin := abrirBasePreparacionDecisionCobertura(
		t,
		ctx,
		dsn,
		nombreBase,
		"postgres",
	)
	ejecutarArchivoPreparacionDecisionCobertura(
		t,
		ctx,
		admin,
		filepath.Join(
			raiz,
			"deploy/postgresql/contratacion_temporal/roles_up.sql",
		),
	)
	crearLoginPreparacionDecisionCobertura(t, ctx, base, login)
	defer func() {
		admin.Close()
		eliminarBasePreparacionDecisionCobertura(t, base, nombreBase)
		eliminarLoginYRolesPreparacionDecisionCobertura(t, base, login)
	}()
	instalarBaseHastaO404BPreparacionDecisionCobertura(
		t,
		ctx,
		admin,
		raiz,
	)
	for _, numero := range []string{"000020", "000021", "000022"} {
		ejecutarMigracionPreparacionDecisionCobertura(
			t,
			ctx,
			admin,
			raiz,
			numero,
			"up",
		)
	}
	for _, numero := range []string{"000023", "000024"} {
		ejecutarMigracionConsumoC1O404D(
			t,
			ctx,
			admin,
			raiz,
			numero,
			"up",
		)
	}
	probarWrapperVECO404D(t, ctx, base, dsn, raiz, sufijo)
	probarCicloYACLO404D(t, ctx, admin, raiz, dsn, nombreBase, login)
	probarCanonesYLimitesO404D(t, ctx, admin)
	probarPersistenciaReplayRollbackO404D(t, ctx, admin)
	probarConcurrenciaYReaperturaO404D(
		t,
		ctx,
		admin,
		dsn,
		nombreBase,
	)
}

func rutaMigracionConsumoC1O404D(
	t *testing.T,
	raiz string,
	numero string,
	direccion string,
) string {
	t.Helper()
	rutas, err := filepath.Glob(filepath.Join(
		raiz,
		"deploy/postgresql/contratacion_temporal/migraciones/"+
			numero+"_*o4_04d."+direccion+".sql",
	))
	if err != nil || len(rutas) != 1 {
		t.Fatalf(
			"migración O4-04D %s %s no unívoca: %v / %v",
			numero,
			direccion,
			rutas,
			err,
		)
	}
	return rutas[0]
}

func ejecutarMigracionConsumoC1O404D(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	raiz string,
	numero string,
	direccion string,
) {
	t.Helper()
	ejecutarArchivoPreparacionDecisionCobertura(
		t,
		ctx,
		pool,
		rutaMigracionConsumoC1O404D(t, raiz, numero, direccion),
	)
}

func probarCicloYACLO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	raiz string,
	dsn string,
	baseDatos string,
	login string,
) {
	t.Helper()
	var version int
	var rlsLote, fuerzaLote, rlsEvidencia, fuerzaEvidencia bool
	var ejecutaPropietario, sinRuntime, publica, configuracion bool
	err := admin.QueryRow(ctx, `
		SELECT c.version_esquema,
		       l.relrowsecurity, l.relforcerowsecurity,
		       e.relrowsecurity, e.relforcerowsecurity,
		       NOT EXISTS (
		         SELECT 1
		           FROM unnest(ARRAY[
		             'vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(jsonb)',
		             'vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(jsonb,jsonb)'
		           ]) f
		          WHERE NOT has_function_privilege(
		            'vec_contratacion_temporal_propietario',f,'EXECUTE'
		          )
		       ),
		       NOT EXISTS (
		         SELECT 1
		           FROM unnest(ARRAY[
		             'vec_contratacion_temporal_ejecutor',
		             'vec_contratacion_temporal_gobernador'
		           ]) r
		          CROSS JOIN unnest(ARRAY[
		             'vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(jsonb)',
		             'vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(jsonb,jsonb)'
		          ]) f
		          WHERE has_function_privilege(r,f,'EXECUTE')
		       ),
		       NOT EXISTS (
		         SELECT 1
		           FROM pg_catalog.aclexplode(p.proacl) a
		          WHERE a.grantee=0 AND a.privilege_type='EXECUTE'
		       ),
		       p.proconfig @> ARRAY[
		         'search_path=pg_catalog',
		         'TimeZone=UTC',
		         'lock_timeout=2s'
		       ]
		  FROM vec_contratacion_temporal.control_migracion_cobertura_o4 c,
		       pg_catalog.pg_class l,
		       pg_catalog.pg_class e,
		       pg_catalog.pg_proc p
		 WHERE c.control
		   AND l.oid='vec_contratacion_temporal.consumo_cobertura_lote'::regclass
		   AND e.oid='vec_contratacion_temporal.consumo_cobertura_evidencia'::regclass
		   AND p.oid=
		     'vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(jsonb,jsonb)'::regprocedure`).Scan(
		&version,
		&rlsLote,
		&fuerzaLote,
		&rlsEvidencia,
		&fuerzaEvidencia,
		&ejecutaPropietario,
		&sinRuntime,
		&publica,
		&configuracion,
	)
	if err != nil || version != 4 || !rlsLote || !fuerzaLote ||
		!rlsEvidencia || !fuerzaEvidencia || !ejecutaPropietario ||
		!sinRuntime || !publica || !configuracion {
		t.Fatalf("ACL/RLS O4-04D inválida: %v", err)
	}
	runtime := abrirBasePreparacionDecisionCobertura(
		t,
		ctx,
		dsn,
		baseDatos,
		login,
	)
	defer runtime.Close()
	if err := runtime.QueryRow(ctx, `
		SELECT count(*) FROM
		  vec_contratacion_temporal.consumo_cobertura_lote`).Scan(
		new(int),
	); err == nil {
		t.Fatal("el runtime leyó el lote C1 interno")
	}
	if err := runtime.QueryRow(ctx, `
		SELECT count(*) FROM
		  vec_contratacion_temporal
		    .persistir_lote_consumo_c1_cobertura_o404d_v1(
		      '{}'::jsonb,'{}'::jsonb
		    )`).Scan(new(int)); codigoPostgreSQLO404D(err) != "42501" {
		t.Fatalf("runtime pudo fabricar p_resultado_vec: %v", err)
	}
	contenido, err := os.ReadFile(rutaMigracionPreparacionDecisionCobertura(
		t,
		raiz,
		"000022",
		"down",
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, string(contenido)); err == nil {
		t.Fatal("000022 down ignoró la barrera v4")
	}
	ejecutarMigracionConsumoC1O404D(
		t,
		ctx,
		admin,
		raiz,
		"000024",
		"down",
	)
	var esquemaPresente bool
	if err := admin.QueryRow(ctx, `
		SELECT c.version_esquema,
		       to_regclass(
		         'vec_contratacion_temporal.consumo_cobertura_lote'
		       ) IS NOT NULL
		  FROM vec_contratacion_temporal.control_migracion_cobertura_o4 c
		 WHERE c.control`).Scan(&version, &esquemaPresente); err != nil ||
		version != 3 || !esquemaPresente {
		t.Fatalf("down 000024 no restauró corte intermedio: %d/%t/%v",
			version, esquemaPresente, err)
	}
	if _, err := admin.Exec(ctx, string(contenido)); err == nil {
		t.Fatal("000022 down ignoró dependencias del esquema 000023")
	}
	ejecutarMigracionConsumoC1O404D(
		t,
		ctx,
		admin,
		raiz,
		"000023",
		"down",
	)
	if err := admin.QueryRow(ctx, `
		SELECT to_regclass(
		  'vec_contratacion_temporal.consumo_cobertura_lote'
		) IS NULL`).Scan(&esquemaPresente); err != nil || !esquemaPresente {
		t.Fatalf("down 000023 incompleto: %t/%v", esquemaPresente, err)
	}
	for _, numero := range []string{"000023", "000024"} {
		ejecutarMigracionConsumoC1O404D(
			t,
			ctx,
			admin,
			raiz,
			numero,
			"up",
		)
	}
}

func transaccionPropietarioO404D(
	ctx context.Context,
	pool *pgxpool.Pool,
	operacion func(pgx.Tx) error,
) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `
		SET LOCAL statement_timeout='15s';
		SET LOCAL idle_in_transaction_session_timeout='20s';
		SET LOCAL ROLE vec_contratacion_temporal_propietario`); err != nil {
		return err
	}
	if err := operacion(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func cargaJSONO404D(t *testing.T, valor map[string]any) []byte {
	t.Helper()
	contenido, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func prevalidarLoteO404D(
	ctx context.Context,
	tx pgx.Tx,
	carga []byte,
) (string, string, int, error) {
	var estado, huella string
	var total int
	err := tx.QueryRow(ctx, `
		SELECT estado,lote_huella_sha256,numero_evidencias
		  FROM vec_contratacion_temporal
		       .prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(
		         $1::jsonb
		       )`, carga).Scan(&estado, &huella, &total)
	return estado, huella, total, err
}

func persistirLoteO404D(
	ctx context.Context,
	tx pgx.Tx,
	carga []byte,
	resultado []byte,
) (string, error) {
	var estado string
	err := tx.QueryRow(ctx, `
		SELECT estado
		  FROM vec_contratacion_temporal
		       .persistir_lote_consumo_c1_cobertura_o404d_v1(
		         $1::jsonb,$2::jsonb
		       )`, carga, resultado).Scan(&estado)
	return estado, err
}

func probarCanonesYLimitesO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
) {
	t.Helper()
	for _, cantidad := range []int{1, 512} {
		lote, err := nuevoLoteFixtureConsumoC1O404D(
			cantidad,
			time.Now().UTC().Add(-100*time.Millisecond),
		)
		if err != nil {
			t.Fatal(err)
		}
		carga := cargaJSONO404D(t, lote)
		var huellaEvidenciaSQL string
		if err := admin.QueryRow(ctx, `
			SELECT coalesce(encode(
			  sha256(
			    vec_contratacion_temporal
			      .o404d_material_evidencia_v1(
			        $1::jsonb -> 'evidencias' -> 0
			      )
			  ),
			  'hex'
			),'NULL')`, carga).Scan(&huellaEvidenciaSQL); err != nil {
			t.Fatal(err)
		}
		primera := lote["evidencias"].([]any)[0].(map[string]any)
		if huellaEvidenciaSQL != primera["evidencia_huella_sha256"] {
			t.Fatalf(
				"canon SQL de evidencia divergente para %d: %s != %s",
				cantidad,
				huellaEvidenciaSQL,
				primera["evidencia_huella_sha256"],
			)
		}
		var huellaSQL string
		if err := admin.QueryRow(ctx, `
			SELECT coalesce(encode(sha256(
			  vec_contratacion_temporal.o404d_material_lote_v1($1::jsonb)
			),'hex'),'NULL')`, carga).Scan(&huellaSQL); err != nil {
			t.Fatal(err)
		}
		if huellaSQL != lote["lote_huella_sha256"] {
			t.Fatalf(
				"canon Go/SQL divergente para %d: %s != %s",
				cantidad,
				huellaSQL,
				lote["lote_huella_sha256"],
			)
		}
		err = transaccionPropietarioO404D(
			ctx,
			admin,
			func(tx pgx.Tx) error {
				estado, _, total, err := prevalidarLoteO404D(ctx, tx, carga)
				if err == nil && (estado != "nueva" || total != cantidad) {
					return fmt.Errorf("prevalidación %d: %s/%d", cantidad, estado, total)
				}
				return err
			},
		)
		if err != nil {
			t.Fatalf("lote %d rechazado: %v", cantidad, err)
		}
	}
	compresible, err := nuevoLoteFixtureConsumoC1O404D(
		74,
		time.Now().UTC().Add(-100*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	bloque := strings.Repeat("61", 65536)
	for _, cruda := range compresible["evidencias"].([]any) {
		evidencia := cruda.(map[string]any)
		for _, clave := range []string{
			"peticion_canon_hex",
			"resultado_canon_hex",
			"atestacion_canon_hex",
			"confirmacion_tcb_canon_hex",
			"catalogo_canon_hex",
			"verificador_canon_hex",
			"resumen_canon_hex",
		} {
			evidencia[clave] = bloque
		}
	}
	cargaCompresible := cargaJSONO404D(t, compresible)
	if len(cargaCompresible) <= 64*1024*1024 {
		t.Fatalf("vector compresible insuficiente: %d bytes", len(cargaCompresible))
	}
	if err := transaccionPropietarioO404D(
		ctx,
		admin,
		func(tx pgx.Tx) error {
			_, _, _, err := prevalidarLoteO404D(
				ctx,
				tx,
				cargaCompresible,
			)
			return err
		},
	); err == nil {
		t.Fatal("carga lógica >64 MiB altamente compresible aceptada")
	}
	base, _ := nuevoLoteFixtureConsumoC1O404D(
		2,
		time.Now().UTC().Add(-100*time.Millisecond),
	)
	for nombre, mutar := range map[string]func(map[string]any){
		"cero": func(lote map[string]any) {
			lote["evidencias"] = []any{}
		},
		"513": func(lote map[string]any) {
			e := lote["evidencias"].([]any)[0]
			lote["evidencias"] = make([]any, 513)
			for i := range 513 {
				lote["evidencias"].([]any)[i] = e
			}
		},
		"prueba": func(lote map[string]any) {
			e := lote["evidencias"].([]any)[0].(map[string]any)
			e["atestacion_canon_hex"] = hex.EncodeToString([]byte("mutada"))
		},
		"posiciones_fuera_de_orden": func(lote map[string]any) {
			evidencias := lote["evidencias"].([]any)
			evidencias[0], evidencias[1] = evidencias[1], evidencias[0]
			resellarLoteFixtureO404D(lote)
		},
		"peticion_duplicada": func(lote map[string]any) {
			evidencias := lote["evidencias"].([]any)
			primera := evidencias[0].(map[string]any)
			segunda := evidencias[1].(map[string]any)
			segunda["peticion_ref"] = primera["peticion_ref"]
			resellarLoteFixtureO404D(lote)
		},
		"respuesta_duplicada": func(lote map[string]any) {
			evidencias := lote["evidencias"].([]any)
			primera := evidencias[0].(map[string]any)
			segunda := evidencias[1].(map[string]any)
			segunda["autoridad_ref"] = primera["autoridad_ref"]
			segunda["generacion"] = primera["generacion"]
			segunda["recibo_respuesta_ref"] =
				primera["recibo_respuesta_ref"]
			resellarLoteFixtureO404D(lote)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			lote := clonarLoteFixtureConsumoC1O404D(t, base)
			mutar(lote)
			err := transaccionPropietarioO404D(
				ctx,
				admin,
				func(tx pgx.Tx) error {
					_, _, _, err := prevalidarLoteO404D(
						ctx,
						tx,
						cargaJSONO404D(t, lote),
					)
					return err
				},
			)
			if err == nil {
				t.Fatal("mutación C1 aceptada")
			}
		})
	}
}

func aislarLoteFixtureO404D(
	t *testing.T,
	lote map[string]any,
	sufijo string,
) map[string]any {
	t.Helper()
	lote = clonarLoteFixtureConsumoC1O404D(t, lote)
	for _, clave := range []string{
		"lote_ref", "preparacion_c1_ref", "decision_vec_ref",
		"correlacion_vec_ref",
	} {
		lote[clave] = lote[clave].(string) + ":" + sufijo
	}
	orden := sha256.Sum256([]byte("orden:" + sufijo))
	lote["huella_orden_sha256"] = hex.EncodeToString(orden[:])
	preparacion := sha256.Sum256([]byte("preparacion:" + sufijo))
	lote["preparacion_c1_huella_sha256"] = hex.EncodeToString(
		preparacion[:],
	)
	for _, cruda := range lote["evidencias"].([]any) {
		evidencia := cruda.(map[string]any)
		for _, clave := range []string{
			"peticion_ref", "autoridad_ref", "recibo_respuesta_ref",
			"verificador_ref",
		} {
			evidencia[clave] = evidencia[clave].(string) + ":" + sufijo
		}
	}
	resellarLoteFixtureO404D(lote)
	return lote
}

func resellarLoteFixtureO404D(lote map[string]any) {
	for _, cruda := range lote["evidencias"].([]any) {
		evidencia := cruda.(map[string]any)
		canon := canonEvidenciaFixtureConsumoC1O404D(evidencia)
		huella := sha256.Sum256(canon)
		evidencia["evidencia_huella_sha256"] = hex.EncodeToString(huella[:])
	}
	canon := canonLoteFixtureConsumoC1O404D(lote)
	huella := sha256.Sum256(canon)
	lote["lote_huella_sha256"] = hex.EncodeToString(huella[:])
}

func probarPersistenciaReplayRollbackO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
) {
	t.Helper()
	lote, _ := nuevoLoteFixtureConsumoC1O404D(
		1,
		time.Now().UTC().Add(-100*time.Millisecond),
	)
	carga := cargaJSONO404D(t, lote)
	resultado := cargaJSONO404D(t, resultadoVECFixtureO404D(lote, ""))
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `
		SET LOCAL statement_timeout='15s';
		SET LOCAL idle_in_transaction_session_timeout='20s';
		SET LOCAL ROLE vec_contratacion_temporal_propietario`); err != nil {
		t.Fatal(err)
	}
	if estado, err := persistirLoteO404D(
		ctx,
		tx,
		carga,
		resultado,
	); err != nil || estado != "persistida" {
		t.Fatalf("persistencia previa a rollback: %s / %v", estado, err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var lotes int
	if err := admin.QueryRow(ctx, `
		SELECT count(*) FROM
		  vec_contratacion_temporal.consumo_cobertura_lote`).Scan(
		&lotes,
	); err != nil || lotes != 0 {
		t.Fatalf("rollback C1 incompleto: %d / %v", lotes, err)
	}
	for _, caso := range []struct {
		esperado string
		repetir  bool
	}{
		{esperado: "persistida", repetir: false},
		{esperado: "repetida", repetir: true},
	} {
		err := transaccionPropietarioO404D(
			ctx,
			admin,
			func(tx pgx.Tx) error {
				estado, err := persistirLoteO404D(ctx, tx, carga, resultado)
				if err == nil && estado != caso.esperado {
					return fmt.Errorf(
						"estado %q, esperado %q",
						estado,
						caso.esperado,
					)
				}
				return err
			},
		)
		if err != nil {
			t.Fatalf("replay=%t: %v", caso.repetir, err)
		}
	}
	resultadoMutado := cargaJSONO404D(
		t,
		resultadoVECFixtureO404D(lote, "mutada"),
	)
	err = transaccionPropietarioO404D(
		ctx,
		admin,
		func(tx pgx.Tx) error {
			_, err := persistirLoteO404D(ctx, tx, carga, resultadoMutado)
			return err
		},
	)
	if codigoPostgreSQLO404D(err) != "23505" {
		t.Fatalf("replay VEC mutado no falló cerrado: %v", err)
	}
	down, err := os.ReadFile(rutaMigracionConsumoC1O404D(
		t,
		raizRepositorioPreparacionDecisionCobertura(t),
		"000024",
		"down",
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, string(down)); err == nil {
		t.Fatal("down O4-04D destruyó historia C1")
	}
}

func probarConcurrenciaYReaperturaO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	dsn string,
	baseDatos string,
) {
	t.Helper()
	base, _ := nuevoLoteFixtureConsumoC1O404D(
		2,
		time.Now().UTC().Add(-100*time.Millisecond),
	)
	lote := aislarLoteFixtureO404D(t, base, "concurrencia")
	carga := cargaJSONO404D(t, lote)
	resultado := cargaJSONO404D(t, resultadoVECFixtureO404D(lote, ""))
	type salida struct {
		estado string
		err    error
	}
	salidas := make(chan salida, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var estado string
			err := transaccionPropietarioO404D(
				ctx,
				admin,
				func(tx pgx.Tx) error {
					var err error
					estado, err = persistirLoteO404D(ctx, tx, carga, resultado)
					return err
				},
			)
			salidas <- salida{estado: estado, err: err}
		}()
	}
	wg.Wait()
	close(salidas)
	exitos := 0
	for salida := range salidas {
		if salida.err == nil {
			exitos++
			if salida.estado != "persistida" && salida.estado != "repetida" {
				t.Fatalf("estado concurrente inesperado: %#v", salida)
			}
			continue
		}
		if codigoPostgreSQLO404D(salida.err) != "40001" {
			t.Fatalf("error concurrente inesperado: %v", salida.err)
		}
	}
	if exitos == 0 {
		t.Fatal("ninguna transacción concurrente persistió")
	}
	reabierta := abrirBasePreparacionDecisionCobertura(
		t,
		ctx,
		dsn,
		baseDatos,
		"postgres",
	)
	defer reabierta.Close()
	var evidencias int
	if err := reabierta.QueryRow(ctx, `
		SELECT count(*)
		  FROM vec_contratacion_temporal.consumo_cobertura_evidencia
		 WHERE lote_ref=$1`, lote["lote_ref"]).Scan(&evidencias); err != nil ||
		evidencias != 2 {
		t.Fatalf("reapertura C1 divergente: %d / %v", evidencias, err)
	}
}

func codigoPostgreSQLO404D(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return ""
	}
	return pgErr.Code
}
