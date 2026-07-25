package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	variableDSNPreparacionDecisionCoberturaPG18         = "VEC_O404C_ADMIN_DSN"
	variableAislamientoPreparacionDecisionCoberturaPG18 = "VEC_O404C_PG18_AISLADO"
	variableRaizPreparacionDecisionCoberturaPG18        = "VEC_O404C_RAIZ"
)

func TestPreparacionDecisionCoberturaDurablePostgreSQL18Real(
	t *testing.T,
) {
	dsn := os.Getenv(variableDSNPreparacionDecisionCoberturaPG18)
	if dsn == "" {
		t.Skip("PostgreSQL 18 efímero O4-04C no solicitado")
	}
	if os.Getenv(variableAislamientoPreparacionDecisionCoberturaPG18) != "1" {
		t.Fatal("la suite O4-04C exige un PostgreSQL 18 efímero aislado")
	}
	raiz := raizRepositorioPreparacionDecisionCobertura(t)
	ctx, cancelar := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer cancelar()
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
		t.Fatalf(
			"entorno aislado inválido: PostgreSQL=%d base=%q",
			version,
			baseDatos,
		)
	}
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())
	basePrincipal := "vec_o404c_principal_" + sufijo
	baseCiclo := "vec_o404c_ciclo_" + sufijo
	login := "vec_o404c_runtime_" + sufijo
	asegurarClusterLimpioPreparacionDecisionCobertura(
		t,
		ctx,
		base,
		basePrincipal,
		baseCiclo,
	)
	crearBasePreparacionDecisionCobertura(t, ctx, base, basePrincipal)
	adminPrincipal := abrirBasePreparacionDecisionCobertura(
		t,
		ctx,
		dsn,
		basePrincipal,
		"postgres",
	)
	ejecutarArchivoPreparacionDecisionCobertura(
		t,
		ctx,
		adminPrincipal,
		filepath.Join(
			raiz,
			"deploy/postgresql/contratacion_temporal/roles_up.sql",
		),
	)
	crearLoginPreparacionDecisionCobertura(
		t,
		ctx,
		base,
		login,
	)
	crearBasePreparacionDecisionCobertura(t, ctx, base, baseCiclo)
	adminCiclo := abrirBasePreparacionDecisionCobertura(
		t,
		ctx,
		dsn,
		baseCiclo,
		"postgres",
	)
	defer func() {
		adminCiclo.Close()
		adminPrincipal.Close()
		eliminarBasePreparacionDecisionCobertura(t, base, baseCiclo)
		eliminarBasePreparacionDecisionCobertura(t, base, basePrincipal)
		eliminarLoginYRolesPreparacionDecisionCobertura(t, base, login)
	}()
	inicializarEsquemaSecundarioPreparacionDecisionCobertura(
		t,
		ctx,
		adminCiclo,
	)
	instalarBaseHastaO404BPreparacionDecisionCobertura(
		t,
		ctx,
		adminPrincipal,
		raiz,
	)
	instalarBaseHastaO404BPreparacionDecisionCobertura(
		t,
		ctx,
		adminCiclo,
		raiz,
	)
	probarCicloMigracionesPreparacionDecisionCobertura(
		t,
		ctx,
		adminCiclo,
		raiz,
	)
	for _, numero := range []string{"000020", "000021", "000022"} {
		ejecutarMigracionPreparacionDecisionCobertura(
			t,
			ctx,
			adminPrincipal,
			raiz,
			numero,
			"up",
		)
	}
	fixture := nuevoFixturePreparacionDecisionCoberturaDurable(t)
	sembrarAnalisisO3PreparacionDecisionCobertura(
		t,
		ctx,
		adminPrincipal,
		fixture,
	)
	runtime := abrirBasePreparacionDecisionCobertura(
		t,
		ctx,
		dsn,
		basePrincipal,
		login,
	)
	defer runtime.Close()
	preparador, err :=
		NuevoPreparadorOperacionDecisionCoberturaDurablePostgreSQL(runtime)
	if err != nil {
		t.Fatal(err)
	}
	probarReservaCercadaPreparacionDecisionCobertura(
		t,
		ctx,
		preparador,
		fixture,
	)
	consultaConcedida := probarReplayRotadoPreparacionDecisionCobertura(
		t,
		ctx,
		adminPrincipal,
		preparador,
		fixture,
	)
	probarReplayDenegadoPreparacionDecisionCobertura(
		t,
		ctx,
		adminPrincipal,
		preparador,
		fixture,
	)
	probarColisionPreparacionDecisionCobertura(
		t,
		ctx,
		preparador,
		fixture,
	)
	probarCorrupcionPreparacionDecisionCobertura(
		t,
		ctx,
		adminPrincipal,
		preparador,
		fixture,
	)
	probarACLRLSPreparacionDecisionCobertura(
		t,
		ctx,
		adminPrincipal,
	)
	probarBarreraPreparacionDecisionCobertura(
		t,
		ctx,
		adminPrincipal,
		preparador,
		consultaConcedida,
	)
}

