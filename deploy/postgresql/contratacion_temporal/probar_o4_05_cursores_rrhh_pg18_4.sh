#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-ct-o405-cursores-${PPID}-${RANDOM}"
clave_archivo="$(mktemp "${TMPDIR:-/tmp}/vec-o405-cursores-clave.XXXXXX")"
temporales=("${clave_archivo}")

limpiar() {
  local temporal
  for temporal in "${temporales[@]}"; do
    rm -f -- "${temporal}"
  done
  docker rm --force --volumes "${contenedor}" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

paso() {
  printf '[O4-05:C2-D1:PG18.4] %s\n' "$1"
}

psql_bd() {
  local base_datos="$1"
  shift
  docker exec --interactive "${contenedor}" psql -X \
    --set ON_ERROR_STOP=1 --username postgres --dbname "${base_datos}" "$@"
}

psql_admin() {
  psql_bd postgres "$@"
}

psql_rol() {
  local rol="$1"
  shift
  docker exec --interactive "${contenedor}" psql -X \
    --set ON_ERROR_STOP=1 --username "${rol}" --dbname postgres "$@"
}

archivo() {
  psql_admin --file "/repo/$1" >/dev/null
}

archivo_bd() {
  local base_datos="$1"
  local ruta="$2"
  psql_bd "${base_datos}" --file "/repo/${ruta}" >/dev/null
}

valor() {
  valor_bd postgres "$1"
}

valor_bd() {
  local base_datos="$1"
  local consulta="$2"
  docker exec "${contenedor}" psql -XAtq \
    --set ON_ERROR_STOP=1 --username postgres --dbname "${base_datos}" \
    --command "${consulta}"
}

esperar_fallo() {
  local descripcion="$1"
  local salida
  shift
  salida="$(mktemp "${TMPDIR:-/tmp}/vec-o405-cursores-fallo.XXXXXX")"
  temporales+=("${salida}")
  if "$@" >"${salida}" 2>&1; then
    printf 'se esperaba rechazo: %s\n' "${descripcion}" >&2
    return 1
  fi
  paso "rechazo verificado: ${descripcion}"
}

par_migracion() {
  local ruta="$1"
  local descripcion="$2"
  local salida_a salida_b estado_a estado_b
  salida_a="$(mktemp "${TMPDIR:-/tmp}/vec-o405-cursores-a.XXXXXX")"
  salida_b="$(mktemp "${TMPDIR:-/tmp}/vec-o405-cursores-b.XXXXXX")"
  temporales+=("${salida_a}" "${salida_b}")
  psql_admin --file "/repo/${ruta}" >"${salida_a}" 2>&1 &
  local pid_a=$!
  psql_admin --file "/repo/${ruta}" >"${salida_b}" 2>&1 &
  local pid_b=$!
  estado_a=0
  estado_b=0
  wait "${pid_a}" || estado_a=$?
  wait "${pid_b}" || estado_b=$?
  if (( (estado_a == 0) + (estado_b == 0) != 1 )); then
    printf 'se esperaba un ganador en %s: %s/%s\n' \
      "${descripcion}" "${estado_a}" "${estado_b}" >&2
    sed -n '1,16p' "${salida_a}" >&2
    sed -n '1,16p' "${salida_b}" >&2
    return 1
  fi
  paso "un único ganador verificado: ${descripcion}"
}

carrera_roles_down_up() {
  local salida_down salida_up estado_down estado_up
  salida_down="$(mktemp "${TMPDIR:-/tmp}/vec-o405-roles-down.XXXXXX")"
  salida_up="$(mktemp "${TMPDIR:-/tmp}/vec-o405-migracion-up.XXXXXX")"
  temporales+=("${salida_down}" "${salida_up}")
  archivo contratacion_temporal/roles_cursor_rrhh_down.sql \
    >"${salida_down}" 2>&1 &
  local pid_down=$!
  archivo contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.up.sql \
    >"${salida_up}" 2>&1 &
  local pid_up=$!
  estado_down=0
  estado_up=0
  wait "${pid_down}" || estado_down=$?
  wait "${pid_up}" || estado_up=$?
  if (( (estado_down == 0) + (estado_up == 0) != 1 )); then
    printf 'carrera roles_down/000038.up sin ganador único: %s/%s\n' \
      "${estado_down}" "${estado_up}" >&2
    sed -n '1,16p' "${salida_down}" >&2
    sed -n '1,16p' "${salida_up}" >&2
    return 1
  fi
  paso 'un único ganador verificado: roles_down contra 000038.up'
}

probar_deriva() {
  local caso="$1"
  local descripcion="$2"
  local base_datos="vec_o405_deriva_${caso}"
  local estado_antes
  psql_bd template1 --command \
    "CREATE DATABASE ${base_datos} TEMPLATE postgres" >/dev/null
  psql_bd "${base_datos}" \
    --set caso="${caso}" --set accion=preparar --file \
    /repo/contratacion_temporal/pruebas_sql/o405_cursores_rrhh_deriva.sql \
    >/dev/null
  estado_antes="$(estado_cursores_bd "${base_datos}")"
  esperar_fallo "${descripcion}" \
    archivo_bd "${base_datos}" \
    contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.down.sql
  [[ "$(estado_cursores_bd "${base_datos}")" == "${estado_antes}" ]]
  psql_bd "${base_datos}" \
    --set caso="${caso}" --set accion=verificar --file \
    /repo/contratacion_temporal/pruebas_sql/o405_cursores_rrhh_deriva.sql \
    >/dev/null
  psql_bd template1 --command "DROP DATABASE ${base_datos}" >/dev/null
}

probar_encadenamiento() {
  local base_datos='vec_o405_encadenamiento'
  psql_bd template1 --command \
    "CREATE DATABASE ${base_datos} TEMPLATE postgres" >/dev/null
  archivo_bd "${base_datos}" \
    contratacion_temporal/pruebas_sql/o405_cursores_rrhh_encadenamiento.sql
  psql_bd template1 --command "DROP DATABASE ${base_datos}" >/dev/null
  paso 'p2, p3 y separación emisión/consumo verificados'
}

estado_cursores() {
  estado_cursores_bd postgres
}

estado_cursores_bd() {
  local base_datos="$1"
  valor_bd "${base_datos}" "SELECT pg_catalog.concat_ws('|',
    (SELECT version_esquema
       FROM vec_contratacion_temporal.control_migracion_cobertura_o4
      WHERE control),
    (SELECT version_esquema
       FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
      WHERE control),
    pg_catalog.to_regclass(
      'vec_contratacion_temporal.control_cursores_cuadro_rrhh'
    ) IS NOT NULL,
    pg_catalog.to_regclass(
      'vec_contratacion_temporal.alcance_acceso_rrhh'
    ) IS NOT NULL,
    pg_catalog.to_regclass(
      'vec_contratacion_temporal.familia_cursor_cuadro_rrhh'
    ) IS NOT NULL,
    pg_catalog.to_regclass(
      'vec_contratacion_temporal.cursor_cuadro_rrhh'
    ) IS NOT NULL,
    pg_catalog.to_regclass(
      'vec_contratacion_temporal.consumo_cursor_cuadro_rrhh'
    ) IS NOT NULL,
    pg_catalog.to_regclass(
      'vec_contratacion_temporal.revocacion_familia_cursor_rrhh'
    ) IS NOT NULL,
    pg_catalog.has_schema_privilege(
      'vec_contratacion_temporal_propietario','public','USAGE'
    ),
    pg_catalog.has_function_privilege(
      'vec_contratacion_temporal_propietario',
      'public.gen_random_bytes(integer)','EXECUTE'
    ))"
}

