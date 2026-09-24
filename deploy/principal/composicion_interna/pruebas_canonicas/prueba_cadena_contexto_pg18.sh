#!/usr/bin/env bash
# Ensayo desechable de las dos cadenas canónicas relacionadas con composición.
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
contenedor="vec-comp-v3-canon-pg18-$$"
limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT

docker run -d --name "$contenedor" --network none \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  postgres:18.4 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 0.5
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres

sql() {
  docker exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 -q -U postgres -d postgres
}
archivo() {
  local ruta=$1
  printf '%s\n' "INSTALANDO $ruta"
  sql < "$raiz/$ruta"
}
archivo_como() {
  local usuario=$1 ruta=$2
  printf '%s\n' "INSTALANDO COMO $usuario $ruta"
  docker exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 -q \
    -U "$usuario" -d postgres < "$raiz/$ruta"
}

sql <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

base=deploy/postgresql/contexto_actor_v1
archivo "$base/roles_up.sql"
archivo "$base/migraciones/000001_contexto_actor_v1.up.sql"
archivo "$base/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
if [[ "${1:-contexto}" == ad3 ]]; then
  archivo "$base/pruebas_sql/fixtures_sinteticos.sql"
  archivo deploy/postgresql/autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
  sql <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
  archivo deploy/postgresql/autorizacion/roles_up.sql
  archivo deploy/postgresql/autorizacion/roles_v2_up.sql
  archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
  archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
  archivo deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql
  archivo deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql
  archivo deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql
  archivo deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql
  archivo deploy/postgresql/autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql
  archivo deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql
  archivo deploy/postgresql/contratacion_temporal/roles_up.sql
  archivo deploy/postgresql/autorizacion_atestada_v3/roles_up.sql
  sql <<'SQL'
CREATE ROLE vec_ad3_canon_migrador LOGIN NOINHERIT;
GRANT CONNECT ON DATABASE postgres TO vec_ad3_canon_migrador;
GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_canon_migrador
  WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL
  ad3=deploy/postgresql/autorizacion_atestada_v3/migraciones
  archivo_como vec_ad3_canon_migrador "$ad3/000001_gobierno_y_registro_v3.up.sql"
  archivo_como vec_ad3_canon_migrador "$ad3/000002_consumidor_capacidad_v3.up.sql"
  archivo "$ad3/000050a_preflight_material_interno.up.sql"
  archivo "$ad3/000053_lectura_configuracion_interna.up.sql"
  sql <<'SQL'
CREATE ROLE vec_interno_preflight_v3_desarrollo LOGIN NOSUPERUSER
  NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_interno
  TO vec_interno_preflight_v3_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SELECT current_setting('server_version_num') AS pg_version,
       pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)') IS NOT NULL AS ad3_53,
       pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_preflight_interno',
         'vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)','EXECUTE') AS lector_tiene_execute,
       pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_consumidor',
         'vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)','EXECUTE') AS consumidor_tiene_execute;
SQL
  sql <<'SQL'
DO $acl$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_class c
     WHERE c.oid = ANY (ARRAY[
       'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass,
       'vec_autorizacion_atestada_v3.configuracion_confianza_version'::regclass,
       'vec_autorizacion_atestada_v3.raiz_confianza_version'::regclass])
       AND (NOT c.relrowsecurity OR NOT c.relforcerowsecurity)
  ) OR pg_catalog.has_table_privilege(
       'vec_interno_preflight_v3_desarrollo',
       'vec_autorizacion_atestada_v3.clave_capacidad_version','SELECT')
    OR pg_catalog.has_table_privilege(
       'vec_interno_preflight_v3_desarrollo',
       'vec_autorizacion_atestada_v3.configuracion_confianza_version','SELECT')
  THEN RAISE EXCEPTION 'RLS o ACL directas AD3 no acreditadas'; END IF;
END $acl$;
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
DO $denegacion$
BEGIN
  BEGIN
    PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v1('{}'::jsonb);
    RAISE EXCEPTION 'material vacío aceptado';
  EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
  END;
END $denegacion$;
RESET SESSION AUTHORIZATION;
SQL
  printf '%s\n' 'REINICIANDO CONTENEDOR PROPIO'
  docker restart "$contenedor" >/dev/null
  for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
    sleep 0.5
  done
  sql <<'SQL'
DO $durable$
BEGIN
  IF pg_catalog.to_regprocedure(
    'vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)') IS NULL
     OR NOT pg_catalog.has_function_privilege(
       'vec_autorizacion_atestada_v3_preflight_interno',
       'vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)',
       'EXECUTE')
  THEN RAISE EXCEPTION 'AD3-53 no persistió tras reinicio'; END IF;
END $durable$;
SQL
  printf '%s\n' 'CADENA AD3 000001, 000002, 000050a, 000053, ACL/RLS Y REINICIO PROBADOS'
  exit 0
fi
archivo "$base/migraciones/000003_organizacion_corporativa_v1.up.sql"
archivo "$base/roles_contexto_corporativo_rrhh_selector_v1_up.sql"
archivo "$base/migraciones/000004_vinculo_corporativo_rrhh_v1.up.sql"
printf '%s\n' 'CADENA CONTEXTO 000004 INSTALADA'
