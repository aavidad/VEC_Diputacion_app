#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-ct-o405-publicacion-${PPID}-${RANDOM}"
temporales=()

limpiar() {
  local temporal
  for temporal in "${temporales[@]}"; do
    rm -f -- "${temporal}"
  done
  docker rm --force --volumes "${contenedor}" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

paso() {
  printf '[O4-05:C2-C:PG18.4] %s\n' "$1"
}

psql_admin() {
  docker exec --interactive "${contenedor}" psql -X \
    --set ON_ERROR_STOP=1 --username postgres --dbname postgres "$@"
}

archivo() {
  psql_admin --file "/repo/$1" >/dev/null
}

valor() {
  docker exec "${contenedor}" psql -XAtq \
    --set ON_ERROR_STOP=1 --username postgres --dbname postgres \
    --command "$1"
}

estado_c2c() {
  valor "SELECT pg_catalog.concat_ws('|',
    (SELECT version_esquema
       FROM vec_contratacion_temporal.control_migracion_cobertura_o4
      WHERE control),
    (SELECT corte_base
       FROM vec_contratacion_temporal.control_publicacion_rrhh
      WHERE control),
    (SELECT ultimo_corte
       FROM vec_contratacion_temporal.control_publicacion_rrhh
      WHERE control),
    (SELECT count(*)
       FROM vec_contratacion_temporal.publicacion_version_rrhh),
    (SELECT coalesce(sum(corte_global),0)
       FROM vec_contratacion_temporal.publicacion_version_rrhh),
    to_regprocedure(
      'vec_contratacion_temporal.extraer_publicacion_rrhh_v1(text,numeric,jsonb,text,text,numeric,text,text,text,timestamp with time zone,text,text)'
    ) IS NOT NULL,
    to_regprocedure(
      'vec_contratacion_temporal.publicar_version_rrhh_v1()'
    ) IS NOT NULL,
    (SELECT count(*) FROM pg_catalog.pg_trigger
      WHERE NOT tgisinternal AND tgenabled='O'
        AND tgname=ANY(ARRAY[
          'expediente_version_integral_publicar_rrhh',
          'publicacion_version_rrhh_inmutable',
          'publicacion_version_rrhh_no_truncar'
        ])),
    (SELECT count(*) FROM pg_catalog.pg_class indice
      JOIN pg_catalog.pg_namespace esquema
        ON esquema.oid=indice.relnamespace
      WHERE esquema.nspname='vec_contratacion_temporal'
        AND indice.relname=ANY(ARRAY[
          'publicacion_rrhh_organizacion_orden_idx',
          'publicacion_rrhh_centro_orden_idx',
          'publicacion_rrhh_unidad_orden_idx',
          'publicacion_rrhh_filtros_idx'
        ])))"
}

esperar_fallo() {
  local descripcion="$1"
  local salida
  shift
  salida="$(mktemp "${TMPDIR:-/tmp}/vec-o405-publicacion.XXXXXX")"
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
  salida_a="$(mktemp "${TMPDIR:-/tmp}/vec-o405-par-a.XXXXXX")"
  salida_b="$(mktemp "${TMPDIR:-/tmp}/vec-o405-par-b.XXXXXX")"
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
    sed -n '1,12p' "${salida_a}" >&2
    sed -n '1,12p' "${salida_b}" >&2
    return 1
  fi
  paso "un único ganador verificado: ${descripcion}"
}

escritor() {
  local indice="$1"
  local pausa="$2"
  psql_admin \
    --set "expediente_ref=expediente:pub:concurrente:${indice}" \
    --set "numero_visible=2026/concurrente-${indice}" \
    --set "marca=${indice}" --set "pausa=${pausa}" \
    --file \
    /repo/contratacion_temporal/pruebas_sql/o405_publicacion_global_rrhh_concurrencia.sql \
    >/dev/null
}