conteos_cursores() {
  valor "SELECT pg_catalog.concat_ws('|',
    (SELECT count(*)
       FROM vec_contratacion_temporal.alcance_acceso_rrhh),
    (SELECT count(*)
       FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh),
    (SELECT count(*)
       FROM vec_contratacion_temporal.cursor_cuadro_rrhh),
    (SELECT count(*)
       FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh),
    (SELECT count(*)
       FROM vec_contratacion_temporal.revocacion_familia_cursor_rrhh))"
}

if [[ ! "${imagen}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
  printf 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256\n' >&2
  exit 64
fi

chmod 0600 "${clave_archivo}"
openssl rand -hex 24 >"${clave_archivo}"

paso "arranque efímero, sin red ni puertos: ${imagen}"
docker run --detach --name "${contenedor}" --network none --read-only \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password \
  --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
  --mount \
  "type=bind,source=${clave_archivo},target=/run/secrets/postgres_password,readonly" \
  --mount "type=bind,source=${raiz}/deploy/postgresql,target=/repo,readonly" \
  --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=1024m \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m \
  --tmpfs /run/postgresql:rw,noexec,nosuid,size=16m \
  "${imagen}" >/dev/null
for _ in {1..60}; do
  if docker exec "${contenedor}" pg_isready --quiet \
    --username postgres --dbname postgres; then
    break
  fi
  sleep 1
done
docker exec "${contenedor}" pg_isready --quiet \
  --username postgres --dbname postgres

psql_admin <<'SQL' >/dev/null
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
DO $version$
BEGIN
  IF pg_catalog.current_setting('server_version_num')::integer <> 180004 THEN
    RAISE EXCEPTION 'se exige PostgreSQL 18.4 exacto';
  END IF;
END
$version$;
SQL

paso 'instalación de dependencias reales'
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
  archivo "${ruta}"
  if [[ "${ruta}" == contexto_actor_v1/migraciones/000002_* ]]; then
    archivo contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql
    archivo autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
    psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o405_contexto LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO vec_o405_contexto
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
    docker exec --interactive "${contenedor}" psql -X \
      --set ON_ERROR_STOP=1 --username vec_o405_contexto \
      --dbname postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT count(*)
FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
  'oca_registro_v3_000000000000000000000000',
  'rca_registro_v3_000000000000000000000000',
  'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
  'prf_sintetico_cccccccccccccccccccccccc',
  'certificado','alto',clock_timestamp());
COMMIT;
SQL
    psql_admin <<'SQL' >/dev/null
REVOKE vec_contexto_actor_v1_runtime FROM vec_o405_contexto;
DROP ROLE vec_o405_contexto;
SQL
  fi
done
archivo autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql
archivo \
  contratacion_temporal/migraciones_autorizacion/000006_wrapper_contexto_exacto_cobertura_o4_04e.up.sql

for numero in $(seq -w 1 34); do
  nombre="$(basename \
    "${raiz}"/deploy/postgresql/contratacion_temporal/migraciones/0000"${numero}"_*.up.sql)"
  if [[ "${numero}" == 03 ]]; then
    psql_admin <<'SQL' >/dev/null
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
    archivo autorizacion_atestada_v3/roles_up.sql
    archivo \
      autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql
    archivo \
      autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql
  fi
  archivo "contratacion_temporal/migraciones/${nombre}"
done
archivo contratacion_temporal/roles_lector_resultado_cobertura_up.sql
archivo \
  contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.up.sql
archivo contratacion_temporal/roles_consultor_rrhh_up.sql
archivo \
  contratacion_temporal/migraciones/000036_registro_accesos_rrhh_o4_05.up.sql

paso 'accesos sintéticos previos y publicación global 000037'
archivo contratacion_temporal/pruebas_sql/o405_registro_accesos_rrhh.sql
psql_admin --set preparar=1 --file \
  /repo/contratacion_temporal/pruebas_sql/o405_publicacion_global_rrhh.sql \
  >/dev/null
archivo \
  contratacion_temporal/migraciones/000037_publicacion_global_rrhh.up.sql
psql_admin --set preparar=0 --file \
  /repo/contratacion_temporal/pruebas_sql/o405_publicacion_global_rrhh.sql \
  >/dev/null
psql_admin --set preparar=1 --file \
  /repo/contratacion_temporal/pruebas_sql/o405_cursores_rrhh.sql \
  >/dev/null

paso 'preflight criptográfico y ACL mínima'
esperar_fallo 'delta sin CSPRNG ejecutable por el propietario' \
  archivo contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.up.sql
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_identidad_sesiones_v1_propietario
  NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
  NOREPLICATION NOBYPASSRLS;
GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer)
  TO vec_identidad_sesiones_v1_propietario WITH GRANT OPTION;
