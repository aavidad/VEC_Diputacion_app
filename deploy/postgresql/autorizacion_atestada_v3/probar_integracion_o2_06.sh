#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-o206-pg-${PPID}-${RANDOM}"
directorio_temporal="$(mktemp -d -t vec-o206.XXXXXXXX)"
chmod 700 "${directorio_temporal}"

limpiar() {
    rm -rf "${directorio_temporal}"
    if [[ "${VEC_CONSERVAR_CONTENEDOR_PRUEBA:-}" == '1' ]]; then
        printf 'contenedor conservado para diagnóstico: %s\n' \
            "${contenedor}" >&2
        return
    fi
    docker rm --force --volumes "${contenedor}" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

paso() {
    printf '[O2-06:PostgreSQL] %s\n' "$1"
}

archivo() {
    local usuario="$1"
    local ruta="$2"
    docker exec --interactive "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username "${usuario}" \
        --dbname postgres --file "/repo/${ruta}"
}

sql() {
    local usuario="$1"
    local consulta="$2"
    docker exec "${contenedor}" \
        psql -X --set ON_ERROR_STOP=1 --username "${usuario}" \
        --dbname postgres --command "${consulta}"
}

valor() {
    docker exec "${contenedor}" \
        psql -XAt --set ON_ERROR_STOP=1 --username postgres \
        --dbname postgres --command "$1"
}

esperar_fallo() {
    local descripcion="$1"
    shift
    local salida
    if salida="$("$@" 2>&1)"; then
        printf 'Se esperaba rechazo: %s\n%s\n' \
            "${descripcion}" "${salida}" >&2
        return 1
    fi
    paso "rechazo verificado: ${descripcion}"
}

if [[ ! "${imagen}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
    printf 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256\n' >&2
    exit 64
fi

paso "arranque efímero PostgreSQL 18 sin red: ${imagen}"
docker run --detach --name "${contenedor}" --network none \
    --env POSTGRES_PASSWORD="$(openssl rand -hex 24)" \
    --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
    --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=768m \
    "${imagen}" >/dev/null
for _ in {1..60}; do
    docker exec "${contenedor}" pg_isready --quiet \
        --username postgres --dbname postgres && break
    sleep 1
done
docker exec "${contenedor}" pg_isready --quiet \
    --username postgres --dbname postgres
docker cp "${raiz}/." "${contenedor}:/repo"

paso 'endurecimiento y comprobación de versión'
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
DO $b$
BEGIN
  IF current_setting('server_version_num')::integer / 10000 <> 18 THEN
    RAISE EXCEPTION 'se exige PostgreSQL 18';
  END IF;
END
$b$;
SQL

paso 'dependencias de contexto de actor y autorización nominal V3'
for ruta in \
    deploy/postgresql/contexto_actor_v1/roles_up.sql \
    deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
    deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
    deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql \
    deploy/postgresql/autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
do
    archivo postgres "${ruta}" >/dev/null
done
sql postgres \
    'CREATE ROLE vec_contexto_o206_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contexto_actor_v1_runtime TO vec_contexto_o206_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' \
    >/dev/null
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_contexto_o206_runtime --dbname postgres <<'SQL' >/dev/null
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT count(*)
  FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    'oca_registro_v3_000000000000000000000000',
    'rca_registro_v3_000000000000000000000000',
    'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    'prf_sintetico_cccccccccccccccccccccccc',
    'certificado','alto',clock_timestamp()
  );
COMMIT;
SQL
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
for ruta in \
    deploy/postgresql/autorizacion/roles_up.sql \
    deploy/postgresql/autorizacion/roles_v2_up.sql \
    deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
    deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
    deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
    deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
    deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql \
    deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql \
    deploy/postgresql/autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql \
    deploy/postgresql/autorizacion/pruebas_sql/integracion_contexto_actor_v3.sql
do
    archivo postgres "${ruta}" >/dev/null
done
archivo postgres \
    deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql \
    >/dev/null

paso 'instalación O2-05 autoritativa con identidades separadas'
archivo postgres deploy/postgresql/contratacion_temporal/roles_up.sql >/dev/null
sql postgres \
    'CREATE ROLE vec_ct_o206_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ct_o206_migrador; GRANT vec_contratacion_temporal_migrador TO vec_ct_o206_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE' \
    >/dev/null
archivo vec_ct_o206_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000001_preparacion_altas.up.sql \
    >/dev/null
archivo vec_ct_o206_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000002_rotacion_hmac.up.sql \
    >/dev/null
archivo postgres deploy/postgresql/autorizacion_atestada_v3/roles_up.sql \
    >/dev/null
sql postgres \
    'CREATE ROLE vec_ad3_o206_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ad3_o206_migrador; GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_o206_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE' \
    >/dev/null
archivo vec_ad3_o206_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql \
    >/dev/null
archivo vec_ad3_o206_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql \
    >/dev/null
archivo vec_ct_o206_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000003_expediente_confirmacion_atestada.up.sql \
    >/dev/null
archivo vec_ct_o206_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000004_integridad_agregado_alta.up.sql \
    >/dev/null
archivo vec_ct_o206_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000005_funcion_confirmar_alta_atestada.up.sql \
    >/dev/null
archivo vec_ct_o206_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000006_candidatura_tecnica_o2_06.up.sql \
    >/dev/null

paso 'gobierno sintético y vectores O2-05 aislados'
sql postgres \
    'GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer) TO vec_autorizacion_atestada_v3_propietario' \
    >/dev/null
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/preparar_entorno_o2_05.sql \
    >/dev/null
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ayudantes_o2_05.sql \
    >/dev/null
sql postgres \
    'CREATE ROLE vec_ct_o206_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contratacion_temporal_ejecutor TO vec_ct_o206_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' \
    >/dev/null

paso 'ACL contractual del pool exclusivo'
[[ "$(valor "SELECT has_function_privilege('vec_ct_o206_runtime','vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')")" == 't' ]]
[[ "$(valor "SELECT has_function_privilege('vec_ct_o206_runtime','vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(text[],text[],text,text,text,text,text,text,text,text)','EXECUTE')")" == 't' ]]
[[ "$(valor "SELECT has_function_privilege('vec_ct_o206_runtime','vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')")" == 'f' ]]
[[ "$(valor "SELECT has_function_privilege('public','vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')")" == 'f' ]]
[[ "$(valor "SELECT has_function_privilege('vec_ct_o206_runtime','vec_contratacion_temporal.preparar_alta_v2(jsonb)','EXECUTE')")" == 'f' ]]
[[ "$(valor "SELECT has_function_privilege('vec_ct_o206_runtime','vec_contratacion_temporal.reconciliar_agregado_alta_v1(bytea,text,text,text,text,text,text)','EXECUTE')")" == 'f' ]]
esperar_fallo 'lectura directa de expedientes' sql vec_ct_o206_runtime \
    'TABLE vec_contratacion_temporal.expediente_alta'
esperar_fallo 'lectura directa de candidaturas técnicas' sql vec_ct_o206_runtime \
    'TABLE vec_contratacion_temporal.candidatura_alta_tecnica'
esperar_fallo 'preparación histórica revocada' sql vec_ct_o206_runtime \
    "SELECT * FROM vec_contratacion_temporal.preparar_alta_v2('{}'::jsonb)"
esperar_fallo 'reconciliador interno revocado' sql vec_ct_o206_runtime \
    "SELECT * FROM vec_contratacion_temporal.reconciliar_agregado_alta_v1(''::bytea,'','','','','','')"

paso 'compilación estática y ejecución del adaptador Go contra PostgreSQL 18'
CGO_ENABLED=0 go test -tags=o206postgresql -c \
    ./internal/modules/contrataciontemporal/adapters/postgres \
    -o "${directorio_temporal}/o206-postgresql.test"
docker cp "${directorio_temporal}/o206-postgresql.test" \
    "${contenedor}:/tmp/o206-postgresql.test"
docker exec \
    --env 'VEC_O206_DSN_ADMIN=user=postgres database=postgres host=/var/run/postgresql sslmode=disable' \
    --env 'VEC_O206_DSN_RUNTIME=user=vec_ct_o206_runtime database=postgres host=/var/run/postgresql sslmode=disable' \
    "${contenedor}" /tmp/o206-postgresql.test \
    -test.run '^TestConfirmacionAltaPostgreSQL18Real$' \
    -test.count=1 -test.v

if [[ "${VEC_EJECUTAR_O207:-}" == '1' ]]; then
    paso 'composición O2-07 y acreditación del pool contra PostgreSQL 18'
    CGO_ENABLED=0 go test -tags=o207postgresql -c \
        ./internal/app/composicion/contrataciontemporal \
        -o "${directorio_temporal}/o207-postgresql.test"
    docker cp "${directorio_temporal}/o207-postgresql.test" \
        "${contenedor}:/tmp/o207-postgresql.test"
    docker exec \
        --env 'VEC_O207_DSN_ADMIN=user=postgres database=postgres host=/var/run/postgresql sslmode=disable' \
        --env 'VEC_O207_DSN_RUNTIME=user=vec_ct_o206_runtime database=postgres host=/var/run/postgresql sslmode=disable' \
        "${contenedor}" /tmp/o207-postgresql.test \
        -test.run '^TestComposicionAltaPostgreSQL18Real$' \
        -test.count=1 -test.v
fi

paso 'retirada protegida ante candidaturas técnicas no confirmadas'
esperar_fallo 'down sin consentimiento destructivo explícito' \
    archivo vec_ct_o206_migrador \
    deploy/postgresql/contratacion_temporal/migraciones/000006_candidatura_tecnica_o2_06.down.sql
[[ "$(valor "SELECT to_regprocedure('vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(text[],text[],text,text,text,text,text,text,text,text)') IS NOT NULL")" == 't' ]]
[[ "$(valor "SELECT has_function_privilege('vec_ct_o206_runtime','vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')")" == 'f' ]]

paso 'PostgreSQL 18 real, ACL y reconciliación O2-06: OK'
