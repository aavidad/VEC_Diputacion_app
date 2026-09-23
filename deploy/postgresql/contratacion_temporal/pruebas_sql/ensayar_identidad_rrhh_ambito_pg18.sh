#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct-identidad2-rrhh-${PPID}-${RANDOM}"
volumen="${contenedor}-datos"
archivo_clave="$(mktemp "${TMPDIR:-/tmp}/vec-ct-identidad.XXXXXX")"
clave="$(openssl rand -hex 24)"

limpiar() {
    rm -f -- "$archivo_clave"
    docker rm --force --volumes "$contenedor" >/dev/null 2>&1 || true
    docker volume rm --force "$volumen" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

chmod 0600 "$archivo_clave"
printf '%s' "$clave" >"$archivo_clave"
unset clave

paso() {
    printf '[IDENTIDAD2:RRHH:PG18] %s\n' "$1"
}

psql_admin() {
    docker exec --interactive "$contenedor" psql -X \
        --set ON_ERROR_STOP=1 --username postgres --dbname postgres "$@"
}

archivo() {
    psql_admin --file "/repo/$1" >/dev/null
}

esperar_fallo() {
    local descripcion=$1
    shift
    if "$@" >/dev/null 2>&1; then
        printf 'se esperaba rechazo: %s\n' "$descripcion" >&2
        return 1
    fi
}

psql_runtime() {
    docker exec "$contenedor" psql -X --no-align --tuples-only \
        --set ON_ERROR_STOP=1 --username vec_c2d2_identidad_runtime \
        --dbname postgres "$@"
}

paso "arranque aislado con $imagen"
docker volume create "$volumen" >/dev/null
docker run --detach --rm --name "$contenedor" --network none \
    --env POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password \
    --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
    --mount \
    "type=bind,source=$archivo_clave,target=/run/secrets/postgres_password,readonly" \
    --mount "type=volume,source=$volumen,target=/var/lib/postgresql" \
    "$imagen" >/dev/null
for _ in {1..60}; do
    if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
docker cp "$raiz/deploy/postgresql/." "$contenedor:/repo"

paso 'instalación mínima de autoridades reales'
psql_admin <<'SQL' >/dev/null
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
archivo autorizacion/roles_up.sql
archivo autorizacion/migraciones/000001_autorizacion.up.sql
archivo \
    ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
archivo identidad_sesiones_v1/roles_up.sql
archivo \
    identidad_sesiones_v1/migraciones_autorizacion/000001_capacidad_tablas_v1.up.sql
archivo identidad_sesiones_v1/migraciones/000001_registro_base_v1.up.sql
archivo identidad_sesiones_v1/migraciones/000002_operaciones_v1.up.sql
archivo \
    identidad_sesiones_v1/migraciones/000003_revalidacion_autenticacion_actor_v1.up.sql
archivo contratacion_temporal/roles_up.sql
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_contratacion_temporal_consultor_rrhh
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
GRANT CONNECT ON DATABASE postgres
    TO vec_contratacion_temporal_consultor_rrhh;
SQL

paso 'instalación y comprobación estructural'
archivo \
    contratacion_temporal/migraciones_identidad/000001_revalidacion_consulta_rrhh_v1.up.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_identidad_consulta_rrhh.sql
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_contratacion_temporal_consultor_rrhh_ambito
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
SQL
archivo \
    contratacion_temporal/migraciones_identidad/000002_revalidacion_consulta_rrhh_ambito_v1.up.sql
archivo \
    contratacion_temporal/pruebas_sql/ensayar_identidad_rrhh_ambito_pg18.sql

paso 'fixture sintético por las operaciones reales de identidad'
psql_admin <<'SQL' >/dev/null
SELECT *
  FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
      'opr_aaaaaaaaaaaaaaaaaaaaaaaa',
      'vec.identidad.hmac-sha256.v1',
      'idh_aaaaaaaaaaaaaaaaaaaaaaaa',
      'clave-hsm-prueba', 1,
      decode(repeat('11', 32), 'hex'),
      decode(repeat('22', 32), 'hex'),
      false, NULL
  );
