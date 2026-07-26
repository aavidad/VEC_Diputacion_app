#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-consultas-rrhh-v3-${PPID}-${RANDOM}"

if [[ ! "${imagen}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
    printf 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256\n' >&2
    exit 64
fi

limpiar() {
    docker rm --force --volumes "${contenedor}" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

paso() {
    printf '[consultas RRHH V3:PostgreSQL] %s\n' "$1"
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

invocar_usuario() {
    local usuario="$1"
    local caso="$2"
    local fachada="$3"
    local pieza="${4:-}"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --username "${usuario}" \
        --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT consumo_nuevo
  FROM vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
    '${caso}','${fachada}','${pieza}'
  );
COMMIT;
SQL
}

preparar() {
    sql postgres \
        "SELECT public.preparar_vector_consulta_rrhh_v3('$1','$2')" \
        >/dev/null
}

estado_vec() {
    valor "SELECT pg_catalog.concat_ws('|',
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.atestacion_decision_v3),
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.consumo_decision_v3),
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3),
        (SELECT secuencia
           FROM vec_autorizacion_atestada_v3.control_cadena_auditoria
          WHERE control_id),
        (SELECT cabeza_sha256
           FROM vec_autorizacion_atestada_v3.control_cadena_auditoria
          WHERE control_id))"
}

recibo_usuario() {
    local usuario="$1"
    local caso="$2"
    local fachada="$3"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --username "${usuario}" \
        --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT pg_catalog.concat_ws(
           '|', decision_ref, efecto_ref, huella_efecto_sha256,
           consumo_huella_sha256, auditoria_ref,
           auditoria_huella_sha256, consumida_en::text,
           consumo_nuevo::text
       )
  FROM vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
    '${caso}','${fachada}',''
  );
COMMIT;
SQL
}

definicion_audiencia() {
    valor "SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\\s+',' ','g')
             FROM pg_constraint c
            WHERE c.conrelid =
                  'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
              AND c.conname =
                  'clave_capacidad_version_audiencia_consumo_check'"
}

paso "arranque efímero sin red: ${imagen}"
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

paso 'endurecimiento y dependencias nominales'
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 --username postgres \
    --dbname postgres <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
DO $version$
BEGIN
  IF current_setting('server_version_num')::integer <> 180004 THEN
    RAISE EXCEPTION 'se exige PostgreSQL 18.4 exacto';
  END IF;
END
$version$;
SQL

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
    "CREATE ROLE vec_contexto_consulta_rrhh LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contexto_actor_v1_runtime TO vec_contexto_consulta_rrhh WITH ADMIN FALSE, INHERIT TRUE, SET FALSE" \
    >/dev/null
docker exec --interactive "${contenedor}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_contexto_consulta_rrhh --dbname postgres <<'SQL' >/dev/null
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
sql postgres \
    'CREATE EXTENSION pgcrypto WITH SCHEMA public; REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC' \
    >/dev/null
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
    deploy/postgresql/autorizacion/pruebas_sql/integracion_contexto_actor_v3.sql \
    deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql
do
    archivo postgres "${ruta}" >/dev/null
done
archivo postgres deploy/postgresql/contratacion_temporal/roles_up.sql >/dev/null
sql postgres \
    "CREATE ROLE vec_ct_consulta_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ct_consulta_migrador; GRANT vec_contratacion_temporal_migrador TO vec_ct_consulta_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE" \
    >/dev/null
for migracion in \
    000001_preparacion_altas \
    000002_rotacion_hmac
do
    archivo vec_ct_consulta_migrador \
        "deploy/postgresql/contratacion_temporal/migraciones/${migracion}.up.sql" \
        >/dev/null
done
sql postgres \
    "CREATE ROLE vec_contratacion_temporal_consultor_rrhh NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; CREATE ROLE vec_consulta_rrhh_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_contratacion_temporal_consultor_rrhh; GRANT vec_contratacion_temporal_consultor_rrhh TO vec_consulta_rrhh_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE" \
    >/dev/null
archivo postgres deploy/postgresql/autorizacion_atestada_v3/roles_up.sql \
    >/dev/null
sql postgres \
    "CREATE ROLE vec_ad3_consulta_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ad3_consulta_migrador; GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_consulta_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE" \
    >/dev/null
for migracion in \
    000001_gobierno_y_registro_v3 \
    000002_consumidor_capacidad_v3
do
    archivo vec_ad3_consulta_migrador \
        "deploy/postgresql/autorizacion_atestada_v3/migraciones/${migracion}.up.sql" \
        >/dev/null
done
for migracion in \
    000003_expediente_confirmacion_atestada \
    000004_integridad_agregado_alta \
    000005_funcion_confirmar_alta_atestada
