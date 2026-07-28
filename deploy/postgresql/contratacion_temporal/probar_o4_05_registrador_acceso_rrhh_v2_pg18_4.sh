#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct-registro-v2-${PPID}-${RANDOM}"
clave_archivo="$(mktemp "${TMPDIR:-/tmp}/vec-ct-registro-v2.XXXXXX")"
temporales=("$clave_archivo")
trazas=()

limpiar() {
    local temporal
    for temporal in "${temporales[@]}"; do
        rm -f -- "$temporal"
    done
    docker rm --force --volumes "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM
paso() {
    printf '[O4-05:C2-D2-C:000039:PG18.4] %s\n' "$1"
}
psql_admin() {
    docker exec --interactive "$contenedor" psql -X \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username postgres --dbname postgres "$@"
}

psql_runtime() {
    docker exec "$contenedor" psql -X --no-align --tuples-only \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres "$@"
}

archivo() {
    psql_admin --file "/repo/$1" >/dev/null
}

valor() {
    docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
        --username postgres --dbname postgres --command "$1"
}

estado_instalacion_000039() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.control_registrador_acceso_rrhh_v2'
        ) IS NOT NULL)::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2'
        ) IS NOT NULL)::text,
        (pg_catalog.to_regclass(
            'vec_contratacion_temporal.'
            'publicacion_rrhh_organizacion_expediente_corte_desc_idx'
        ) IS NOT NULL)::text,
        (pg_catalog.to_regprocedure(
            'vec_contratacion_temporal.'
            'registrar_acceso_rrhh_interno_v2(jsonb)'
        ) IS NOT NULL)::text,
        cadena.ultima_secuencia::text,
        cadena.cabeza_sha256,
        (SELECT pg_catalog.count(*)::text
           FROM vec_contratacion_temporal.registro_acceso_rrhh),
        (SELECT pg_catalog.count(*)::text
           FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2)
    )
    FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
    CROSS JOIN
        vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
    CROSS JOIN
        vec_contratacion_temporal.control_cadena_accesos_rrhh cadena
    WHERE cobertura.control AND consultas.control AND cadena.control"
}

comprobar_estado_000039() {
    local esperado=$1
    local contexto=$2
    local obtenido
    obtenido="$(estado_instalacion_000039)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado 000039 alterado tras %s\nesperado=%s\nobtenido=%s\n' \
            "$contexto" "$esperado" "$obtenido" >&2
        return 1
    fi
}

esperar_fallo() {
    local descripcion=$1
    local sqlstate=$2
    local mensaje=$3
    local salida
    shift 3
    salida="$(mktemp "${TMPDIR:-/tmp}/vec-ct-registro-fallo.XXXXXX")"
    temporales+=("$salida")
    trazas+=("$salida")
    if "$@" >"$salida" 2>&1; then
        printf 'se esperaba rechazo: %s\n' "$descripcion" >&2
        return 1
    fi
    comprobar_fallo_archivo "$descripcion" "$sqlstate" "$mensaje" "$salida"
}

comprobar_fallo_archivo() {
    local descripcion=$1
    local sqlstate=$2
    local mensaje=$3
    local salida=$4
    if ! rg -Fq "ERROR:  ${sqlstate}:" "$salida" ||
        ! rg -Fq "$mensaje" "$salida"; then
        printf 'rechazo inesperado para %s\n' "$descripcion" >&2
        sed -n '1,16p' "$salida" >&2
        return 1
    fi
}

contiene_privado() {
    local valor_privado=$1 hexadecimal
    shift
    hexadecimal="$(printf '%s' "$valor_privado" |
        od -An -tx1 | tr -d ' \n')"
    rg -Fq -- "$valor_privado" "$@" || rg -Fq -- "$hexadecimal" "$@"
}

esperar_actividad() {
    local marca=$1
    local espera=$2
    for _ in {1..120}; do
        if [[ "$(valor "SELECT count(*) FROM pg_catalog.pg_stat_activity
          WHERE pid <> pg_catalog.pg_backend_pid()
            AND application_name = '${marca}'
            AND (wait_event = '${espera}'
                 OR wait_event_type = '${espera}')")" == '1' ]]; then
            return
        fi
        sleep 0.05
    done
    printf 'no se observó %s en %s\n' "$espera" "$marca" >&2
    return 1
}

