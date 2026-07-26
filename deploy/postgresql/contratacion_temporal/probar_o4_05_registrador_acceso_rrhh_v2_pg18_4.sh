#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct-registro-v2-${PPID}-${RANDOM}"
clave_archivo="$(mktemp "${TMPDIR:-/tmp}/vec-ct-registro-v2.XXXXXX")"
temporales=("$clave_archivo")

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
        --set ON_ERROR_STOP=1 --username postgres --dbname postgres "$@"
}

psql_runtime() {
    docker exec "$contenedor" psql -X --no-align --tuples-only \
        --set ON_ERROR_STOP=1 --username vec_c2d2_registro_runtime \
        --dbname postgres "$@"
}

archivo() {
    psql_admin --file "/repo/$1" >/dev/null
}

valor() {
    docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
        --username postgres --dbname postgres --command "$1"
}

esperar_fallo() {
    local descripcion=$1
    local salida
    shift
    salida="$(mktemp "${TMPDIR:-/tmp}/vec-ct-registro-fallo.XXXXXX")"
    temporales+=("$salida")
    if "$@" >"$salida" 2>&1; then
        printf 'se esperaba rechazo: %s\n' "$descripcion" >&2
        return 1
    fi
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
    "$imagen" >/dev/null
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

paso 'up, estructura, deriva, dependencia y down con baseline'
archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.up.sql
psql_admin --set antes=0 --file \
    /repo/contratacion_temporal/pruebas_sql/o405_registrador_acceso_rrhh_v2.sql \
    >/dev/null
psql_admin --command \
    'GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb) TO vec_contratacion_temporal_consultor_rrhh' \
    >/dev/null
esperar_fallo 'down con ACL derivada' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
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
esperar_fallo 'down con función futura dependiente' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
psql_admin --command \
    'DROP FUNCTION vec_contratacion_temporal.c2d2_dependencia_000039()' \
    >/dev/null
archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql
[[ "$(valor "SELECT ultima_secuencia || '|' || cabeza_sha256
  FROM vec_contratacion_temporal.control_cadena_accesos_rrhh
 WHERE control")" == "$estado_base" ]]
[[ "$(valor \
    'SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh')" \
    == "$conteo_base" ]]
archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.up.sql

paso 'sesión corporativa real y fachada CT de prueba'
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
SELECT autenticacion_ref, sesion_ref, control_sesion_ref,
       control_sesion_revision_texto, control_sesion_huella_sha256
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
IFS='|' read -r autenticacion sesion control revision control_huella \
    <<<"$refs"
if [[ -z $autenticacion || -z $sesion || -z $control_huella ]]; then
    printf 'fixture de identidad incompleto\n' >&2
    exit 1
fi

psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_c2d2_registro_runtime
    LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_consultor_rrhh
    TO vec_c2d2_registro_runtime
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE FUNCTION vec_contratacion_temporal.c2d2_registrar_prueba(
    p_indice integer, p_tipo text, p_autenticacion text,
    p_sesion text, p_control text, p_revision numeric,
    p_control_huella text
)
RETURNS jsonb
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
BEGIN ATOMIC
SELECT vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(
    jsonb_build_object(
        'registro', jsonb_build_object(
            'accion', CASE p_tipo WHEN 'cuadro' THEN
                'contratacion_temporal.cuadro.consultar'
                ELSE 'contratacion_temporal.expediente.consultar' END,
            'actor_ref', 'actor:rrhh:' || p_indice::text,
            'ambito_ref', 'organizacion:rrhh:principal',
            'audiencia', CASE p_tipo WHEN 'cuadro' THEN
                'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
                ELSE
                'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
                END,
            'auditoria_vec_huella_sha256',
                lpad(to_hex(p_indice + 80), 64, '8'),
            'auditoria_vec_ref', 'auditoria:vec:rrhh:' || p_indice::text,
            'capacidad_huella_sha256',
                lpad(to_hex(p_indice + 50), 64, '5'),
            'consulta_huella_sha256',
                lpad(to_hex(p_indice + 40), 64, '4'),
            'consumo_vec_huella_sha256',
                lpad(to_hex(p_indice + 60), 64, '6'),
            'correlacion_ref', 'correlacion:rrhh:' || p_indice::text,
            'decision_huella_sha256',
                lpad(to_hex(p_indice + 10), 64, '1'),
            'decision_ref', 'decision:rrhh:' || p_indice::text,
            'dominio_huella_consulta', CASE p_tipo WHEN 'cuadro' THEN
                'vec.contratacion_temporal.consulta_rrhh.cuadro.v1'
                ELSE
                'vec.contratacion_temporal.consulta_rrhh.detalle.v1' END,
            'expediente_ref', CASE p_tipo WHEN 'detalle'
                THEN 'expediente:rrhh:' || p_indice::text ELSE NULL END,
            'finalidad', CASE p_tipo WHEN 'cuadro' THEN
                'gestion_operativa_contratacion_temporal'
                ELSE 'tramitacion_expediente_contratacion_temporal' END,
            'modulo_id', 'contratacion_temporal',
            'organizacion_ref', 'organizacion:rrhh:principal',
            'perfil_id', 'perfil:rrhh:principal', 'perfil_version', 1,
            'recurso_ref', CASE p_tipo WHEN 'detalle'
                THEN 'expediente:rrhh:' || p_indice::text
                ELSE 'organizacion:rrhh:principal' END,
            'recurso_tipo', CASE p_tipo WHEN 'cuadro' THEN
                'cuadro_rrhh_contratacion_temporal'
                ELSE 'expediente_contratacion_temporal' END,
            'resultado_generico', 'entregado',
            'resultado_huella_sha256',
                lpad(to_hex(p_indice + 70), 64, '7'),
            'sesion_huella_sha256',
                lpad(to_hex(p_indice + 20), 64, '2'),
            'sesion_id', p_sesion, 'tipo_consulta', p_tipo,
            'total', 1, 'version_expediente',
                CASE p_tipo WHEN 'detalle' THEN 3 ELSE NULL END
        ),
        'alcance', jsonb_build_object(
            'clase_ambito', 'organizacion', 'familia_ref', NULL
        ),
        'identidad', jsonb_build_object(
            'actor_ref', 'actor:rrhh:' || p_indice::text,
            'autenticacion_huella_sha256', repeat('a', 64),
            'autenticacion_ref', p_autenticacion,
            'control_sesion_huella_sha256', p_control_huella,
            'control_sesion_ref', p_control,
            'control_sesion_revision', p_revision,
            'organizacion_ref', 'organizacion:rrhh:principal',
            'perfil_ref', 'perfil:rrhh:principal', 'perfil_version', 1,
            'sesion_ref', p_sesion
        )
    )
);
END;
ALTER FUNCTION vec_contratacion_temporal.c2d2_registrar_prueba(
    integer, text, text, text, text, numeric, text
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.c2d2_registrar_prueba(
        integer, text, text, text, text, numeric, text
    ) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.c2d2_registrar_prueba(
        integer, text, text, text, text, numeric, text
    ) TO vec_contratacion_temporal_consultor_rrhh;
SQL

invocar() {
    local indice=$1
    local tipo=$2
    psql_runtime --command \
        "SELECT vec_contratacion_temporal.c2d2_registrar_prueba(
            $indice, '$tipo', '$autenticacion', '$sesion',
            '$control', $revision, '$control_huella'
        )"
}

esperar_fallo 'runtime invoca directamente el registrador interno' \
    psql_runtime --command \
    "SELECT vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(
        '{}'::jsonb
    )"
