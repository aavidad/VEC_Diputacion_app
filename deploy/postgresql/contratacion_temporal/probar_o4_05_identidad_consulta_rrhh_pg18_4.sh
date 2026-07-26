#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct-identidad-rrhh-${PPID}-${RANDOM}"
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
    printf '[O4-05:C2-D2-C:IDENTIDAD:PG18.4] %s\n' "$1"
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

paso 'fachada CT de prueba y LOGIN técnico mínimo'
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_c2d2_identidad_runtime
    LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_consultor_rrhh
    TO vec_c2d2_identidad_runtime
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE FUNCTION vec_contratacion_temporal.c2d2_identidad_prueba(
    p_autenticacion_ref text,
    p_sesion_ref text
)
RETURNS TABLE(
    autenticacion_ref text,
    autenticacion_huella_sha256 text,
    asercion_ref text,
    sesion_ref text,
    control_sesion_ref text,
    control_sesion_revision text,
    control_sesion_huella_sha256 text,
    cuenta_ref text,
    cuenta_ordinaria_ref text,
    cuenta_privilegiada boolean,
    superficie text,
    metodo_observado text,
    garantia_observada text,
    politica_garantia_ref text,
    politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz,
    sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz,
    sesion_revalidada_en timestamptz,
    login_tecnico text
)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT *
      FROM vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(
          p_autenticacion_ref, p_sesion_ref
      );
END;
ALTER FUNCTION
    vec_contratacion_temporal.c2d2_identidad_prueba(text, text)
    OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.c2d2_identidad_prueba(text, text)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.c2d2_identidad_prueba(text, text)
    TO vec_contratacion_temporal_consultor_rrhh;
SQL

salida="$(
    psql_runtime --command \
        "SELECT superficie || '|' || garantia_observada || '|' ||
                cuenta_privilegiada::text || '|' || login_tecnico
           FROM vec_contratacion_temporal.c2d2_identidad_prueba(
               '$autenticacion', '$sesion'
           )"
)"
if [[ $salida != \
    'interna_corporativa|alto|false|vec_c2d2_identidad_runtime' ]]; then
    printf 'resultado nominal inesperado: %s\n' "$salida" >&2
    exit 1
fi

paso 'rechazos por ACL, identidad inválida y topología'
esperar_fallo 'runtime llama directamente a identidad' \
    psql_runtime --command \
    "SELECT * FROM vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(
        '$autenticacion', '$sesion'
    )"
if [[ $(
    psql_runtime --command \
        "SELECT count(*)
           FROM vec_contratacion_temporal.c2d2_identidad_prueba(
               '$autenticacion', 'ses_0000000000000000000000'
           )"
) != 0 ]]; then
    printf 'una sesión ajena produjo identidad\n' >&2
    exit 1
fi
psql_admin --command \
    'CREATE ROLE vec_c2d2_rol_ajeno NOLOGIN; GRANT vec_c2d2_rol_ajeno TO vec_c2d2_identidad_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' \
    >/dev/null
esperar_fallo 'LOGIN con segunda membresía' \
    psql_runtime --command \
    "SELECT * FROM vec_contratacion_temporal.c2d2_identidad_prueba(
        '$autenticacion', '$sesion'
    )"
psql_admin --command \
    'REVOKE vec_c2d2_rol_ajeno FROM vec_c2d2_identidad_runtime; DROP ROLE vec_c2d2_rol_ajeno' \
    >/dev/null

paso 'down rechaza dependencia y luego revierte sin CASCADE'
esperar_fallo 'función CT dependiente' archivo \
    contratacion_temporal/migraciones_identidad/000001_revalidacion_consulta_rrhh_v1.down.sql
psql_admin <<'SQL' >/dev/null
DROP FUNCTION
    vec_contratacion_temporal.c2d2_identidad_prueba(text, text);
REVOKE USAGE ON SCHEMA vec_contratacion_temporal
    FROM vec_contratacion_temporal_consultor_rrhh;
SQL
archivo \
    contratacion_temporal/migraciones_identidad/000001_revalidacion_consulta_rrhh_v1.down.sql
if [[ $(
    psql_admin --no-align --tuples-only --command \
        "SELECT to_regprocedure(
            'vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text,text)'
        ) IS NULL"
) != t ]]; then
    printf 'el down no retiró la función nominal\n' >&2
    exit 1
fi

paso 'ciclo completo superado'