if [[ ! $imagen =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
    printf 'la imagen PostgreSQL debe fijarse por digest\n' >&2
    exit 64
fi
chmod 0600 "$clave_archivo"
openssl rand -hex 24 >"$clave_archivo"

paso "arranque efímero sin red: $imagen"
docker run --detach --name "$contenedor" --network none --read-only \
    --env POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password \
    --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
    --mount \
    "type=bind,source=$clave_archivo,target=/run/secrets/postgres_password,readonly" \
    --mount "type=bind,source=$raiz/deploy/postgresql,target=/repo,readonly" \
    --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=1024m \
    --tmpfs /tmp:rw,noexec,nosuid,size=64m \
    --tmpfs /run/postgresql:rw,noexec,nosuid,size=16m \
    "$imagen" -c wal_level=logical -c max_replication_slots=2 \
    -c max_wal_senders=2 >/dev/null
for _ in {1..60}; do
    if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres

psql_admin <<'SQL' >/dev/null
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
DO $$
BEGIN
    IF current_setting('server_version_num')::integer <> 180004 THEN
        RAISE EXCEPTION 'se exige PostgreSQL 18.4 exacto';
    END IF;
END
$$;
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL

paso 'instalación mínima del agregado y autoridades'
for ruta in \
    contexto_actor_v1/roles_up.sql \
    contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
    contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
    autorizacion/roles_up.sql \
    autorizacion/roles_v2_up.sql \
    contratacion_temporal/roles_up.sql \
    autorizacion/migraciones/000001_autorizacion.up.sql \
    ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
    autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
    autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
    autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql \
    autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql \
    autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql \
    contratacion_temporal/migraciones_autorizacion/000001_revalidacion_analisis_v3.up.sql \
    contratacion_temporal/migraciones_autorizacion/000002_proyeccion_motivos_cobertura_v1.up.sql \
    contratacion_temporal/migraciones_autorizacion/000003_barrera_motivos_cobertura_v1.up.sql \
    contratacion_temporal/migraciones_autorizacion/000004_wrapper_vec_cobertura_o4_04d.up.sql \
    contratacion_temporal/migraciones_autorizacion/000005_enlace_probatorio_vec_cobertura_o4_04e.up.sql
do
    archivo "$ruta"
    if [[ $ruta == contexto_actor_v1/migraciones/000002_* ]]; then
        archivo contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql
        archivo autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
        psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_c2d2_contexto LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO vec_c2d2_contexto
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
        docker exec --interactive "$contenedor" psql -X \
            --set ON_ERROR_STOP=1 -U vec_c2d2_contexto -d postgres \
            <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT count(*)
FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    'oca_registro_v3_000000000000000000000000',
    'rca_registro_v3_000000000000000000000000',
    'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    'prf_sintetico_cccccccccccccccccccccccc',
    'certificado', 'alto', clock_timestamp()
);
COMMIT;
SQL
        psql_admin <<'SQL' >/dev/null
REVOKE vec_contexto_actor_v1_runtime FROM vec_c2d2_contexto;
DROP ROLE vec_c2d2_contexto;
SQL
    fi
done
archivo autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql
archivo \
    contratacion_temporal/migraciones_autorizacion/000006_wrapper_contexto_exacto_cobertura_o4_04e.up.sql

archivo identidad_sesiones_v1/roles_up.sql
archivo \
    identidad_sesiones_v1/migraciones_autorizacion/000001_capacidad_tablas_v1.up.sql
archivo identidad_sesiones_v1/migraciones/000001_registro_base_v1.up.sql
archivo identidad_sesiones_v1/migraciones/000002_operaciones_v1.up.sql
archivo \
    identidad_sesiones_v1/migraciones/000003_revalidacion_autenticacion_actor_v1.up.sql

archivo autorizacion_atestada_v3/roles_up.sql
archivo \
    autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql
archivo \
    autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql
for numero in $(seq -w 1 34); do
    nombre="$(basename \
        "$raiz"/deploy/postgresql/contratacion_temporal/migraciones/0000"${numero}"_*.up.sql)"
    archivo "contratacion_temporal/migraciones/$nombre"
done
archivo contratacion_temporal/roles_lector_resultado_cobertura_up.sql
archivo \
    contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.up.sql
archivo contratacion_temporal/roles_consultor_rrhh_up.sql
archivo \
    contratacion_temporal/migraciones_identidad/000001_revalidacion_consulta_rrhh_v1.up.sql
archivo \
    contratacion_temporal/migraciones/000036_registro_accesos_rrhh_o4_05.up.sql

paso 'historia v1 previa, publicación y cursores'
archivo contratacion_temporal/pruebas_sql/o405_registro_accesos_rrhh.sql
psql_admin --set preparar=1 --file \
    /repo/contratacion_temporal/pruebas_sql/o405_publicacion_global_rrhh.sql \
    >/dev/null
archivo \
    contratacion_temporal/migraciones/000037_publicacion_global_rrhh.up.sql
psql_admin --set preparar=0 --file \
    /repo/contratacion_temporal/pruebas_sql/o405_publicacion_global_rrhh.sql \
    >/dev/null
archivo contratacion_temporal/roles_cursor_rrhh_up.sql
archivo \
    contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.up.sql
psql_admin --set antes=1 --file \
    /repo/contratacion_temporal/pruebas_sql/o405_registrador_acceso_rrhh_v2.sql \
    >/dev/null

estado_base="$(valor "SELECT ultima_secuencia || '|' || cabeza_sha256
  FROM vec_contratacion_temporal.control_cadena_accesos_rrhh
 WHERE control")"
conteo_base="$(valor \
    'SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh')"
huella_historia_base="$(valor "SELECT pg_catalog.encode(
    pg_catalog.sha256(pg_catalog.convert_to(
        COALESCE(pg_catalog.jsonb_agg(
            pg_catalog.to_jsonb(registro)
            ORDER BY registro.secuencia
        )::text, '[]'),
        'UTF8'
    )),
    'hex'
) FROM vec_contratacion_temporal.registro_acceso_rrhh registro")"

paso 'up, estructura, deriva, dependencia y down con baseline'
archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.up.sql
psql_admin --set antes=0 --file \
    /repo/contratacion_temporal/pruebas_sql/o405_registrador_acceso_rrhh_v2.sql \
    >/dev/null
estado_limpio_000039="$(estado_instalacion_000039)"
psql_admin --command \
    'GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb) TO vec_contratacion_temporal_consultor_rrhh' \
    >/dev/null
esperar_fallo 'down con ACL derivada' 55000 \
    'down del registrador RRHH v2 rechazado por deriva' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
comprobar_estado_000039 "$estado_limpio_000039" 'ACL de función derivada'
psql_admin --command \
    'REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb) FROM vec_contratacion_temporal_consultor_rrhh' \
    >/dev/null
psql_admin <<'SQL' >/dev/null
CREATE FUNCTION vec_contratacion_temporal.c2d2_dependencia_000039()
RETURNS jsonb
LANGUAGE sql VOLATILE
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT vec_contratacion_temporal
           .registrar_acceso_rrhh_interno_v2('{}'::jsonb);
END;
ALTER FUNCTION vec_contratacion_temporal.c2d2_dependencia_000039()
    OWNER TO vec_contratacion_temporal_propietario;
SQL
esperar_fallo 'down con función futura dependiente' 55000 \
    'dependencias impiden retirar registrador RRHH v2' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
comprobar_estado_000039 "$estado_limpio_000039" 'dependencia futura'
psql_admin --command \
    'DROP FUNCTION vec_contratacion_temporal.c2d2_dependencia_000039()' \
    >/dev/null

paso 'down rechaza deriva de estructura, política, trigger y ACL'
psql_admin --command \
    'ALTER TABLE vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2 ADD CONSTRAINT vinculo_identidad_deriva_c2d2 CHECK (true) NOT VALID' \
    >/dev/null
esperar_fallo 'down con restricción añadida' 55000 \
    'down del registrador RRHH v2 rechazado por deriva' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
comprobar_estado_000039 "$estado_limpio_000039" 'deriva de restricción'
[[ "$(valor "SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
 WHERE conrelid = 'vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2'::regclass
   AND conname = 'vinculo_identidad_deriva_c2d2' AND NOT convalidated")" == '1' ]]
psql_admin --command \
    'ALTER TABLE vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2 DROP CONSTRAINT vinculo_identidad_deriva_c2d2' \
    >/dev/null

psql_admin <<'SQL' >/dev/null
DROP POLICY propietario_total
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2;
CREATE POLICY propietario_total
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    AS RESTRICTIVE FOR SELECT
    TO vec_contratacion_temporal_propietario USING (false);
SQL
esperar_fallo 'down con política sustituida' 55000 \
    'down del registrador RRHH v2 rechazado por deriva' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
comprobar_estado_000039 "$estado_limpio_000039" 'deriva de política'
psql_admin <<'SQL' >/dev/null
DROP POLICY propietario_total
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2;
CREATE POLICY propietario_total
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    AS PERMISSIVE FOR ALL TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
SQL

psql_admin <<'SQL' >/dev/null
CREATE FUNCTION vec_contratacion_temporal.c2d2_trigger_derivado_000039()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog
AS $funcion$ BEGIN RETURN OLD; END $funcion$;
ALTER FUNCTION vec_contratacion_temporal.c2d2_trigger_derivado_000039()
    OWNER TO vec_contratacion_temporal_propietario;
DROP TRIGGER vinculo_identidad_acceso_rrhh_v2_inmutable
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2;
CREATE TRIGGER vinculo_identidad_acceso_rrhh_v2_inmutable
    BEFORE DELETE
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    FOR EACH ROW EXECUTE FUNCTION
        vec_contratacion_temporal.c2d2_trigger_derivado_000039();
SQL
esperar_fallo 'down con trigger sustituido' 55000 \
    'down del registrador RRHH v2 rechazado por deriva' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
comprobar_estado_000039 "$estado_limpio_000039" 'deriva de trigger'
psql_admin <<'SQL' >/dev/null
DROP TRIGGER vinculo_identidad_acceso_rrhh_v2_inmutable
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2;
DROP FUNCTION vec_contratacion_temporal.c2d2_trigger_derivado_000039();
CREATE TRIGGER vinculo_identidad_acceso_rrhh_v2_inmutable
    BEFORE UPDATE OR DELETE
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    FOR EACH ROW EXECUTE FUNCTION
        vec_contratacion_temporal.rechazar_mutacion_historia_v1();
SQL

psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_c2d2_deriva_acl NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT SELECT
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    TO vec_c2d2_deriva_acl WITH GRANT OPTION;
SQL
esperar_fallo 'down con ACL arbitraria en tabla' 55000 \
    'down del registrador RRHH v2 rechazado por deriva' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
comprobar_estado_000039 "$estado_limpio_000039" 'deriva de ACL de tabla'
psql_admin <<'SQL' >/dev/null
REVOKE ALL
    ON vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    FROM vec_c2d2_deriva_acl;
DROP ROLE vec_c2d2_deriva_acl;
SQL

paso 'down rechaza las barreras futuras 000040 y 000041'
for barreras in '20 4' '21 5'; do
    read -r cobertura_futura consultas_futuras <<<"$barreras"
    psql_admin --set=cobertura="$cobertura_futura" \
        --set=consultas="$consultas_futuras" <<'SQL' >/dev/null
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = :cobertura WHERE control;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = :consultas WHERE control;
SQL
    estado_futuro="$(estado_instalacion_000039)"
    esperar_fallo \
        "down con barrera futura $cobertura_futura/$consultas_futuras" \
        55000 'down del registrador RRHH v2 rechazado por deriva' archivo \
        contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
    comprobar_estado_000039 "$estado_futuro" \
        "barrera futura $cobertura_futura/$consultas_futuras"
    psql_admin <<'SQL' >/dev/null
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 19 WHERE control;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 3 WHERE control;
SQL
done
comprobar_estado_000039 "$estado_limpio_000039" 'restauración de barreras'

archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
[[ "$(valor "SELECT ultima_secuencia || '|' || cabeza_sha256
  FROM vec_contratacion_temporal.control_cadena_accesos_rrhh
 WHERE control")" == "$estado_base" ]]
[[ "$(valor \
    'SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh')" \
    == "$conteo_base" ]]
[[ "$(valor "SELECT pg_catalog.encode(
    pg_catalog.sha256(pg_catalog.convert_to(
        COALESCE(pg_catalog.jsonb_agg(
            pg_catalog.to_jsonb(registro)
            ORDER BY registro.secuencia
        )::text, '[]'),
        'UTF8'
    )),
    'hex'
) FROM vec_contratacion_temporal.registro_acceso_rrhh registro")" \
    == "$huella_historia_base" ]]
[[ "$(valor "SELECT pg_catalog.concat_ws('|',
    (pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_registrador_acceso_rrhh_v2'
    ) IS NULL)::text,
    (pg_catalog.to_regclass(
        'vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2'
    ) IS NULL)::text,
    (pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)'
    ) IS NULL)::text
)")" == 'true|true|true' ]]
archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.up.sql