recibo="$(invocar 101 cuadro)"
[[ $recibo == *'recibo-acceso-rrhh.o4-05.v2'* ]]
invocar 102 detalle >/dev/null

paso 'dos escrituras concurrentes conservan una sola cadena'
salida_a="$(mktemp "${TMPDIR:-/tmp}/vec-ct-registro-a.XXXXXX")"
salida_b="$(mktemp "${TMPDIR:-/tmp}/vec-ct-registro-b.XXXXXX")"
temporales+=("$salida_a" "$salida_b")
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

psql_admin <<SQL >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DO \$\$
DECLARE
    base record;
BEGIN
    SELECT * INTO STRICT base
      FROM vec_contratacion_temporal.control_registrador_acceso_rrhh_v2
     WHERE control;
    IF (
        SELECT count(*)
          FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    ) <> 4 OR (
        SELECT count(*)
          FROM vec_contratacion_temporal.alcance_acceso_rrhh
         WHERE acceso_ref IN (
             SELECT acceso_ref
               FROM vec_contratacion_temporal
                    .vinculo_identidad_acceso_rrhh_v2
         )
    ) <> 3 OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .vinculo_identidad_acceso_rrhh_v2
         WHERE login_tecnico <> 'vec_c2d2_registro_runtime'
            OR prueba_huella_sha256 <> encode(
                sha256(prueba_canonica), 'hex'
            )
    ) OR EXISTS (
        SELECT 1
          FROM (
              SELECT acceso.*,
                     lag(
                         huella_sha256, 1, base.cabeza_base_sha256
                     ) OVER (ORDER BY secuencia) anterior_esperado
                FROM vec_contratacion_temporal.registro_acceso_rrhh acceso
               WHERE acceso.secuencia > base.secuencia_base
          ) cadena
         WHERE cadena.anterior_sha256 <> cadena.anterior_esperado
    ) THEN
        RAISE EXCEPTION 'cadena/vínculo v2 incorrectos';
    END IF;
    BEGIN
        UPDATE vec_contratacion_temporal
               .vinculo_identidad_acceso_rrhh_v2
           SET login_tecnico = login_tecnico;
        RAISE EXCEPTION 'vínculo mutable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal
                 .vinculo_identidad_acceso_rrhh_v2;
        RAISE EXCEPTION 'vínculo truncable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
\$\$;
RESET ROLE;
SQL
esperar_fallo 'runtime lee vínculo técnico' \
    psql_runtime --command \
    'SELECT count(*) FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2'

paso 'historia v2 bloquea la reversión'
psql_admin <<'SQL' >/dev/null
DROP FUNCTION vec_contratacion_temporal.c2d2_registrar_prueba(
    integer, text, text, text, text, numeric, text
);
SQL
esperar_fallo 'down con accesos v2' archivo \
    contratacion_temporal/migraciones/000039_registrador_acceso_rrhh_v2_y_snapshot_asof.down.sql

paso 'registrador v2 y snapshot as-of superados'