func raizRepositorioPreparacionDecisionCobertura(t *testing.T) string {
	t.Helper()
	if raiz := os.Getenv(variableRaizPreparacionDecisionCoberturaPG18); raiz != "" {
		return raiz
	}
	raiz, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "go.mod")); err != nil {
		t.Fatalf(
			"no se localizó la raíz; defina %s: %v",
			variableRaizPreparacionDecisionCoberturaPG18,
			err,
		)
	}
	return raiz
}

func asegurarClusterLimpioPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	base *pgxpool.Pool,
	bases ...string,
) {
	t.Helper()
	var existe bool
	if err := base.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1
		    FROM pg_catalog.pg_roles
		   WHERE rolname = ANY(ARRAY[
		     'vec_contratacion_temporal_propietario',
		     'vec_contratacion_temporal_migrador',
		     'vec_contratacion_temporal_ejecutor',
		     'vec_contratacion_temporal_gobernador'
		   ])
		)`).Scan(&existe); err != nil {
		t.Fatal(err)
	}
	if existe {
		t.Fatal("el cluster efímero contiene roles técnicos previos")
	}
	for _, nombre := range bases {
		if err := base.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM pg_catalog.pg_database WHERE datname=$1
			)`, nombre).Scan(&existe); err != nil {
			t.Fatal(err)
		}
		if existe {
			t.Fatalf("base efímera previa inesperada: %s", nombre)
		}
	}
}

func crearBasePreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	base *pgxpool.Pool,
	nombre string,
) {
	t.Helper()
	if _, err := base.Exec(
		ctx,
		"CREATE DATABASE "+pgx.Identifier{nombre}.Sanitize()+
			" TEMPLATE template0",
	); err != nil {
		t.Fatal(err)
	}
}

func eliminarBasePreparacionDecisionCobertura(
	t *testing.T,
	base *pgxpool.Pool,
	nombre string,
) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if _, err := base.Exec(
		ctx,
		"DROP DATABASE IF EXISTS "+pgx.Identifier{nombre}.Sanitize()+
			" WITH (FORCE)",
	); err != nil {
		t.Errorf("retirar base efímera %s: %v", nombre, err)
	}
}

func abrirBasePreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	dsn string,
	baseDatos string,
	usuario string,
) *pgxpool.Pool {
	t.Helper()
	configuracion, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	configuracion.ConnConfig.Database = baseDatos
	configuracion.ConnConfig.User = usuario
	configuracion.ConnConfig.Password = ""
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	return pool
}

func crearLoginPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	base *pgxpool.Pool,
	login string,
) {
	t.Helper()
	identificador := pgx.Identifier{login}.Sanitize()
	_, err := base.Exec(ctx, `
		CREATE ROLE `+identificador+`
		  LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
		  NOREPLICATION NOBYPASSRLS;
		GRANT vec_contratacion_temporal_ejecutor TO `+identificador+`
		  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE`)
	if err != nil {
		t.Fatal(err)
	}
}