paso 'dos identidades corporativas reales y fachada CT de prueba'
mapfile -t identidades < <(
    psql_admin --no-align --tuples-only --quiet --field-separator='|' \
        --file \
        /repo/contratacion_temporal/pruebas_sql/o405_registrador_acceso_rrhh_v2_fixtures.sql
)
[[ ${#identidades[@]} == 2 ]]
IFS='|' read -r _ autenticacion sesion control revision control_huella \
    cuenta_ref cuenta_ordinaria_ref <<<"${identidades[0]}"
IFS='|' read -r _ autenticacion_b sesion_b control_b revision_b \
    control_huella_b cuenta_ref_b cuenta_ordinaria_ref_b \
    <<<"${identidades[1]}"
if [[ -z $autenticacion || -z $autenticacion_b ||
    $cuenta_ref != "$cuenta_ordinaria_ref" ||
    $cuenta_ref_b != "$cuenta_ordinaria_ref_b" ||
    $cuenta_ref == "$cuenta_ref_b" ]]; then
    printf 'fixtures de identidades independientes incompletos\n' >&2
    exit 1
fi
psql_admin <<'SQL' >/dev/null
ALTER DATABASE postgres SET log_parameter_max_length_on_error = 4096;
CREATE ROLE vec_c2d2_registro_runtime
    LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
ALTER ROLE vec_c2d2_registro_runtime
    SET log_parameter_max_length_on_error = 0;
GRANT vec_contratacion_temporal_consultor_rrhh
    TO vec_c2d2_registro_runtime
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
[[ "$(psql_runtime --command \
    'SHOW log_parameter_max_length_on_error')" == '0' ]]

invocar() {
    local indice=$1
    local tipo=$2
    psql_runtime --command \
        "SELECT vec_contratacion_temporal.c2d2_registrar_prueba(
            $indice, '$tipo', '$autenticacion', '$sesion',
            '$control', $revision, '$control_huella'
        )"
}

invocar_parametrizado() {
    local indice=$1 tipo=$2 auth=$3 ses=$4 ctl=$5 rev=$6 huella=$7
    local dormir=${8:-0} marca=${9:-c2d2_parametrizada}
    docker exec --interactive \
        --env VEC_INDICE="$indice" --env VEC_TIPO="$tipo" \
        --env VEC_AUTH="$auth" --env VEC_SESION="$ses" \
        --env VEC_CONTROL="$ctl" --env VEC_REVISION="$rev" \
        --env VEC_HUELLA="$huella" --env VEC_DORMIR="$dormir" \
        --env VEC_MARCA="$marca" \
        "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
        --set VERBOSITY=verbose --username vec_c2d2_registro_runtime \
        --dbname postgres <<'SQL'
\getenv indice VEC_INDICE
\getenv tipo VEC_TIPO
\getenv auth VEC_AUTH
\getenv sesion VEC_SESION
\getenv control VEC_CONTROL
\getenv revision VEC_REVISION
\getenv huella VEC_HUELLA
\getenv dormir VEC_DORMIR
\getenv marca VEC_MARCA
SET application_name = :'marca';
BEGIN;
SELECT vec_contratacion_temporal.c2d2_registrar_prueba(
    $1::integer,$2::text,$3::text,$4::text,$5::text,$6::numeric,$7::text)
\parse llamada
\bind_named llamada :indice :tipo :auth :sesion :control :revision :huella
\g
SELECT pg_sleep(:'dormir');
COMMIT;
SQL
}

valor "SELECT slot_name FROM pg_catalog.pg_create_logical_replication_slot(
    'c2d2_privacidad', 'test_decoding')" >/dev/null

centinela_filtro='FILTRO-PRIVADO-C2D2'
centinela_material='MATERIAL-VEC-PRIVADO-C2D2'
centinela_token='TOKEN-PRIVADO-C2D2'
centinela_pii='DNI-SINTETICO-C2D2-00000000T'
centinelas=("$centinela_filtro" "$centinela_material" \
    "$centinela_token" "$centinela_pii")
opacos_wal=("$cuenta_ref" "$cuenta_ref_b" "$autenticacion"
    "$autenticacion_b" "$sesion" "$sesion_b" "$control" "$control_b"
    "$control_huella" "$control_huella_b" "$(printf 'a%.0s' {1..64})")
privados_salida=(vec_c2d2_registro_runtime login_tecnico cuenta_ref
    cuenta_ordinaria_ref "${opacos_wal[@]}"
    "${centinelas[@]}" clave-hsm-prueba)
salida_centinelas="$(mktemp "${TMPDIR:-/tmp}/vec-ct-centinelas.XXXXXX")"
temporales+=("$salida_centinelas")
trazas+=("$salida_centinelas")
docker exec --interactive --env VEC_FILTRO="$centinela_filtro" \
    --env VEC_MATERIAL="$centinela_material" \
    --env VEC_TOKEN="$centinela_token" --env VEC_PII="$centinela_pii" \
    --env VEC_AUTH="$autenticacion" --env VEC_SESION="$sesion" \
    --env VEC_CONTROL="$control" --env VEC_REVISION="$revision" \
    --env VEC_HUELLA="$control_huella" "$contenedor" \
    psql -XAt --set ON_ERROR_STOP=1 --username vec_c2d2_registro_runtime \
    --dbname postgres >"$salida_centinelas" <<'SQL'
\getenv filtro VEC_FILTRO
\getenv material VEC_MATERIAL
\getenv token VEC_TOKEN
\getenv pii VEC_PII
\getenv auth VEC_AUTH
\getenv sesion VEC_SESION
\getenv control VEC_CONTROL
\getenv revision VEC_REVISION
\getenv huella VEC_HUELLA
SELECT vec_contratacion_temporal.c2d2_centinela_prueba(
 $1::integer,$2::text,$3::text,$4::text,$5::text,$6::numeric,$7::text)
\parse centinela
\bind_named centinela 301 :filtro :auth :sesion :control :revision :huella
\g
\bind_named centinela 302 :material :auth :sesion :control :revision :huella
\g
\bind_named centinela 303 :token :auth :sesion :control :revision :huella
\g
\bind_named centinela 304 :pii :auth :sesion :control :revision :huella
\g
SQL

paso 'registrador v2 sobrevive a 000040 y 000041 sin dejar efectos'
estado_antes_barreras="$(estado_instalacion_000039)"
psql_admin --set=autenticacion="$autenticacion" --set=sesion="$sesion" \
    --set=control="$control" --set=revision="$revision" \
    --set=control_huella="$control_huella" <<'SQL' >/dev/null
BEGIN;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 20 WHERE control AND version_esquema = 19;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 4 WHERE control AND version_esquema = 3;
SET SESSION AUTHORIZATION vec_c2d2_registro_runtime;
SELECT vec_contratacion_temporal.c2d2_registrar_prueba(
    105, 'cuadro', :'autenticacion', :'sesion', :'control',
    :revision, :'control_huella'
);
RESET SESSION AUTHORIZATION;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 21 WHERE control AND version_esquema = 20;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 5 WHERE control AND version_esquema = 4;
SET SESSION AUTHORIZATION vec_c2d2_registro_runtime;
SELECT vec_contratacion_temporal.c2d2_registrar_prueba(
    106, 'detalle', :'autenticacion', :'sesion', :'control',
    :revision, :'control_huella'
);
RESET SESSION AUTHORIZATION;
ROLLBACK;
SQL
comprobar_estado_000039 "$estado_antes_barreras" \
    'rollback de consultas con barreras 000040/000041'
[[ "$(valor "SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh
 WHERE decision_ref IN ('decision:rrhh:105', 'decision:rrhh:106')")" == '0' ]]

esperar_fallo 'runtime invoca directamente el registrador interno' 42501 \
    'permission denied for function registrar_acceso_rrhh_interno_v2' \
    psql_runtime --command \
    "SELECT vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(
        '{}'::jsonb
    )"

