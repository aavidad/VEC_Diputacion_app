#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct141-pg18-${PPID}-${RANDOM}"
volumen="${contenedor}-datos"
clave="$(openssl rand -hex 24)"
archivo_clave=
binario_go=
limpiar() {
  if [[ -n $binario_go ]]; then
    rm -f -- "$binario_go"
  fi
  if [[ -n $archivo_clave ]]; then
    rm -f -- "$archivo_clave"
  fi
  docker exec "$contenedor" rm -rf /run/vec-o405 >/dev/null 2>&1 || true
  if [[ ${VEC_MANTENER_CONTENEDOR_FALLIDO:-0} == 1 ]]; then
    printf 'contenedor conservado: %s\n' "$contenedor" >&2
    return
  fi
  docker rm --force --volumes "$contenedor" >/dev/null 2>&1 || true
  docker volume rm --force "$volumen" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM
archivo_clave="$(mktemp "${TMPDIR:-/tmp}/vec-ct141-clave.XXXXXX")"
chmod 0600 "$archivo_clave"
printf '%s' "$clave" >"$archivo_clave"
unset clave
paso() {
  printf '[CT141:PG18.4] %s\n' "$1"
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
  --env POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password \
  --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
  --mount \
  "type=bind,source=$archivo_clave,target=/run/secrets/postgres_password,readonly" \
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

paso 'preimagen V1 y huella antes del delta CT000141'
psql_admin <<'SQL' >/dev/null
CREATE SCHEMA vec_ct141_prueba;
CREATE TABLE vec_ct141_prueba.v1 AS
WITH p AS (
 SELECT jsonb_build_object(
  'canon',jsonb_build_object(
    'dominio','vec.dipgra.contratacion-temporal.catalogo-vias-cobertura',
    'version_esquema',1,'algoritmo','sha-256'),
  'referencia','catalogo:ct141:golden','version',1,
  'huella_sha256',repeat('a',64),
  'publicado_en','2026-07-23T09:00:00Z',
  'vigencia',jsonb_build_object(
    'desde','2026-08-01T00:00:00Z','hasta','2027-01-01T00:00:00Z'),
  'procedencia_ref','procedencia:ct141:golden',
  'vias',jsonb_build_array(jsonb_build_object(
    'clave','via_ct141','orden',1,
    'comprobaciones',jsonb_build_array(jsonb_build_object(
      'clave','comprobacion_ct141','orden',1,'obligatoria',true,
      'procedencia',jsonb_build_object(
        'clave','fuente_ct141',
        'definicion_fuente_ref','fuente:ct141:golden')))))) AS j
)
SELECT j, vec_contratacion_temporal.gobi_o404b_material_catalogo(j) AS m
FROM p;
ALTER SCHEMA vec_ct141_prueba OWNER TO vec_contratacion_temporal_propietario;
ALTER TABLE vec_ct141_prueba.v1 OWNER TO vec_contratacion_temporal_propietario;
DO $$BEGIN
 IF (SELECT m IS NULL FROM vec_ct141_prueba.v1) THEN
   RAISE EXCEPTION 'V1 original no acepta fixture'; END IF;
END$$;
SQL

paso 'instalar delta CT000141 en PostgreSQL 18 efímera'
archivo contratacion_temporal/migraciones/000141_requisitos_via_expediente.up.sql
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_ct141_gob LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
 INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_gobernador TO vec_ct141_gob
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
archivo contratacion_temporal/pruebas_sql/000141_requisitos_via_expediente_pg18.sql
paso 'DOWN/UP reversible antes de historia V2'
archivo contratacion_temporal/migraciones/000141_requisitos_via_expediente.down.sql
psql_admin <<'SQL' >/dev/null
DO $$ BEGIN
 IF pg_catalog.to_regprocedure(
   'vec_contratacion_temporal.gobi_o404b_material_catalogo_v1(jsonb)'
 ) IS NOT NULL
 OR pg_catalog.to_regprocedure(
   'vec_contratacion_temporal.gobi_o404b_material_catalogo_v2(jsonb)'
 ) IS NOT NULL
 OR (SELECT vec_contratacion_temporal.gobi_o404b_material_catalogo(j)
       IS DISTINCT FROM m FROM vec_ct141_prueba.v1 LIMIT 1) THEN
   RAISE EXCEPTION 'CT141 DOWN no restauró V1';
 END IF;
END $$;
SQL
archivo contratacion_temporal/migraciones/000141_requisitos_via_expediente.up.sql
paso 'ACL de ayudantes y RLS de catálogo'
esperar_sqlstate 'gobernador sin SELECT directo de catálogo' 42501 \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose -U vec_ct141_gob -d postgres \
    -c 'SELECT count(*) FROM vec_contratacion_temporal.gobi_o404b_catalogo'
esperar_sqlstate 'gobernador sin EXECUTE en material V2 interno' 42501 \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose -U vec_ct141_gob -d postgres \
    -c "SELECT vec_contratacion_temporal.gobi_o404b_material_catalogo_v2('{}'::jsonb)"

paso 'publicar V1 y V2 por rol nominal; replay exacto'
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
  -U vec_ct141_gob -d postgres \
  -c "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'; SELECT vec_ct141_prueba.publicar_v1(); COMMIT" >/dev/null
archivo contratacion_temporal/pruebas_sql/000141_requisitos_via_expediente_publicacion_pg18.sql
for esperado in publicada repetida; do
  resultado=$(docker exec "$contenedor" psql -X -At --set ON_ERROR_STOP=1 \
    -U vec_ct141_gob -d postgres \
    -c "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'; SELECT vec_ct141_prueba.publicar_v2(); COMMIT" | rg '^(publicada|repetida)$')
  if [[ $resultado != "$esperado" ]]; then
    printf 'replay CT141 esperado %s, observado %s\n' "$esperado" "$resultado" >&2
    exit 1
  fi
done
archivo contratacion_temporal/pruebas_sql/000141_requisitos_via_expediente_historia_pg18.sql
esperar_sqlstate 'DOWN con historia V2' 55000 \
  psql_admin --set VERBOSITY=verbose \
    --file /repo/contratacion_temporal/migraciones/000141_requisitos_via_expediente.down.sql
paso 'reinicio controlado de PostgreSQL efímero'
docker restart "$contenedor" >/dev/null
for _ in {1..60}; do
  docker exec "$contenedor" pg_isready -q -U postgres -d postgres && break
  sleep 1
done
archivo contratacion_temporal/pruebas_sql/000141_requisitos_via_expediente_reinicio_pg18.sql
paso 'CT000141 PG18: correcto'