func eliminarLoginYRolesPreparacionDecisionCobertura(
	t *testing.T,
	base *pgxpool.Pool,
	login string,
) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	_, err := base.Exec(ctx, `
		DROP ROLE IF EXISTS `+pgx.Identifier{login}.Sanitize()+`;
		DROP ROLE IF EXISTS vec_contratacion_temporal_gobernador;
		DROP ROLE IF EXISTS vec_contratacion_temporal_ejecutor;
		DROP ROLE IF EXISTS vec_contratacion_temporal_migrador;
		DROP ROLE IF EXISTS vec_contratacion_temporal_propietario`)
	if err != nil {
		t.Errorf("retirar roles efímeros: %v", err)
	}
}

func inicializarEsquemaSecundarioPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		CREATE SCHEMA vec_contratacion_temporal
		  AUTHORIZATION vec_contratacion_temporal_propietario;
		REVOKE ALL ON SCHEMA vec_contratacion_temporal FROM PUBLIC;
		GRANT CONNECT ON DATABASE `+
		pgx.Identifier{pool.Config().ConnConfig.Database}.Sanitize()+`
		  TO vec_contratacion_temporal_migrador,
		     vec_contratacion_temporal_ejecutor,
		     vec_contratacion_temporal_gobernador`)
	if err != nil {
		t.Fatal(err)
	}
}

func ejecutarArchivoPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	ruta string,
) {
	t.Helper()
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(contenido)); err != nil {
		t.Fatalf("ejecutar %s: %v", filepath.Base(ruta), err)
	}
}

func instalarBaseHastaO404BPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	raiz string,
) {
	t.Helper()
	patron := filepath.Join(
		raiz,
		"deploy/postgresql/contratacion_temporal/migraciones/*.up.sql",
	)
	rutas, err := filepath.Glob(patron)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(rutas)
	for _, ruta := range rutas {
		base := filepath.Base(ruta)
		if base[:6] > "000019" {
			continue
		}
		if strings.HasPrefix(base, "000003_") {
			_, err := pool.Exec(ctx, `
				CREATE SCHEMA vec_autorizacion_atestada_v3;
				CREATE FUNCTION
				vec_autorizacion_atestada_v3
				  .registrar_y_consumir_decision_v3_atestada(
				    bytea,bytea,bytea,bytea,numeric,numeric,
				    bytea,bytea,bytea,bytea
				  )
				RETURNS void LANGUAGE plpgsql
				SET search_path=pg_catalog AS $$
				BEGIN RETURN; END
				$$;
				GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
				  TO vec_contratacion_temporal_propietario;
				GRANT EXECUTE ON FUNCTION
				  vec_autorizacion_atestada_v3
				    .registrar_y_consumir_decision_v3_atestada(
				      bytea,bytea,bytea,bytea,numeric,numeric,
				      bytea,bytea,bytea,bytea
				    )
				  TO vec_contratacion_temporal_propietario`)
			if err != nil {
				t.Fatal(err)
			}
		}
		if strings.HasPrefix(base, "000012_") {
			_, err := pool.Exec(ctx, `
				CREATE SCHEMA vec_autorizacion;
				CREATE FUNCTION vec_autorizacion
				  .revalidar_decision_analisis_contratacion_temporal_v1(
				    bytea,bytea,numeric,numeric,jsonb
				  )
				RETURNS TABLE (
				  revalidada_en timestamptz,
				  decision_ref text,
				  decision_huella_sha256 text
				)
				LANGUAGE sql SET search_path=pg_catalog AS $$
				  SELECT NULL::timestamptz,NULL::text,NULL::text
				   WHERE false
				$$;
				GRANT USAGE ON SCHEMA vec_autorizacion
				  TO vec_contratacion_temporal_propietario;
				GRANT EXECUTE ON FUNCTION vec_autorizacion
				  .revalidar_decision_analisis_contratacion_temporal_v1(
				    bytea,bytea,numeric,numeric,jsonb
				  )
				  TO vec_contratacion_temporal_propietario`)
			if err != nil {
				t.Fatal(err)
			}
		}
		ejecutarArchivoPreparacionDecisionCobertura(
			t,
			ctx,
			pool,
			ruta,
		)
	}
}

func rutaMigracionPreparacionDecisionCobertura(
	t *testing.T,
	raiz string,
	numero string,
	direccion string,
) string {
	t.Helper()
	rutas, err := filepath.Glob(filepath.Join(
		raiz,
		"deploy/postgresql/contratacion_temporal/migraciones/"+
			numero+"_*o4_04c."+direccion+".sql",
	))
	if err != nil || len(rutas) != 1 {
		t.Fatalf(
			"migración %s %s no unívoca: %v / %v",
			numero,
			direccion,
			rutas,
			err,
		)
	}
	return rutas[0]
}

func ejecutarMigracionPreparacionDecisionCobertura(
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
		rutaMigracionPreparacionDecisionCobertura(
			t,
			raiz,
			numero,
			direccion,
		),
	)
}

func probarCicloMigracionesPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	raiz string,
) {
	t.Helper()
	ejecutarMigracionPreparacionDecisionCobertura(
		t, ctx, pool, raiz, "000020", "up",
	)
	var version int
	var lectorAusente, preparadorAusente bool
	if err := pool.QueryRow(ctx, `
		SELECT version_esquema,
		       pg_catalog.to_regprocedure(
		         'vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(jsonb,jsonb)'
		       ) IS NULL,
		       pg_catalog.to_regprocedure(
		         'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)'
		       ) IS NULL
		  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
		 WHERE control`).Scan(
		&version,
		&preparadorAusente,
		&lectorAusente,
	); err != nil || version != 1 || !preparadorAusente || !lectorAusente {
		t.Fatalf(
			"estado posterior a 000020 inseguro: v=%d preparador_ausente=%t lector_ausente=%t err=%v",
			version,
			preparadorAusente,
			lectorAusente,
			err,
		)
	}
	ejecutarMigracionPreparacionDecisionCobertura(
		t, ctx, pool, raiz, "000021", "up",
	)
	var ejecutaPreparador, lectorPrimarioAusente, ayudaNoPublica bool
	if err := pool.QueryRow(ctx, `
		SELECT version_esquema,
		       pg_catalog.has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',
		         'vec_contratacion_temporal.preparar_operacion_decision_cobertura_v1(jsonb,jsonb)',
		         'EXECUTE'
		       ),
		       pg_catalog.to_regprocedure(
		         'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)'
		       ) IS NULL,
		       pg_catalog.to_regprocedure(
		         'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(jsonb)'
		       ) IS NULL,
		       NOT pg_catalog.has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',
		         'vec_contratacion_temporal.o404c_carga_terminal_v1(text)',
		         'EXECUTE'
		       )
		  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
		 WHERE control`).Scan(
		&version,
		&ejecutaPreparador,
		&lectorAusente,
		&lectorPrimarioAusente,
		&ayudaNoPublica,
	); err != nil || version != 2 || !ejecutaPreparador ||
		!lectorAusente || !lectorPrimarioAusente || !ayudaNoPublica {
		t.Fatalf(
			"grant prematuro O4-04C: v=%d prep=%t lector_ausente=%t primario_ausente=%t ayuda_cerrada=%t err=%v",
			version,
			ejecutaPreparador,
			lectorAusente,
			lectorPrimarioAusente,
			ayudaNoPublica,
			err,
		)
	}
	ejecutarMigracionPreparacionDecisionCobertura(
		t, ctx, pool, raiz, "000022", "up",
	)
	var ejecutaLector, ejecutaPrimario bool
	if err := pool.QueryRow(ctx, `
		SELECT version_esquema,
		       pg_catalog.has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',
		         'vec_contratacion_temporal.consultar_operacion_decision_cobertura_confirmada_v1(jsonb)',
		         'EXECUTE'
		       ),
		       pg_catalog.has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',
		         'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404c_v1(jsonb)',
		         'EXECUTE'
		       )
		  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
		 WHERE control`).Scan(
		&version,
		&ejecutaLector,
		&ejecutaPrimario,
	); err != nil || version != 3 || !ejecutaLector || ejecutaPrimario {
		t.Fatalf(
			"ACL de lectores O4-04C insegura: v=%d lector=%t primario=%t err=%v",
			version,
			ejecutaLector,
			ejecutaPrimario,
			err,
		)
	}
	contenido, err := os.ReadFile(rutaMigracionPreparacionDecisionCobertura(
		t, raiz, "000021", "down",
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(contenido)); err == nil {
		t.Fatal("000021 down alteró un esquema todavía en versión 3")
	}
	if err := pool.QueryRow(ctx, `
		SELECT version_esquema
		  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
		 WHERE control`).Scan(&version); err != nil || version != 3 {
		t.Fatalf("000021 down parcial: v=%d err=%v", version, err)
	}
	ejecutarMigracionPreparacionDecisionCobertura(
		t, ctx, pool, raiz, "000022", "down",
	)
	ejecutarMigracionPreparacionDecisionCobertura(
		t, ctx, pool, raiz, "000021", "down",
	)
	ejecutarMigracionPreparacionDecisionCobertura(
		t, ctx, pool, raiz, "000020", "down",
	)
	var objetosAusentes bool
	if err := pool.QueryRow(ctx, `
		SELECT pg_catalog.to_regclass(
		  'vec_contratacion_temporal.reserva_operacion_decision_cobertura'
		) IS NULL`).Scan(&objetosAusentes); err != nil || !objetosAusentes {
		t.Fatalf("down vacío O4-04C incompleto: %t / %v", objetosAusentes, err)
	}
	ejecutarMigracionPreparacionDecisionCobertura(
		t, ctx, pool, raiz, "000020", "up",
	)
	_, err = pool.Exec(ctx, `
		BEGIN;
		SET LOCAL session_replication_role = 'replica';
		SET LOCAL ROLE vec_contratacion_temporal_propietario;
		INSERT INTO
		  vec_contratacion_temporal.reserva_operacion_decision_cobertura (
		    ambito_raiz_hmac,reserva_ref,recibo_ref,actuacion_ref,
		    auditoria_ref,evento_ref,correlacion_vec_ref,decision_vec_ref,
		    organizacion_ref,expediente_ref,version_expediente,
		    analisis_ref,analisis_huella_sha256,
		    huella_semantica_raiz_hmac,creada_en
		  ) VALUES (
		    'hmac-sha256:vec.contratacion-temporal.cobertura-decision.ambito/v1:'
		      || pg_catalog.repeat('a',64),
		    'reserva:o404c:down','recibo:o404c:down',
		    'actuacion:o404c:down','auditoria:o404c:down',
		    'evento:o404c:down','correlacion:o404c:down',
		    'decision:o404c:down','organizacion:o404c:down',
		    'expediente:o404c:down',2,'analisis:o404c:down',
		    pg_catalog.repeat('b',64),
		    'hmac-sha256:vec.contratacion-temporal.cobertura-decision.semantica/v1:'
		      || pg_catalog.repeat('c',64),
		    pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
		  );
		COMMIT`)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err = os.ReadFile(rutaMigracionPreparacionDecisionCobertura(
		t, raiz, "000020", "down",
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(contenido)); err == nil {
		t.Fatal("000020 down destruyó una reserva durable")
	}
	if err := pool.QueryRow(ctx, `
		SELECT version_esquema
		  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
		 WHERE control`).Scan(&version); err != nil || version != 1 {
		t.Fatalf("down con datos dejó estado parcial: v=%d err=%v", version, err)
	}
}