paso 'identidades cruzadas no dejan historia parcial'
estado_antes_cruce="$(estado_instalacion_000039)"
esperar_fallo 'autenticación A con sesión B' 42501 \
    'identidad de acceso RRHH v2 no disponible' invocar_parametrizado \
    107 cuadro "$autenticacion" "$sesion_b" "$control_b" \
    "$revision_b" "$control_huella_b"
esperar_fallo 'autenticación B con sesión A' 42501 \
    'identidad de acceso RRHH v2 no disponible' invocar_parametrizado \
    108 cuadro "$autenticacion_b" "$sesion" "$control" \
    "$revision" "$control_huella"
comprobar_estado_000039 "$estado_antes_cruce" 'identidades cruzadas'

recibo="$(invocar 101 cuadro)"
[[ $recibo == *'recibo-acceso-rrhh.o4-05.v2'* ]]
for privado in "${privados_salida[@]}"; do
    [[ $recibo != *"$privado"* ]]
done
invocar 102 detalle >/dev/null

paso 'dos escrituras concurrentes conservan una sola cadena'
salida_a="$(mktemp "${TMPDIR:-/tmp}/vec-ct-registro-a.XXXXXX")"
salida_b="$(mktemp "${TMPDIR:-/tmp}/vec-ct-registro-b.XXXXXX")"
temporales+=("$salida_a" "$salida_b")
trazas+=("$salida_a" "$salida_b")
invocar 103 cuadro >"$salida_a" 2>&1 &
pid_a=$!
invocar 104 cuadro >"$salida_b" 2>&1 &
pid_b=$!
estado_a=0
estado_b=0
wait "$pid_a" || estado_a=$?
wait "$pid_b" || estado_b=$?
if ((estado_a != 0 || estado_b != 0)); then
    sed -n '1,20p' "$salida_a" >&2
    sed -n '1,20p' "$salida_b" >&2
    exit 1