SQL
esperar_fallo 'identidad con GRANT OPTION impide el delta' \
  archivo contratacion_temporal/roles_cursor_rrhh_up.sql
[[ "$(valor "SELECT EXISTS (
  SELECT 1
    FROM pg_catalog.pg_proc funcion
   CROSS JOIN LATERAL pg_catalog.aclexplode(funcion.proacl) privilegio
    JOIN pg_catalog.pg_roles rol ON rol.oid = privilegio.grantee
   WHERE funcion.oid =
         'public.gen_random_bytes(integer)'::regprocedure
     AND rol.rolname = 'vec_identidad_sesiones_v1_propietario'
     AND privilegio.privilege_type = 'EXECUTE'
     AND privilegio.is_grantable)")" == 't' ]]
psql_admin <<'SQL' >/dev/null
REVOKE GRANT OPTION FOR EXECUTE
  ON FUNCTION public.gen_random_bytes(integer)
  FROM vec_identidad_sesiones_v1_propietario;
CREATE ROLE vec_o405_miembro_identidad_hostil
  LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
  NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_propietario
  TO vec_o405_miembro_identidad_hostil
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
esperar_fallo 'LOGIN miembro del propietario de identidad impide el delta' \
  archivo contratacion_temporal/roles_cursor_rrhh_up.sql