do
    archivo vec_ct_consulta_migrador \
        "deploy/postgresql/contratacion_temporal/migraciones/${migracion}.up.sql" \
        >/dev/null
done
sql postgres \
    'GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer) TO vec_autorizacion_atestada_v3_propietario' \
    >/dev/null
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/preparar_entorno_o2_05.sql \
    >/dev/null

alta="CHECK (audiencia_consumo = 'vec_contratacion_temporal.confirmar_alta_atestada.v1'::text)"
alta_cuadro="CHECK (audiencia_consumo = ANY (ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1'::text, 'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'::text]))"
alta_cuadro_detalle="CHECK (audiencia_consumo = ANY (ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1'::text, 'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'::text, 'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'::text]))"

paso 'cuatro fotos exactas de migración y reversión limpia'
[[ "$(definicion_audiencia)" == "${alta}" ]]
archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.up.sql \
    >/dev/null
[[ "$(definicion_audiencia)" == "${alta_cuadro}" ]]
archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.up.sql \
    >/dev/null
[[ "$(definicion_audiencia)" == "${alta_cuadro_detalle}" ]]

sql postgres \
    "SET ROLE vec_autorizacion_atestada_v3_propietario; ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check; ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1','audiencia_adulterada'))" \
    >/dev/null
esperar_fallo 'down rechaza CHECK adulterado' \
    archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.down.sql
[[ "$(valor "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL")" == 't' ]]
sql postgres \
    "SET ROLE vec_autorizacion_atestada_v3_propietario; ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check; ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'))" \
    >/dev/null
archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.down.sql \
    >/dev/null
[[ "$(definicion_audiencia)" == "${alta_cuadro}" ]]
sql postgres \
    "SET ROLE vec_autorizacion_atestada_v3_propietario; ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check; ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','audiencia_adulterada'))" \
    >/dev/null
esperar_fallo 'down cuadro rechaza CHECK adulterado' \
    archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.down.sql
[[ "$(valor "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL")" == 't' ]]
sql postgres \
    "SET ROLE vec_autorizacion_atestada_v3_propietario; ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check; ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'))" \
    >/dev/null
archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.down.sql \
    >/dev/null