fi

psql_admin --set=cuenta_ref="$cuenta_ref" \
    --set=cuenta_ordinaria_ref="$cuenta_ordinaria_ref" <<'SQL' >/dev/null
SELECT vec_contratacion_temporal.c2d2_verificar_prueba(
    :'cuenta_ref', :'cuenta_ordinaria_ref'
);
SQL
esperar_fallo 'runtime lee vínculo técnico' 42501 \
    'permission denied for table vinculo_identidad_acceso_rrhh_v2' \
    psql_runtime --command \
    'SELECT count(*) FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2'

paso 'carreras reales de registro y revocación tienen orden único'
salida_registro="$(mktemp "${TMPDIR:-/tmp}/vec-ct-carrera-reg.XXXXXX")"
salida_revoca="$(mktemp "${TMPDIR:-/tmp}/vec-ct-carrera-rev.XXXXXX")"
temporales+=("$salida_registro" "$salida_revoca")
trazas+=("$salida_registro" "$salida_revoca")
invocar_parametrizado 201 cuadro "$autenticacion_b" "$sesion_b" \
    "$control_b" "$revision_b" "$control_huella_b" 1.2 \
    c2d2_registro_primero \
    >"$salida_registro" 2>&1 &
pid_registro=$!
esperar_actividad c2d2_registro_primero PgSleep
psql_admin --no-align --tuples-only --command \
    "SET application_name='c2d2_revocacion_despues'; SELECT
     vec_identidad_sesiones_v1.revocar_sesion_v1(
      '$sesion_b','$control_b','$revision_b',
      'opr_eeeeeeeeeeeeeeeeeeeeeeee')" >"$salida_revoca" 2>&1 &