[[ "$(valor "SELECT EXISTS (
  SELECT 1
    FROM pg_catalog.pg_auth_members membresia
    JOIN pg_catalog.pg_roles grupo ON grupo.oid = membresia.roleid
    JOIN pg_catalog.pg_roles miembro ON miembro.oid = membresia.member
   WHERE grupo.rolname = 'vec_identidad_sesiones_v1_propietario'
     AND miembro.rolname = 'vec_o405_miembro_identidad_hostil'
     AND membresia.inherit_option)")" == 't' ]]
psql_admin <<'SQL' >/dev/null
REVOKE vec_identidad_sesiones_v1_propietario
  FROM vec_o405_miembro_identidad_hostil;
DROP ROLE vec_o405_miembro_identidad_hostil;
CREATE ROLE vec_o405_criptografia_hostil
  LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
  NOREPLICATION NOBYPASSRLS;
GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer)
  TO vec_o405_criptografia_hostil;
SQL
esperar_fallo 'LOGIN hostil con CSPRNG impide el delta' \
  archivo contratacion_temporal/roles_cursor_rrhh_up.sql
[[ "$(valor "SELECT
  NOT has_schema_privilege(
    'vec_contratacion_temporal_propietario','public','USAGE')
  AND NOT has_function_privilege(
    'vec_contratacion_temporal_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')
  AND has_function_privilege(
    'vec_identidad_sesiones_v1_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')
  AND has_function_privilege(
    'vec_o405_criptografia_hostil',
    'public.gen_random_bytes(integer)','EXECUTE')")" == 't' ]]
psql_admin <<'SQL' >/dev/null
REVOKE EXECUTE ON FUNCTION public.gen_random_bytes(integer)
  FROM vec_o405_criptografia_hostil;
DROP ROLE vec_o405_criptografia_hostil;
SQL

paso 'privilegios CT preexistentes se rechazan y conservan'
psql_admin <<'SQL' >/dev/null
GRANT USAGE ON SCHEMA public
  TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer)
  TO vec_contratacion_temporal_propietario;
SQL
esperar_fallo 'CSPRNG preexistente impide atribuir procedencia al delta' \
  archivo contratacion_temporal/roles_cursor_rrhh_up.sql
[[ "$(valor "SELECT
  has_schema_privilege(
    'vec_contratacion_temporal_propietario','public','USAGE')
  AND has_function_privilege(
    'vec_contratacion_temporal_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')
  AND has_function_privilege(
    'vec_identidad_sesiones_v1_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')")" == 't' ]]
psql_admin <<'SQL' >/dev/null
REVOKE EXECUTE ON FUNCTION public.gen_random_bytes(integer)
  FROM vec_contratacion_temporal_propietario;
REVOKE USAGE ON SCHEMA public
  FROM vec_contratacion_temporal_propietario;
SQL
archivo contratacion_temporal/roles_cursor_rrhh_up.sql
[[ "$(valor "SELECT
  has_schema_privilege(
    'vec_contratacion_temporal_propietario','public','USAGE')
  AND has_function_privilege(
    'vec_contratacion_temporal_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')
  AND NOT has_schema_privilege('public','public','USAGE')
  AND NOT has_function_privilege(
    'public','public.gen_random_bytes(integer)','EXECUTE')
  AND has_function_privilege(
    'vec_identidad_sesiones_v1_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')")" == 't' ]]

paso 'carrera roles_down contra instalación 000038'
carrera_roles_down_up
case "$(estado_cursores)" in
  '18|2|t|t|t|t|t|t|t|t')
    archivo contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.down.sql
    archivo contratacion_temporal/roles_cursor_rrhh_down.sql
    ;;
  '17|1|f|f|f|f|f|f|f|f')
    ;;
  *)
    printf 'estado inconsistente tras carrera roles_down/000038.up\n' >&2
    exit 1
    ;;
esac
archivo contratacion_temporal/roles_cursor_rrhh_up.sql