SQL
refs="$(
    psql_admin --no-align --tuples-only --field-separator='|' <<'SQL'
SELECT autenticacion_ref, sesion_ref
  FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
      'opr_bbbbbbbbbbbbbbbbbbbbbbbb',
      'vec.identidad.hmac-sha256.v1',
      'idh_aaaaaaaaaaaaaaaaaaaaaaaa',
      'clave-hsm-prueba', 1,
      decode(repeat('33', 32), 'hex'),
      decode(repeat('44', 32), 'hex'),
      decode(repeat('22', 32), 'hex'),
      decode(repeat('11', 32), 'hex'),
      NULL, false, 'interna_corporativa', 'kerberos_ad', 'alto',
      repeat('a', 64),
      date_trunc('microseconds', clock_timestamp() - interval '2 seconds'),
      date_trunc('microseconds', clock_timestamp() - interval '1 second'),
      date_trunc('microseconds', clock_timestamp() + interval '4 minutes'),
      'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
  );
SQL
)"
autenticacion=${refs%%|*}
sesion=${refs#*|}
if [[ -z $autenticacion || -z $sesion || $refs != *'|'* ]]; then
    printf 'no se creó la sesión interna sintética\n' >&2
    exit 1
fi

paso 'LOGIN legacy, ámbito, doble y sin grupo'
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_identidad2_legacy LOGIN INHERIT;
CREATE ROLE vec_identidad2_ambito LOGIN INHERIT;
CREATE ROLE vec_identidad2_doble LOGIN INHERIT;
CREATE ROLE vec_identidad2_sin_grupo LOGIN INHERIT;
GRANT CONNECT ON DATABASE postgres
    TO vec_contratacion_temporal_consultor_rrhh_ambito,
       vec_identidad2_sin_grupo;
GRANT vec_contratacion_temporal_consultor_rrhh TO vec_identidad2_legacy
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_contratacion_temporal_consultor_rrhh_ambito TO vec_identidad2_ambito
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_contratacion_temporal_consultor_rrhh TO vec_identidad2_doble
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_contratacion_temporal_consultor_rrhh_ambito TO vec_identidad2_doble
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE FUNCTION vec_contratacion_temporal.identidad2_prueba(text,text)
RETURNS text LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT login_tecnico FROM
      vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1($1,$2)
$funcion$;
ALTER FUNCTION vec_contratacion_temporal.identidad2_prueba(text,text)
    OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.identidad2_prueba(text,text)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_consultor_rrhh,
       vec_contratacion_temporal_consultor_rrhh_ambito,
       vec_identidad2_sin_grupo;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.identidad2_prueba(text,text)
    TO vec_contratacion_temporal_consultor_rrhh,
       vec_contratacion_temporal_consultor_rrhh_ambito,
       vec_identidad2_sin_grupo;
SQL

psql_login() {
    local rol=$1
    shift
    docker exec "$contenedor" psql -X --no-align --tuples-only \
        --set ON_ERROR_STOP=1 --username "$rol" --dbname postgres "$@"
}
probar_ok() {
    local rol=$1
    local esperado=$2
    local obtenido
    obtenido=$(psql_login "$rol" --command \
        "SELECT vec_contratacion_temporal.identidad2_prueba(
            '$autenticacion','$sesion')")
    if [[ $obtenido != "$esperado" ]]; then
        printf 'identidad inesperada para %s: %s\n' "$rol" "$obtenido" >&2
        exit 1
    fi
}
probar_ok vec_identidad2_legacy vec_identidad2_legacy
probar_ok vec_identidad2_ambito vec_identidad2_ambito
if [[ -n $(psql_login vec_identidad2_ambito --command \
    "SELECT vec_contratacion_temporal.identidad2_prueba(
        '$autenticacion','ses_0000000000000000000000')") ]]; then
    printf 'la sesión inválida produjo identidad\n' >&2
    exit 1
fi
esperar_fallo 'doble grupo técnico' psql_login vec_identidad2_doble \
    --command "SELECT vec_contratacion_temporal.identidad2_prueba('$autenticacion','$sesion')"
esperar_fallo 'sin grupo técnico' psql_login vec_identidad2_sin_grupo \
    --command "SELECT vec_contratacion_temporal.identidad2_prueba('$autenticacion','$sesion')"
esperar_fallo 'SET ROLE de grupo sin opción SET' psql_login vec_identidad2_ambito \
    --command 'SET ROLE vec_contratacion_temporal_consultor_rrhh_ambito'
esperar_fallo 'EXECUTE directo de identidad' psql_login vec_identidad2_ambito \
    --command "SELECT * FROM vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1('$autenticacion','$sesion')"

paso 'deriva de membresía, atributos y ACL'
psql_admin --command \
    'CREATE ROLE vec_identidad2_extra NOLOGIN; GRANT vec_identidad2_extra TO vec_identidad2_ambito WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' >/dev/null
esperar_fallo 'segunda membresía no nominal' psql_login vec_identidad2_ambito \
    --command "SELECT vec_contratacion_temporal.identidad2_prueba('$autenticacion','$sesion')"
psql_admin --command \
    'REVOKE vec_identidad2_extra FROM vec_identidad2_ambito; DROP ROLE vec_identidad2_extra' >/dev/null
psql_admin --command \
    'ALTER ROLE vec_identidad2_ambito NOINHERIT' >/dev/null
esperar_fallo 'LOGIN sin herencia obligatoria' psql_login vec_identidad2_ambito \
    --command "SELECT vec_contratacion_temporal.identidad2_prueba('$autenticacion','$sesion')"
psql_admin --command \
    'ALTER ROLE vec_identidad2_ambito INHERIT' >/dev/null
psql_admin --command \
    'ALTER ROLE vec_contratacion_temporal_consultor_rrhh_ambito LOGIN' >/dev/null
esperar_fallo 'grupo nominal con LOGIN' psql_login vec_identidad2_ambito \
    --command "SELECT vec_contratacion_temporal.identidad2_prueba('$autenticacion','$sesion')"
psql_admin --command \
    'ALTER ROLE vec_contratacion_temporal_consultor_rrhh_ambito NOLOGIN' >/dev/null
probar_ok vec_identidad2_ambito vec_identidad2_ambito
archivo contratacion_temporal/pruebas_sql/ensayar_identidad_rrhh_ambito_pg18.sql

paso 'DOWN protegido con LOGIN y reversión vacía'
esperar_fallo 'DOWN con pool nuevo en uso' archivo \
    contratacion_temporal/migraciones_identidad/000002_revalidacion_consulta_rrhh_ambito_v1.down.sql
psql_admin <<'SQL' >/dev/null
DROP FUNCTION vec_contratacion_temporal.identidad2_prueba(text,text);
REVOKE vec_contratacion_temporal_consultor_rrhh_ambito FROM vec_identidad2_ambito;
REVOKE vec_contratacion_temporal_consultor_rrhh_ambito FROM vec_identidad2_doble;
SQL
archivo \
    contratacion_temporal/migraciones_identidad/000002_revalidacion_consulta_rrhh_ambito_v1.down.sql
if [[ $(psql_admin --no-align --tuples-only --command \
    "SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')
       FROM pg_proc p WHERE p.oid = to_regprocedure(
       'vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text,text)')") != \
    69e0c769db76e8eee040334bba764f5374360f30b01c4d1d59db14899fce6c44 ]]; then
    printf 'la inversión no recuperó la preimagen exacta\n' >&2
    exit 1
fi
paso 'ensayo superado'