pid_revoca=$!
esperar_actividad c2d2_revocacion_despues Lock
wait "$pid_registro"
wait "$pid_revoca"
rg -Fq 'recibo-acceso-rrhh.o4-05.v2' "$salida_registro"
if rg -q "vec_c2d2_registro_runtime|login_tecnico|cuenta(_ordinaria)?_ref|$cuenta_ref|$cuenta_ref_b" "$salida_registro"; then
    printf 'recibo de carrera expone identidad técnica\n' >&2; exit 1
fi
[[ "$(valor "SELECT count(*) FROM
 vec_contratacion_temporal.registro_acceso_rrhh acceso
 JOIN vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2 vinculo
 USING (acceso_ref) WHERE acceso.decision_ref='decision:rrhh:201'
 AND vinculo.cuenta_ref='$cuenta_ref_b'")" == '1' ]]

estado_antes_revoca="$(estado_instalacion_000039)"
psql_admin --no-align --tuples-only --command \
    "SET application_name='c2d2_revocacion_primero'; BEGIN; SELECT
     vec_identidad_sesiones_v1.revocar_sesion_v1(
      '$sesion','$control','$revision',
      'opr_ffffffffffffffffffffffff'); SELECT pg_sleep(0.4); COMMIT" \
    >"$salida_revoca" 2>&1 &