paso 'up/down concurrentes con un único ganador'
par_migracion \
  contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.up.sql \
  'dos up concurrentes'
[[ "$(estado_cursores)" == \
  '18|2|t|t|t|t|t|t|t|t' ]]
[[ "$(conteos_cursores)" == '0|0|0|0|0' ]]
esperar_fallo 'retirada del delta antes del bloque 000038' \
  archivo contratacion_temporal/roles_cursor_rrhh_down.sql
par_migracion \
  contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.down.sql \
  'dos down concurrentes'
[[ "$(estado_cursores)" == \
  '17|1|f|f|f|f|f|f|t|t' ]]
archivo contratacion_temporal/roles_cursor_rrhh_down.sql
[[ "$(valor "SELECT NOT has_schema_privilege(
  'vec_contratacion_temporal_propietario','public','USAGE')
  AND NOT has_function_privilege(
    'vec_contratacion_temporal_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')
  AND has_function_privilege(
    'vec_identidad_sesiones_v1_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')")" == 't' ]]

paso 'reinstalación y matriz estructural/hostil'
archivo contratacion_temporal/roles_cursor_rrhh_up.sql
archivo \
  contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.up.sql
psql_admin --set preparar=0 --file \
  /repo/contratacion_temporal/pruebas_sql/o405_cursores_rrhh.sql \
  >/dev/null
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o405_cursor_runtime
  LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
  NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_consultor_rrhh
  TO vec_o405_cursor_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
esperar_fallo 'runtime sin acceso directo a historia de cursor' \
  psql_rol vec_o405_cursor_runtime --command \
  'SELECT count(*) FROM vec_contratacion_temporal.cursor_cuadro_rrhh'
esperar_fallo 'runtime sin acceso al CSPRNG' \
  psql_rol vec_o405_cursor_runtime --command \
  'SELECT public.gen_random_bytes(1)'

paso 'encadenamiento exacto de accesos por página'
probar_encadenamiento

paso 'cada historia impide down sin cambios parciales'
psql_admin <<'SQL' >/dev/null
CREATE UNLOGGED TABLE public.vec_o405_respaldo_alcance AS
  TABLE vec_contratacion_temporal.alcance_acceso_rrhh;
CREATE UNLOGGED TABLE public.vec_o405_respaldo_familia AS
  TABLE vec_contratacion_temporal.familia_cursor_cuadro_rrhh;
CREATE UNLOGGED TABLE public.vec_o405_respaldo_cursor AS
  TABLE vec_contratacion_temporal.cursor_cuadro_rrhh;
CREATE UNLOGGED TABLE public.vec_o405_respaldo_consumo AS
  TABLE vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;
CREATE UNLOGGED TABLE public.vec_o405_respaldo_revocacion AS
  TABLE vec_contratacion_temporal.revocacion_familia_cursor_rrhh;
SQL
for caso in \
  alcance:alcance_acceso_rrhh \
  familia:familia_cursor_cuadro_rrhh \
  cursor:cursor_cuadro_rrhh \
  consumo:consumo_cursor_cuadro_rrhh \
  revocacion:revocacion_familia_cursor_rrhh
do
  sufijo="${caso%%:*}"
  tabla="${caso#*:}"
  psql_admin <<'SQL' >/dev/null
SET session_replication_role = replica;
DELETE FROM vec_contratacion_temporal.revocacion_familia_cursor_rrhh;
DELETE FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;
DELETE FROM vec_contratacion_temporal.cursor_cuadro_rrhh;
DELETE FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh;
DELETE FROM vec_contratacion_temporal.alcance_acceso_rrhh;
RESET session_replication_role;
SQL
  if [[ "${sufijo}" == cursor ]]; then
    psql_admin <<'SQL' >/dev/null
SET session_replication_role = replica;
INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
    token_huella_sha256, familia_ref, padre_token_huella_sha256,
    pagina, padre_emitida_en, ultimo_actualizado_en,
    ultimo_expediente_ref, familia_creada_en, familia_valida_hasta,
    emitida_en, acceso_emision_ref, prueba_canonica, prueba_huella_sha256
)
SELECT
    token_huella_sha256, familia_ref, padre_token_huella_sha256,
    pagina, padre_emitida_en, ultimo_actualizado_en,
    ultimo_expediente_ref, familia_creada_en, familia_valida_hasta,
    emitida_en, acceso_emision_ref, prueba_canonica, prueba_huella_sha256