if [[ ! "${imagen}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
  printf 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256\n' >&2
  exit 64
fi

paso "arranque efímero sin red: ${imagen}"
docker run --detach --name "${contenedor}" --network none \
  --env POSTGRES_PASSWORD="$(openssl rand -hex 24)" \
  --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
  --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=1024m \
  "${imagen}" >/dev/null
for _ in {1..60}; do
  docker exec "${contenedor}" pg_isready --quiet \
    --username postgres --dbname postgres && break
  sleep 1
done
docker exec "${contenedor}" pg_isready --quiet \
  --username postgres --dbname postgres
docker cp "${raiz}/deploy/postgresql/." "${contenedor}:/repo"

psql_admin <<'SQL' >/dev/null
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

paso 'accesos C2-B válidos preexistentes'
archivo \
  contratacion_temporal/pruebas_sql/o405_registro_accesos_rrhh.sql
[[ "$(valor "SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh")" == '2' ]]

paso 'historia vacía, singleton cero y ciclo 16→17→16 con accesos'
archivo \
  contratacion_temporal/migraciones/000037_publicacion_global_rrhh.up.sql
[[ "$(valor "SELECT concat_ws('|',m.version_esquema,p.corte_base,p.ultimo_corte,(SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh)) FROM vec_contratacion_temporal.control_migracion_cobertura_o4 m CROSS JOIN vec_contratacion_temporal.control_publicacion_rrhh p WHERE m.control AND p.control")" == '17|0|0|0' ]]
archivo \
  contratacion_temporal/migraciones/000037_publicacion_global_rrhh.down.sql
[[ "$(valor "SELECT version_esquema FROM vec_contratacion_temporal.control_migracion_cobertura_o4 WHERE control")" == '16' ]]

paso 'up/down concurrentes: un solo ganador'
par_migracion \
  contratacion_temporal/migraciones/000037_publicacion_global_rrhh.up.sql \
  'dos up concurrentes'
[[ "$(valor "SELECT version_esquema FROM vec_contratacion_temporal.control_migracion_cobertura_o4 WHERE control")" == '17' ]]
par_migracion \
  contratacion_temporal/migraciones/000037_publicacion_global_rrhh.down.sql \
  'dos down concurrentes'
[[ "$(valor "SELECT version_esquema FROM vec_contratacion_temporal.control_migracion_cobertura_o4 WHERE control")" == '16' ]]

paso 'fixture histórico no vacío y backfill determinista'
psql_admin --set preparar=1 --file \
  /repo/contratacion_temporal/pruebas_sql/o405_publicacion_global_rrhh.sql \
  >/dev/null
archivo \
  contratacion_temporal/migraciones/000037_publicacion_global_rrhh.up.sql
psql_admin --set preparar=0 --file \
  /repo/contratacion_temporal/pruebas_sql/o405_publicacion_global_rrhh.sql \
  >/dev/null

paso 'dos escritores: bloqueo, orden de COMMIT y cortes 7→8'
escritor 1 1.2 &
pid_primero=$!
sleep 0.2
escritor 2 0 &
pid_segundo=$!
sleep 0.2
[[ "$(valor "SELECT count(*) > 0 FROM pg_catalog.pg_stat_activity WHERE datname=current_database() AND usename='postgres' AND wait_event_type='Lock' AND query LIKE '%INSERT INTO vec_contratacion_temporal.expediente_version_integral%'")" == 't' ]]
wait "${pid_primero}"
wait "${pid_segundo}"
[[ "$(valor "SELECT string_agg(expediente_ref || '=' || corte_global::text,',' ORDER BY corte_global) FROM vec_contratacion_temporal.publicacion_version_rrhh WHERE corte_global BETWEEN 7 AND 8")" == 'expediente:pub:concurrente:1=7,expediente:pub:concurrente:2=8' ]]

paso 'ocho escritores concurrentes sin huecos'
pids=()
for indice in $(seq 3 10); do
  escritor "${indice}" 0 &
  pids+=("$!")
done
for pid in "${pids[@]}"; do
  wait "${pid}"
done
[[ "$(valor "SELECT count(*)=16 AND min(corte_global)=1 AND max(corte_global)=16 AND count(DISTINCT corte_global)=16 FROM vec_contratacion_temporal.publicacion_version_rrhh")" == 't' ]]
[[ "$(valor "SELECT corte_base=4 AND ultimo_corte=16 FROM vec_contratacion_temporal.control_publicacion_rrhh WHERE control")" == 't' ]]

paso 'down rechaza publicaciones posteriores sin cambio parcial'
estado_antes="$(estado_c2c)"
esperar_fallo 'historia posterior al corte base' \
  psql_admin --file \
  /repo/contratacion_temporal/migraciones/000037_publicacion_global_rrhh.down.sql
[[ "$(estado_c2c)" == "${estado_antes}" ]]

paso 'limpieza sintética de publicaciones posteriores'
psql_admin <<'SQL' >/dev/null
SET session_replication_role=replica;
DELETE FROM vec_contratacion_temporal.publicacion_version_rrhh
 WHERE corte_global > (
   SELECT corte_base
     FROM vec_contratacion_temporal.control_publicacion_rrhh
    WHERE control
 );
DELETE FROM vec_contratacion_temporal.expediente_version_integral historia
 WHERE NOT EXISTS (
   SELECT 1
     FROM vec_contratacion_temporal.publicacion_version_rrhh publicacion
    WHERE publicacion.expediente_ref = historia.expediente_ref
      AND publicacion.version = historia.version
 );
UPDATE vec_contratacion_temporal.control_publicacion_rrhh
   SET ultimo_corte=corte_base,
       actualizada_en=date_trunc('microseconds',clock_timestamp())
 WHERE control;
SET session_replication_role=origin;
SQL

paso 'dependencia futura impide down sin cambio parcial'
psql_admin <<'SQL' >/dev/null
CREATE VIEW vec_contratacion_temporal.prueba_dependencia_publicacion_rrhh AS
SELECT expediente_ref,version
  FROM vec_contratacion_temporal.publicacion_version_rrhh;
SQL
estado_antes="$(estado_c2c)"
esperar_fallo 'vista dependiente' \
  psql_admin --file \
  /repo/contratacion_temporal/migraciones/000037_publicacion_global_rrhh.down.sql
[[ "$(estado_c2c)" == "${estado_antes}" ]]
[[ "$(valor "SELECT to_regclass('vec_contratacion_temporal.prueba_dependencia_publicacion_rrhh') IS NOT NULL")" == 't' ]]
psql_admin --command \
  'DROP VIEW vec_contratacion_temporal.prueba_dependencia_publicacion_rrhh' \
  >/dev/null

paso 'divergencia impide down sin cambio parcial'
psql_admin <<'SQL' >/dev/null
SET session_replication_role=replica;
UPDATE vec_contratacion_temporal.publicacion_version_rrhh
   SET centro_ref='centro:publicacion:adulterado'
 WHERE corte_global=1;
SET session_replication_role=origin;
SQL
estado_antes="$(estado_c2c)"
esperar_fallo 'proyección divergente' \
  psql_admin --file \
  /repo/contratacion_temporal/migraciones/000037_publicacion_global_rrhh.down.sql
[[ "$(estado_c2c)" == "${estado_antes}" ]]
psql_admin <<'SQL' >/dev/null
SET session_replication_role=replica;
UPDATE vec_contratacion_temporal.publicacion_version_rrhh
   SET centro_ref='centro:publicacion:rrhh'
 WHERE corte_global=1;
SET session_replication_role=origin;
SQL

paso 'down permite retirar exclusivamente el bloque base'
archivo \
  contratacion_temporal/migraciones/000037_publicacion_global_rrhh.down.sql
[[ "$(valor "SELECT version_esquema=16 AND to_regclass('vec_contratacion_temporal.publicacion_version_rrhh') IS NULL AND (SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh)=2 FROM vec_contratacion_temporal.control_migracion_cobertura_o4 WHERE control")" == 't' ]]

paso 'resultado verde'
printf 'O4-05 C2-C PostgreSQL 18.4: GO técnico\n'
