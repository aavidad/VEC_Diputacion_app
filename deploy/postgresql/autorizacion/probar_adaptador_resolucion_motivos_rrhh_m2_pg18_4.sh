#!/usr/bin/env bash
set -Eeuo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd -P)
cd "$raiz"

imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-m23-adaptador-motivos-${USER:-usuario}-$$"
base=vec_m23_adaptador_motivos
resolutor=vec_m23_resolutor
proyector=vec_m23_proyector
temporales=$(mktemp -d "${TMPDIR:-/tmp}/vec-m23-adaptador.XXXXXX")
socket="$temporales/socket"
barrera="$temporales/barrera"
secreto="$temporales/bootstrap"
binario="$temporales/postgres.test"
salida_reinicio="$temporales/reinicio.log"
pid_prueba=''
fase='arranque'

limpiar() {
  if [[ -n $pid_prueba ]]; then
    kill "$pid_prueba" >/dev/null 2>&1 || true
    wait "$pid_prueba" >/dev/null 2>&1 || true
  fi
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf "$temporales"
}
informar_fallo() {
  echo "M2.3 falló durante: $fase" >&2
}
trap informar_fallo ERR
trap limpiar EXIT INT TERM

mkdir -m 0777 "$socket"
mkdir -m 0700 "$barrera"
umask 077
od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]' >"$secreto"

docker run --detach --network none --name "$contenedor" \
  --env POSTGRES_DB="$base" \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/bootstrap \
  --mount "type=bind,src=$secreto,dst=/run/secrets/bootstrap,readonly" \
  --mount "type=bind,src=$socket,dst=/socket-m23" \
  "$imagen" postgres \
  -c unix_socket_directories=/var/run/postgresql,/socket-m23 >/dev/null

esperar_postgresql() {
  local disponible=false
  for _ in $(seq 1 120); do
    if docker exec "$contenedor" psql -XAtq -h /socket-m23 \
        -U postgres -d "$base" \
        -c 'SELECT 1' >/dev/null 2>&1; then
      disponible=true
      break
    fi
    sleep 0.1
  done
  if [[ $disponible != true ]]; then
    docker logs "$contenedor" >&2 || true
    echo 'PostgreSQL 18.4 efímero M2.3 no quedó disponible' >&2
    return 1
  fi
  [[ $(docker exec "$contenedor" psql -XAtq -h /socket-m23 \
    -U postgres -d "$base" \
    -c "SELECT current_setting('server_version_num')") == 180004 ]]
}

psql_archivo() {
  docker exec --interactive "$contenedor" psql -Xq -h /socket-m23 \
    --set ON_ERROR_STOP=1 -U postgres -d "$base" <"$raiz/$1"
}

psql_admin() {
  docker exec --interactive "$contenedor" psql -Xq -h /socket-m23 \
    --set ON_ERROR_STOP=1 -U postgres -d "$base"
}

psql_valor() {
  docker exec "$contenedor" psql -XAtq -h /socket-m23 \
    --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" -c "$1"
}

psql_actor() {
  local actor=$1 consulta=$2
  docker exec "$contenedor" psql -XAtq -h /socket-m23 \
    --set ON_ERROR_STOP=1 \
    -U "$actor" -d "$base" -c "$consulta"
}

ejecutar_prueba() {
  local nombre=$1
  env \
    VEC_M23_PG18_AISLADO=1 \
    VEC_M23_RESOLUTOR_LOGIN="$resolutor" \
    VEC_M23_RESOLUTOR_DSN="$dsn_resolutor" \
    VEC_M23_RESOLUTOR_UNICO_DSN="$dsn_resolutor_unico" \
    VEC_M23_ADMIN_DSN="$dsn_admin" \
    VEC_M23_PROYECTOR_DSN="$dsn_proyector" \
    "$binario" -test.run "^${nombre}$" -test.count=1
}

esperar_postgresql

psql_admin <<'SQL'
DO $bloque$ BEGIN
  EXECUTE format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    current_database()
  );
END $bloque$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

for archivo in \
  deploy/postgresql/contexto_actor_v1/roles_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/roles_v2_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
  deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql \
  deploy/postgresql/autorizacion/migraciones/000008_vinculaciones_motivo_consultas_rrhh.up.sql \
  deploy/postgresql/autorizacion/migraciones/000009_publicacion_retirada_vinculaciones_motivo_consultas_rrhh.up.sql \
  deploy/postgresql/autorizacion/roles_motivos_rrhh_resolutor_v1_up.sql \
  deploy/postgresql/autorizacion/migraciones/000010_resolucion_vinculaciones_motivo_consultas_rrhh.up.sql
do
  psql_archivo "$archivo" >/dev/null
done

psql_admin <<SQL
CREATE ROLE $resolutor LOGIN PASSWORD NULL NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1;
CREATE ROLE $proyector LOGIN PASSWORD NULL NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1;
CREATE ROLE vec_m23_autoridad_extra NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_motivos_proyector TO $proyector
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

dsn_resolutor="host=$socket port=5432 dbname=$base user=$resolutor sslmode=disable connect_timeout=2"
dsn_resolutor_unico="$dsn_resolutor pool_max_conns=1"
dsn_admin="host=$socket port=5432 dbname=$base user=postgres sslmode=disable connect_timeout=2"
dsn_proyector="host=$socket port=5432 dbname=$base user=$proyector sslmode=disable connect_timeout=2"

go test -c -race -o "$binario" \
  ./internal/modules/contrataciontemporal/adapters/postgres

ejecutar_prueba \
  TestIntegracionResolutorMotivosRRHHPostgreSQLSinMembresia