FROM public.vec_o405_respaldo_cursor LIMIT 1;
RESET session_replication_role;
SQL
  else
    psql_admin --command \
      "SET session_replication_role=replica;
       INSERT INTO vec_contratacion_temporal.${tabla}
       SELECT * FROM public.vec_o405_respaldo_${sufijo} LIMIT 1;
       RESET session_replication_role" >/dev/null
  fi
  estado_antes="$(estado_cursores)"
  esperar_fallo "historia aislada: ${sufijo}" \
    archivo \
    contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.down.sql
  [[ "$(estado_cursores)" == "${estado_antes}" ]]
done
psql_admin <<'SQL' >/dev/null
SET session_replication_role = replica;
DELETE FROM vec_contratacion_temporal.revocacion_familia_cursor_rrhh;
DELETE FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;
DELETE FROM vec_contratacion_temporal.cursor_cuadro_rrhh;
DELETE FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh;
DELETE FROM vec_contratacion_temporal.alcance_acceso_rrhh;
RESET session_replication_role;
DROP TABLE public.vec_o405_respaldo_alcance;
DROP TABLE public.vec_o405_respaldo_familia;
DROP TABLE public.vec_o405_respaldo_cursor;
DROP TABLE public.vec_o405_respaldo_consumo;
DROP TABLE public.vec_o405_respaldo_revocacion;
SQL
[[ "$(conteos_cursores)" == '0|0|0|0|0' ]]

paso 'deriva estructural o de seguridad bloquea down sin mutación'
probar_deriva constraint 'constraint homónimo y FK retirada'
probar_deriva trigger_definicion 'trigger homónimo redefinido'
probar_deriva trigger_deshabilitado 'trigger deshabilitado'
probar_deriva trigger_ri 'trigger RI interno deshabilitado'
probar_deriva regla 'regla ON INSERT añadida'
probar_deriva columna 'columna extra'
probar_deriva indice 'índice extra'
probar_deriva rls 'RLS y FORCE desactivados'
probar_deriva politica 'policy homónima ampliada a PUBLIC'
probar_deriva acl 'GRANT extra a PUBLIC'
probar_deriva propietario 'propietario de tabla sustituido'

paso 'dependencia futura sobre modelo vacío no cambia estado'
psql_admin <<'SQL' >/dev/null
CREATE VIEW vec_contratacion_temporal.prueba_dependencia_cursor_rrhh AS
SELECT control,version_esquema
  FROM vec_contratacion_temporal.control_cursores_cuadro_rrhh;
SQL
estado_antes="$(estado_cursores)"
esperar_fallo 'vista futura dependiente' \
  archivo contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.down.sql
[[ "$(estado_cursores)" == "${estado_antes}" ]]
psql_admin --command \
  'DROP VIEW vec_contratacion_temporal.prueba_dependencia_cursor_rrhh' \
  >/dev/null

paso 'barreras futuras sobre modelo vacío no cambian estado'
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=19 WHERE control AND version_esquema=18' \
  >/dev/null
estado_antes="$(estado_cursores)"
esperar_fallo 'barrera de cobertura futura' \
  archivo contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.down.sql
[[ "$(estado_cursores)" == "${estado_antes}" ]]
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=18 WHERE control AND version_esquema=19' \
  >/dev/null
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh SET version_esquema=3 WHERE control AND version_esquema=2' \
  >/dev/null
estado_antes="$(estado_cursores)"
esperar_fallo 'barrera de consultas futura' \
  archivo contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.down.sql
[[ "$(estado_cursores)" == "${estado_antes}" ]]
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh SET version_esquema=2 WHERE control AND version_esquema=3' \
  >/dev/null

paso 'down vacío y retirada final del delta'
archivo contratacion_temporal/migraciones/000038_cursores_cuadro_rrhh.down.sql
[[ "$(estado_cursores)" == '17|1|f|f|f|f|f|f|t|t' ]]
archivo contratacion_temporal/roles_cursor_rrhh_down.sql
[[ "$(valor "SELECT
  NOT has_schema_privilege(
    'vec_contratacion_temporal_propietario','public','USAGE')
  AND NOT has_function_privilege(
    'vec_contratacion_temporal_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')
  AND has_function_privilege(
    'vec_identidad_sesiones_v1_propietario',
    'public.gen_random_bytes(integer)','EXECUTE')")" == 't' ]]

paso 'resultado verde'
printf 'O4-05 C2-D1 PostgreSQL 18.4: GO técnico\n'
