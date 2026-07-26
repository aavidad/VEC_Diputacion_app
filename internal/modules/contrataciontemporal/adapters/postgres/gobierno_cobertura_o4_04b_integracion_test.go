package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestGobiernoCoberturaO404BPostgreSQL18Real(t *testing.T) {
	dsnPublicador := os.Getenv("VEC_O404B_PUBLICADOR_DSN")
	dsnEjecutor := os.Getenv("VEC_O404B_EJECUTOR_DSN")
	dsnAdministrador := os.Getenv("VEC_O404B_ADMIN_DSN")
	if dsnPublicador == "" || dsnEjecutor == "" || dsnAdministrador == "" {
		t.Skip("PostgreSQL 18 efímero O4-04B no solicitado")
	}
	if os.Getenv("VEC_O404B_PG18_AISLADO") != "1" {
		t.Fatal("la suite O4-04B exige un PostgreSQL 18 efímero aislado")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	publicador, err := pgxpool.New(ctx, dsnPublicador)
	if err != nil {
		t.Fatal(err)
	}
	defer publicador.Close()
	ejecutor, err := pgxpool.New(ctx, dsnEjecutor)
	if err != nil {
		t.Fatal(err)
	}
	defer ejecutor.Close()
	administrador, err := pgxpool.New(ctx, dsnAdministrador)
	if err != nil {
		t.Fatal(err)
	}
	defer administrador.Close()
	var versionMayor int
	if err := ejecutor.QueryRow(
		ctx,
		"SELECT current_setting('server_version_num')::integer / 10000",
	).Scan(&versionMayor); err != nil || versionMayor != 18 {
		t.Fatalf("requiere PostgreSQL 18, obtenido %d: %v", versionMayor, err)
	}

	fixture := nuevoFixtureGobiernoCoberturaO404B(
		t,
		time.Now().UTC().Truncate(time.Microsecond),
	)
	catalogoJSON, politicaJSON, actuacionJSON := fixture.JSON(t)
	publicacion := map[string]any{
		"esquema": "vec.contratacion-temporal." +
			"gobierno-cobertura.o4-04b.v1",
		"secuencia": 1,
		"evento_ref": "evento_gobi_o404b_" +
			"0123456789abcdef0123456789abcdef",
		"catalogo":  json.RawMessage(catalogoJSON),
		"politica":  json.RawMessage(politicaJSON),
		"actuacion": json.RawMessage(actuacionJSON),
	}
	carga, err := json.Marshal(publicacion)
	if err != nil {
		t.Fatal(err)
	}
	verificarCanonesGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		catalogoJSON,
		politicaJSON,
		actuacionJSON,
	)
	verificarBarreraGobiernoCoberturaO404B(t, ctx, administrador)
	verificarDownFueraDeOrdenGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"000017_gobierno_cobertura_o4_04b.down.sql",
	)
	verificarDownFueraDeOrdenGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"000018_politicas_gobierno_cobertura_o4_04b.down.sql",
	)
	verificarBloqueoCheckpointGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		publicador,
		carga,
	)
	var adulterada map[string]any
	if err := json.Unmarshal(carga, &adulterada); err != nil {
		t.Fatal(err)
	}
	adulterada["actuacion"].(map[string]any)["unidad_ejecutora_ref"] =
		"unidad_adulterada_o404b"
	cargaAdulterada, err := json.Marshal(adulterada)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ejecutarPublicacionGobiernoCoberturaO404BReal(
		ctx,
		publicador,
		cargaAdulterada,
	); err == nil {
		t.Fatal("una publicación con canon adulterado fue aceptada")
	}

	type resultadoPublicacion struct {
		resultado string
		evento    string
		huella    string
		err       error
	}
	resultados := make(chan resultadoPublicacion, 2)
	for range 2 {
		go func() {
			var resultado, evento, huella string
			var err error
			for intento := 1; intento <= 3; intento++ {
				resultado, evento, huella, err =
					ejecutarPublicacionGobiernoCoberturaO404BReal(
						ctx,
						publicador,
						carga,
					)
				if err == nil || !errorPostgreSQLReintentable(err) {
					break
				}
			}
			resultados <- resultadoPublicacion{
				resultado: resultado,
				evento:    evento,
				huella:    huella,
				err:       err,
			}
		}()
	}
	primero, segundo := <-resultados, <-resultados
	close(resultados)
	if primero.err != nil || segundo.err != nil ||
		primero.evento != segundo.evento ||
		primero.huella != segundo.huella ||
		!((primero.resultado == "publicada" &&
			segundo.resultado == "repetida") ||
			(primero.resultado == "repetida" &&
				segundo.resultado == "publicada")) {
		t.Fatalf(
			"concurrencia de publicación divergente: %#v / %#v",
			primero,
			segundo,
		)
	}

	resolutor, err := NuevoResolutorGobiernoCoberturaO404BPostgreSQL(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := cobertura.NuevaSolicitudGobiernoDecisionCobertura(
		"organizacion:dipgra",
		"expediente:ct:o404b:01",
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	gobierno, err := cobertura.ObtenerGobiernoOperacionCobertura(
		ctx,
		relojGobiernoCoberturaO404BPrueba{instante: fixture.instante},
		resolutor,
		solicitud,
	)
	if err != nil {
		t.Fatalf("resolver gobierno real: %v", err)
	}
	datos, err := gobierno.DesplegarPara(
		ctx,
		relojGobiernoCoberturaO404BPrueba{
			instante: fixture.instante.Add(time.Second),
		},
		solicitud,
	)
	if err != nil ||
		datos.Catalogo.Identidad() != fixture.catalogo.Identidad() ||
		datos.Politica.Identidad() != fixture.politica.Identidad() ||
		datos.PoliticaActuacion.HuellaSHA256 !=
			fixture.actuacion.HuellaSHA256 {
		t.Fatalf("gobierno real divergente: %#v, %v", datos, err)
	}
	verificarAislamientoGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		ejecutor,
		fixture,
		true,
	)
	var cantidad int
	if err := ejecutor.QueryRow(
		ctx,
		"SELECT count(*) FROM vec_contratacion_temporal.gobi_o404b_catalogo",
	).Scan(&cantidad); err == nil {
		t.Fatal("el ejecutor pudo leer una tabla interna O4-04B")
	}

	retiradaEn := time.Now().UTC().Truncate(time.Microsecond)
	retirada := map[string]any{
		"esquema": "vec.contratacion-temporal." +
			"retirar-gobierno-cobertura.o4-04b.v1",
		"secuencia":               2,
		"evento_ref":              "evento_gobi_o404b_" + strings.Repeat("f", 32),
		"organizacion_ref":        fixture.actuacion.OrganizacionRef,
		"accion":                  fixture.actuacion.Accion,
		"actuacion_ref":           fixture.actuacion.Referencia,
		"actuacion_version":       fixture.actuacion.Version,
		"actuacion_huella_sha256": fixture.actuacion.HuellaSHA256,
		"retirada_en":             retiradaEn,
	}
	cargaRetirada, err := json.Marshal(retirada)
	if err != nil {
		t.Fatal(err)
	}
	resultadoRetirada, eventoRetirada, huellaRetirada :=
		retirarGobiernoCoberturaO404BReal(
			t,
			ctx,
			publicador,
			cargaRetirada,
		)
	if resultadoRetirada != "retirada" ||
		eventoRetirada != retirada["evento_ref"] ||
		len(huellaRetirada) != 64 {
		t.Fatal("retirada durable O4-04B inesperada")
	}
	resultadoRetirada, eventoRetiradaRepetido, huellaRetiradaRepetida :=
		retirarGobiernoCoberturaO404BReal(
			t,
			ctx,
			publicador,
			cargaRetirada,
		)
	if resultadoRetirada != "repetida" ||
		eventoRetiradaRepetido != eventoRetirada ||
		huellaRetiradaRepetida != huellaRetirada {
		t.Fatal("replay de retirada O4-04B divergente")
	}
	_, err = cobertura.ObtenerGobiernoOperacionCobertura(
		ctx,
		relojGobiernoCoberturaO404BPrueba{instante: retiradaEn},
		resolutor,
		solicitud,
	)
	if !errors.Is(err, cobertura.ErrGobiernoOperacionCoberturaNoDisponible) {
		t.Fatalf("una publicación retirada siguió resolviéndose: %v", err)
	}
	verificarAislamientoGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		ejecutor,
		fixture,
		false,
	)
	verificarDownProtegidoGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"000019_resolucion_gobierno_cobertura_o4_04b.down.sql",
	)
	instalarBarreraCPosteriorGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
	)
	verificarDownFueraDeOrdenGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
		"000019_resolucion_gobierno_cobertura_o4_04b.down.sql",
	)
}