psql_admin <<SQL
GRANT vec_autorizacion_motivos_rrhh_resolutor TO $resolutor
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

env \
  VEC_POSTGRES_TEST_MOTIVOS_RRHH_RESOLUTOR_DSN="$dsn_resolutor" \
  VEC_POSTGRES_TEST_MOTIVOS_RRHH_RESOLUTOR_LOGIN="$resolutor" \
  "$binario" \
  -test.run '^TestIntegracionPoolResolucionMotivosRRHHPostgreSQL$' \
  -test.count=1

ejecutar_prueba TestIntegracionResolutorMotivosRRHHPostgreSQLAusencia

fase='publicación del catálogo sintético inicial'
publicacion_inicial=$(psql_actor "$proyector" \
  "SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
    'evento_11111111111111111111111111111111',1,repeat('1',64),
    'motivos_rrhh_m23',1,repeat('2',64),
    clock_timestamp()-interval '31 days',
    jsonb_build_array(
      jsonb_build_object(
        'clave','motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        'vigente_desde',to_char(
          (clock_timestamp()-interval '30 days') AT TIME ZONE 'UTC',
          'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'
        ),
        'vigente_hasta',NULL
      ),
      jsonb_build_object(
        'clave','motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        'vigente_desde',to_char(
          (clock_timestamp()-interval '30 days') AT TIME ZONE 'UTC',
          'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'
        ),
        'vigente_hasta',NULL
      )
    )
  )")
if [[ $publicacion_inicial != t ]]; then
  psql_valor \
    "SELECT ultima_secuencia,
      vec_autorizacion.motivo_v2_entradas_validas(
        jsonb_build_array(
          jsonb_build_object(
            'clave','motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
            'vigente_desde',to_char(
              (clock_timestamp()-interval '30 days') AT TIME ZONE 'UTC',
              'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'
            ),
            'vigente_hasta',NULL
          ),
          jsonb_build_object(
            'clave','motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
            'vigente_desde',to_char(
              (clock_timestamp()-interval '30 days') AT TIME ZONE 'UTC',
              'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'
            ),
            'vigente_hasta',NULL
          )
        )
      )
    FROM vec_autorizacion.motivo_v2_checkpoint_origen" >&2
  echo 'no se pudo publicar el catálogo sintético inicial M2.3' >&2
  exit 1
fi

fase='publicación de la vinculación de cuadro'
[[ $(psql_actor "$proyector" \
  "SELECT vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(
    'evento_vinculacion_motivo_rrhh_11111111111111111111111111111111',
    repeat('3',64),1,
    'publicacion_motivo_rrhh_11111111111111111111111111111111',
    repeat('4',64),'motivos_rrhh_m23',1,repeat('2',64),
    'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    clock_timestamp()-interval '20 days'
  )") == t ]]

fase='publicación de la vinculación de detalle'
[[ $(psql_actor "$proyector" \
  "SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
    'evento_vinculacion_motivo_rrhh_22222222222222222222222222222222',
    repeat('5',64),1,
    'publicacion_motivo_rrhh_22222222222222222222222222222222',
    repeat('6',64),'motivos_rrhh_m23',1,repeat('2',64),
    'motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
    clock_timestamp()-interval '19 days'
  )") == t ]]

fase='resolución positiva y matriz de derivas'
ejecutar_prueba TestIntegracionResolutorMotivosRRHHPostgreSQLPositivo

rm -f "$barrera/listo" "$barrera/continuar"
env \
  VEC_M23_PG18_AISLADO=1 \
  VEC_M23_RESOLUTOR_LOGIN="$resolutor" \
  VEC_M23_RESOLUTOR_DSN="$dsn_resolutor" \
  VEC_M23_RESOLUTOR_UNICO_DSN="$dsn_resolutor_unico" \
  VEC_M23_ADMIN_DSN="$dsn_admin" \
  VEC_M23_PROYECTOR_DSN="$dsn_proyector" \
  VEC_M23_BARRERA="$barrera" \
  "$binario" \
  -test.run '^TestIntegracionResolutorMotivosRRHHPostgreSQLReinicioVivo$' \
  -test.count=1 >"$salida_reinicio" 2>&1 &
pid_prueba=$!

listo=false
for _ in $(seq 1 300); do
  if [[ -s $barrera/listo ]]; then
    listo=true
    break
  fi
  if ! kill -0 "$pid_prueba" >/dev/null 2>&1; then
    break
  fi
  sleep 0.05
done
if [[ $listo != true ]]; then
  sed -n '1,80p' "$salida_reinicio" >&2
  echo 'el test M2.3 no alcanzó la barrera de reinicio' >&2
  exit 1
fi

fase='reinicio vivo de PostgreSQL'
docker restart "$contenedor" >/dev/null
esperar_postgresql
touch "$barrera/continuar"
if ! wait "$pid_prueba"; then
  pid_prueba=''
  sed -n '1,120p' "$salida_reinicio" >&2
  exit 1
fi
pid_prueba=''

fase='retiradas y vigencias'
ejecutar_prueba TestIntegracionResolutorMotivosRRHHPostgreSQLRetiradas

fase='comprobación final de privilegios'
[[ $(psql_valor \
  "SELECT (NOT has_table_privilege(
    '$resolutor',
    'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1',
    'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
  ) AND NOT pg_has_role(
    '$resolutor','vec_autorizacion_propietario','SET'
  ))::text") == true ]]

echo 'OK: adaptador nominal de motivos RRHH M2 sobre PostgreSQL 18.4'