pid_revoca=$!
esperar_actividad c2d2_revocacion_primero PgSleep
invocar_parametrizado 202 cuadro "$autenticacion" "$sesion" "$control" \
    "$revision" "$control_huella" 0 c2d2_registro_despues \
    >"$salida_registro" 2>&1 &
pid_registro=$!
esperar_actividad c2d2_registro_despues Lock
wait "$pid_revoca"
estado_registro=0
wait "$pid_registro" || estado_registro=$?
((estado_registro != 0))
comprobar_fallo_archivo 'registro posterior a revocación' 42501 \
    'identidad de acceso RRHH v2 no disponible' "$salida_registro"
comprobar_estado_000039 "$estado_antes_revoca" \
    'revocación anterior al registro'

paso 'WAL lógico, logs, recibos y trazas permanecen minimizados'
salida_wal="$(mktemp "${TMPDIR:-/tmp}/vec-ct-wal.XXXXXX")"
salida_log="$(mktemp "${TMPDIR:-/tmp}/vec-ct-log.XXXXXX")"
temporales+=("$salida_wal" "$salida_log")
psql_admin --no-align --tuples-only --command \
    "SELECT data FROM pg_catalog.pg_logical_slot_get_changes(
     'c2d2_privacidad', NULL, NULL)" >"$salida_wal"