func ejecutarPublicacionGobiernoCoberturaO404BReal(
	ctx context.Context,
	pool *pgxpool.Pool,
	carga []byte,
) (string, string, string, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return "", "", "", err
	}
	defer revertirTransaccion(tx)
	if _, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`); err != nil {
		return "", "", "", err
	}
	var resultado, evento, huella string
	if err := tx.QueryRow(
		ctx,
		`SELECT resultado, evento_ref, huella_evento_sha256
		   FROM vec_contratacion_temporal.gobi_o404b_publicar($1::jsonb)`,
		carga,
	).Scan(&resultado, &evento, &huella); err != nil {
		return "", "", "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", "", err
	}
	return resultado, evento, huella, nil
}

func retirarGobiernoCoberturaO404BReal(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	carga []byte,
) (string, string, string) {
	t.Helper()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(tx)
	if _, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`); err != nil {
		t.Fatal(err)
	}
	var resultado, evento, huella string
	if err := tx.QueryRow(ctx, `
		SELECT resultado, evento_ref, huella_evento_sha256
		  FROM vec_contratacion_temporal.gobi_o404b_retirar($1::jsonb)`,
		carga,
	).Scan(&resultado, &evento, &huella); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return resultado, evento, huella
}

func verificarAislamientoGobiernoCoberturaO404B(
	t *testing.T,
	ctx context.Context,
	administrador *pgxpool.Pool,
	ejecutor *pgxpool.Pool,
	fixture fixtureGobiernoCoberturaO404B,
	esperado bool,
) {
	t.Helper()
	const wrapper = "vec_contratacion_temporal." +
		"gobi_o404b_revalidar_prueba_efimera"
	_, err := administrador.Exec(ctx, `
		CREATE OR REPLACE FUNCTION `+wrapper+`(
		    text, text, text, numeric, text, text, numeric, text,
		    text, numeric, text
		) RETURNS boolean
		LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$
		  SELECT vec_contratacion_temporal.gobi_o404b_revalidar_actual(
		    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		$$;
		ALTER FUNCTION `+wrapper+`(
		    text,text,text,numeric,text,text,numeric,text,text,numeric,text
		) OWNER TO vec_contratacion_temporal_propietario;
		REVOKE ALL ON FUNCTION `+wrapper+`(
		    text,text,text,numeric,text,text,numeric,text,text,numeric,text
		) FROM PUBLIC;
		GRANT EXECUTE ON FUNCTION `+wrapper+`(
		    text,text,text,numeric,text,text,numeric,text,text,numeric,text
		) TO vec_contratacion_temporal_ejecutor`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = administrador.Exec(
			context.Background(),
			"DROP FUNCTION IF EXISTS "+wrapper+
				"(text,text,text,numeric,text,text,numeric,text,"+
				"text,numeric,text)",
		)
	}()
	tx, err := ejecutor.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(tx)
	if _, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`); err != nil {
		t.Fatal(err)
	}
	catalogo := fixture.catalogo.Identidad()
	politica := fixture.politica.Identidad()
	actuacion := fixture.actuacion
	var obtenido bool
	err = tx.QueryRow(
		ctx,
		"SELECT "+wrapper+"($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
		actuacion.OrganizacionRef,
		string(actuacion.Accion),
		catalogo.Referencia,
		catalogo.Version,
		catalogo.HuellaSHA256,
		politica.Referencia,
		politica.Version,
		politica.HuellaSHA256,
		actuacion.Referencia,
		actuacion.Version,
		actuacion.HuellaSHA256,
	).Scan(&obtenido)
	if err != nil || obtenido != esperado {
		t.Fatalf(
			"barrera TCB O4-04B inesperada: obtenido=%t esperado=%t err=%v",
			obtenido,
			esperado,
			err,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var fuerzaRLS, aclFuncionesValida, rolGobernadorValido bool
	err = administrador.QueryRow(ctx, `
			SELECT bool_and(c.relrowsecurity AND c.relforcerowsecurity),
			       NOT has_function_privilege('public',
			         'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)',
			         'EXECUTE')
			       AND NOT has_function_privilege('public',
			         'vec_contratacion_temporal.gobi_o404b_retirar(jsonb)',
			         'EXECUTE')
			       AND NOT has_function_privilege('public',
			         'vec_contratacion_temporal.gobi_o404b_resolver(text,text,numeric,text,timestamp with time zone)',
			         'EXECUTE')
			       AND NOT has_function_privilege('public',
			         'vec_contratacion_temporal.gobi_o404b_politica_ligada(jsonb,jsonb)',
			         'EXECUTE')
			       AND has_function_privilege(
			         'vec_contratacion_temporal_gobernador',
			         'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)',
			         'EXECUTE')
			       AND has_function_privilege(
			         'vec_contratacion_temporal_gobernador',
			         'vec_contratacion_temporal.gobi_o404b_retirar(jsonb)',
			         'EXECUTE')
			       AND NOT has_function_privilege(
			         'vec_contratacion_temporal_gobernador',
			         'vec_contratacion_temporal.gobi_o404b_resolver(text,text,numeric,text,timestamp with time zone)',
			         'EXECUTE')
			       AND NOT has_function_privilege(
			         'vec_contratacion_temporal_migrador',
			         'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)',
			         'EXECUTE')
			       AND NOT has_function_privilege(
			         'vec_contratacion_temporal_migrador',
			         'vec_contratacion_temporal.gobi_o404b_retirar(jsonb)',
			         'EXECUTE')
			       AND NOT has_function_privilege(
			         'vec_contratacion_temporal_ejecutor',
			         'vec_contratacion_temporal.gobi_o404b_publicar(jsonb)',
			         'EXECUTE')
			       AND NOT has_function_privilege(
			         'vec_contratacion_temporal_ejecutor',
			         'vec_contratacion_temporal.gobi_o404b_retirar(jsonb)',
			         'EXECUTE'),
			       (
			         SELECT NOT r.rolcanlogin AND NOT r.rolsuper
			                AND NOT r.rolcreaterole AND NOT r.rolcreatedb
			                AND NOT r.rolreplication AND NOT r.rolbypassrls
			                AND NOT pg_has_role(
			                  'vec_contratacion_temporal_gobernador',
			                  'vec_contratacion_temporal_propietario',
			                  'MEMBER')
			                AND has_database_privilege(
			                  'vec_contratacion_temporal_gobernador',
			                  current_database(), 'CONNECT')
			                AND has_schema_privilege(
			                  'vec_contratacion_temporal_gobernador',
			                  'vec_contratacion_temporal', 'USAGE')
			                AND NOT has_table_privilege(
			                  'vec_contratacion_temporal_gobernador',
			                  'vec_contratacion_temporal.gobi_o404b_actual',
			                  'SELECT')
			                AND NOT has_table_privilege(
			                  'vec_contratacion_temporal_gobernador',
			                  'vec_contratacion_temporal.gobi_o404b_actual',
			                  'INSERT')
			                AND NOT has_table_privilege(
			                  'vec_contratacion_temporal_gobernador',
			                  'vec_contratacion_temporal.gobi_o404b_actual',
			                  'UPDATE')
			                AND NOT has_table_privilege(
			                  'vec_contratacion_temporal_gobernador',
			                  'vec_contratacion_temporal.gobi_o404b_actual',
			                  'DELETE')
			           FROM pg_catalog.pg_roles r
			          WHERE r.rolname =
			                'vec_contratacion_temporal_gobernador')
			  FROM pg_catalog.pg_class c
			  JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
			 WHERE n.nspname='vec_contratacion_temporal'
			   AND c.relname LIKE 'gobi_o404b_%'
			   AND c.relkind IN ('r','p')`).Scan(
		&fuerzaRLS,
		&aclFuncionesValida,
		&rolGobernadorValido,
	)
	if err != nil || !fuerzaRLS || !aclFuncionesValida ||
		!rolGobernadorValido {
		t.Fatalf("ACL/RLS O4-04B incompletas: %v", err)
	}
	verificarSuperficieFuncionesGobiernoCoberturaO404B(
		t,
		ctx,
		administrador,
	)
}
