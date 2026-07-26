#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ct-o405-registro-pg18-${PPID}-${RANDOM}"
volumen="${contenedor}-datos"
archivo_clave="$(mktemp "${TMPDIR:-/tmp}/vec-o405-registro-clave.XXXXXX")"
clave="$(openssl rand -hex 24)"
archivos_temporales=("$archivo_clave")

limpiar() {
  local temporal
  for temporal in "${archivos_temporales[@]}"; do
    rm -f -- "$temporal"
  done
  if [[ ${VEC_MANTENER_CONTENEDOR_FALLIDO:-0} == 1 ]]; then
    printf 'contenedor conservado: %s\n' "$contenedor" >&2
    return
  fi
  docker rm --force --volumes "$contenedor" >/dev/null 2>&1 || true
  docker volume rm --force "$volumen" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

chmod 0600 "$archivo_clave"
printf '%s' "$clave" >"$archivo_clave"
unset clave

paso() {
  printf '[O4-05:C2-B:PG18.4] %s\n' "$1"
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
  local salida
  shift
  salida="$(mktemp "${TMPDIR:-/tmp}/vec-o405-registro-fallo.XXXXXX")"
  archivos_temporales+=("$salida")
  if "$@" >"$salida" 2>&1; then
    printf 'se esperaba rechazo: %s\n' "$descripcion" >&2
    return 1
  fi
}

esperar_sqlstate() {
  local descripcion=$1
  local codigo=$2
  local salida
  shift 2
  salida="$(mktemp "${TMPDIR:-/tmp}/vec-o405-registro-estado.XXXXXX")"
  archivos_temporales+=("$salida")
  if "$@" >"$salida" 2>&1; then
    printf 'se esperaba SQLSTATE %s: %s\n' "$codigo" "$descripcion" >&2
    return 1
  fi
  if ! rg -q "ERROR:  ${codigo}:" "$salida"; then
    printf 'SQLSTATE inesperado para %s:\n' "$descripcion" >&2
    sed -n '1,16p' "$salida" >&2
    return 1
  fi
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
psql_admin --command \
  'REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC' >/dev/null
psql_admin --command 'REVOKE ALL ON SCHEMA public FROM PUBLIC' >/dev/null

paso 'delta de rol falla sin instalación CT'
psql_admin --command 'CREATE DATABASE o405_registro_sin_ct' >/dev/null
esperar_fallo 'consultor RRHH fuera de una instalación CT' \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    -U postgres -d o405_registro_sin_ct \
    -f /repo/contratacion_temporal/roles_consultor_rrhh_up.sql
psql_admin <<'SQL' >/dev/null
DO $$
BEGIN
  IF EXISTS(
    SELECT 1 FROM pg_roles
     WHERE rolname='vec_contratacion_temporal_consultor_rrhh'
  ) THEN
    RAISE EXCEPTION 'el delta fallido dejó rol residual';
  END IF;
END
$$;
DROP DATABASE o405_registro_sin_ct;
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
  archivo "$ruta"
  if [[ $ruta == contexto_actor_v1/migraciones/000002_* ]]; then
    archivo contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql
    archivo autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
    psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o405_contexto LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO vec_o405_contexto
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
    docker exec --interactive "$contenedor" psql -X \
      --set ON_ERROR_STOP=1 -U vec_o405_contexto -d postgres \
      <<'SQL' >/dev/null
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

paso 'migraciones CT 000001-000034'
for numero in $(seq -w 1 34); do
  nombre="$(basename \
    "$raiz"/deploy/postgresql/contratacion_temporal/migraciones/0000"${numero}"_*.up.sql)"
  if [[ $numero == 03 ]]; then
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
  archivo "contratacion_temporal/migraciones/$nombre"
done
archivo contratacion_temporal/roles_lector_resultado_cobertura_up.sql
archivo \
  contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.up.sql

paso 'ciclo limpio del rol y la migración C2-B'
archivo contratacion_temporal/roles_consultor_rrhh_up.sql
archivo \
  contratacion_temporal/migraciones/000036_registro_accesos_rrhh_o4_05.up.sql
archivo \
  contratacion_temporal/migraciones/000036_registro_accesos_rrhh_o4_05.down.sql
archivo contratacion_temporal/roles_consultor_rrhh_down.sql
archivo contratacion_temporal/roles_consultor_rrhh_up.sql
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o405_consultor_pre LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_o405_puente_pre LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_consultor_rrhh
 TO vec_o405_consultor_pre WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
GRANT vec_o405_consultor_pre
 TO vec_o405_puente_pre WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
esperar_fallo 'migración con herencia transitiva vía LOGIN' \
  archivo \
  contratacion_temporal/migraciones/000036_registro_accesos_rrhh_o4_05.up.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_o405_consultor_pre FROM vec_o405_puente_pre;
REVOKE vec_contratacion_temporal_consultor_rrhh
 FROM vec_o405_consultor_pre;
DROP ROLE vec_o405_puente_pre,vec_o405_consultor_pre;
SQL
archivo \
  contratacion_temporal/migraciones/000036_registro_accesos_rrhh_o4_05.up.sql
esperar_sqlstate 'down 000035 bajo C2-B sin historia' 55000 \
  psql_admin --set VERBOSITY=verbose \
    --file \
    /repo/contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.down.sql

paso 'topología nominativa, ACL y ataques de membresía'
psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o405_consultor LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_o405_login_puente LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_consultor_rrhh
 TO vec_o405_consultor WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
archivo contratacion_temporal/roles_consultor_rrhh_up.sql
psql_admin <<'SQL' >/dev/null
GRANT vec_o405_consultor TO vec_o405_login_puente
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
esperar_fallo 'LOGIN consultor usado como puente transitivo' \
  archivo contratacion_temporal/roles_consultor_rrhh_up.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_o405_consultor FROM vec_o405_login_puente;
SQL
archivo contratacion_temporal/roles_consultor_rrhh_up.sql
esperar_sqlstate 'LOGIN consulta tabla directamente' 42501 \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose -U vec_o405_consultor -d postgres \
    -c 'SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh'
esperar_sqlstate 'LOGIN invoca función interna directamente' 42501 \
  docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose -U vec_o405_consultor -d postgres \
    -c \
    "SELECT vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1('{}')"
esperar_sqlstate 'down de rol bajo migración instalada' 55000 \
  psql_admin --set VERBOSITY=verbose \
    --file /repo/contratacion_temporal/roles_consultor_rrhh_down.sql
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=15 WHERE control AND version_esquema=16' \
  >/dev/null
esperar_fallo 'rol sobre barreras 15 más C2-B incoherentes' \
  archivo contratacion_temporal/roles_consultor_rrhh_up.sql
psql_admin --command \
  'UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4 SET version_esquema=16 WHERE control AND version_esquema=15' \
  >/dev/null
archivo contratacion_temporal/roles_consultor_rrhh_up.sql

psql_admin <<'SQL' >/dev/null
CREATE ROLE vec_o405_grupo_ajeno NOLOGIN;
GRANT vec_o405_grupo_ajeno TO vec_o405_consultor
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
esperar_fallo 'LOGIN con segunda membresía directa' \
  archivo contratacion_temporal/roles_consultor_rrhh_up.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_o405_grupo_ajeno FROM vec_o405_consultor;
REVOKE vec_contratacion_temporal_consultor_rrhh FROM vec_o405_consultor;
GRANT vec_contratacion_temporal_consultor_rrhh TO vec_o405_consultor
 WITH ADMIN FALSE,INHERIT TRUE,SET TRUE;
SQL
esperar_fallo 'membresía consultora con SET ROLE' \
  archivo contratacion_temporal/roles_consultor_rrhh_up.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_contratacion_temporal_consultor_rrhh FROM vec_o405_consultor;
GRANT vec_contratacion_temporal_consultor_rrhh TO vec_o405_consultor
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
GRANT vec_o405_grupo_ajeno TO vec_contratacion_temporal_consultor_rrhh
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
esperar_fallo 'grupo consultor miembro de otro grupo' \
  archivo contratacion_temporal/roles_consultor_rrhh_up.sql
psql_admin <<'SQL' >/dev/null
REVOKE vec_o405_grupo_ajeno
 FROM vec_contratacion_temporal_consultor_rrhh;
DROP ROLE vec_o405_grupo_ajeno;
SQL
archivo contratacion_temporal/roles_consultor_rrhh_up.sql

paso 'estructura, RLS, cadena, límites e inmutabilidad'
archivo contratacion_temporal/pruebas_sql/o405_registro_accesos_rrhh.sql

paso 'ocho escritores concurrentes con reintento explícito de prueba'
ejecutar_con_reintento() {
  local indice=$1
  for _ in {1..12}; do
    if docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
      -U postgres -d postgres --set "indice=$indice" \
      --file \
      /repo/contratacion_temporal/pruebas_sql/o405_registrar_acceso_concurrente.sql \
      >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.05
  done
  printf 'escritor concurrente %s no confirmó\n' "$indice" >&2
  return 1
}
procesos=()
for indice in $(seq 1001 1008); do
  ejecutar_con_reintento "$indice" &
  procesos+=("$!")
done
for proceso in "${procesos[@]}"; do
  wait "$proceso"
done

psql_admin <<'SQL' >/dev/null
SET ROLE vec_contratacion_temporal_propietario;
DO $concurrencia$
BEGIN
  IF (SELECT count(*) FROM
        vec_contratacion_temporal.registro_acceso_rrhh) <> 10
     OR (SELECT min(secuencia) FROM
          vec_contratacion_temporal.registro_acceso_rrhh) <> 1
     OR (SELECT max(secuencia) FROM
          vec_contratacion_temporal.registro_acceso_rrhh) <> 10
     OR (SELECT count(DISTINCT secuencia) FROM
          vec_contratacion_temporal.registro_acceso_rrhh) <> 10
     OR EXISTS(
       SELECT 1
         FROM (
           SELECT registro.*,
                  lag(huella_sha256,1,repeat('0',64))
                    OVER(ORDER BY secuencia) AS anterior_esperado
             FROM vec_contratacion_temporal.registro_acceso_rrhh registro
         ) ordenado
        WHERE anterior_sha256<>anterior_esperado
           OR huella_sha256<>encode(
             sha256(decode(anterior_sha256,'hex')||prueba_canonica),'hex')
     ) OR NOT EXISTS(
       SELECT 1
         FROM vec_contratacion_temporal.control_cadena_accesos_rrhh c
         JOIN vec_contratacion_temporal.registro_acceso_rrhh r
           ON r.secuencia=c.ultima_secuencia
          AND r.huella_sha256=c.cabeza_sha256
        WHERE c.control AND c.ultima_secuencia=10
     ) THEN
    RAISE EXCEPTION 'cadena concurrente no es íntegra y gapless';
  END IF;
END
$concurrencia$;
RESET ROLE;
SQL

paso 'reversión protegida por historia'
esperar_sqlstate 'down 000035 bajo C2-B con historia' 55000 \
  psql_admin --set VERBOSITY=verbose \
    --file \
    /repo/contratacion_temporal/migraciones/000035_recuperacion_propia_cobertura_o4_05.down.sql
esperar_sqlstate 'down con accesos durables' 55000 \
  psql_admin --set VERBOSITY=verbose \
    --file \
    /repo/contratacion_temporal/migraciones/000036_registro_accesos_rrhh_o4_05.down.sql

paso 'resultado verde'
printf 'O4-05 C2-B PostgreSQL 18.4: GO técnico\n'