docker logs "$contenedor" >"$salida_log" 2>&1
rg -Fq 'vinculo_identidad_acceso_rrhh_v2' "$salida_wal"
for opaco in "${opacos_wal[@]}"; do
    rg -Fq "$opaco" "$salida_wal"
    if rg -F "$opaco" "$salida_wal" | rg -qv \
        'table (vec_contratacion_temporal\.(registro_acceso_rrhh|alcance_acceso_rrhh|vinculo_identidad_acceso_rrhh_v2)|vec_autorizacion\.control_sesion_(v1|actual_v1)):'; then
        printf 'referencia opaca fuera del WAL técnico permitido\n' >&2; exit 1
    fi
done
for marcador in {301..304}; do
    rg -Fq "marcador:privacidad:$marcador" "$salida_wal"
done
for privado in "${centinelas[@]}" vec_c2d2_registro_runtime \
    clave-hsm-prueba; do
    if contiene_privado "$privado" "$salida_wal"; then
        printf 'material privado inesperado en WAL lógico\n' >&2
        exit 1
    fi
done
for privado in "${privados_salida[@]}"; do
    if contiene_privado "$privado" "$salida_log" "${trazas[@]}"; then
        printf 'material privado inesperado en logs o trazas\n' >&2
        exit 1
    fi
done
valor "SELECT pg_catalog.pg_drop_replication_slot(
    'c2d2_privacidad')" >/dev/null

paso 'historia v2 bloquea la reversión'
psql_admin <<'SQL' >/dev/null
DROP FUNCTION vec_contratacion_temporal.c2d2_centinela_prueba(
    integer, text, text, text, text, numeric, text
);
DROP FUNCTION vec_contratacion_temporal.c2d2_verificar_prueba(text, text);
DROP FUNCTION vec_contratacion_temporal.c2d2_registrar_prueba(
    integer, text, text, text, text, numeric, text
);
DROP TABLE vec_contratacion_temporal.c2d2_marcador_privacidad_prueba;
SQL
estado_con_historia_v2="$(estado_instalacion_000039)"
esperar_fallo 'down con accesos v2' 55000 \
    'historia v2 impide retirar el registrador RRHH' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
comprobar_estado_000039 "$estado_con_historia_v2" \
    'retirada rechazada con historia v2'

paso 'registrador v2 y snapshot as-of superados'
