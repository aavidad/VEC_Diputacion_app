#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct-o404e-pg18-${PPID}-${RANDOM}"
volumen="${contenedor}-datos"
clave="$(openssl rand -hex 24)"
limpiar() {
  if [[ ${VEC_MANTENER_CONTENEDOR_FALLIDO:-0} == 1 ]]; then
    printf 'contenedor conservado: %s\n' "$contenedor" >&2
    return
  fi
  docker rm --force --volumes "$contenedor" >/dev/null 2>&1 || true
  docker volume rm --force "$volumen" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM
paso() {
  printf '[O4-04E:PG18.4] %s\n' "$1"
}
psql_admin() {
  docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname postgres "$@"
}
archivo() {
  psql_admin --file "/repo/$1" >/dev/null
}
esperar_fallo() {
  local descripcion=$1
  shift
  if "$@" >/tmp/o404e-fallo.$$ 2>&1; then
    printf 'se esperaba rechazo: %s\n' "$descripcion" >&2
    rm -f /tmp/o404e-fallo.$$
    return 1
  fi
  rm -f /tmp/o404e-fallo.$$
}
esperar_sqlstate() {
  local descripcion=$1 codigo=$2
  shift 2
  if "$@" >/tmp/o404e-fallo.$$ 2>&1; then
    printf 'se esperaba SQLSTATE %s: %s\n' "$codigo" "$descripcion" >&2
    rm -f /tmp/o404e-fallo.$$
    return 1
  fi
  if ! rg -q "ERROR:  ${codigo}:" /tmp/o404e-fallo.$$; then
    printf 'SQLSTATE inesperado para %s:\n' "$descripcion" >&2
    sed -n '1,12p' /tmp/o404e-fallo.$$ >&2
    rm -f /tmp/o404e-fallo.$$
    return 1
  fi
  rm -f /tmp/o404e-fallo.$$
}
paso "arranque desde cero con $imagen"
docker volume create "$volumen" >/dev/null
docker run --detach --rm --name "$contenedor" --network none \
  --env POSTGRES_PASSWORD="$clave" \
  --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
  --mount "type=volume,source=$volumen,target=/var/lib/postgresql" \
  "$imagen" >/dev/null
for _ in {1..60}; do
  docker exec "$contenedor" pg_isready -q -U postgres -d postgres &&
    break
  sleep 1
done
docker cp "$raiz/deploy/postgresql/." "$contenedor:/repo"
psql_admin --command \
  'REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC' >/dev/null
psql_admin --command 'REVOKE ALL ON SCHEMA public FROM PUBLIC' >/dev/null

paso 'delta de rol falla limpio fuera de una instalación CT'
psql_admin --command 'CREATE DATABASE o404e_sin_ct' >/dev/null
esperar_fallo 'delta de rol en base sin CT' \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    -U postgres -d o404e_sin_ct -f \
    /repo/contratacion_temporal/roles_confirmador_cobertura_up.sql
esperar_fallo 'delta de lector O4-05 en base sin CT' \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    -U postgres -d o404e_sin_ct -f \
    /repo/contratacion_temporal/roles_lector_resultado_cobertura_up.sql
psql_admin <<'SQL' >/dev/null
DO $$
BEGIN
  IF EXISTS(SELECT 1 FROM pg_roles
             WHERE rolname='vec_contratacion_temporal_confirmador_cobertura')
  THEN RAISE EXCEPTION 'delta fuera de orden dejó rol residual'; END IF;
  IF EXISTS(SELECT 1 FROM pg_roles
             WHERE rolname=
               'vec_contratacion_temporal_lector_resultado_cobertura')
  THEN RAISE EXCEPTION 'delta lector fuera de orden dejó rol residual'; END IF;
END
$$;
DROP DATABASE o404e_sin_ct;
SQL

paso 'dependencias reales de ContextoActor y Autorización'
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
    paso 'fixture y resolución sintética ContextoActor antes de otros esquemas'
    archivo contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql
    archivo autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
    psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o404e_contexto LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO vec_o404e_contexto
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
    docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
      -U vec_o404e_contexto -d postgres <<'SQL' >/dev/null
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
REVOKE vec_contexto_actor_v1_runtime FROM vec_o404e_contexto;
DROP ROLE vec_o404e_contexto;
SQL
  fi
done

paso 'fixture sintético de Autorización'
archivo autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql

paso 'wrapper exacto de Autorización para instalación fresca'
archivo contratacion_temporal/migraciones_autorizacion/000006_wrapper_contexto_exacto_cobertura_o4_04e.up.sql

paso 'migraciones CT 000001–000034'
mapfile -t migraciones < <(
  find "$raiz/deploy/postgresql/contratacion_temporal/migraciones" \
    -maxdepth 1 -type f -name '*.up.sql' -printf '%f\n' | sort
)
for nombre in "${migraciones[@]}"; do
  # O4-05 tiene rol y pruebas propios; se aplica después de cerrar O4-04E.
  if [[ $nombre > 000034_lector_fuerte_acl_cobertura_o4_04e.up.sql ]]; then
    continue
  fi
  if [[ $nombre == 000003_* ]]; then
    paso 'dependencia real de Autorización Atestada V3'
    psql_admin <<'SQL' >/dev/null
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
    archivo autorizacion_atestada_v3/roles_up.sql
    archivo autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql
    archivo autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql
  fi
  paso "aplicar $nombre"
  archivo "contratacion_temporal/migraciones/$nombre"
done

psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o404e_tcb LOGIN INHERIT;
CREATE ROLE vec_o404e_runtime LOGIN INHERIT;
CREATE ROLE vec_o404e_conflictivo LOGIN INHERIT;
GRANT vec_contratacion_temporal_confirmador_cobertura
 TO vec_o404e_tcb WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
GRANT vec_contratacion_temporal_ejecutor
 TO vec_o404e_runtime WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
GRANT vec_contratacion_temporal_confirmador_cobertura,
      vec_contratacion_temporal_migrador
 TO vec_o404e_conflictivo WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL

paso 'matriz ACL/RLS y primario fuerte'
archivo contratacion_temporal/pruebas_sql/o404e_acl_barrera.sql
esperar_sqlstate 'runtime genérico invoca confirmación' 42501 \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose \
    -U vec_o404e_runtime -d postgres -c \
    "SELECT * FROM vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1('{}')"
esperar_sqlstate 'credencial con confirmador y migrador' 42501 \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose \
    -U vec_o404e_conflictivo -d postgres -c \
    "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'; SELECT * FROM vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1('{}')"
psql_admin <<'SQL' >/dev/null
SET SESSION AUTHORIZATION vec_o404e_tcb;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT resultado_json
FROM vec_contratacion_temporal
 .leer_terminal_primario_decision_cobertura_o404e_v1(
  jsonb_build_object(
   'esquema',
    'vec.contratacion-temporal.consulta-primaria-decision-cobertura.o4-04e.v1',
   'organizacion_ref','organizacion:ausente',
   'expediente_ref','expediente:ausente','version_expediente',2,
   'reserva_ref','reserva:ausente','recibo_ref','recibo:ausente',
   'correlacion_vec_ref','correlacion:ausente',
   'decision_vec_ref','decision:ausente','revision_cercado',1,
   'huella_orden_sha256',repeat('a',64)
  )
)
\gset
SELECT :'resultado_json'::jsonb->>'encontrado'='false' AS ausente
\gset
\if :ausente
\else
  SELECT 1/0;
\endif
COMMIT;
RESET SESSION AUTHORIZATION;
SQL

paso 'ciclo O4-05 reversible sin historia terminal'
archivo contratacion_temporal/roles_lector_resultado_cobertura_up.sql
archivo contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.up.sql
archivo contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.down.sql
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o405_lector_ciclo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_lector_resultado_cobertura
 TO vec_o405_lector_ciclo WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
esperar_sqlstate 'rol lector no baja con LOGIN miembro vigente' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/roles_lector_resultado_cobertura_down.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_contratacion_temporal_lector_resultado_cobertura
 FROM vec_o405_lector_ciclo;
DROP ROLE vec_o405_lector_ciclo;
SQL
archivo contratacion_temporal/roles_lector_resultado_cobertura_down.sql

paso 'historia O4-D previa no bloquea down del wrapper 000006'
psql_admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL session_replication_role='replica';
INSERT INTO vec_autorizacion.enlace_decision_cobertura_ct_o404e(
 decision_ref,rama,concedida,codigo,accion,decision_huella_sha256,
 decision_concedida_ref,decision_denegada_ref,correlacion_ref,
 organizacion_ref,expediente_ref,version_expediente,reserva_ref,
 contexto_recurso_huella_sha256,huella_orden_sha256,
 lote_huella_sha256,prueba_vinculo_sha256,registrada_en,
 revalidada_en,vinculada_en
) VALUES(
 'decision:o404d:historica','denegada',false,'accion_no_concedida',
 'contratacion_temporal.cobertura.decidir',repeat('1',64),NULL,
 'decision:o404d:historica','correlacion:o404d:historica',
 'organizacion:o404d','expediente:o404d',2,'reserva:o404d',
 repeat('2',64),repeat('3',64),NULL,repeat('4',64),
 '2026-01-01 00:00:00+00',NULL,'2026-01-01 00:00:00+00');
COMMIT;
SQL

paso 'ciclo down completo 000034→000028→auth 000006→rol delta'
for numero in 28 29 30 31 32 33; do
  barrera=$((numero-20))
  nombre="$(basename "$raiz"/deploy/postgresql/contratacion_temporal/migraciones/0000${numero}_*.down.sql)"
  psql_admin --command \
    "UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=$barrera WHERE control AND version_esquema=14" \
    >/dev/null
  esperar_sqlstate \
    "down 0000$numero con capas superiores del stack 14" 55000 \
    psql_admin --set VERBOSITY=verbose \
    --file "/repo/contratacion_temporal/migraciones/$nombre"
  psql_admin --command \
    "UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=14 WHERE control AND version_esquema=$barrera" \
    >/dev/null
done
esperar_sqlstate 'down auth 000006 con stack CT 14' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/migraciones_autorizacion/000006_wrapper_contexto_exacto_cobertura_o4_04e.down.sql
esperar_sqlstate 'down del rol confirmador con stack CT 14' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/roles_confirmador_cobertura_down.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_contratacion_temporal_confirmador_cobertura
 FROM vec_o404e_tcb,vec_o404e_conflictivo;
REVOKE vec_contratacion_temporal_migrador FROM vec_o404e_conflictivo;
DROP ROLE vec_o404e_tcb,vec_o404e_runtime,vec_o404e_conflictivo;
SQL
for numero in 34 33 32 31 30 29 28; do
  nombre="$(basename "$raiz"/deploy/postgresql/contratacion_temporal/migraciones/0000${numero}_*.down.sql)"
  if [[ $numero == 33 ]]; then
    psql_admin <<'SQL' >/dev/null
GRANT USAGE ON SCHEMA vec_contratacion_temporal
  TO vec_contratacion_temporal_confirmador_cobertura;
GRANT EXECUTE ON FUNCTION
  vec_contratacion_temporal
    .confirmar_operacion_decision_cobertura_o404e_v1(jsonb)
  TO vec_contratacion_temporal_confirmador_cobertura;
SQL
    esperar_sqlstate 'down 000033 con ACL residual de 000034' 55000 \
      psql_admin --set VERBOSITY=verbose \
      --file "/repo/contratacion_temporal/migraciones/$nombre"
    psql_admin <<'SQL' >/dev/null
REVOKE EXECUTE ON FUNCTION
  vec_contratacion_temporal
    .confirmar_operacion_decision_cobertura_o404e_v1(jsonb)
  FROM vec_contratacion_temporal_confirmador_cobertura;
REVOKE USAGE ON SCHEMA vec_contratacion_temporal
  FROM vec_contratacion_temporal_confirmador_cobertura;
SQL
  fi
  if [[ $numero == 32 || $numero == 28 ]]; then
    barrera=$((numero-20))
    psql_admin --command \
      "UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=99 WHERE control AND version_esquema=$barrera" \
      >/dev/null
    esperar_sqlstate "down 0000$numero con barrera adulterada" 55000 \
      psql_admin --set VERBOSITY=verbose \
      --file "/repo/contratacion_temporal/migraciones/$nombre"
    psql_admin --command \
      "UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=$barrera WHERE control AND version_esquema=99" \
      >/dev/null
  fi
  archivo "contratacion_temporal/migraciones/$nombre"
done
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=8 WHERE control AND version_esquema=7' \
  >/dev/null
esperar_sqlstate 'down auth 000006 con barrera CT incompatible' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/migraciones_autorizacion/000006_wrapper_contexto_exacto_cobertura_o4_04e.down.sql
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=7 WHERE control AND version_esquema=8' \
  >/dev/null
archivo contratacion_temporal/migraciones_autorizacion/000006_wrapper_contexto_exacto_cobertura_o4_04e.down.sql
psql_admin --command \
  'ALTER ROLE vec_contratacion_temporal_confirmador_cobertura LOGIN' \
  >/dev/null
esperar_sqlstate 'down del rol confirmador con atributos no mínimos' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/roles_confirmador_cobertura_down.sql
psql_admin --command \
  'ALTER ROLE vec_contratacion_temporal_confirmador_cobertura NOLOGIN' \
  >/dev/null
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o404e_grupo_down NOLOGIN;
GRANT vec_o404e_grupo_down
  TO vec_contratacion_temporal_confirmador_cobertura;
SQL
esperar_sqlstate 'down del rol confirmador como miembro saliente' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/roles_confirmador_cobertura_down.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_o404e_grupo_down
  FROM vec_contratacion_temporal_confirmador_cobertura;
DROP ROLE vec_o404e_grupo_down;
SQL
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o404e_miembro_down LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_confirmador_cobertura
  TO vec_o404e_miembro_down;
SQL
esperar_sqlstate 'down del rol confirmador con miembro vigente' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/roles_confirmador_cobertura_down.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_contratacion_temporal_confirmador_cobertura
  FROM vec_o404e_miembro_down;
DROP ROLE vec_o404e_miembro_down;
SQL
archivo contratacion_temporal/roles_confirmador_cobertura_down.sql
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=99 WHERE control AND version_esquema=7' \
  >/dev/null
esperar_sqlstate \
  'down del rol ausente no omite la validación de barrera' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/roles_confirmador_cobertura_down.sql
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=7 WHERE control AND version_esquema=99' \
  >/dev/null
archivo contratacion_temporal/roles_confirmador_cobertura_down.sql

paso 'upgrade 000027: rol delta→auth 000006→CT 000028–000034'
archivo contratacion_temporal/roles_confirmador_cobertura_up.sql
archivo contratacion_temporal/migraciones_autorizacion/000006_wrapper_contexto_exacto_cobertura_o4_04e.up.sql
for numero in 28 29 30 31 32 33 34; do
  nombre="$(basename "$raiz"/deploy/postgresql/contratacion_temporal/migraciones/0000${numero}_*.up.sql)"
  archivo "contratacion_temporal/migraciones/$nombre"
done
archivo contratacion_temporal/pruebas_sql/o404e_canones_cruzados.sql
archivo contratacion_temporal/pruebas_sql/o404e_acl_barrera.sql
psql_admin <<'SQL' >/dev/null
DO $$
BEGIN
 IF NOT EXISTS(
   SELECT 1 FROM vec_autorizacion.enlace_decision_cobertura_ct_o404e
    WHERE decision_ref='decision:o404d:historica'
 ) THEN RAISE EXCEPTION 'upgrade perdió historia O4-D'; END IF;
END
$$;
SQL

paso 'credencial nominativa mínima y golden denegado/replay'
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o404e_tcb LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_confirmador_cobertura TO vec_o404e_tcb
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
archivo contratacion_temporal/roles_confirmador_cobertura_up.sql
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o404e_grupo NOLOGIN;
GRANT vec_contratacion_temporal_confirmador_cobertura
 TO vec_o404e_grupo WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
esperar_fallo 'miembro entrante NOLOGIN del confirmador' \
  archivo contratacion_temporal/roles_confirmador_cobertura_up.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_contratacion_temporal_confirmador_cobertura
 FROM vec_o404e_grupo;
GRANT vec_o404e_grupo
 TO vec_contratacion_temporal_confirmador_cobertura
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
esperar_fallo 'confirmador miembro saliente de otro grupo' \
  archivo contratacion_temporal/roles_confirmador_cobertura_up.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_o404e_grupo
 FROM vec_contratacion_temporal_confirmador_cobertura;
DROP ROLE vec_o404e_grupo;
GRANT vec_contratacion_temporal_confirmador_cobertura TO vec_o404e_tcb
 WITH ADMIN TRUE,INHERIT TRUE,SET TRUE;
SQL
esperar_fallo 'membresía confirmadora delegable o con SET' \
  archivo contratacion_temporal/roles_confirmador_cobertura_up.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_contratacion_temporal_confirmador_cobertura FROM vec_o404e_tcb;
GRANT vec_contratacion_temporal_confirmador_cobertura TO vec_o404e_tcb
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
psql_admin --set o404e_solo_preparar=1 \
  --file /repo/contratacion_temporal/pruebas_sql/o404e_denegacion_replay.sql \
  >/dev/null

psql_admin <<'SQL' >/dev/null
CREATE SCHEMA vec_o404e_fallos;
REVOKE ALL ON SCHEMA vec_o404e_fallos FROM PUBLIC;
CREATE FUNCTION vec_o404e_fallos.despues_escritura()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
AS $$
BEGIN
  RAISE EXCEPTION USING ERRCODE='Z04E1',
    MESSAGE='fallo inyectado O4-04E después de escritura';
END
$$;
REVOKE ALL ON FUNCTION vec_o404e_fallos.despues_escritura() FROM PUBLIC;
SQL

source "$raiz/deploy/postgresql/contratacion_temporal/pruebas_o4_04e_failpoints_carreras.sh"

paso 'recuperación propia O4-05: rol, ACL, alias y recibo fuerte'
archivo contratacion_temporal/roles_lector_resultado_cobertura_up.sql
archivo contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.up.sql
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o405_lector LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_lector_resultado_cobertura
 TO vec_o405_lector WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
archivo contratacion_temporal/pruebas_sql/o405_recuperacion_propia.sql
esperar_sqlstate 'confirmador no puede recuperar por la función O4-05' 42501 \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose -U vec_o404e_tcb -d postgres -c \
    "BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'; SELECT * FROM vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1('{}')"
esperar_sqlstate 'rol lector no baja antes que la migración O4-05' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/roles_lector_resultado_cobertura_down.sql
esperar_sqlstate 'down 000035 con historia terminal O4-04E' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.down.sql
esperar_sqlstate 'down 000034 bajo la capa O4-05' 55000 \
  psql_admin --set VERBOSITY=verbose \
  --file /repo/contratacion_temporal/migraciones/000034_lector_fuerte_acl_cobertura_o4_04e.down.sql