[[ "$(definicion_audiencia)" == "${alta}" ]]
[[ "$(valor "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_decision_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND has_function_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_decision_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')")" == 't' ]]
archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.up.sql \
    >/dev/null
archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.up.sql \
    >/dev/null

paso 'regresión funcional de alta tras ambas migraciones de consulta'
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ayudantes_o2_05.sql \
    >/dev/null
sql postgres \
    "CREATE ROLE vec_alta_regresion_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_alta_regresion_runtime; GRANT vec_contratacion_temporal_ejecutor TO vec_alta_regresion_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE; GRANT USAGE ON SCHEMA public TO vec_alta_regresion_runtime; REVOKE ALL ON FUNCTION public.invocar_vector_o2_05(text) FROM PUBLIC; GRANT EXECUTE ON FUNCTION public.invocar_vector_o2_05(text) TO vec_alta_regresion_runtime" \
    >/dev/null
sql postgres \
    "SELECT public.preparar_vector_o2_05('alta_regresion','valido',1)" \
    >/dev/null
[[ "$(docker exec --interactive "${contenedor}" psql -XAtq \
    --set ON_ERROR_STOP=1 --username vec_alta_regresion_runtime \
    --dbname postgres <<'SQL'
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT count(*) FROM public.invocar_vector_o2_05('alta_regresion');
COMMIT;
SQL
)" == '1' ]]

paso 'fixtures mínimos, fachada CT aislada y ACL'
archivo postgres \
    deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/consultas_rrhh_v3.sql \
    >/dev/null
[[ "$(valor "WITH funciones(nombre) AS (VALUES
    ('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    ('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
), denegados(rol) AS (VALUES
    ('public'), ('vec_contratacion_temporal_consultor_rrhh'),
    ('vec_contratacion_temporal_ejecutor'),
    ('vec_autorizacion_atestada_v3_consumidor'),
    ('vec_autorizacion_atestada_v3_emisor')
) SELECT
    NOT has_function_privilege(
        'vec_contratacion_temporal_propietario',
        'vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
        'EXECUTE')
    AND NOT EXISTS (
        SELECT 1 FROM denegados d
         WHERE has_function_privilege(
             d.rol,
             'vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
             'EXECUTE'))
    AND NOT EXISTS (
        SELECT 1 FROM funciones f, denegados d
         WHERE has_function_privilege(d.rol, f.nombre, 'EXECUTE'))
    AND NOT EXISTS (
        SELECT 1 FROM funciones f
         WHERE NOT has_function_privilege(
             'vec_contratacion_temporal_propietario',
             f.nombre, 'EXECUTE'))")" == 't' ]]
[[ "$(valor "SELECT
    (SELECT count(*) FROM pg_auth_members m
      JOIN pg_roles u ON u.oid=m.member
     WHERE u.rolname='vec_consulta_rrhh_runtime') = 1
    AND EXISTS (
        SELECT 1 FROM pg_auth_members m
        JOIN pg_roles u ON u.oid=m.member
        JOIN pg_roles r ON r.oid=m.roleid
        WHERE u.rolname='vec_consulta_rrhh_runtime'
          AND r.rolname='vec_contratacion_temporal_consultor_rrhh'
          AND NOT m.admin_option AND m.inherit_option
          AND NOT m.set_option)
    AND EXISTS (
        SELECT 1 FROM pg_roles
         WHERE rolname='vec_contratacion_temporal_consultor_rrhh'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
           AND NOT rolcreaterole AND rolinherit
           AND NOT rolreplication AND NOT rolbypassrls)")" == 't' ]]
esperar_fallo 'consultor no entra directamente en VEC-AD-3' \
    sql vec_consulta_rrhh_runtime \
    "SELECT * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)"

paso 'atomicidad de rollback sin trazas parciales'
preparar rollback_atomico cuadro
estado_antes="$(estado_vec)"
docker exec --interactive "${contenedor}" \
    psql -XAtq --set ON_ERROR_STOP=1 \
    --username vec_consulta_rrhh_runtime --dbname postgres <<'SQL' >/dev/null
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT count(*)
  FROM vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
    'rollback_atomico','cuadro',''
  );
ROLLBACK;
SQL
[[ "$(estado_vec)" == "${estado_antes}" ]]

paso 'éxito nominal, recibo íntegro y replay solo de conciliación'
preparar cuadro_valido cuadro
recibo_primero="$(
    recibo_usuario vec_consulta_rrhh_runtime cuadro_valido cuadro
)"
[[ "${recibo_primero##*|}" == 'true' ]]
estado_primero="$(estado_vec)"
recibo_replay="$(
    recibo_usuario vec_consulta_rrhh_runtime cuadro_valido cuadro
)"
[[ "${recibo_replay##*|}" == 'false' ]]
[[ "${recibo_primero%|*}" == "${recibo_replay%|*}" ]]
[[ "$(estado_vec)" == "${estado_primero}" ]]
IFS='|' read -r _ _ _ _ auditoria_ref auditoria_huella _ _ \
    <<<"${recibo_primero}"
[[ "$(valor "SELECT huella_sha256
    FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
    WHERE auditoria_ref='${auditoria_ref}'")" == "${auditoria_huella}" ]]
preparar detalle_valido detalle
[[ "$(invocar_usuario vec_consulta_rrhh_runtime detalle_valido detalle)" == 't' ]]

paso 'cruces A/B y listas positivas de decisión/capacidad'
preparar cruce_a detalle
esperar_fallo 'detalle no entra por fachada cuadro' \
    invocar_usuario vec_consulta_rrhh_runtime cruce_a cuadro
preparar cruce_b cuadro
esperar_fallo 'cuadro no entra por fachada detalle' \
    invocar_usuario vec_consulta_rrhh_runtime cruce_b detalle
for especificacion in \
    'modulo_id:otro_modulo' \
    'tipo_recurso:otro_tipo' \
    'finalidad:otra_finalidad' \
    'accion:otra_accion'
do
    campo="${especificacion%%:*}"
    contenido="${especificacion#*:}"
    caso="decision_${campo}"
    preparar "${caso}" cuadro
    sql postgres \
        "SELECT public.adulterar_decision_consulta_rrhh_v3('${caso}','${campo}','${contenido}')" \
        >/dev/null
    esperar_fallo "decisión rechaza ${campo}" \
        invocar_usuario vec_consulta_rrhh_runtime "${caso}" cuadro
done
for especificacion in \
    'audiencia_consumo:audiencia_incorrecta' \
    'operacion:operacion_incorrecta'
do
    campo="${especificacion%%:*}"
    contenido="${especificacion#*:}"
    caso="capacidad_${campo}"
    preparar "${caso}" cuadro
    sql postgres \
        "SELECT public.adulterar_capacidad_consulta_rrhh_v3('${caso}','${campo}','${contenido}')" \
        >/dev/null
    esperar_fallo "capacidad rechaza ${campo}" \
        invocar_usuario vec_consulta_rrhh_runtime "${caso}" cuadro
done

paso 'adulteración independiente de las diez piezas y DER-SPKI'
estado_antes="$(estado_vec)"
for pieza in \
    capacidad decision motivo contexto persona_version perfil_version \
    payload cose evidencia spki decision_null spki_null \
    spki_x25519 spki_der_no_canonico spki_rsa
do
    caso="pieza_${pieza}"
    preparar "${caso}" cuadro
    esperar_fallo "pieza adulterada: ${pieza}" \
        invocar_usuario vec_consulta_rrhh_runtime "${caso}" cuadro "${pieza}"
done
[[ "$(estado_vec)" == "${estado_antes}" ]]

paso 'replay cruzado múltiple falla de forma determinista'
preparar replay_a cuadro
preparar replay_b cuadro
invocar_usuario vec_consulta_rrhh_runtime replay_a cuadro >/dev/null
invocar_usuario vec_consulta_rrhh_runtime replay_b cuadro >/dev/null
sql postgres \
    "SELECT public.crear_colision_replay_consulta_rrhh_v3('replay_colision','replay_a','replay_b')" \
    >/dev/null
esperar_fallo 'dos coincidencias de replay' \
    invocar_usuario vec_consulta_rrhh_runtime replay_colision cuadro

paso 'rol adicional y caducidad durante revalidación fallan cerrados'
sql postgres \
    "CREATE ROLE vec_consulta_rrhh_rol_invalido LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contratacion_temporal_consultor_rrhh TO vec_consulta_rrhh_rol_invalido WITH ADMIN FALSE, INHERIT TRUE, SET FALSE; GRANT vec_contratacion_temporal_ejecutor TO vec_consulta_rrhh_rol_invalido WITH ADMIN FALSE, INHERIT TRUE, SET FALSE" \
    >/dev/null
preparar rol_invalido cuadro
esperar_fallo 'LOGIN con segundo grupo queda rechazado' \
    invocar_usuario vec_consulta_rrhh_rol_invalido rol_invalido cuadro

preparar expira_concurrente cuadro
sleep 3.6
sql postgres \
    "BEGIN; SET ROLE vec_autorizacion_propietario; SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref='prf_sintetico_cccccccccccccccccccccccc' FOR UPDATE; SELECT pg_sleep(1.7); COMMIT" \
    >/dev/null &
pid_bloqueo=$!
sleep 0.15
estado_antes="$(estado_vec)"
esperar_fallo 'capacidad expira durante revalidación RBAC' \
    invocar_usuario vec_consulta_rrhh_runtime expira_concurrente cuadro
wait "${pid_bloqueo}"
[[ "$(estado_vec)" == "${estado_antes}" ]]

paso 'linealización: consumo primero y revocación confirmada después'
preparar revocacion_concurrente detalle
salida_consumidor="$(mktemp)"
docker exec --interactive "${contenedor}" \
    psql -XAtq --set ON_ERROR_STOP=1 \
    --username vec_consulta_rrhh_runtime --dbname postgres \
    >"${salida_consumidor}" <<'SQL' &
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT consumo_nuevo
  FROM vec_contratacion_temporal.prueba_consumir_consulta_rrhh_v3(
    'revocacion_concurrente','detalle',''
  );
SELECT pg_sleep(1.2);
COMMIT;
SQL
pid_consumidor=$!
sleep 0.2
sql postgres \
    "BEGIN; SET ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad VALUES ('clave-consulta-detalle-rrhh-v3',3,clock_timestamp(),'motivo:prueba-concurrente','acto:revocacion:consulta-detalle',clock_timestamp()); COMMIT" \
    >/dev/null
wait "${pid_consumidor}"
[[ "$(tr -d '[:space:]' <"${salida_consumidor}")" == 't' ]]
rm -f "${salida_consumidor}"

paso 'linealización: revocación confirmada primero impide consumo'
preparar revocacion_primero cuadro
sql postgres \
    "BEGIN; SET ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad VALUES ('clave-consulta-cuadro-rrhh-v3',2,clock_timestamp(),'motivo:prueba-revocacion-previa','acto:revocacion:consulta-cuadro',clock_timestamp()); COMMIT" \
    >/dev/null
estado_antes="$(estado_vec)"
esperar_fallo 'revocación confirmada antes de consumir' \
    invocar_usuario vec_consulta_rrhh_runtime revocacion_primero cuadro
[[ "$(estado_vec)" == "${estado_antes}" ]]

paso 'down queda absolutamente protegido por claves, punteros e historia'
esperar_fallo 'down detalle protegido' \
    archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.down.sql
esperar_fallo 'down cuadro protegido' \
    archivo vec_ad3_consulta_migrador \
    deploy/postgresql/autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.down.sql
[[ "$(definicion_audiencia)" == "${alta_cuadro_detalle}" ]]
[[ "$(valor "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL")" == 't' ]]

paso 'todas las puertas PostgreSQL 18.4 superadas'
